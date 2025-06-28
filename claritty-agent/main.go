package main

import (
    // "fmt"
    "time"
    "github.com/Vaishnav88sk/claritty/claritty-agent/config"
    "github.com/Vaishnav88sk/claritty/claritty-agent/metrics"
    "github.com/Vaishnav88sk/claritty/claritty-agent/logs"
    "github.com/Vaishnav88sk/claritty/claritty-agent/sender"
)

func main() {
    cfg := config.LoadConfig()

    for {
        // fmt.Println("Hello")   -- for testing -- prints 'hello' after every 10 secs
        // 1. Collect metrics
        nodeMetrics := metrics.CollectNodeMetrics()

        // 2. Collect logs
        containerLogs := logs.CollectLogs()

        // 3. Send to backend
        sender.SendData(cfg.BackendURL, nodeMetrics, containerLogs)

        time.Sleep(cfg.Interval)
    }
}
