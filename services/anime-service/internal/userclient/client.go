package userclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"
)

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

func New(rawURL string) (*Client, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseURL: u,
		HTTPClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}, nil
}

func (c *Client) GetUserByID(id string, headers http.Header) (*User, error) {
	rel := &url.URL{Path: path.Join("/internal/users", id)}
	u := c.BaseURL.ResolveReference(rel)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	// прокидываем корреляционные заголовки при необходимости
	if reqID := headers.Get("X-Request-ID"); reqID != "" {
		req.Header.Set("X-Request-ID", reqID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var uresp User
	if err := json.NewDecoder(resp.Body).Decode(&uresp); err != nil {
		return nil, err
	}
	return &uresp, nil
}
