package request

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

var client = &http.Client{ Timeout: time.Second * 10 }

func SendRequest(method, url string, data map[string]interface{}) (int, []byte, error) {
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}
