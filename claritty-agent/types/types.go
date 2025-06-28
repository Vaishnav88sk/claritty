package types

type Metrics struct {
    CPU    float64 `json:"cpu"`
    Memory int     `json:"memory"`
    Node   string  `json:"node"`
}

type Log struct {
    Pod  string `json:"pod"`
    Text string `json:"text"`
}
