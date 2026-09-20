package entity

// Bucket is one aggregated time slice of a series.
type Bucket struct {
	T       int64    `json:"t"`
	Samples int      `json:"n"`
	RHMin   float64  `json:"rh_min"`
	RHAvg   float64  `json:"rh_avg"`
	RHMax   float64  `json:"rh_max"`
	TMin    *float64 `json:"t_min,omitempty"`
	TAvg    *float64 `json:"t_avg,omitempty"`
	TMax    *float64 `json:"t_max,omitempty"`
	VBatMin *float64 `json:"vbat_min,omitempty"`
}

// Series is a bucketed history for one node.
type Series struct {
	Slug    string   `json:"slug"`
	From    int64    `json:"from"`
	To      int64    `json:"to"`
	BucketS int      `json:"bucket_s"`
	Points  []Bucket `json:"points"`
}
