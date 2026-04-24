package drift

// SyncStatus is a service synchronization status.
type SyncStatus string

const (
	// SyncStatusSynced means runtime state matches desired specification.
	SyncStatusSynced SyncStatus = "Synced"
	// SyncStatusOutOfSync means runtime state diverges from desired specification.
	SyncStatusOutOfSync SyncStatus = "OutOfSync"
	// SyncStatusSyncFailed means remediation attempt failed.
	SyncStatusSyncFailed SyncStatus = "SyncFailed"
)

// Drift describes divergence between desired and live service state.
type Drift struct {
	// OutOfSync is true when at least one drift condition is detected.
	OutOfSync bool
	// ServiceMissed is true when service is not found in cluster runtime.
	ServiceMissed bool
	// Replicas contains replicas-specific drift details.
	Replicas Replicas
}

// Replicas describes desired and live replicas drift details.
type Replicas struct {
	// OutOfSync is true when desired and live replicas counts differ.
	OutOfSync bool
	// Desired is expected replicas value from compose specification.
	Desired uint
	// Live is actual replicas value from cluster runtime.
	Live uint
}
