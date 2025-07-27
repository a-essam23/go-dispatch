package config

import (
	"time"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type Config struct {
	Server    ServerConfig
	Transport TransportConfig
	// raw representation from YAML (only used when loading)
	Events map[string]EventConfig `mapstructure:"events"`
	// compiled, ready-to-execute action pipelines (populated by the compiler)
	Pipelines   map[string][]pipeline.Step `mapstructure:"-"`
	Permissions []string                   `mapstructure:"permissions"`
}

type ServerConfig struct {
	Address         string
	Auth            AuthConfig
	ConnectionLimit ConnectionLimitConfig `mapstructure:"connectionLimit"`
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwtSecret"`
}

type ConnectionLimitConfig struct {
	MaxPerUser int    `mapstructure:"maxPerUser"`
	Mode       string `mapstructure:"mode"` // "reject" or "cycle"
}

type TransportConfig struct {
	ReadTimeout time.Duration `mapstructure:"readTimeout"`
}

type EventConfig struct {
	Actions []ActionConfig `mapstructure:"actions"`
}

type ActionConfig struct {
	Name   string   `mapstructure:"name"`
	Params []string `mapstructure:"params"`
}
