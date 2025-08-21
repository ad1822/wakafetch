package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sahaj-b/wakafetch/types"
)

// Todo : This is not working for local
// Replace this with our SQLITE version
func fetchSummary(apiKey, apiURL string, days int) (*types.SummaryResponse, error) {
	apiURL = strings.TrimSuffix(apiURL, "/")
	today := time.Now()
	todayDate := today.Format("2006-01-02")
	startDate := today.AddDate(0, 0, -days+1).Format("2006-01-02")
	requestURL := fmt.Sprintf("%s/v1/users/current/summaries?start=%s&end=%s", apiURL, startDate, todayDate)
	if strings.HasSuffix(apiURL, "/v1") {
		requestURL = fmt.Sprintf("%s/users/current/summaries?start=%s&end=%s", apiURL, startDate, todayDate)
	}
	response, err := fetchApi[types.SummaryResponse](apiKey, requestURL)
	fmt.Println("Respose : ", response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	return response, nil
}

// fetchSQLiteSummary queries heartbeats and returns counts per day
func fetchSQLiteSummary(userID string, from, to time.Time) (*types.SummaryResponse, error) {
	db, err := sql.Open("sqlite3", "/home/ad/.local/share/wakapi/wakapi_db.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Count heartbeats per day
	query := `
		SELECT strftime('%Y-%m-%d', time) as day, COUNT(*) 
		FROM heartbeats 
		WHERE user_id = ? 
		  AND time >= ? 
		  AND time < ? 
		GROUP BY day
		ORDER BY day ASC;
	`
	rows, err := db.Query(query, userID, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build SummaryResponse
	summary := &types.SummaryResponse{}
	var totalSeconds float64
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}

		// Each heartbeat is ~1 second (adjust if your schema is different)
		totalSeconds += float64(count)

		// Build a DayData record
		var dayData types.DayData
		dayData.Range.Date = day
		dayData.GrandTotal.TotalSeconds = float64(count)
		dayData.GrandTotal.Text = fmt.Sprintf("%d secs", count)
		dayData.GrandTotal.Digital = fmt.Sprintf("0:%02d", count) // crude formatting
		dayData.GrandTotal.Hours = 0
		dayData.GrandTotal.Minutes = count

		summary.Data = append(summary.Data, dayData)
	}

	// Fill cumulative total
	summary.CumulativeTotal.Seconds = totalSeconds
	summary.CumulativeTotal.Text = fmt.Sprintf("%.0f secs", totalSeconds)
	summary.CumulativeTotal.Digital = fmt.Sprintf("0:%02.0f", totalSeconds)

	summary.Start = from.Format("2006-01-02")
	summary.End = to.Format("2006-01-02")

	return summary, rows.Err()
}

func fetchStats(apiKey, apiURL, rangeStr string) (*types.StatsResponse, error) {
	apiURL = strings.TrimSuffix(apiURL, "/")
	requestURL := fmt.Sprintf("%s/v1/users/current/stats/%s", apiURL, rangeStr)
	if strings.HasSuffix(apiURL, "/v1") {
		requestURL = fmt.Sprintf("%s/users/current/stats/%s", apiURL, rangeStr)
	}
	// fmt.Println(requestURL)
	response, err := fetchApi[types.StatsResponse](apiKey, requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats: %w", err)
	}
	return response, nil
}

func fetchApi[T any](apiKey, requestURL string) (*T, error) {
	const timeout = 10 * time.Second
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	encodedKey := base64.StdEncoding.EncodeToString([]byte(apiKey))
	req.Header.Set("Authorization", "Basic "+encodedKey)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
			return nil, fmt.Errorf("Request timed out after %s while contacting server", timeout)
		}
		return nil, fmt.Errorf("Unable to reach server. Check your internet connection")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("Authentication failed (401). Check your API key")
		case http.StatusForbidden:
			return nil, fmt.Errorf("Access forbidden (403). Your API key might not have permission")
		case http.StatusNotFound:
			return nil, fmt.Errorf("Endpoint not found (404). Verify the API URL")
		case http.StatusTooManyRequests:
			return nil, fmt.Errorf("Rate limit exceeded (429). Please try again later")
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return nil, fmt.Errorf("Server unavailable (%s). Please try again later", resp.Status)
		default:
			return nil, fmt.Errorf("Api request failed: %s", resp.Status)
		}
	}

	var apiResponse T
	if err = json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("Invalid response from server (failed to decode JSON)")
	}

	return &apiResponse, nil
}
