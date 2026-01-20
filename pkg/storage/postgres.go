package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Storage provides database operations for FusionGuard
type Storage struct {
	db *sql.DB
}

// New creates a new Storage instance
func New(dsn string) (*Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// CreateShot creates or updates a shot record
func (s *Storage) CreateShot(ctx context.Context, shotID string, startedAt *time.Time) error {
	query := `
		INSERT INTO shots (shot_id, started_at)
		VALUES ($1, $2)
		ON CONFLICT (shot_id) DO UPDATE
		SET started_at = COALESCE(EXCLUDED.started_at, shots.started_at)
	`
	_, err := s.db.ExecContext(ctx, query, shotID, startedAt)
	return err
}

// FinishShot marks a shot as finished
func (s *Storage) FinishShot(ctx context.Context, shotID string, finishedAt time.Time) error {
	query := `UPDATE shots SET finished_at = $1 WHERE shot_id = $2`
	_, err := s.db.ExecContext(ctx, query, finishedAt, shotID)
	return err
}

// StoreTelemetryPoints stores telemetry points in batch
func (s *Storage) StoreTelemetryPoints(ctx context.Context, points []TelemetryPoint) error {
	if len(points) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO telemetry_points (shot_id, ts_unix_ns, channel_name, value, quality)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (shot_id, ts_unix_ns, channel_name) DO UPDATE
		SET value = EXCLUDED.value, quality = EXCLUDED.quality
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, p := range points {
		if _, err := stmt.ExecContext(ctx, p.ShotID, p.TsUnixNs, p.ChannelName, p.Value, p.Quality); err != nil {
			return fmt.Errorf("execute statement: %w", err)
		}
	}

	return tx.Commit()
}

// StoreRiskPoint stores a risk prediction point
func (s *Storage) StoreRiskPoint(ctx context.Context, rp RiskPoint) error {
	query := `
		INSERT INTO risks (shot_id, ts_unix_ns, risk_h50, risk_h200, model_version, calibration_version)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (shot_id, ts_unix_ns) DO UPDATE
		SET risk_h50 = EXCLUDED.risk_h50,
		    risk_h200 = EXCLUDED.risk_h200,
		    model_version = EXCLUDED.model_version,
		    calibration_version = EXCLUDED.calibration_version
	`
	_, err := s.db.ExecContext(ctx, query, rp.ShotID, rp.TsUnixNs, rp.RiskH50, rp.RiskH200, rp.ModelVersion, rp.CalibrationVersion)
	return err
}

// CreateEvent creates an event record
func (s *Storage) CreateEvent(ctx context.Context, event Event) (int64, error) {
	query := `
		INSERT INTO events (shot_id, ts_unix_ns, kind, message, severity)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id int64
	err := s.db.QueryRowContext(ctx, query, event.ShotID, event.TsUnixNs, event.Kind, event.Message, event.Severity).Scan(&id)
	return id, err
}

// ListShots returns a list of all shots
func (s *Storage) ListShots(ctx context.Context) ([]Shot, error) {
	query := `SELECT shot_id, started_at, finished_at FROM shots ORDER BY started_at DESC NULLS LAST`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shots []Shot
	for rows.Next() {
		var shot Shot
		var startedAt, finishedAt sql.NullTime
		if err := rows.Scan(&shot.ShotID, &startedAt, &finishedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			shot.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			shot.FinishedAt = &finishedAt.Time
		}
		shots = append(shots, shot)
	}

	return shots, rows.Err()
}

// GetRiskSeries returns risk predictions for a shot within a time range
func (s *Storage) GetRiskSeries(ctx context.Context, shotID string, fromUnixNs, toUnixNs *int64) (RiskSeries, error) {
	var query string
	var args []interface{}

	if fromUnixNs != nil && toUnixNs != nil {
		query = `
			SELECT shot_id, ts_unix_ns, risk_h50, risk_h200, model_version, calibration_version
			FROM risks
			WHERE shot_id = $1 AND ts_unix_ns >= $2 AND ts_unix_ns <= $3
			ORDER BY ts_unix_ns ASC
		`
		args = []interface{}{shotID, *fromUnixNs, *toUnixNs}
	} else {
		query = `
			SELECT shot_id, ts_unix_ns, risk_h50, risk_h200, model_version, calibration_version
			FROM risks
			WHERE shot_id = $1
			ORDER BY ts_unix_ns ASC
		`
		args = []interface{}{shotID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return RiskSeries{}, err
	}
	defer rows.Close()

	series := RiskSeries{ShotID: shotID}
	for rows.Next() {
		var rp RiskPoint
		var modelVersion, calibrationVersion sql.NullString
		if err := rows.Scan(&rp.ShotID, &rp.TsUnixNs, &rp.RiskH50, &rp.RiskH200, &modelVersion, &calibrationVersion); err != nil {
			return RiskSeries{}, err
		}
		if modelVersion.Valid {
			rp.ModelVersion = modelVersion.String
		}
		if calibrationVersion.Valid {
			rp.CalibrationVersion = calibrationVersion.String
		}
		series.Points = append(series.Points, rp)
	}

	return series, rows.Err()
}

// GetTelemetrySeries returns telemetry data for a shot within a time range
func (s *Storage) GetTelemetrySeries(ctx context.Context, shotID string, fromUnixNs, toUnixNs *int64) (TelemetrySeries, error) {
	var query string
	var args []interface{}

	if fromUnixNs != nil && toUnixNs != nil {
		query = `
			SELECT channel_name, ts_unix_ns, value
			FROM telemetry_points
			WHERE shot_id = $1 AND ts_unix_ns >= $2 AND ts_unix_ns <= $3
			ORDER BY channel_name, ts_unix_ns ASC
		`
		args = []interface{}{shotID, *fromUnixNs, *toUnixNs}
	} else {
		query = `
			SELECT channel_name, ts_unix_ns, value
			FROM telemetry_points
			WHERE shot_id = $1
			ORDER BY channel_name, ts_unix_ns ASC
		`
		args = []interface{}{shotID}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return TelemetrySeries{}, err
	}
	defer rows.Close()

	series := TelemetrySeries{
		ShotID:   shotID,
		Channels: make(map[string][]TelemetryChannelPoint),
	}

	for rows.Next() {
		var channelName string
		var point TelemetryChannelPoint
		if err := rows.Scan(&channelName, &point.TsUnixNs, &point.Value); err != nil {
			return TelemetrySeries{}, err
		}
		series.Channels[channelName] = append(series.Channels[channelName], point)
	}

	return series, rows.Err()
}

// GetEvents returns events for a shot
func (s *Storage) GetEvents(ctx context.Context, shotID string) ([]Event, error) {
	query := `
		SELECT id, shot_id, ts_unix_ns, kind, message, severity, created_at
		FROM events
		WHERE shot_id = $1
		ORDER BY ts_unix_ns ASC
	`
	rows, err := s.db.QueryContext(ctx, query, shotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.ShotID, &event.TsUnixNs, &event.Kind, &event.Message, &event.Severity, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

// GetRiskPointAt returns a risk point at a specific timestamp
func (s *Storage) GetRiskPointAt(ctx context.Context, shotID string, atUnixNs int64) (RiskPoint, error) {
	query := `
		SELECT shot_id, ts_unix_ns, risk_h50, risk_h200, model_version, calibration_version
		FROM risks
		WHERE shot_id = $1 AND ts_unix_ns <= $2
		ORDER BY ts_unix_ns DESC
		LIMIT 1
	`

	var rp RiskPoint
	var modelVersion, calibrationVersion sql.NullString
	err := s.db.QueryRowContext(ctx, query, shotID, atUnixNs).Scan(
		&rp.ShotID, &rp.TsUnixNs, &rp.RiskH50, &rp.RiskH200, &modelVersion, &calibrationVersion,
	)
	if err != nil {
		return RiskPoint{}, err
	}

	if modelVersion.Valid {
		rp.ModelVersion = modelVersion.String
	}
	if calibrationVersion.Valid {
		rp.CalibrationVersion = calibrationVersion.String
	}

	return rp, nil
}
