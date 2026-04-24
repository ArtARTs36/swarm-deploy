package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/swarm-deploy/swarm-deploy/internal/compose"
	"github.com/swarm-deploy/swarm-deploy/internal/config"
	"github.com/swarm-deploy/swarm-deploy/internal/controller/drift"
	"github.com/swarm-deploy/swarm-deploy/internal/deployer"
	"github.com/swarm-deploy/swarm-deploy/internal/event/dispatcher"
	"github.com/swarm-deploy/swarm-deploy/internal/event/events"
	gitx "github.com/swarm-deploy/swarm-deploy/internal/git"
	"github.com/swarm-deploy/swarm-deploy/internal/metrics"
	"github.com/swarm-deploy/swarm-deploy/internal/security"
)

type TriggerReason string

const (
	TriggerStartup TriggerReason = "startup"
	TriggerPoll    TriggerReason = "poll"
	TriggerWebhook TriggerReason = "webhook"
	TriggerManual  TriggerReason = "manual"
)

const eventShutdownTimeout = 5 * time.Second

type StackView struct {
	Name         string        `json:"name"`
	ComposeFile  string        `json:"compose_file"`
	LastStatus   string        `json:"last_status"`
	LastError    string        `json:"last_error,omitempty"`
	LastCommit   string        `json:"last_commit,omitempty"`
	LastDeployAt time.Time     `json:"last_deploy_at,omitempty"`
	SourceDigest string        `json:"source_digest,omitempty"`
	Services     []ServiceView `json:"services"`
}

type ServiceView struct {
	Name         string    `json:"name"`
	Image        string    `json:"image,omitempty"`
	ImageVersion string    `json:"image_version,omitempty"`
	LastStatus   string    `json:"last_status,omitempty"`
	LastDeployAt time.Time `json:"last_deploy_at,omitempty"`
}

type Controller struct {
	cfg      *config.Config
	git      gitx.Repository
	deployer *deployer.Deployer
	metrics  *metrics.Group
	event    dispatcher.Dispatcher

	stateStore      *runtimeStateStore
	stackReconciler *stackReconciler
	driftAnalyzer   *drift.Analyzer

	triggerCh chan triggerTask
}

type triggerTask struct {
	triggeredBy string
	reason      TriggerReason
}

func New(
	cfg *config.Config,
	git gitx.Repository,
	deployer *deployer.Deployer,
	metricGroup *metrics.Group,
	eventDispatcher dispatcher.Dispatcher,
	serviceReader drift.ServiceReader,
) *Controller {
	var driftAnalyzer *drift.Analyzer
	if serviceReader != nil {
		driftAnalyzer = drift.NewAnalyzer(serviceReader)
	}

	return &Controller{
		cfg:        cfg,
		git:        git,
		deployer:   deployer,
		metrics:    metricGroup,
		event:      eventDispatcher,
		stateStore: newRuntimeStateStore(),
		stackReconciler: newStackReconciler(
			cfg,
			git,
			deployer,
		),
		driftAnalyzer: driftAnalyzer,
		triggerCh:     make(chan triggerTask, 1),
	}
}

func (c *Controller) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.cfg.Spec.Sync.Interval.Value)

	slog.InfoContext(ctx, "[controller] trigger startup sync")

	c.trigger(triggerTask{
		reason: TriggerStartup,
	})

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), eventShutdownTimeout)
			if err := c.event.Shutdown(shutdownCtx); err != nil {
				slog.ErrorContext(
					context.Background(),
					"[controller] failed to shutdown event dispatcher",
					slog.Any("err", err),
				)
			}
			cancel()
			return nil
		case task := <-c.triggerCh:
			c.syncOnce(ctx, task)
		case <-tickerC(ticker):
			c.trigger(triggerTask{
				reason: TriggerPoll,
			})
		}
	}
}

func tickerC(t *time.Ticker) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

func (c *Controller) Manual(ctx context.Context) bool {
	user, _ := security.UserFromContext(ctx)

	return c.trigger(triggerTask{
		triggeredBy: user.Name,
		reason:      TriggerManual,
	})
}

func (c *Controller) Webhook() bool {
	return c.trigger(triggerTask{
		reason: TriggerWebhook,
	})
}

