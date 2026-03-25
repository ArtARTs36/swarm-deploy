package webroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolverResolveNginxProxyRoutes(t *testing.T) {
	resolver := NewResolver()

	routes := resolver.Resolve([]string{
		"VIRTUAL_HOST=api.example.com, admin.example.com",
		"VIRTUAL_PATH=/v1",
		"VIRTUAL_PORT=8080",
	})

	assert.Equal(t, []Route{
		{
			Domain:  "api.example.com",
			Address: "api.example.com/v1",
			Port:    "8080",
		},
		{
			Domain:  "admin.example.com",
			Address: "admin.example.com/v1",
			Port:    "8080",
		},
	}, routes, "unexpected resolved web routes")
}

func TestResolverResolveWithoutPath(t *testing.T) {
	resolver := NewResolver()

	routes := resolver.Resolve([]string{
		"VIRTUAL_HOST=app.example.com",
		"VIRTUAL_PORT=80",
	})

	assert.Equal(t, []Route{
		{
			Domain:  "app.example.com",
			Address: "app.example.com/",
			Port:    "80",
		},
	}, routes, "unexpected resolved web routes without path")
}

func TestResolverResolveWithoutHost(t *testing.T) {
	resolver := NewResolver()

	routes := resolver.Resolve([]string{
		"VIRTUAL_PATH=/v1",
		"VIRTUAL_PORT=8080",
	})

	assert.Nil(t, routes, "expected no routes without virtual host")
}
