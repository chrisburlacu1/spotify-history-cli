package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	tokenMutex   sync.RWMutex
	accessToken  string
	tokenExpiry  time.Time
}

type authResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func NewClient(clientID, clientSecret string) (*Client, error) {
	if clientID == "" {
		clientID = os.Getenv("SPOTIFY_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Spotify API credentials missing. Set SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables or pass via flags")
	}

	c := &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	if err := c.authenticate(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return c, nil
}

func (c *Client) authenticate() error {
	authHeader := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth error (%d): %s", resp.StatusCode, string(body))
	}

	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return err
	}

	c.tokenMutex.Lock()
	c.accessToken = authResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn-60) * time.Second)
	c.tokenMutex.Unlock()

	return nil
}

func (c *Client) getValidToken() (string, error) {
	c.tokenMutex.RLock()
	token := c.accessToken
	expiry := c.tokenExpiry
	c.tokenMutex.RUnlock()

	if token == "" || time.Now().After(expiry) {
		if err := c.authenticate(); err != nil {
			return "", err
		}
		c.tokenMutex.RLock()
		token = c.accessToken
		c.tokenMutex.RUnlock()
	}

	return token, nil
}

func (c *Client) Get(endpoint string, query url.Values) ([]byte, error) {
	reqURL := "https://api.spotify.com/v1" + endpoint
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	for retries := 0; retries < 4; retries++ {
		token, err := c.getValidToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return io.ReadAll(resp.Body)
		}

		// Handle Rate Limiting (HTTP 429)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfterSec := 2
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				if s, err := strconv.Atoi(retryHeader); err == nil && s > 0 {
					retryAfterSec = s
				}
			}
			resp.Body.Close()
			time.Sleep(time.Duration(retryAfterSec) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			_ = c.authenticate()
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("spotify API error (%d): %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded for %s", endpoint)
}
