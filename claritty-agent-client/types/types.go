package types

type Metrics struct {
	CPU    float64 `json:"cpu"`    // cores
	Memory int     `json:"memory"` // MB
	Node   string  `json:"node"`
}

type Log struct {
	Pod  string `json:"pod"`
	Text string `json:"text"`
}
