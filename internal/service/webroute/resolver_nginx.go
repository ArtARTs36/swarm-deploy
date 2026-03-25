package webroute

import "strings"

const (
	nginxVirtualHostKey = "VIRTUAL_HOST"
	nginxVirtualPathKey = "VIRTUAL_PATH"
	nginxVirtualPortKey = "VIRTUAL_PORT"
)

// nginxProxyProvider resolves routes configured for nginx-proxy.
type nginxProxyProvider struct{}

// ResolveRoutes resolves nginx-proxy routes from env values.
func (*nginxProxyProvider) ResolveRoutes(env map[string]string) []Route {
	if len(env) == 0 {
		return nil
	}

	virtualHosts := strings.TrimSpace(env[nginxVirtualHostKey])
	if virtualHosts == "" {
		return nil
	}

	virtualPath := normalizeNginxPath(env[nginxVirtualPathKey])
	virtualPort := strings.TrimSpace(env[nginxVirtualPortKey])

	routes := make([]Route, 0)
	for _, host := range strings.Split(virtualHosts, ",") {
		domain := strings.TrimSpace(host)
		if domain == "" {
			continue
		}

		routes = append(routes, Route{
			Domain:  domain,
			Address: domain + "/" + virtualPath,
			Port:    virtualPort,
		})
	}

	return routes
}

func normalizeNginxPath(path string) string {
	normalized := strings.TrimSpace(path)
	normalized = strings.TrimPrefix(normalized, "/")

	return normalized
}
