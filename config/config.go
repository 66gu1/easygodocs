package config

import (
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Config struct {
	Port        string          `mapstructure:"port" json:"port"`
	DatabaseDSN string          `mapstructure:"database_dsn" json:"database_dsn"`
	LogLevel    LogLevel        `mapstructure:"log_level" json:"log_level"`
	MaxBodySize int64           `mapstructure:"max_body_size" json:"max_body_size"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit" json:"rate_limit"`
	DBPool      DBPoolConfig    `mapstructure:"db_pool" json:"db_pool"`
	// TrustedProxies are CIDR prefixes of reverse proxies whose X-Forwarded-For entries are trusted.
	// Empty means the service is reached directly and forwarding headers are ignored.
	TrustedProxies []string `mapstructure:"trusted_proxies" json:"trusted_proxies"`
}

type DBPoolConfig struct {
	MaxOpenConns           int `mapstructure:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `mapstructure:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeMinutes int `mapstructure:"conn_max_lifetime_minutes" json:"conn_max_lifetime_minutes"`
}

type RateLimitConfig struct {
	// PublicRequestsPerMinute limits unauthenticated endpoints (login, refresh, register) per client IP.
	PublicRequestsPerMinute int `mapstructure:"public_requests_per_minute" json:"public_requests_per_minute"`
}

func LoadConfig() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")
	viper.SetDefault("rate_limit.public_requests_per_minute", 20)
	viper.SetDefault("db_pool.max_open_conns", 20)
	viper.SetDefault("db_pool.max_idle_conns", 20)
	viper.SetDefault("db_pool.conn_max_lifetime_minutes", 5)

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var Cfg Config
	if err := viper.Unmarshal(&Cfg); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return Cfg
}

func GetUserConfigs() (user.Config, user.ValidationConfig) {
	var userCfg user.Config
	if err := viper.Sub("user").Unmarshal(&userCfg); err != nil {
		panic(fmt.Errorf("fatal error user config: %w", err))
	}

	var userValCfg user.ValidationConfig
	if err := viper.Sub("user").Unmarshal(&userValCfg); err != nil {
		panic(fmt.Errorf("fatal error user validation config: %w", err))
	}

	return userCfg, userValCfg
}

func GetAuthConfigs() auth.Config {
	var authCfg auth.Config
	if err := viper.Sub("auth").Unmarshal(&authCfg); err != nil {
		panic(fmt.Errorf("fatal error auth config: %w", err))
	}

	return authCfg
}

func GetEntityConfigs() (entity.Config, entity.ValidationConfig) {
	var entityValidationCfg entity.ValidationConfig
	if err := viper.Sub("entity").Unmarshal(&entityValidationCfg); err != nil {
		panic(fmt.Errorf("fatal error entity config: %w", err))
	}

	var entityCfg entity.Config
	if err := viper.Sub("entity").Unmarshal(&entityCfg); err != nil {
		panic(fmt.Errorf("fatal error entity repo config: %w", err))
	}

	return entityCfg, entityValidationCfg
}

type LogLevel string

const (
	logLevelDebug LogLevel = "debug"
	logLevelInfo  LogLevel = "info"
	logLevelWarn  LogLevel = "warn"
	logLevelError LogLevel = "error"
)

func (l LogLevel) ZeroLog() zerolog.Level {
	switch l {
	case logLevelDebug:
		return zerolog.DebugLevel
	case logLevelInfo:
		return zerolog.InfoLevel
	case logLevelWarn:
		return zerolog.WarnLevel
	case logLevelError:
		return zerolog.ErrorLevel

	default:
		return zerolog.InfoLevel
	}
}