func (c *Controller) trigger(task triggerTask) bool {
	select {
	case c.triggerCh <- task:
		return true
	default:
		return false
	}
}

func (c *Controller) syncOnce(ctx context.Context, task triggerTask) { //nolint:funlen // not need
	startedAt := time.Now()

	slog.InfoContext(ctx, "[controller] run sync", slog.String("reason", string(task.reason)))
	if task.reason == TriggerManual {
		c.event.Dispatch(ctx, &events.SyncManualStarted{
			TriggeredBy: task.triggeredBy,
		})
	}

	syncResult, err := c.git.Pull(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "sync failed at git stage",
			slog.String("reason", string(task.reason)),
			slog.String("repository", c.cfg.Spec.Git.Repository),
			slog.Any("err", err),
		)
		c.metrics.Git.RecordGitUpdate(c.cfg.Spec.Git.Repository, "error")
		c.metrics.Sync.RecordSyncRun(string(task.reason), "error", time.Since(startedAt))
		c.updateState(func(s *runtimeState) {
			s.LastSyncAt = time.Now()
			s.LastSyncReason = string(task.reason)
			s.LastSyncResult = "error"
			s.LastSyncError = err.Error()
		})
		return
	}

	slog.InfoContext(ctx, "[controller] git synced", slog.Any("result", syncResult))

	updateResult := "no_change"
	if syncResult.Updated {
		updateResult = "updated"
	}
	c.metrics.Git.RecordGitUpdate(c.cfg.Spec.Git.Repository, updateResult)

	reloadedFrom, reloadErr := c.reloadStacks()
	if reloadErr != nil {
		slog.ErrorContext(ctx, "sync failed at stacks reload stage",
			slog.String("reason", string(task.reason)),
			slog.String("stacks.file", c.cfg.Spec.StacksSource.File),
			slog.Any("err", reloadErr),
		)
		c.metrics.Sync.RecordSyncRun(string(task.reason), "error", time.Since(startedAt))
		c.updateState(func(s *runtimeState) {
			s.LastSyncAt = time.Now()
			s.LastSyncReason = string(task.reason)
			s.LastSyncResult = "error"
			s.LastSyncError = reloadErr.Error()
			s.GitRevision = syncResult.NewRevision
		})
		return
	}

	slog.InfoContext(ctx, "[controller] stacks reloaded",
		slog.String("path", reloadedFrom),
		slog.Int("count", len(c.cfg.Spec.Stacks)),
	)

	var deployErrs []error
	for _, stackCfg := range c.cfg.Spec.Stacks {
		err = c.syncStack(ctx, stackCfg, syncResult.NewRevision)
		if err != nil {
			deployErrs = append(deployErrs, err)
			slog.ErrorContext(ctx, "sync failed for stack",
				slog.String("reason", string(task.reason)),
				slog.String("stack", stackCfg.Name),
				slog.String("commit", syncResult.NewRevision),
				slog.Any("err", err),
			)
		}
	}

	driftErrs := c.runPollDriftPass(ctx, syncResult.NewRevision)
	deployErrs = append(deployErrs, driftErrs...)
	for _, driftErr := range driftErrs {
		slog.ErrorContext(ctx, "poll drift handling failed",
			slog.String("reason", string(task.reason)),
			slog.String("commit", syncResult.NewRevision),
			slog.Any("err", driftErr),
		)
	}

	result := "success"
	combinedErr := errors.Join(deployErrs...)
	if combinedErr != nil {
		result = "partial_error"
		slog.ErrorContext(ctx, "sync finished with errors",
			slog.String("reason", string(task.reason)),
			slog.String("commit", syncResult.NewRevision),
			slog.Any("err", combinedErr),
		)
	}

	c.metrics.Sync.RecordSyncRun(string(task.reason), result, time.Since(startedAt))
	c.updateState(func(s *runtimeState) {
		s.LastSyncAt = time.Now()
		s.LastSyncReason = string(task.reason)
		s.LastSyncResult = result
		s.LastSyncError = ""
		if combinedErr != nil {
			s.LastSyncError = combinedErr.Error()
		}
		s.GitRevision = syncResult.NewRevision
	})
}

func (c *Controller) reloadStacks() (string, error) {
	return c.cfg.ReloadStacks(c.git.WorkingDir())
}

