package serviceupdater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/artarts36/swarm-deploy/internal/config"
	gitx "github.com/artarts36/swarm-deploy/internal/git"
	"github.com/artarts36/swarm-deploy/internal/githosting"
	"github.com/artarts36/swarm-deploy/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceUpdaterUpdateImageVersion(t *testing.T) {
	repositoryDir := t.TempDir()
	composePath := filepath.Join(repositoryDir, "deploy", "docker-compose.yaml")
	writeComposeFile(t, composePath, `
services:
  api:
    image: ghcr.io/acme/api:1.0.0
`)

	repository := &fakeRepository{
		workingDir: repositoryDir,
		commitHash: "abc123",
	}
	resolver := &fakeImageVersionResolver{}

	updater := NewServiceUpdater(
		func() []config.StackSpec {
			return []config.StackSpec{
				{Name: "core", ComposeFile: "deploy/docker-compose.yaml"},
			}
		},
		repository,
		resolver,
		"https://github.com/acme/swarm-config.git",
		"main",
		"",
		nil,
	)

	result, err := updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "core",
		ServiceName:  "api",
		ImageVersion: "2.0.0",
		Reason:       "please update image",
		UserName:     "alice",
	})
	require.NoError(t, err, "update image version")

	assert.Equal(t, "core", result.StackName, "unexpected stack name")
	assert.Equal(t, "api", result.ServiceName, "unexpected service name")
	assert.Equal(t, "ghcr.io/acme/api:1.0.0", result.OldImage, "unexpected old image")
	assert.Equal(t, "ghcr.io/acme/api:2.0.0", result.NewImage, "unexpected new image")
	assert.Equal(t, "api-up-image-2.0.0", result.BranchName, "unexpected branch name")
	assert.Equal(t, "https://github.com/acme/swarm-config/tree/api-up-image-2.0.0", result.BranchURL, "unexpected branch url")
	assert.Equal(t, "abc123", result.CommitHash, "unexpected commit hash")
	assert.Empty(t, result.MergeRequestURL, "merge request url must be empty without token")

	assert.Equal(t, 1, repository.syncBranchCalled, "sync branch should be called once")
	assert.Equal(t, "main", repository.syncedBranch, "unexpected synced branch")
	assert.Equal(t, 1, repository.createBranchCalled, "create branch should be called once")
	assert.Equal(t, "api-up-image-2.0.0", repository.createdBranch, "unexpected created branch")
	assert.Equal(t, 1, repository.addCalled, "add should be called once")
	assert.Equal(t, "deploy/docker-compose.yaml", repository.addedPath, "unexpected added path")
	assert.Equal(t, 1, repository.commitCalled, "commit should be called once")
	assert.Equal(t, "chore(api): up image to 2.0.0", repository.commitMessage, "unexpected commit message")
	assert.Equal(t, gitx.CommitAuthor{Name: "alice", Email: defaultCommitAuthorEmail}, repository.commitAuthor, "unexpected commit author")
	assert.Equal(t, 1, repository.pushCalled, "push should be called once")
	assert.Equal(t, "api-up-image-2.0.0", repository.pushedBranch, "unexpected pushed branch")
	assert.Equal(t, "ghcr.io/acme/api:2.0.0", resolver.image, "unexpected image checked in registry")

	updatedComposeRaw, err := os.ReadFile(composePath)
	require.NoError(t, err, "read updated compose")
	assert.Contains(t, string(updatedComposeRaw), "ghcr.io/acme/api:2.0.0", "compose image should be updated")
}

