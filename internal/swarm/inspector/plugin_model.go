package inspector

import (
	"sort"
)

// PluginInfo is a runtime snapshot of Docker plugin metadata.
type PluginInfo struct {
	// ID is a unique Docker plugin identifier.
	ID string `json:"id"`
	// Name is a Docker plugin name.
	Name string `json:"name"`
	// Description is a plugin description from plugin config.
	Description string `json:"description"`
	// Enabled indicates whether plugin is enabled.
	Enabled bool `json:"enabled"`
	// PluginReference is a plugin reference used for push/pull.
	PluginReference string `json:"plugin_reference"`
	// Capabilities contains plugin interface capabilities.
	Capabilities []string `json:"capabilities"`
}

func sortPluginInfos(plugins []PluginInfo) {
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Name != plugins[j].Name {
			return plugins[i].Name < plugins[j].Name
		}

		return plugins[i].ID < plugins[j].ID
	})
}
