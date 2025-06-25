package main

import (
	"github.com/gin-gonic/gin"
	"github.com/vaishnav88sk/claritty/backend/db"
	"github.com/vaishnav88sk/claritty/backend/routes"
)

func main() {
	db.Init()

	r := gin.Default()
	r.POST("/api/metrics", routes.PostMetrics)
	r.GET("/api/metrics", routes.GetMetrics)

	r.Run(":8088") // Runs on port 8080
}
