package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vaishnav88sk/claritty/backend/db"
	"github.com/vaishnav88sk/claritty/backend/routes"
	"time"
)

func main() {
	db.Init()

	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://127.0.0.1:5500"},
        AllowMethods:     []string{"GET", "POST"},
        AllowHeaders:     []string{"Origin", "Content-Type"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))

	r.POST("/api/metrics", routes.PostMetrics)
	r.GET("/api/metrics", routes.GetMetrics)

	r.Run(":8088") // Runs on port 8080
}
