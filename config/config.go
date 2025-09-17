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
	Port        string   `mapstructure:"port" json:"port"`
	DatabaseDSN string   `mapstructure:"database_dsn" json:"database_dsn"`
	LogLevel    LogLevel `mapstructure:"log_level" json:"log_level"`
	MaxBodySize int64    `mapstructure:"max_body_size" json:"max_body_size"`
}

func LoadConfig() Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")

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
