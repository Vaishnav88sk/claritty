package db

import (
	"github.com/glebarez/sqlite"
	"github.com/vaishnav88sk/claritty/backend/models"
	"gorm.io/gorm"
	// _ "modernc.org/sqlite"
)

var DB *gorm.DB

func Init() {
	database, err := gorm.Open(sqlite.Open("/tmp/metrics.db"), &gorm.Config{})
	// database, err := gorm.Open(sqlite.Open("file:/tmp/metrics.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout=10000"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to DB" + err.Error())
	}

	database.AutoMigrate(&models.Metric{})
	DB = database
}
