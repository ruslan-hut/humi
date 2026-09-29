package core

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"humi/entity"
	"humi/internal/config"
	"humi/internal/lib/clock"
)

// stubDB fakes the ingest and series paths; the embedded interface panics on
// anything else, which the integration tests in internal/database cover.
type stubDB struct {
	Database
	saved []entity.Reading
	calls []struct {
		from, to int64
		bucketS  int
	}
}

func (s *stubDB) NodeByToken(context.Context, string) (*entity.Node, error) { return nil, nil }

func (s *stubDB) NodeBySlug(context.Context, string) (*entity.Node, error) {
	return &entity.Node{ID: 1, Slug: "bedroom", IntervalS: 900}, nil
}

func (s *stubDB) Nodes(context.Context) ([]entity.NodeState, error) { return nil, nil }

func (s *stubDB) SaveReadings(_ context.Context, _ int64, rs []entity.Reading) error {
	s.saved = append(s.saved, rs...)
	return nil
}

func (s *stubDB) Series(_ context.Context, _ int64, from, to int64, bucketS int) ([]entity.Bucket, error) {
	s.calls = append(s.calls, struct {
		from, to int64
		bucketS  int
	}{from, to, bucketS})
	return nil, nil
}

func (s *stubDB) Stat(context.Context) (map[string]any, error) { return nil, nil }

func testCore(db Database) *Core {
	conf := &config.Config{}
	conf.Ingest.MaxBatch = 4
	conf.Ingest.MaxAgeS = 3600
	conf.Ingest.OfflineRatio = 3
	return New(db, conf, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func f(v float64) *float64 { return &v }

func TestIngestValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      []entity.ReadingInput
		want    int
		wantErr bool
	}{
		{"single reading", []entity.ReadingInput{{RH: 54.2, Temp: f(21.8)}}, 1, false},
		{"batch within limit", []entity.ReadingInput{{RH: 50}, {RH: 51}, {RH: 52}}, 3, false},
		{"empty batch", nil, 0, true},
		{"batch over limit", make([]entity.ReadingInput, 5), 0, true},
		{"rh above range", []entity.ReadingInput{{RH: 101}}, 0, true},
		{"rh below range", []entity.ReadingInput{{RH: -1}}, 0, true},
		{"temp out of range", []entity.ReadingInput{{RH: 50, Temp: f(140)}}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &stubDB{}
			n, err := testCore(db).Ingest(context.Background(), &entity.Node{ID: 1}, tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if n != tt.want {
				t.Errorf("stored = %d, want %d", n, tt.want)
			}
		})
	}
}

func TestIngestBackdatesByAge(t *testing.T) {
	tests := []struct {
		name    string
		ageS    int
		wantOff int64
	}{
		{"no age", 0, 0},
		{"buffered 15 min", 900, 900},
		{"negative age ignored", -10, 0},
		{"age over limit ignored", 999_999, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &stubDB{}
			before := clock.Unix()
			if _, err := testCore(db).Ingest(context.Background(), &entity.Node{ID: 1},
				[]entity.ReadingInput{{RH: 50, AgeS: tt.ageS}}); err != nil {
				t.Fatalf("ingest: %v", err)
			}
			if len(db.saved) != 1 {
				t.Fatalf("saved %d readings, want 1", len(db.saved))
			}

			off := clock.Unix() - db.saved[0].ReceivedAt
			if off < tt.wantOff || off > tt.wantOff+2 {
				t.Errorf("backdated by %ds, want %ds", off, tt.wantOff)
			}
			if db.saved[0].ReceivedAt < before-tt.wantOff-2 {
				t.Errorf("received_at %d predates the call", db.saved[0].ReceivedAt)
			}
		})
	}
}

func TestSeriesWidensBucket(t *testing.T) {
	tests := []struct {
		name     string
		from, to int64
		bucketS  int
		want     int
	}{
		{"fits under the cap", 0, 24 * 3600, 900, 900},
		{"zero falls back to default", 0, 3600, 0, 300},
		{"30 days at 5 min is widened", 0, 30 * 24 * 3600, 300, 5401},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &stubDB{}
			s, err := testCore(db).Series(context.Background(), "bedroom", tt.from, tt.to, tt.bucketS)
			if err != nil {
				t.Fatalf("series: %v", err)
			}
			if s.BucketS != tt.want {
				t.Errorf("bucket = %d, want %d", s.BucketS, tt.want)
			}
			if got := (tt.to - tt.from) / int64(s.BucketS); got > maxPoints {
				t.Errorf("%d points exceeds the cap of %d", got, maxPoints)
			}
		})
	}
}

func TestSeriesRejectsEmptyRange(t *testing.T) {
	if _, err := testCore(&stubDB{}).Series(context.Background(), "bedroom", 100, 100, 300); err == nil {
		t.Fatal("expected an error for an empty range")
	}
}
