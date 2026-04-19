package swarm

import "fmt"

// ServiceReference identifies a stack service.
type ServiceReference struct {
	stackName   string
	serviceName string
}

// NewServiceReference creates a service reference from stack and service names.
func NewServiceReference(stackName, serviceName string) ServiceReference {
	return ServiceReference{
		stackName:   stackName,
		serviceName: serviceName,
	}
}

// StackName returns stack name part.
func (r ServiceReference) StackName() string {
	return r.stackName
}

// ServiceName returns service name part.
func (r ServiceReference) ServiceName() string {
	return r.serviceName
}

// Name returns full service name as "<stack>_<service>".
func (r ServiceReference) Name() string {
	return fmt.Sprintf("%s_%s", r.stackName, r.serviceName)
}