func TestServiceUpdaterUpdateImageVersionWithMergeRequest(t *testing.T) {
	repositoryDir := t.TempDir()
	composePath := filepath.Join(repositoryDir, "docker-compose.yaml")
	writeComposeFile(t, composePath, `
services:
  api:
    image: ghcr.io/acme/api:1.0.0
`)

	provider := &fakeMergeRequestProvider{
		supported: true,
		url:       "https://github.com/acme/swarm-config/pull/10",
	}
	updater := NewServiceUpdater(
		func() []config.StackSpec {
			return []config.StackSpec{
				{Name: "core", ComposeFile: "docker-compose.yaml"},
			}
		},
		&fakeRepository{
			workingDir: repositoryDir,
			commitHash: "abc123",
		},
		&fakeImageVersionResolver{},
		"https://github.com/acme/swarm-config.git",
		"main",
		"token-1",
		[]githosting.Provider{provider},
	)

	result, err := updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "core",
		ServiceName:  "api",
		ImageVersion: "2.1.0",
		Reason:       "Обнови сервис до стабильной версии",
		UserName:     "artem",
	})
	require.NoError(t, err, "update image version")
	assert.Equal(t, "https://github.com/acme/swarm-config/pull/10", result.MergeRequestURL, "unexpected merge request url")
	assert.Equal(t, 1, provider.called, "merge request provider should be called once")
	assert.Equal(t, "https://github.com/acme/swarm-config.git", provider.request.RepositoryURL, "unexpected provider repository")
	assert.Equal(t, "main", provider.request.BaseBranch, "unexpected provider base branch")
	assert.Equal(t, "api-up-image-2.1.0", provider.request.HeadBranch, "unexpected provider head branch")
	assert.Equal(t, "chore(api): up image to 2.1.0", provider.request.Title, "unexpected provider title")
	assert.Equal(t, "Обнови сервис до стабильной версии by artem", provider.request.Body, "unexpected provider body")
}

func TestServiceUpdaterUpdateImageVersionFailsOnMissingStack(t *testing.T) {
	repositoryDir := t.TempDir()

	updater := NewServiceUpdater(
		func() []config.StackSpec {
			return []config.StackSpec{
				{Name: "core", ComposeFile: "docker-compose.yaml"},
			}
		},
		&fakeRepository{
			workingDir: repositoryDir,
		},
		&fakeImageVersionResolver{},
		"https://github.com/acme/swarm-config.git",
		"main",
		"",
		nil,
	)

	_, err := updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "missing",
		ServiceName:  "api",
		ImageVersion: "1.0.0",
		Reason:       "update",
		UserName:     "alice",
	})
	require.Error(t, err, "missing stack must fail")
	assert.Contains(t, err.Error(), `stack "missing" not found`, "unexpected error")
}

func TestServiceUpdaterUpdateImageVersionFailsOnRegistryError(t *testing.T) {
	repositoryDir := t.TempDir()
	writeComposeFile(t, filepath.Join(repositoryDir, "docker-compose.yaml"), `
services:
  api:
    image: ghcr.io/acme/api:1.0.0
`)

	repository := &fakeRepository{
		workingDir: repositoryDir,
		commitHash: "abc123",
	}
	updater := NewServiceUpdater(
		func() []config.StackSpec {
			return []config.StackSpec{
				{Name: "core", ComposeFile: "docker-compose.yaml"},
			}
		},
		repository,
		&fakeImageVersionResolver{err: errors.New("registry unavailable")},
		"https://github.com/acme/swarm-config.git",
		"main",
		"",
		nil,
	)

	_, err := updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "core",
		ServiceName:  "api",
		ImageVersion: "2.0.0",
		Reason:       "update",
		UserName:     "alice",
	})
	require.Error(t, err, "registry error must fail")
	assert.Contains(t, err.Error(), "registry unavailable", "unexpected error")
	assert.Equal(t, 0, repository.createBranchCalled, "branch should not be created when registry check fails")
}

