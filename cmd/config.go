package main

import (
	"fmt"

	"github.com/66gu1/easygodocs/internal/app/auth"
	entity "github.com/66gu1/easygodocs/internal/app/entity/repo/gorm"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type config struct {
	Port        string   `mapstructure:"port" json:"port"`
	DatabaseDSN string   `mapstructure:"database_dsn" json:"database_dsn"`
	LogLevel    logLevel `mapstructure:"log_level" json:"log_level"`
	MaxBodySize int64    `mapstructure:"max_body_size" json:"max_body_size"`

	User   user.Config   `mapstructure:"user" json:"user"`
	Auth   auth.Config   `mapstructure:"auth" json:"auth"`
	Entity entity.Config `mapstructure:"entity" json:"entity"`
}

func getConfig() config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var Cfg config
	if err := viper.Unmarshal(&Cfg); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return Cfg
}

type logLevel string

const (
	logLevelDebug logLevel = "debug"
	logLevelInfo  logLevel = "info"
	logLevelWarn  logLevel = "warn"
	logLevelError logLevel = "error"
)

func (l logLevel) zeroLog() zerolog.Level {
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
