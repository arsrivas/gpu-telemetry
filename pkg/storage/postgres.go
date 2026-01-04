package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"gpu-telemetry/model"

	_ "github.com/lib/pq"
)

// PostgresStore implements the Store interface backed by PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgres initializes a new PostgresStore with retry and backoff logic.
func NewPostgres(dsn string) (*PostgresStore, error) {
	backoff := time.Second
	var lastErr error

	for attempt := 1; attempt <= 6; attempt++ {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			lastErr = err
		} else if err := db.Ping(); err != nil {
			lastErr = err
		} else {
			// success path
			db.SetMaxOpenConns(20)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(10 * time.Minute)

			ps := &PostgresStore{db: db}
			return ps, ps.init()
		}

		log.Printf(
			"[postgres] connection failed (attempt %d/5): %v — retrying in %s",
			attempt,
			lastErr,
			backoff,
		)

		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, fmt.Errorf("failed to connect to postgres after 5 attempts: %w", lastErr)
}

// init initializes the database schema if it does not already exist.
func (p *PostgresStore) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS telemetry (
		id TEXT PRIMARY KEY,
		gpu_id TEXT NOT NULL,
		ts TIMESTAMPTZ NOT NULL,
		metric TEXT NOT NULL,
		value DOUBLE PRECISION,
		labels TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_gpu_ts ON telemetry (gpu_id, ts);
	`
	_, err := p.db.Exec(schema)
	return err
}

// Insert persists a telemetry record into PostgreSQL.
func (p *PostgresStore) Insert(t model.Telemetry) error {
	_, err := p.db.Exec(`
		INSERT INTO telemetry (id, gpu_id, ts, metric, value, labels)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO NOTHING
	`,
		t.ID, t.GPUId, t.Timestamp, t.Metric, t.Value, t.Labels,
	)
	return err
}

// GPUs returns a list of distinct GPU IDs for which telemetry exists.
func (p *PostgresStore) GPUs() ([]string, error) {
	rows, err := p.db.Query(`SELECT DISTINCT gpu_id FROM telemetry`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []string
	for rows.Next() {
		var gpu string
		rows.Scan(&gpu)
		res = append(res, gpu)
	}
	return res, nil
}

// Telemetry retrieves telemetry records for a specific GPU,
// optionally filtered by a start and/or end timestamp.
func (p *PostgresStore) Telemetry(gpu string, startTs, endTs *int64) ([]model.Telemetry, error) {
	query := `
	SELECT id, gpu_id, ts, metric, value, labels
	FROM telemetry
	WHERE gpu_id = $1
	`
	args := []any{gpu}
	argPos := 2

	if startTs != nil {
		query += " AND ts >= to_timestamp($" + itoa(argPos) + ")"
		args = append(args, *startTs)
		argPos++
	}
	if endTs != nil {
		query += " AND ts <= to_timestamp($" + itoa(argPos) + ")"
		args = append(args, *endTs)
	}

	query += " ORDER BY ts"

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Telemetry
	for rows.Next() {
		var t model.Telemetry
		rows.Scan(
			&t.ID,
			&t.GPUId,
			&t.Timestamp,
			&t.Metric,
			&t.Value,
			&t.Labels,
		)
		out = append(out, t)
	}
	return out, nil
}

// GPUExists checks whether any telemetry exists for the given GPU ID.
func (p *PostgresStore) GPUExists(gpuID string) (bool, error) {
	var exists bool
	err := p.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM telemetry WHERE gpu_id = $1
		)
	`, gpuID).Scan(&exists)

	return exists, err
}

// Ping verifies database connectivity.
func (p *PostgresStore) Ping() error {
	return p.db.Ping()
}

// Close releases all database resources and connections.
func (p *PostgresStore) Close() error {
	return p.db.Close()
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
