package config

import (
	"fmt"

	"github.com/spf13/viper"
)

const defaultAPIServerPort = "8081"

type APIConfig struct {
	ServerPort  string
	PostgresDSN string
	LogLevel    string
}

type CollectorConfig struct {
	PostgresDSN string
	MQURL       string
	LogLevel    string
}

type StreamerConfig struct {
	MQURL              string
	MetricsCsvFilePath string
	LogLevel           string
}

type MQConfig struct {
	MQPartitions int
	LogLevel     string
}

func LoadAPIConfig() (*APIConfig, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("SERVER_PORT", defaultAPIServerPort)
	cfg := &APIConfig{
		ServerPort:  v.GetString("SERVER_PORT"),
		PostgresDSN: v.GetString("POSTGRES_DSN"),
		LogLevel:    v.GetString("LOG_LEVEL"),
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
		LogLevel:    v.GetString("LOG_LEVEL"),
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
		LogLevel:           v.GetString("LOG_LEVEL"),
	}
	if cfg.MetricsCsvFilePath == "" {
		return nil, fmt.Errorf("METRICS_CSV_FILEPATH is required")
	}
	if cfg.MQURL == "" {
		return nil, fmt.Errorf("MQ_URL is required")
	}
	return cfg, nil
}

func LoadMQConfig() *MQConfig {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("MQ_PARTITIONS", 3)
	cfg := &MQConfig{
		MQPartitions: v.GetInt("MQ_PARTITIONS"),
		LogLevel:     v.GetString("LOG_LEVEL"),
	}
	return cfg
}
