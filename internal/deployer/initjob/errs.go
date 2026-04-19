package initjob

import "fmt"

type JobFailedError struct {
	ID     string
	Name   string
	Reason string
	logs   []string
}

func (e *JobFailedError) Error() string {
	return fmt.Sprintf("job %q with id %q failed: %s", e.Name, e.ID, e.Reason)
}

func (e *JobFailedError) Logs() []string {
	return e.logs
}