func TestServiceUpdaterUpdateImageVersionUsesFreshStacksOnEveryCall(t *testing.T) {
	repositoryDir := t.TempDir()
	composePath := filepath.Join(repositoryDir, "docker-compose.yaml")

	writeComposeFile(t, composePath, `
services:
  api:
    image: ghcr.io/acme/api:1.0.0
`)

	stacks := []config.StackSpec{
		{Name: "core", ComposeFile: "docker-compose.yaml"},
	}
	updater := NewServiceUpdater(
		func() []config.StackSpec {
			out := make([]config.StackSpec, len(stacks))
			copy(out, stacks)
			return out
		},
		&fakeRepository{
			workingDir: repositoryDir,
			commitHash: "abc123",
		},
		&fakeImageVersionResolver{},
		"https://github.com/acme/swarm-config.git",
		"main",
		"",
		nil,
	)

	_, err := updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "core",
		ServiceName:  "api",
		ImageVersion: "2.0.0",
		Reason:       "update",
		UserName:     "alice",
	})
	require.NoError(t, err, "first update should use current stacks file")

	writeComposeFile(t, composePath, `
services:
  api:
    image: ghcr.io/acme/api:2.0.0
`)
	stacks = []config.StackSpec{
		{Name: "next", ComposeFile: "docker-compose.yaml"},
	}
	_, err = updater.UpdateImageVersion(context.Background(), UpdateImageVersionInput{
		StackName:    "next",
		ServiceName:  "api",
		ImageVersion: "3.0.0",
		Reason:       "update again",
		UserName:     "alice",
	})
	require.NoError(t, err, "second update should pick updated stacks file")
}

func writeComposeFile(t *testing.T, path string, payload string) {
	t.Helper()

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err, "create compose dir")

	err = os.WriteFile(path, []byte(payload), 0o600)
	require.NoError(t, err, "write compose file")
}

type fakeRepository struct {
	workingDir string
	commitHash string
	err        error

	syncBranchCalled int
	syncedBranch     string

	createBranchCalled int
	createdBranch      string

	addCalled int
	addedPath string

	commitCalled    int
	commitMessage   string
	commitAuthor    gitx.CommitAuthor
	commitReturnErr error

	pushCalled   int
	pushedBranch string
}

func (f *fakeRepository) SyncBranch(_ context.Context, branch string) error {
	f.syncBranchCalled++
	f.syncedBranch = branch
	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeRepository) CreateBranch(_ context.Context, branch string) error {
	f.createBranchCalled++
	f.createdBranch = branch
	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeRepository) Add(_ context.Context, path string) error {
	f.addCalled++
	f.addedPath = path
	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeRepository) Commit(
	_ context.Context,
	message string,
	author gitx.CommitAuthor,
) (string, error) {
	f.commitCalled++
	f.commitMessage = message
	f.commitAuthor = author
	if f.commitReturnErr != nil {
		return "", f.commitReturnErr
	}
	if f.err != nil {
		return "", f.err
	}

	return f.commitHash, nil
}

func (f *fakeRepository) Push(_ context.Context, branch string) error {
	f.pushCalled++
	f.pushedBranch = branch
	if f.err != nil {
		return f.err
	}

	return nil
}

func (f *fakeRepository) WorkingDir() string {
	return f.workingDir
}

type fakeImageVersionResolver struct {
	image string
	err   error
}

func (f *fakeImageVersionResolver) ResolveActualVersion(
	_ context.Context,
	image string,
) (registry.ImageVersion, error) {
	f.image = image
	if f.err != nil {
		return registry.ImageVersion{}, f.err
	}

	return registry.ImageVersion{
		Image: image,
	}, nil
}

type fakeMergeRequestProvider struct {
	supported bool
	url       string
	err       error
	called    int
	request   githosting.CreateMergeRequestRequest
}

func (f *fakeMergeRequestProvider) Supports(_ string) bool {
	return f.supported
}

func (f *fakeMergeRequestProvider) CreateMergeRequest(
	_ context.Context,
	request githosting.CreateMergeRequestRequest,
) (string, error) {
	f.called++
	f.request = request
	if f.err != nil {
		return "", f.err
	}

	return f.url, nil
}
