package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Logger     *Logger     `mapstructure:"logger"`
	HTTPServer *HTTPServer `mapstructure:"http_server"`
	Database   *Database   `mapstructure:"database"`
}

type Logger struct {
	Level int `mapstructure:"level"`
}

type HTTPServer struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
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
