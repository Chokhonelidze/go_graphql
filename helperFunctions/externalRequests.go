package helperFunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"io"
	"strings"
	"time"
)


func MakeHTTPRequest(client *http.Client, method, url, body string, headers map[string]string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	maxRetries := 3
	var response *http.Response
	var err error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %v", err)
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		response, err = client.Do(req)
		if err == nil && response.StatusCode < 500 {
			return response, nil
		}
		if attempt < maxRetries {
			fmt.Printf("Attempt %d failed, retrying in 2 seconds...\n", attempt)
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request after %d attempts: %v", maxRetries, err)
	}
	return response, nil
}


func InsertNotification(client *http.Client,ctx context.Context, message string, message_type string, link string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	jwtToken, ok := ctx.Value("jwtToken").(string)
	if !ok {
		fmt.Printf("jwtToken not found or invalid in context\n")
		return fmt.Errorf("jwtToken not found or invalid in context")
	}
	userToken, ok := ctx.Value("userToken").(string)
	if !ok {
		fmt.Printf("userToken not found or invalid in context\n")
		return fmt.Errorf("userToken not found or invalid in context")
	}
	notificationAPIURL := os.Getenv("NOTIFICATION_API_URL")
	if notificationAPIURL == "" {
		fmt.Printf("NOTIFICATION_API_URL not found or invalid in context\n")
		return fmt.Errorf("NOTIFICATION_API_URL not found or invalid in context")
	}
	
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization":"Bearer " + jwtToken,
		"user_token": userToken,
	}
	q := `mutation InsertNotification($notification: notification_input!) {
		insertNotification(notification: $notification) {
			notification {
			id
			message_type
			type
			title
			time
			short
			long
			link
			notification_type
			}
			success
			errors
		}
		}`
	var link_url string
	if(link != "") {
		link_url = "/mmCoverageChecklist/" + link
	} 
	variables := map[string]interface{}{
		"notification": map[string]interface{}{
			"message_type": "global",
			"type": message_type,
			"title": "Coverage Checklist Notification",
			"time": time.Now().Format(time.RFC3339),
			"short": message,
			"long": message,
			"link": link_url,
			"notification_type": "notification",
		},
	}
	payload := map[string]interface{}{
		"query":     q,
		"variables": variables,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshaling payload: %v\n", err)
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	fmt.Printf("Notification API request payload: %s\n", string(payloadBytes))
	response, err := http.NewRequestWithContext(ctx, "POST", notificationAPIURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		fmt.Printf("Error creating HTTP request: %v\n", err)
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	for key, value := range headers {
		fmt.Printf("Setting header: %s=%s\n", key, value)
		response.Header.Set(key, value)
	}
	resp, err := client.Do(response)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	fmt.Printf("Notification API response status: %s\n", resp.Status)
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %v\n", err)
		return fmt.Errorf("failed to read response body: %v", err)
	}
	fmt.Printf("Notification API response body: %s\n", string(respBody))
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		return fmt.Errorf("failed to make HTTP request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Received non-OK response from notification API: %s\n", resp.Status)
		return fmt.Errorf("received non-OK response from notification API: %s", resp.Status)
	}
	return nil
}
   