package config

import (
	"os"
	"testing"
)

func TestLoadAPIConfig(t *testing.T) {
	tests := []struct {
		name        string
		postgresDSN string
		wantErr     bool
	}{
		{
			name:        "missing postgres dsn",
			postgresDSN: "",
			wantErr:     true,
		},
		{
			name:        "valid config",
			postgresDSN: "postgres://user:pass@localhost/db",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("POSTGRES_DSN")
			os.Unsetenv("SERVER_PORT")

			if tt.postgresDSN != "" {
				os.Setenv("POSTGRES_DSN", tt.postgresDSN)
			}

			_, err := LoadAPIConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadAPIConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadCollectorConfig(t *testing.T) {
	tests := []struct {
		name        string
		postgresDSN string
		mqURL       string
		wantErr     bool
	}{
		{
			name:        "missing postgres dsn",
			postgresDSN: "",
			mqURL:       "amqp://localhost",
			wantErr:     true,
		},
		{
			name:        "missing mq url",
			postgresDSN: "postgres://user:pass@localhost/db",
			mqURL:       "",
			wantErr:     true,
		},
		{
			name:        "valid config",
			postgresDSN: "postgres://user:pass@localhost/db",
			mqURL:       "amqp://localhost",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("POSTGRES_DSN")
			os.Unsetenv("MQ_URL")

			if tt.postgresDSN != "" {
				os.Setenv("POSTGRES_DSN", tt.postgresDSN)
			}
			if tt.mqURL != "" {
				os.Setenv("MQ_URL", tt.mqURL)
			}

			_, err := LoadCollectorConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadCollectorConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadStreamerConfig(t *testing.T) {
	tests := []struct {
		name       string
		mqURL      string
		metricsCSV string
		wantErr    bool
	}{
		{
			name:       "missing metrics csv",
			mqURL:      "amqp://localhost",
			metricsCSV: "",
			wantErr:    true,
		},
		{
			name:       "missing mq url",
			mqURL:      "",
			metricsCSV: "/tmp/metrics.csv",
			wantErr:    true,
		},
		{
			name:       "valid config",
			mqURL:      "amqp://localhost",
			metricsCSV: "/tmp/metrics.csv",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("MQ_URL")
			os.Unsetenv("METRICS_CSV_FILEPATH")

			if tt.mqURL != "" {
				os.Setenv("MQ_URL", tt.mqURL)
			}
			if tt.metricsCSV != "" {
				os.Setenv("METRICS_CSV_FILEPATH", tt.metricsCSV)
			}

			_, err := LoadStreamerConfig()
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadStreamerConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
