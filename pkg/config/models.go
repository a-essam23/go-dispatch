package config

import "time"

type Config struct {
	Server      ServerConfig
	Transport   TransportConfig
	Events      map[string]EventConfig `mapstructure:"events"`
	Permissions []string               `mapstructure:"permissions"`
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
