package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPGetSkill fetches a URL and returns the response body.
type HTTPGetSkill struct{}

func (h HTTPGetSkill) Name() string { return "http_get" }

func (h HTTPGetSkill) Description() string {
	return "Fetch a URL and return the response body (text only, 10s timeout, 8KB max). Usage: [!http_get: https://example.com]"
}

func (h HTTPGetSkill) Safety() SafetyLevel { return SafetyReadOnly }

func (h HTTPGetSkill) Execute(ctx context.Context, args string) (string, error) {
	url := strings.TrimSpace(args)
	if url == "" {
		return "", fmt.Errorf("http_get: no URL provided")
	}

	// Minimal validation.
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("http_get: bad URL: %w", err)
	}
	req.Header.Set("User-Agent", "Clawkido/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_get: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read up to 8KB.
	const maxBytes = 8192
	limited := io.LimitReader(resp.Body, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("http_get: read failed: %w", err)
	}

	result := string(body)
	if len(body) > maxBytes {
		result = result[:maxBytes] + "\n...(truncated at 8KB)"
	}

	return fmt.Sprintf("[%d %s]\n%s", resp.StatusCode, resp.Status, result), nil
}
