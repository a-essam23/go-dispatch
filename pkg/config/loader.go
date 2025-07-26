package config

import (
	"log/slog"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration from a file and environment variables.
func Load(logger *slog.Logger, fileName string) (*Config, error) {
	v := viper.New()

	// 1. Set default values
	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.auth.jwtSecret", "default-secret-key-change-me")
	v.SetDefault("server.ratelimit.maxConnsPerIP", 5)
	v.SetDefault("transport.readTimeout", "60s")

	// 2. Set config file details
	v.SetConfigName(fileName)
	v.SetConfigType("yaml")
	v.AddConfigPath(".") // look for config in the working directory

	// 3. Set up environment variable handling
	v.SetEnvPrefix("GODISPATCH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 4. Read the configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, err
		}
		logger.Warn("Config file not found. ignoring error and relying on defaults/env vars")
	}

	// 5. Unmarshal the configuration into our struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	for _, name := range cfg.Permissions {
		if err := RegisterPermission(name); err != nil {
			return nil, err
		}
	}
	slog.Info("Permission registry loaded", "total_permissions", len(GetAllRegistered()))

	return &cfg, nil
}
