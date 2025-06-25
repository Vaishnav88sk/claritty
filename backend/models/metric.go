package models

import "time"

type Metric struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Node      string    `json:"node"`
	CPU       float64   `json:"cpu"`
	Memory    int       `json:"memory"`
	Logs      string    `json:"logs"`
}
