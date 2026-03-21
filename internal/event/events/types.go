package events

type Type string

const (
	TypeDeploySuccess     = "deploySuccess"
	TypeDeployFailed      = "deployFailed"
	TypeSyncManualStarted = "syncManualStarted"
	TypeUserAuthenticated = "userAuthenticated"
)

type Event interface {
	Type() Type
}
