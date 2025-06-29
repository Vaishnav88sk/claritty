package sender

import (
	"bytes"
	"encoding/json"
	"github.com/Vaishnav88sk/claritty/claritty-agent-client/types"
	"net/http"
)

// type Payload struct {
//     Metrics types.Metrics
//     Logs    []types.Log
// }

type Payload struct {
	Node   string   `json:"node"`
	CPU    float64  `json:"cpu"`
	Memory int      `json:"memory"`
	Logs   []string `json:"logs"`
}

// func SendData(url string, m types.Metrics, l []types.Log) {
//     payload := Payload{Metrics: m, Logs: l}
//     jsonData, _ := json.Marshal(payload)
//     http.Post(url, "application/json", bytes.NewBuffer(jsonData))
// }

func SendData(url string, m types.Metrics, l []types.Log) {
	logs := []string{}
	for _, log := range l {
		logs = append(logs, log.Text)
	}

	payload := Payload{
		Node:   m.Node,
		CPU:    m.CPU,
		Memory: m.Memory,
		Logs:   logs,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url+"/api/metrics", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		println("Error sending:", err.Error())
	} else {
		println("Sent metrics to backend, status:", resp.Status)
	}
}
