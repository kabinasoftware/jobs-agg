package util

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type exchangeRate struct {
	rate      float64
	timestamp time.Time
}

var (
	rateCache     = make(map[string]exchangeRate)
	cacheMux      sync.RWMutex
	cacheDuration = 1 * time.Hour
)

func GetExchangeRate(from, to string) (float64, error) {
	from = strings.ToLower(from)

	cacheMux.RLock()
	if cached, ok := rateCache[from]; ok {
		if time.Since(cached.timestamp) < cacheDuration {
			cacheMux.RUnlock()
			return cached.rate, nil
		}
	}
	cacheMux.RUnlock()

	rate, err := fetchExchangeRate(from)
	if err != nil {
		return 0, err
	}

	cacheMux.Lock()
	rateCache[from] = exchangeRate{
		rate:      rate,
		timestamp: time.Now(),
	}
	cacheMux.Unlock()

	return rate, nil
}

func fetchExchangeRate(from string) (float64, error) {
	url := fmt.Sprintf("https://api.nbp.pl/api/exchangerates/rates/a/%s/?format=json", from)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected content type: %s, body: %s", contentType, string(body))
	}

	var result struct {
		Rates []struct {
			Mid float64 `json:"mid"`
		} `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode exchange rate response: %w", err)
	}

	if len(result.Rates) == 0 {
		return 0, fmt.Errorf("no exchange rate data available")
	}

	return result.Rates[0].Mid, nil
}
