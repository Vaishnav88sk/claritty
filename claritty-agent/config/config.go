package config

import (
    "os"
    "time"
)

type Config struct {
    BackendURL string
    Interval   time.Duration
}

func LoadConfig() Config {
    backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8088" // fallback default
	}

    return Config{
        // BackendURL: "http://backend.default.svc.cluster.local:8088",
        // BackendURL: "http://localhost:8088", // for local testing
        BackendURL: backendURL,  // ec2 hosted backend
        Interval:   10 * time.Second,
    }
}
