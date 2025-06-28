// package metrics

// import (
//     "encoding/json"
//     "net/http"
//     "github.com/Vaishnav88sk/claritty/claritty-agent/types"
// )

// func CollectNodeMetrics() types.Metrics {
//     resp, err := http.Get("https://127.0.0.1:10250/stats/summary")
//     if err != nil {
//         return types.Metrics{}
//     }
//     defer resp.Body.Close()

//     var data map[string]interface{}
//     json.NewDecoder(resp.Body).Decode(&data)

//     // Parse actual CPU/Memory here
//     return types.Metrics{
//         CPU: 0.42,  // dummy
//         Memory: 128,
//     }
// }


package metrics

import (
    "crypto/tls"
    "encoding/json"
    "io/ioutil"
    "log"
    "net/http"

    "github.com/Vaishnav88sk/claritty/claritty-agent/types"
)

func CollectNodeMetrics() types.Metrics {
    // Custom HTTP client that skips TLS verification (dev only!)
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ⚠️ for dev only
        },
    }

    req, err := http.NewRequest("GET", "http://127.0.0.1:10255/stats/summary", nil)
    if err != nil {
        log.Println("Failed to create request:", err)
        return types.Metrics{}
    }

    resp, err := client.Do(req)
    if err != nil {
        log.Println("Failed to fetch stats summary:", err)
        return types.Metrics{}
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)

    var summary struct {
        Node struct {
            NodeName string `json:"nodeName"`
            CPU struct {
                UsageNanoCores uint64 `json:"usageNanoCores"`
            } `json:"cpu"`
            Memory struct {
                UsageBytes uint64 `json:"usageBytes"`
            } `json:"memory"`
        } `json:"node"`
    }

    err = json.Unmarshal(body, &summary)
    if err != nil {
        log.Println("Failed to parse summary:", err)
        return types.Metrics{}
    }

    return types.Metrics{
        CPU:    float64(summary.Node.CPU.UsageNanoCores) / 1e9,  // Convert to cores
        Memory: int(summary.Node.Memory.UsageBytes / 1024 / 1024), // Convert to MB
        Node:   summary.Node.NodeName,
    }
}
