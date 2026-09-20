package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"humi/entity"
)

func newTestDB(t *testing.T) *SQLite {
	t.Helper()
	// Integration tests run against a real SQLite file, never a mock.
	db, err := NewSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestNodeByToken(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	token, err := db.CreateNode(ctx, "bedroom", "Bedroom", "1st floor", 900)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{"issued token resolves", token, nil},
		{"unknown token rejected", "deadbeef", ErrNotFound},
		{"empty token rejected", "", ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := db.NodeByToken(ctx, tt.token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && node.Slug != "bedroom" {
				t.Fatalf("slug = %q, want bedroom", node.Slug)
			}
		})
	}
}

func TestSaveReadingsAdvancesLastSeen(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	if _, err := db.CreateNode(ctx, "cellar", "Cellar", "", 900); err != nil {
		t.Fatalf("create node: %v", err)
	}
	node, err := db.NodeBySlug(ctx, "cellar")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	temp := 14.2
	// Out of order on purpose: last_seen must track the newest, not the last written.
	err = db.SaveReadings(ctx, node.ID, []entity.Reading{
		{ReceivedAt: 2_000, RH: 71.0, Temp: &temp},
		{ReceivedAt: 1_000, RH: 70.0, Temp: &temp},
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	states, err := db.Nodes(ctx)
	if err != nil {
		t.Fatalf("nodes: %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("got %d nodes, want 1", len(states))
	}
	if states[0].LastSeen != 2_000 {
		t.Errorf("last_seen = %d, want 2000", states[0].LastSeen)
	}
	if states[0].Last == nil || states[0].Last.RH != 71.0 {
		t.Errorf("latest reading = %+v, want rh 71", states[0].Last)
	}
}

func TestSeriesBuckets(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	if _, err := db.CreateNode(ctx, "bath", "Bath", "", 900); err != nil {
		t.Fatalf("create node: %v", err)
	}
	node, err := db.NodeBySlug(ctx, "bath")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}

	// Four readings inside one 3600s bucket, one in the next.
	rs := []entity.Reading{
		{ReceivedAt: 3_600, RH: 60},
		{ReceivedAt: 4_000, RH: 64},
		{ReceivedAt: 5_000, RH: 62},
		{ReceivedAt: 6_000, RH: 58},
		{ReceivedAt: 7_300, RH: 50},
	}
	if err = db.SaveReadings(ctx, node.ID, rs); err != nil {
		t.Fatalf("save: %v", err)
	}

	points, err := db.Series(ctx, node.ID, 0, 10_000, 3600)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("got %d buckets, want 2", len(points))
	}
	first := points[0]
	if first.T != 3600 || first.Samples != 4 {
		t.Errorf("bucket = %d with %d samples, want 3600 with 4", first.T, first.Samples)
	}
	if first.RHMin != 58 || first.RHMax != 64 {
		t.Errorf("rh range = %v–%v, want 58–64", first.RHMin, first.RHMax)
	}
	if first.RHAvg != 61 {
		t.Errorf("rh avg = %v, want 61", first.RHAvg)
	}
}

func TestSeriesIncludesNewestReading(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)

	if _, err := db.CreateNode(ctx, "hall", "Hall", "", 900); err != nil {
		t.Fatalf("create node: %v", err)
	}
	node, err := db.NodeBySlug(ctx, "hall")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if err = db.SaveReadings(ctx, node.ID, []entity.Reading{{ReceivedAt: 5_000, RH: 44}}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// The dashboard asks for [from, now]; a half-open range would drop the
	// reading that arrived exactly at now.
	points, err := db.Series(ctx, node.ID, 0, 5_000, 300)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("got %d buckets, want the reading at the upper bound", len(points))
	}
}
