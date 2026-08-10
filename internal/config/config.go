package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

var ErrInvalidConfig = errors.New("invalid configuration")

type Config struct {
	Logger     *Logger     `mapstructure:"logger"`
	HTTPServer *HTTPServer `mapstructure:"http_server"`
	Database   *Database   `mapstructure:"database"`
	Auth       Auth        `mapstructure:"auth"`
	Demo       Demo        `mapstructure:"demo"`
}

type Auth struct {
	// SessionSecret задаётся только переменной окружения AUTH_SESSION_SECRET,
	// в файле конфига его нет: закоммиченный секрет доступен всем, кто открыл
	// репозиторий.
	SessionSecret string `mapstructure:"session_secret"`
	// SecureCookie — флаг Secure на куке сессии. Обязателен на HTTPS-стенде,
	// на локальном http его включать нельзя: браузер такую куку не сохранит.
	SecureCookie bool `mapstructure:"secure_cookie"`
}

type Demo struct {
	Enabled bool `mapstructure:"enabled"`
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

// dbEnvBindings — явные привязки, потому что AutomaticEnv() сам реплейсит "."
// на "_" (database.host → DATABASE_HOST), а в compose уже используются
// короткие имена вроде DB_HOST.
var dbEnvBindings = map[string]string{
	"database.host":     "DB_HOST",
	"database.port":     "DB_PORT",
	"database.user":     "DB_USER",
	"database.password": "DB_PASSWORD",
	"database.name":     "DB_NAME",

	"auth.session_secret": "AUTH_SESSION_SECRET",
	"auth.secure_cookie":  "AUTH_SECURE_COOKIE",
	"demo.enabled":        "DEMO_ENABLED",
}

// minSessionSecretLen — минимальная длина секрета подписи сессии. HMAC-SHA256
// работает с ключом любой длины, но короткий ключ перебирается офлайн по
// одному перехваченному токену, после чего подделывается любая личность.
const minSessionSecretLen = 32

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

	// Переменные окружения перекрывают YAML: тот же configuration.yaml
	// (host: postgres, верно внутри docker-сети) работает и с хоста через
	// DB_HOST=localhost go run ./cmd.
	for key, env := range dbEnvBindings {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("bind env %s: %w", env, err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate роняет приложение на старте, если сессии нечем подписывать —
// иначе оно выглядело бы рабочим, не имея никакой защиты личности.
func (c *Config) validate() error {
	if len(c.Auth.SessionSecret) < minSessionSecretLen {
		return fmt.Errorf(
			"AUTH_SESSION_SECRET must be at least %d bytes long, got %d: %w",
			minSessionSecretLen, len(c.Auth.SessionSecret), ErrInvalidConfig,
		)
	}

	return nil
}
