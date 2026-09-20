package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"humi/entity"
	"humi/internal/lib/token"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("not found")

// SQLite is the storage backend. SQLite is enough here: a handful of nodes
// posting a few times per hour produce well under a million rows per year.
type SQLite struct {
	db *sql.DB
}

// NewSQLite opens the database file, applies pragmas and runs pending migrations.
func NewSQLite(path string) (*SQLite, error) {
	dsn := fmt.Sprintf("file:%s?%s", path, strings.Join([]string{
		"_pragma=journal_mode(WAL)",
		"_pragma=synchronous(NORMAL)",
		"_pragma=busy_timeout(5000)",
		"_pragma=foreign_keys(ON)",
	}, "&"))

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One writer avoids SQLITE_BUSY entirely; the read volume here is trivial.
	db.SetMaxOpenConns(1)

	if err = db.Ping(); err != nil {
		return nil, err
	}
	if err = migrate(db); err != nil {
		return nil, err
	}

	return &SQLite{db: db}, nil
}

// Close releases the database handle.
func (s *SQLite) Close() error {
	return s.db.Close()
}

// CreateNode registers a node and returns the plaintext token, which is shown once.
func (s *SQLite) CreateNode(ctx context.Context, slug, name, location string, intervalS int) (string, error) {
	t, err := token.New()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO nodes (slug, name, location, token_hash, interval_s, created_at)
		 VALUES (?, ?, ?, ?, ?, unixepoch())`,
		slug, name, location, token.Hash(t), intervalS)
	if err != nil {
		return "", err
	}
	return t, nil
}

// NodeByToken resolves an ingest bearer token to an enabled node.
func (s *SQLite) NodeByToken(ctx context.Context, t string) (*entity.Node, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, location, interval_s, enabled, created_at, COALESCE(last_seen, 0)
		 FROM nodes WHERE token_hash = ? AND enabled = 1`, token.Hash(t))
	return scanNode(row)
}

// NodeBySlug returns a node by its slug.
func (s *SQLite) NodeBySlug(ctx context.Context, slug string) (*entity.Node, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, slug, name, location, interval_s, enabled, created_at, COALESCE(last_seen, 0)
		 FROM nodes WHERE slug = ?`, slug)
	return scanNode(row)
}

func scanNode(row *sql.Row) (*entity.Node, error) {
	var n entity.Node
	err := row.Scan(&n.ID, &n.Slug, &n.Name, &n.Location, &n.IntervalS, &n.Enabled, &n.CreatedAt, &n.LastSeen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// SaveReadings stores a batch and advances the node's last_seen.
func (s *SQLite) SaveReadings(ctx context.Context, nodeID int64, rs []entity.Reading) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO readings (node_id, received_at, rh, temp, vbat, rssi) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	var newest int64
	for _, r := range rs {
		if _, err = stmt.ExecContext(ctx, nodeID, r.ReceivedAt, r.RH, r.Temp, r.VBat, r.RSSI); err != nil {
			return err
		}
		if r.ReceivedAt > newest {
			newest = r.ReceivedAt
		}
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE nodes SET last_seen = MAX(COALESCE(last_seen, 0), ?) WHERE id = ?`, newest, nodeID); err != nil {
		return err
	}

	return tx.Commit()
}

// Nodes returns every node with its most recent reading attached.
func (s *SQLite) Nodes(ctx context.Context) ([]entity.NodeState, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT n.id, n.slug, n.name, n.location, n.interval_s, n.enabled, n.created_at, COALESCE(n.last_seen, 0),
		        r.received_at, r.rh, r.temp, r.vbat, r.rssi
		 FROM nodes n
		 LEFT JOIN readings r ON r.id = (
		     SELECT id FROM readings WHERE node_id = n.id ORDER BY received_at DESC LIMIT 1)
		 ORDER BY n.slug`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	states := make([]entity.NodeState, 0, 8)
	for rows.Next() {
		var st entity.NodeState
		var at sql.NullInt64
		var rh sql.NullFloat64
		var temp, vbat sql.NullFloat64
		var rssi sql.NullInt64

		if err = rows.Scan(&st.ID, &st.Slug, &st.Name, &st.Location, &st.IntervalS, &st.Enabled,
			&st.CreatedAt, &st.LastSeen, &at, &rh, &temp, &vbat, &rssi); err != nil {
			return nil, err
		}
		if at.Valid {
			st.Last = &entity.Reading{ReceivedAt: at.Int64, RH: rh.Float64}
			if temp.Valid {
				st.Last.Temp = &temp.Float64
			}
			if vbat.Valid {
				st.Last.VBat = &vbat.Float64
			}
			if rssi.Valid {
				v := int(rssi.Int64)
				st.Last.RSSI = &v
			}
		}
		states = append(states, st)
	}

	return states, rows.Err()
}

// Series aggregates readings of one node into fixed time buckets. The range is
// inclusive on both ends so the newest reading always shows up on the chart.
func (s *SQLite) Series(ctx context.Context, nodeID int64, from, to int64, bucketS int) ([]entity.Bucket, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT (received_at / ?) * ? AS bucket, COUNT(*),
		        MIN(rh), AVG(rh), MAX(rh),
		        MIN(temp), AVG(temp), MAX(temp),
		        MIN(vbat)
		 FROM readings
		 WHERE node_id = ? AND received_at BETWEEN ? AND ?
		 GROUP BY bucket ORDER BY bucket`,
		bucketS, bucketS, nodeID, from, to)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	points := make([]entity.Bucket, 0, 256)
	for rows.Next() {
		var b entity.Bucket
		var tMin, tAvg, tMax, vMin sql.NullFloat64

		if err = rows.Scan(&b.T, &b.Samples, &b.RHMin, &b.RHAvg, &b.RHMax,
			&tMin, &tAvg, &tMax, &vMin); err != nil {
			return nil, err
		}
		if tMin.Valid {
			b.TMin, b.TAvg, b.TMax = &tMin.Float64, &tAvg.Float64, &tMax.Float64
		}
		if vMin.Valid {
			b.VBatMin = &vMin.Float64
		}
		points = append(points, b)
	}

	return points, rows.Err()
}

// Stat reports the database file name for the health endpoint.
func (s *SQLite) Stat(ctx context.Context) (map[string]any, error) {
	var nodes, readings int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&nodes); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM readings`).Scan(&readings); err != nil {
		return nil, err
	}
	return map[string]any{"nodes": nodes, "readings": readings}, nil
}
