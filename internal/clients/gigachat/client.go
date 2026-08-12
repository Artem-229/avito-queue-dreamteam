package gigachat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	authURL       = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	embeddingsURL = "https://api.giga.chat/v1/embeddings"
	defaultModel  = "Embeddings"
)

type Client struct {
	authKey    string
	scope      string
	httpClient *http.Client

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

type Config struct {
	AuthKey            string
	Scope              string
	InsecureSkipVerify bool
}

func NewClient(cfg Config) *Client {
	if cfg.Scope == "" {
		cfg.Scope = "GIGACHAT_API_PERS"
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
	}

	return &Client{
		authKey: cfg.AuthKey,
		scope:   cfg.Scope,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}
}

func (c *Client) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string]interface{}{
		"model": defaultModel,
		"input": []string{text},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, embeddingsURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", uuid.New().String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gigachat api error: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var responseData struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(responseData.Data) == 0 || len(responseData.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned from API")
	}

	return responseData.Data[0].Embedding, nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	isValid := c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt)
	token := c.token
	c.mu.RUnlock()

	if isValid {
		return token, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Add(1*time.Minute).Before(c.expiresAt) {
		return c.token, nil
	}

	return c.requestNewToken(ctx)
}

func (c *Client) requestNewToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("scope", c.scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Authorization", "Basic "+c.authKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.token = authResp.AccessToken
	c.expiresAt = time.Unix(authResp.ExpiresAt, 0)

	return c.token, nil
}
