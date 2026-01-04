package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{
			name:    "valid info level",
			level:   "info",
			wantErr: false,
		},
		{
			name:    "valid debug level",
			level:   "debug",
			wantErr: false,
		},
		{
			name:    "invalid level",
			level:   "invalid-level",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.level)

			if (err != nil) != tt.wantErr {
				t.Fatalf("NewLogger(%q) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}

			if err == nil && logger == nil {
				t.Fatalf("expected non-nil logger for level %q", tt.level)
			}
		})
	}
}
