package entity

// Node is a single measuring device.
type Node struct {
	ID        int64  `json:"-"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Location  string `json:"location,omitempty"`
	IntervalS int    `json:"interval_s"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
	LastSeen  int64  `json:"last_seen,omitempty"`
}

// NodeState is a node together with its most recent reading, as shown on the dashboard.
type NodeState struct {
	Node
	Online bool     `json:"online"`
	Last   *Reading `json:"last,omitempty"`
	RHLow  *float64 `json:"rh_low,omitempty"`  // threshold of the enabled "rh lt" rule
	RHHigh *float64 `json:"rh_high,omitempty"` // threshold of the enabled "rh gt" rule
}

// NodeInput registers a node.
type NodeInput struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	IntervalS int    `json:"interval_s"`
}

// NodePatch changes the settings of a node; nil fields stay as they are.
type NodePatch struct {
	Name      *string `json:"name"`
	Location  *string `json:"location"`
	IntervalS *int    `json:"interval_s"`
	Enabled   *bool   `json:"enabled"`
}
