package config

import "github.com/artarts36/specw"

type WebSpec struct {
	// Address is an HTTP listen address for UI and API server.
	Address string `yaml:"address"`
	// Security contains UI and API access settings.
	Security SecuritySpec `yaml:"security"`
}

type SecuritySpec struct {
	// Authentication contains web authentication strategy settings.
	Authentication AuthenticationSpec `yaml:"authentication"`
}

type AuthenticationSpec struct {
	// Basic contains HTTP Basic authentication settings.
	Basic BasicAuthenticationSpec `yaml:"basic"`
	// Passkey contains WebAuthn passkey authentication settings.
	Passkey PasskeyAuthenticationSpec `yaml:"passkey"`
}

type BasicAuthenticationSpec struct {
	// HTPasswdFile is a path to htpasswd file with user credentials.
	HTPasswdFile specw.File `yaml:"htpasswdFile"`
}

type PasskeyAuthenticationSpec struct {
	// Enabled enables passkey-based WebAuthn authentication.
	Enabled bool `yaml:"enabled"`
	// RPID is relying party id used by WebAuthn.
	RPID string `yaml:"rpId"`
	// RPDisplayName is relying party human-readable display name.
	RPDisplayName string `yaml:"rpDisplayName"`
	// RPOrigins contains allowed origins for WebAuthn ceremonies.
	RPOrigins []string `yaml:"rpOrigins"`
	// StoragePath is a directory path with passkey users and sessions storage files.
	StoragePath string `yaml:"storagePath"`
	// InsecureCookie disables secure flag on passkey cookies for local development.
	InsecureCookie bool `yaml:"insecureCookie"`
}
