package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type PythonClient struct {
	baseURL string
	client  *http.Client
}

type PythonAnalysisResult struct {
	Frame   int       `json:"frame"`
	Mode    string    `json:"mode"`
	Centers []float32 `json:"centers"`
}

func NewPythonClient() *PythonClient {
	baseURL := os.Getenv("PYTHON_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://python-app:5000"
	}
	return &PythonClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Minute}, // Long timeout for video processing
	}
}

func (c *PythonClient) ProcessVideo(filename string) ([]PythonAnalysisResult, error) {
	payload := map[string]string{"filename": filename}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/process", c.baseURL)
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to call python service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python service returned status: %d", resp.StatusCode)
	}

	var results []PythonAnalysisResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}
