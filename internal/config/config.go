package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Logger     *Logger     `mapstructure:"logger" validate:"required"`
	HTTPServer *HTTPServer `mapstructure:"http_server" validate:"required"`
	Database   *Database   `mapstructure:"database" validate:"required"`
}

type Logger struct {
	Level int `mapstructure:"level" validate:"required"`
}

type HTTPServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port" validate:"required"`
}

type Database struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required"`
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name     string `mapstructure:"name" validate:"required"`
	SSLMode  string `mapstructure:"sslmode"`
}

func ReadConfig() (*Config, error) {
	v := viper.New()

	v.SetConfigType("yaml")
	v.SetConfigName("configuration")
	v.AddConfigPath("./configs/")

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}
