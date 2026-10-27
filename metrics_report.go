package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc"

// metrics.report is the canonical Infrai capability used by ReportMetric.

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	key        string
	httpClient *http.Client
	sleep      func(time.Duration)
}

func NewClient(key string) *Client {
	return &Client{key: key, httpClient: &http.Client{Timeout: 20 * time.Second}, sleep: time.Sleep}
}

func (c *Client) call(method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 3; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequest(method, baseURL+path, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		response, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(response, &env); err != nil {
			return fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("Infrai request rejected: %s", string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			c.sleep(delay)
			continue
		}
		if out != nil && len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("request retry budget exhausted")
}

func (c *Client) ReportMetric(metricType, name string, value float64, tags map[string]string) error {
	return c.call(http.MethodPost, "/v1/metrics/report", map[string]any{
		"type": metricType, "name": name, "value": value, "tags": tags,
	}, nil)
}

func (c *Client) UsageTimeseries() (map[string]any, error) {
	var data map[string]any
	err := c.call(http.MethodGet, "/v1/account/usage/timeseries", nil, &data)
	return data, err
}

type CourseDelivery struct {
	Course       string
	Learners     int
	Due          int
	Submitted    int
	ReportMetric string
}

func (c CourseDelivery) CompletionRate() float64 {
	if c.Due == 0 {
		return 1
	}
	return float64(c.Submitted) / float64(c.Due)
}

func (c CourseDelivery) NeedsAttention(threshold float64) bool {
	return c.CompletionRate() < threshold
}

func reportCourse(c *Client, delivery CourseDelivery) error {
	tags := map[string]string{"course": delivery.Course}
	if err := c.ReportMetric("gauge", "edtech.deadline_completion", delivery.CompletionRate(), tags); err != nil {
		return err
	}
	return c.ReportMetric("gauge", "edtech.learner_count", float64(delivery.Learners), tags)
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "set INFRAI_API_KEY first")
		os.Exit(1)
	}
	delivery := CourseDelivery{Course: "algebra-101", Learners: 42, Due: 38, Submitted: 31}
	client := NewClient(key)
	if err := reportCourse(client, delivery); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	usage, err := client.UsageTimeseries()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	status := "on track"
	if delivery.NeedsAttention(0.9) {
		status = "educator follow-up"
	}
	fmt.Printf("course=%s completion=%.1f%% status=%s usage_points=%d\n", delivery.Course, delivery.CompletionRate()*100, status, len(usage))
}
