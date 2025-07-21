package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type config struct {
	Port        string   `mapstructure:"port" json:"port"`
	DatabaseDSN string   `mapstructure:"database_dsn" json:"database_dsn"`
	LogLevel    logLevel `mapstructure:"log_level" json:"log_level"`
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
	LogLevelDebug logLevel = "debug"
)

func (l logLevel) zeroLog() zerolog.Level {
	switch l {
	case LogLevelDebug:
		return zerolog.DebugLevel
	default:
		return zerolog.InfoLevel
	}
}
