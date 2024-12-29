package datum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type apiTokenSource struct {
	APIToken string

	Hostname string
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`

	Message string `json:"message"`
}

func (s *apiTokenSource) Token() (*oauth2.Token, error) {
	client := http.DefaultClient

	url := fmt.Sprintf("https://%s/oauth/token/exchange", s.Hostname)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, fmt.Errorf("unexpected status code %d from token endpoint", resp.StatusCode)
	}

	var r tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New(r.Message)
	}

	if r.AccessToken == "" {
		return nil, fmt.Errorf("no access_token field returned by %s", url)
	}

	return &oauth2.Token{
		AccessToken: r.AccessToken,
		TokenType:   "Bearer",
	}, nil
}

func NewAPITokenSource(token, hostname string) oauth2.TokenSource {
	return &apiTokenSource{
		APIToken: token,
		Hostname: hostname,
	}
}
