package webroute

import "strings"

const (
	envSplitParts = 2
)

// Provider resolves web routes for a specific reverse proxy from env values.
type Provider interface {
	// ResolveRoutes resolves routes from normalized environment map.
	ResolveRoutes(env map[string]string) []Route
}

// Resolver resolves public routes using a list of proxy-specific providers.
type Resolver struct {
	providers []Provider
}

// NewResolver creates a resolver with built-in providers.
func NewResolver() *Resolver {
	return &Resolver{
		providers: []Provider{
			&nginxProxyProvider{},
		},
	}
}

// Resolve resolves all routes from container env vars.
func (r *Resolver) Resolve(containerEnv []string) []Route {
	env := parseContainerEnv(containerEnv)
	if len(env) == 0 {
		return nil
	}

	out := make([]Route, 0)
	seen := map[string]struct{}{}

	for _, provider := range r.providers {
		for _, route := range provider.ResolveRoutes(env) {
			key := route.Domain + "\x00" + route.Address + "\x00" + route.Port
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, route)
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func parseContainerEnv(containerEnv []string) map[string]string {
	if len(containerEnv) == 0 {
		return nil
	}

	env := make(map[string]string, len(containerEnv))
	for _, item := range containerEnv {
		keyValue := strings.SplitN(item, "=", envSplitParts)
		if len(keyValue) != envSplitParts {
			continue
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		if key == "" {
			continue
		}

		env[key] = value
	}

	return env
}
