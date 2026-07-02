package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"os"
	"app/graph/model"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"content"`
	UserID    model.MSSQLUUID    `json:"user.id"`
	Level     model.LogLevel  `json:"level"`
	Service   string    `json:"service.name"`
}

func SendLogToDynatrace(message string, level model.LogLevel, userID model.MSSQLUUID) {
	endpoint := os.Getenv("DT_SERVER_URL")
	token := os.Getenv("DT_API_TOKEN")
	if endpoint == "" || token == "" {
		fmt.Println("Dynatrace endpoint or API token not set in environment variables")
		return
	}
	url := fmt.Sprintf("%s/api/v2/logs/ingest", endpoint)
	fmt.Printf("Sending log to Dynatrace at %s with level %s: %s\n", url, level, message)
	logData := []LogEntry{
		{
			Timestamp: time.Now().UTC(),
			Message:   message,
			UserID:    userID,
			Level:     level,
			Service:   "mm-coverage-checklist-api",
		},
	}

	jsonData, _ := json.Marshal(logData)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Api-Token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	fmt.Printf("Request: %v\n", req)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending log to Dynatrace:", err)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Printf("Error sending log to Dynatrace: received status code %d\n", resp.StatusCode)
	}
	defer resp.Body.Close()
}
