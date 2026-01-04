package config

import (
	"fmt"

	"github.com/spf13/viper"
)

const defaultAPIServerPort = "8081"

type APIConfig struct {
	ServerPort  string
	PostgresDSN string
}

type CollectorConfig struct {
	PostgresDSN string
	MQURL       string
}

type StreamerConfig struct {
	MQURL              string
	MetricsCsvFilePath string
}

func LoadAPIConfig() (*APIConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("SERVER_PORT", defaultAPIServerPort)
	cfg := &APIConfig{
		ServerPort:  v.GetString("SERVER_PORT"),
		PostgresDSN: v.GetString("POSTGRES_DSN"),
	}
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	return cfg, nil
}

func LoadCollectorConfig() (*CollectorConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	cfg := &CollectorConfig{
		MQURL:       v.GetString("MQ_URL"),
		PostgresDSN: v.GetString("POSTGRES_DSN"),
	}
	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	if cfg.MQURL == "" {
		return nil, fmt.Errorf("MQ_URL is required")
	}
	return cfg, nil
}

func LoadStreamerConfig() (*StreamerConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	cfg := &StreamerConfig{
		MQURL:              v.GetString("MQ_URL"),
		MetricsCsvFilePath: v.GetString("METRICS_CSV_FILEPATH"),
	}
	if cfg.MetricsCsvFilePath == "" {
		return nil, fmt.Errorf("METRICS_CSV_FILEPATH is required")
	}
	if cfg.MQURL == "" {
		return nil, fmt.Errorf("MQ_URL is required")
	}
	return cfg, nil
}
