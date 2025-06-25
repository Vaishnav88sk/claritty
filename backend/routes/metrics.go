package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vaishnav88sk/claritty/backend/db"
	"github.com/vaishnav88sk/claritty/backend/models"
	"net/http"
	"strings"
	"time"
)

type MetricInput struct {
	Node   string   `json:"node"`
	CPU    float64  `json:"cpu"`
	Memory int      `json:"memory"`
	Logs   []string `json:"logs"`
}

func PostMetrics(c *gin.Context) {
	var input MetricInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metric := models.Metric{
		Timestamp: time.Now(),
		Node:      input.Node,
		CPU:       input.CPU,
		Memory:    input.Memory,
		Logs:      joinLogs(input.Logs),
	}

	db.DB.Create(&metric)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func GetMetrics(c *gin.Context) {
	var metrics []models.Metric
	db.DB.Order("timestamp desc").Limit(50).Find(&metrics)
	c.JSON(http.StatusOK, metrics)
}

func joinLogs(logs []string) string {
	return strings.Join(logs, "\n")
}
