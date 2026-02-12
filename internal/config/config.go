package config

import (
	"github.com/Kilat-Pet-Delivery/lib-common/config"
)

// ServiceConfig holds all configuration for the tracking service.
type ServiceConfig struct {
	Port        string
	AppEnv      string
	DBConfig    config.DatabaseConfig
	JWTConfig   config.JWTConfig
	KafkaConfig config.KafkaConfig
}

// Load reads configuration from environment variables and returns ServiceConfig.
func Load() (*ServiceConfig, error) {
	v, err := config.Load("tracking")
	if err != nil {
		return nil, err
	}

	return &ServiceConfig{
		Port:        config.GetServicePort(v, "SERVICE_PORT"),
		AppEnv:      config.GetAppEnv(v),
		DBConfig:    config.LoadDatabaseConfig(v, "DB_NAME"),
		JWTConfig:   config.LoadJWTConfig(v),
		KafkaConfig: config.LoadKafkaConfig(v),
	}, nil
}
