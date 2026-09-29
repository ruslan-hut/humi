package core

import (
	"errors"
	"strings"
	"testing"

	"humi/entity"
)

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		valid bool
	}{
		{"slug", validateSlug("bedroom-2"), true},
		{"slug uppercase", validateSlug("Bedroom"), false},
		{"slug leading dash", validateSlug("-bath"), false},
		{"slug too long", validateSlug(strings.Repeat("a", 33)), false},
		{"name", validateName("Спальня"), true},
		{"name empty", validateName(""), false},
		{"name 64 runes", validateName(strings.Repeat("я", 64)), true},
		{"name 65 runes", validateName(strings.Repeat("я", 65)), false},
		{"interval minimum", validateInterval(60), true},
		{"interval too short", validateInterval(59), false},
		{"interval too long", validateInterval(86401), false},
		{"username", validateUsername("ruslan.hut"), true},
		{"username too short", validateUsername("ru"), false},
		{"username with space", validateUsername("ruslan hut"), false},
		{"password", validatePassword("12345678"), true},
		{"password too short", validatePassword("1234567"), false},
		{"password over bcrypt limit", validatePassword(strings.Repeat("x", 73)), false},
		{"role viewer", validateRole("viewer"), true},
		{"role unknown", validateRole("root"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.err == nil) != tt.valid {
				t.Fatalf("err = %v, want valid %v", tt.err, tt.valid)
			}
			if tt.err != nil && !errors.Is(tt.err, entity.ErrInvalid) {
				t.Fatalf("err kind = %v, want ErrInvalid", tt.err)
			}
		})
	}
}

func TestNormalizeRules(t *testing.T) {
	rh := func(op string, v float64) entity.Rule {
		return entity.Rule{Metric: entity.MetricRH, Op: op, Threshold: v, ForMin: 60, Enabled: true}
	}

	tests := []struct {
		name    string
		in      []entity.Rule
		wantErr bool
	}{
		{"defaults-like set", []entity.Rule{rh("gt", 65), rh("lt", 30),
			{Metric: "vbat", Op: "lt", Threshold: 3.4, ForMin: 180}}, false},
		{"empty set", nil, false},
		{"rh above 100", []entity.Rule{rh("gt", 101)}, true},
		{"temp below range", []entity.Rule{{Metric: "temp", Op: "lt", Threshold: -60}}, true},
		{"vbat above range", []entity.Rule{{Metric: "vbat", Op: "lt", Threshold: 6}}, true},
		{"unknown metric", []entity.Rule{{Metric: "co2", Op: "gt", Threshold: 1}}, true},
		{"bad op", []entity.Rule{{Metric: "rh", Op: "ge", Threshold: 50}}, true},
		{"negative duration", []entity.Rule{{Metric: "rh", Op: "gt", Threshold: 50, ForMin: -1}}, true},
		{"duration over a day", []entity.Rule{{Metric: "rh", Op: "gt", Threshold: 50, ForMin: 1441}}, true},
		{"duplicate direction", []entity.Rule{rh("gt", 65), rh("gt", 70)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeRules(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeRulesClearsOfflineThreshold(t *testing.T) {
	out, err := normalizeRules([]entity.Rule{{ID: 9, Metric: "offline", Op: "lt", Threshold: 42, Enabled: true}})
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if r := out[0]; r.Op != entity.OpGT || r.Threshold != 0 || r.ID != 0 || !r.Enabled {
		t.Fatalf("offline rule = %+v, want gt 0 without id", r)
	}
}
