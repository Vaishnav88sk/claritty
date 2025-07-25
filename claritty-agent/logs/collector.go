package logs

import (
    "os"
    "github.com/Vaishnav88sk/claritty/claritty-agent/types"
)

func CollectLogs() []types.Log {
    logs := []types.Log{}
    files, _ := os.ReadDir("/var/log/containers/")
    for _, f := range files {
        data, err := os.ReadFile("/var/log/containers/" + f.Name())
        if err == nil {
            logs = append(logs, types.Log{
                Pod:  f.Name(),
                Text: string(data),
            })
        }
    }
    return logs
}
