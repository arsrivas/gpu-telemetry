package storage

import (
	"errors"
	"testing"
	"time"

	"gpu-telemetry/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func newMockStore(t *testing.T) (*PostgresStore, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	store := &PostgresStore{db: db}
	cleanup := func() { db.Close() }

	return store, mock, cleanup
}

func TestPostgresStore_Insert(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	msg := model.Telemetry{
		ID:        "1",
		GPUId:     "gpu-1",
		Timestamp: time.Now(),
		Metric:    "temp",
		Value:     42.5,
		Labels:    "x=y",
	}

	mock.ExpectExec("INSERT INTO telemetry").
		WithArgs(
			msg.ID,
			msg.GPUId,
			msg.Timestamp,
			msg.Metric,
			msg.Value,
			msg.Labels,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Insert(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPostgresStore_GPUs(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"gpu_id"}).
		AddRow("gpu-1").
		AddRow("gpu-2")

	mock.ExpectQuery("SELECT DISTINCT gpu_id FROM telemetry").
		WillReturnRows(rows)

	gpus, err := store.GPUs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gpus) != 2 {
		t.Fatalf("got %d gpus, want 2", len(gpus))
	}
}

func TestPostgresStore_GPUExists(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{"gpu exists", true},
		{"gpu does not exist", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock, cleanup := newMockStore(t)
			defer cleanup()

			mock.ExpectQuery("SELECT EXISTS").
				WithArgs("gpu-1").
				WillReturnRows(
					sqlmock.NewRows([]string{"exists"}).AddRow(tt.exists),
				)

			ok, err := store.GPUExists("gpu-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.exists {
				t.Fatalf("exists = %v, want %v", ok, tt.exists)
			}
		})
	}
}

func TestPostgresStore_Telemetry(t *testing.T) {
	now := time.Now()
	start := now.Add(-time.Hour).Unix()
	end := now.Unix()

	tests := []struct {
		name      string
		startTs   *int64
		endTs     *int64
		expectErr bool
	}{
		{
			name:    "no filters",
			startTs: nil,
			endTs:   nil,
		},
		{
			name:    "startTs only",
			startTs: &start,
			endTs:   nil,
		},
		{
			name:    "endTs only",
			startTs: nil,
			endTs:   &end,
		},
		{
			name:    "start and end",
			startTs: &start,
			endTs:   &end,
		},
		{
			name:      "query error",
			startTs:   nil,
			endTs:     nil,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, mock, cleanup := newMockStore(t)
			defer cleanup()

			if tc.expectErr {
				mock.ExpectQuery("FROM telemetry").
					WillReturnError(errors.New("query failed"))
			} else {
				rows := sqlmock.NewRows([]string{
					"id", "gpu_id", "ts", "metric", "value", "labels",
				}).AddRow(
					"1", "gpu-1", now, "temp", 42.0, "a=b",
				)

				mock.ExpectQuery("FROM telemetry").
					WillReturnRows(rows)
			}

			out, err := store.Telemetry("gpu-1", tc.startTs, tc.endTs)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out) != 1 {
				t.Fatalf("got %d rows, want 1", len(out))
			}
		})
	}
}

func TestPostgresStore_Ping(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectPing().WillReturnError(nil)

	if err := store.Ping(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostgresStore_GPUs_Error(t *testing.T) {
	store, mock, cleanup := newMockStore(t)
	defer cleanup()

	mock.ExpectQuery("SELECT DISTINCT gpu_id FROM telemetry").
		WillReturnError(errors.New("query failed"))

	_, err := store.GPUs()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want string
	}{
		{"zero", 0, "0"},
		{"positive", 5, "5"},
		{"large", 12345, "12345"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := itoa(tc.in); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestPostgresStore_init(t *testing.T) {
	tests := []struct {
		name      string
		execErr   error
		expectErr bool
	}{
		{
			name:    "schema creation success",
			execErr: nil,
		},
		{
			name:      "schema creation failure",
			execErr:   errors.New("exec failed"),
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, mock, cleanup := newMockStore(t)
			defer cleanup()

			if tc.execErr != nil {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS telemetry").
					WillReturnError(tc.execErr)
			} else {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS telemetry").
					WillReturnResult(sqlmock.NewResult(0, 0))
			}

			err := store.init()

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}