func (c *Controller) syncStack(
	ctx context.Context,
	stackCfg config.StackSpec,
	commit string,
) error {
	currentState := c.snapshotState()
	prev, exists := currentState.Stacks[stackCfg.Name]
	reconcileResult, err := c.stackReconciler.Reconcile(
		ctx,
		stackCfg,
		prev.SourceDigest,
		exists,
	)
	if err != nil {
		c.recordStackFailure(stackCfg.Name, commit, failedServicesFromReconcileError(err), err)
		return fmt.Errorf("stack %s %w", stackCfg.Name, err)
	}

	if reconcileResult.Skipped {
		return nil
	}

	err = c.deployer.DeployStack(ctx, stackCfg.Name, reconcileResult.DeployCompose, reconcileResult.Services)
	if err != nil {
		c.recordStackFailure(stackCfg.Name, commit, reconcileResult.Services, err)
		return fmt.Errorf("deploy stack %s: %w", stackCfg.Name, err)
	}

	now := time.Now()
	servicesState := map[string]serviceState{}
	for _, service := range reconcileResult.Services {
		servicesState[service.Name] = serviceState{
			Image:        service.Image,
			LastStatus:   string(drift.SyncStatusSynced),
			LastDeployAt: now,
		}
		c.metrics.Deploys.RecordDeploy(stackCfg.Name, service.Name, "success")
	}

	c.updateState(func(s *runtimeState) {
		s.Stacks[stackCfg.Name] = stackState{
			SourceDigest: reconcileResult.SourceDigest,
			LastCommit:   commit,
			LastStatus:   "success",
			LastError:    "",
			LastDeployAt: now,
			Services:     servicesState,
		}
	})

	c.event.Dispatch(ctx, &events.DeploySuccess{
		StackName: stackCfg.Name,
		Commit:    commit,
		Services:  reconcileResult.Services,
	})
	return nil
}

func (c *Controller) runPollDriftPass(ctx context.Context, commit string) []error {
	if c.driftAnalyzer == nil {
		return nil
	}

	driftErrs := make([]error, 0)
	for _, stackCfg := range c.cfg.Spec.Stacks {
		currentState := c.snapshotState()
		prev, exists := currentState.Stacks[stackCfg.Name]

		reconcileResult, err := c.stackReconciler.Reconcile(ctx, stackCfg, prev.SourceDigest, exists)
		if err != nil {
			driftErrs = append(driftErrs, fmt.Errorf("stack %s reconcile for drift: %w", stackCfg.Name, err))
			continue
		}

		err = c.analyzeStackDrift(ctx, stackCfg, commit, reconcileResult)
		if err != nil {
			driftErrs = append(driftErrs, fmt.Errorf("stack %s drift analysis: %w", stackCfg.Name, err))
		}
	}

	return driftErrs
}

