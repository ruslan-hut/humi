package entity

// ReadingInput is a measurement as posted by a node.
type ReadingInput struct {
	RH   float64  `json:"rh"`
	Temp *float64 `json:"temp,omitempty"`
	VBat *float64 `json:"vbat,omitempty"`
	RSSI *int     `json:"rssi,omitempty"`
	AgeS int      `json:"age_s,omitempty"` // seconds before now, for readings buffered while offline
	FW   string   `json:"fw,omitempty"`
}

// Reading is a stored measurement. Time is assigned by the server, nodes have no RTC.
type Reading struct {
	ReceivedAt int64    `json:"received_at"`
	RH         float64  `json:"rh"`
	Temp       *float64 `json:"temp,omitempty"`
	VBat       *float64 `json:"vbat,omitempty"`
	RSSI       *int     `json:"rssi,omitempty"`
}
