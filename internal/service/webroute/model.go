package webroute

// Route is a single public endpoint route for a service.
type Route struct {
	// Domain is a public domain where service is available.
	Domain string `json:"domain"`
	// Address is a full host and path value used for direct HTTP calls.
	Address string `json:"address"`
	// Port is a service container port exposed by a reverse proxy.
	Port string `json:"port"`
}
