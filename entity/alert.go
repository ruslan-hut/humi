package entity

// Metric names a value a rule can watch.
const (
	MetricRH      = "rh"
	MetricTemp    = "temp"
	MetricVBat    = "vbat"
	MetricOffline = "offline"
)

// Comparison operators of a rule.
const (
	OpGT = "gt"
	OpLT = "lt"
)

// Rule is a threshold that raises an alert while it holds.
type Rule struct {
	ID        int64   `json:"id"`
	NodeSlug  string  `json:"node,omitempty"` // empty applies to every node
	Metric    string  `json:"metric"`
	Op        string  `json:"op"` // gt | lt
	Threshold float64 `json:"threshold"`
	ForMin    int     `json:"for_min"`
	Enabled   bool    `json:"enabled"`
}

// Alert is one firing of a rule, open until ResolvedAt is set.
type Alert struct {
	ID         int64   `json:"id"`
	RuleID     int64   `json:"rule_id"`
	NodeSlug   string  `json:"node"`
	Metric     string  `json:"metric"`
	FiredAt    int64   `json:"fired_at"`
	ResolvedAt int64   `json:"resolved_at,omitempty"`
	Value      float64 `json:"value"`
}