func (c *Controller) analyzeStackDrift(
	ctx context.Context,
	stackCfg config.StackSpec,
	commit string,
	reconcileResult stackReconcileResult,
) error {
	if c.driftAnalyzer == nil {
		return nil
	}

	currentState := c.snapshotState()
	prev := currentState.Stacks[stackCfg.Name]
	servicesState := cloneServiceStates(prev.Services)
	now := time.Now()
	hasRemediationRun := false
	var remediationErrs []error

	for _, service := range reconcileResult.Services {
		driftResult, err := c.driftAnalyzer.Analyze(ctx, stackCfg.Name, service)
		if err != nil {
			remediationErrs = append(remediationErrs, fmt.Errorf("analyze service %s: %w", service.Name, err))
			continue
		}

		currentServiceState := servicesState[service.Name]
		currentServiceState.Image = service.Image
		if currentServiceState.LastStatus == "" {
			currentServiceState.LastStatus = string(drift.SyncStatusSynced)
		}

		if !driftResult.OutOfSync {
			currentServiceState.LastStatus = string(drift.SyncStatusSynced)
			servicesState[service.Name] = currentServiceState
			continue
		}

		if driftResult.ServiceMissed {
			c.event.Dispatch(ctx, &events.ServiceMissed{
				StackName:   stackCfg.Name,
				ServiceName: service.Name,
			})
		}
		if driftResult.Replicas.OutOfSync {
			c.event.Dispatch(ctx, &events.ServiceReplicasDiverged{
				StackName:   stackCfg.Name,
				ServiceName: service.Name,
			})
		}

		currentServiceState.LastStatus = string(drift.SyncStatusOutOfSync)
		servicesState[service.Name] = currentServiceState

		if !resolveSelfHealEnabled(service.SyncPolicy.SelfHeal, c.cfg.Spec.Sync.Policy.SelfHeal) {
			continue
		}

		hasRemediationRun = true
		err = c.deployer.DeployService(ctx, stackCfg.Name, reconcileResult.DeployCompose, service)
		if err != nil {
			c.metrics.Deploys.RecordDeploy(stackCfg.Name, service.Name, "failed")

			if driftResult.ServiceMissed {
				c.event.Dispatch(ctx, &events.ServiceRestoreFailed{
					StackName:   stackCfg.Name,
					ServiceName: service.Name,
				})
			}

			currentServiceState.LastStatus = string(drift.SyncStatusSyncFailed)
			currentServiceState.LastDeployAt = now
			servicesState[service.Name] = currentServiceState
			remediationErrs = append(remediationErrs, fmt.Errorf("restore service %s: %w", service.Name, err))
			continue
		}

		c.metrics.Deploys.RecordDeploy(stackCfg.Name, service.Name, "success")
		currentServiceState.LastStatus = string(drift.SyncStatusSynced)
		currentServiceState.LastDeployAt = now
		servicesState[service.Name] = currentServiceState

		if driftResult.ServiceMissed {
			c.event.Dispatch(ctx, &events.ServiceRestored{
				StackName:   stackCfg.Name,
				ServiceName: service.Name,
			})
		}
	}

	c.updateState(func(s *runtimeState) {
		stackRuntimeState := s.Stacks[stackCfg.Name]
		stackRuntimeState.SourceDigest = reconcileResult.SourceDigest
		if commit != "" {
			stackRuntimeState.LastCommit = commit
		}
		if stackRuntimeState.LastStatus == "" {
			stackRuntimeState.LastStatus = "success"
		}
		stackRuntimeState.LastError = ""
		if hasRemediationRun {
			stackRuntimeState.LastDeployAt = now
		}
		stackRuntimeState.Services = servicesState
		s.Stacks[stackCfg.Name] = stackRuntimeState
	})

	if len(remediationErrs) > 0 {
		return errors.Join(remediationErrs...)
	}

	return nil
}

func cloneServiceStates(in map[string]serviceState) map[string]serviceState {
	if len(in) == 0 {
		return map[string]serviceState{}
	}

	out := make(map[string]serviceState, len(in))
	for name, state := range in {
		out[name] = state
	}

	return out
}

func resolveSelfHealEnabled(servicePolicy *bool, globalPolicy bool) bool {
	if servicePolicy != nil {
		return *servicePolicy
	}

	return globalPolicy
}

func (c *Controller) recordStackFailure(stackName, commit string, services []compose.Service, reason error) {
	now := time.Now()
	servicesState := map[string]serviceState{}
	for _, service := range services {
		servicesState[service.Name] = serviceState{
			Image:        service.Image,
			LastStatus:   string(drift.SyncStatusSyncFailed),
			LastDeployAt: now,
		}
		c.metrics.Deploys.RecordDeploy(stackName, service.Name, "failed")
	}
	if len(servicesState) == 0 {
		c.metrics.Deploys.RecordDeploy(stackName, "unknown", "failed")
	}

	c.updateState(func(s *runtimeState) {
		s.Stacks[stackName] = stackState{
			SourceDigest: "",
			LastCommit:   commit,
			LastStatus:   "failed",
			LastError:    reason.Error(),
			LastDeployAt: now,
			Services:     servicesState,
		}
	})

	logs := []string{}

	var logsErr containsLogsError
	if errors.As(reason, &logsErr) {
		logs = logsErr.Logs()
	}

	c.event.Dispatch(context.Background(), &events.DeployFailed{
		StackName: stackName,
		Commit:    commit,
		Services:  services,
		Error:     reason,
		Logs:      logs,
	})
}

type containsLogsError interface {
	Logs() []string
}
