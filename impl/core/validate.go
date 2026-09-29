package core

import (
	"regexp"
	"strings"

	"humi/entity"
)

var (
	slugRe     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
	usernameRe = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)
)

const (
	minIntervalS = 60
	maxIntervalS = 86400
	maxNameLen   = 64
	minPassword  = 8
	maxPassword  = 72 // bcrypt ignores everything past 72 bytes
	maxForMin    = 1440
)

func validateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return entity.Errorf(entity.ErrInvalid, "slug must be 1–32 lowercase letters, digits or dashes")
	}
	return nil
}

func validateName(name string) error {
	if name == "" || len([]rune(name)) > maxNameLen {
		return entity.Errorf(entity.ErrInvalid, "name must be 1–%d characters", maxNameLen)
	}
	return nil
}

func validateLocation(loc string) error {
	if len([]rune(loc)) > maxNameLen {
		return entity.Errorf(entity.ErrInvalid, "location must be at most %d characters", maxNameLen)
	}
	return nil
}

func validateInterval(s int) error {
	if s < minIntervalS || s > maxIntervalS {
		return entity.Errorf(entity.ErrInvalid, "interval must be %d–%d seconds", minIntervalS, maxIntervalS)
	}
	return nil
}

func validateUsername(u string) error {
	if !usernameRe.MatchString(u) {
		return entity.Errorf(entity.ErrInvalid, "username must be 3–32 letters, digits, dots, dashes or underscores")
	}
	return nil
}

func validatePassword(p string) error {
	if len(p) < minPassword || len(p) > maxPassword {
		return entity.Errorf(entity.ErrInvalid, "password must be %d–%d bytes", minPassword, maxPassword)
	}
	return nil
}

func validateRole(role string) error {
	if role != entity.RoleAdmin && role != entity.RoleViewer {
		return entity.Errorf(entity.ErrInvalid, "role must be admin or viewer")
	}
	return nil
}

// ruleRange is the accepted threshold span per metric.
var ruleRange = map[string][2]float64{
	entity.MetricRH:   {0, 100},
	entity.MetricTemp: {-50, 100},
	entity.MetricVBat: {2, 5},
}

// normalizeRules validates a rule set and returns it cleaned up: offline rules
// carry no threshold, and each metric/direction pair appears at most once.
func normalizeRules(in []entity.Rule) ([]entity.Rule, error) {
	out := make([]entity.Rule, 0, len(in))
	seen := make(map[string]bool, len(in))

	for _, r := range in {
		r.Metric = strings.TrimSpace(r.Metric)
		if r.Op != entity.OpGT && r.Op != entity.OpLT {
			return nil, entity.Errorf(entity.ErrInvalid, "op must be gt or lt")
		}
		if r.Metric == entity.MetricOffline {
			r.Op, r.Threshold = entity.OpGT, 0
		} else {
			span, ok := ruleRange[r.Metric]
			if !ok {
				return nil, entity.Errorf(entity.ErrInvalid, "unknown metric %q", r.Metric)
			}
			if r.Threshold < span[0] || r.Threshold > span[1] {
				return nil, entity.Errorf(entity.ErrInvalid, "%s threshold must be %g…%g", r.Metric, span[0], span[1])
			}
		}
		if r.ForMin < 0 || r.ForMin > maxForMin {
			return nil, entity.Errorf(entity.ErrInvalid, "duration must be 0–%d minutes", maxForMin)
		}

		key := r.Metric + " " + r.Op
		if seen[key] {
			return nil, entity.Errorf(entity.ErrInvalid, "more than one %s rule", key)
		}
		seen[key] = true

		out = append(out, entity.Rule{Metric: r.Metric, Op: r.Op, Threshold: r.Threshold, ForMin: r.ForMin, Enabled: r.Enabled})
	}
	return out, nil
}
