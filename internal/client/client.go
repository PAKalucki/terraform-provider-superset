package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginProvider = "db"
	loginPath            = "/api/v1/security/login"
	csrfTokenPath        = "/api/v1/security/csrf_token/"
)

type Config struct {
	Endpoint    string
	Username    string
	Password    string
	AccessToken string
	HTTPClient  *http.Client
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	username   string
	password   string

	mu          sync.Mutex
	accessToken string
	csrfToken   string

	maxPaginationPages int
}

type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("superset API %s %s returned status %d", e.Method, e.Path, e.StatusCode)
	}

	return fmt.Sprintf("superset API %s %s returned status %d: %s", e.Method, e.Path, e.StatusCode, body)
}

func New(config Config) (*Client, error) {
	endpoint, err := normalizeEndpoint(config.Endpoint)
	if err != nil {
		return nil, err
	}

	accessToken := strings.TrimSpace(config.AccessToken)
	username := strings.TrimSpace(config.Username)
	password := strings.TrimSpace(config.Password)

	if accessToken == "" && (username == "" || password == "") {
		return nil, errors.New("authentication requires either access_token or username and password")
	}

	if accessToken != "" && (username != "" || password != "") {
		return nil, errors.New("authentication requires either access_token or username and password, not both")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	httpClient = cloneHTTPClient(httpClient)

	return &Client{
		baseURL:     endpoint,
		httpClient:  httpClient,
		username:    username,
		password:    password,
		accessToken: accessToken,
	}, nil
}

func (c *Client) Endpoint() string {
	return c.baseURL.String()
}

func (c *Client) AccessToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.accessToken
}

func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" {
		return nil
	}

	var loginResp struct {
		AccessToken string `json:"access_token"`
	}

	loginReq := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Provider string `json:"provider"`
	}{
		Username: c.username,
		Password: c.password,
		Provider: defaultLoginProvider,
	}

	if err := c.execute(ctx, http.MethodPost, loginPath, loginReq, &loginResp, "", false, ""); err != nil {
		return fmt.Errorf("authenticate with Superset API: %w", err)
	}

	if strings.TrimSpace(loginResp.AccessToken) == "" {
		return errors.New("authenticate with Superset API: empty access token in response")
	}

	c.accessToken = loginResp.AccessToken

	return nil
}

func (c *Client) ensureCSRFToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.csrfToken != "" {
		return c.csrfToken, nil
	}

	var csrfResp struct {
		Result string `json:"result"`
	}

	if err := c.execute(ctx, http.MethodGet, csrfTokenPath, nil, &csrfResp, c.accessToken, true, ""); err != nil {
		return "", fmt.Errorf("retrieve Superset CSRF token: %w", err)
	}

	if strings.TrimSpace(csrfResp.Result) == "" {
		return "", errors.New("retrieve Superset CSRF token: empty token in response")
	}

	c.csrfToken = csrfResp.Result

	return c.csrfToken, nil
}

func (c *Client) Get(ctx context.Context, requestPath string, responseBody any) error {
	return c.do(ctx, http.MethodGet, requestPath, nil, responseBody)
}

func (c *Client) Post(ctx context.Context, requestPath string, requestBody any, responseBody any) error {
	return c.do(ctx, http.MethodPost, requestPath, requestBody, responseBody)
}

func (c *Client) Put(ctx context.Context, requestPath string, requestBody any, responseBody any) error {
	return c.do(ctx, http.MethodPut, requestPath, requestBody, responseBody)
}

func (c *Client) Delete(ctx context.Context, requestPath string, responseBody any) error {
	return c.do(ctx, http.MethodDelete, requestPath, nil, responseBody)
}

func (c *Client) do(ctx context.Context, method string, requestPath string, requestBody any, responseBody any) error {
	if err := c.Authenticate(ctx); err != nil {
		return err
	}

	csrfToken := ""

	if requiresCSRF(method) {
		var err error

		csrfToken, err = c.ensureCSRFToken(ctx)
		if err != nil {
			return err
		}
	}

	return c.execute(ctx, method, requestPath, requestBody, responseBody, c.AccessToken(), true, csrfToken)
}

func (c *Client) execute(ctx context.Context, method string, requestPath string, requestBody any, responseBody any, token string, includeAuth bool, csrfToken string) error {
	req, err := c.newRequest(ctx, method, requestPath, requestBody, token, includeAuth, csrfToken)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform Superset API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read Superset API response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       requestPath,
			Body:       string(body),
		}
	}

	if responseBody == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, responseBody); err != nil {
		return fmt.Errorf("decode Superset API response: %w", err)
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method string, requestPath string, requestBody any, token string, includeAuth bool, csrfToken string) (*http.Request, error) {
	var body io.Reader

	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("marshal Superset API request: %w", err)
		}

		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.resolveURL(requestPath), body)
	if err != nil {
		return nil, fmt.Errorf("create Superset API request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if includeAuth && token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if csrfToken != "" {
		req.Header.Set("X-CSRFToken", csrfToken)
	}

	return req, nil
}

func (c *Client) resolveURL(requestPath string) string {
	relative, err := url.Parse(strings.TrimSpace(requestPath))
	if err != nil {
		relative = &url.URL{Path: requestPath}
	}

	relative.Path = strings.TrimLeft(relative.Path, "/")
	base := *c.baseURL

	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	return base.ResolveReference(relative).String()
}

func normalizeEndpoint(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("endpoint is required")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("endpoint must be a valid URL: %w", err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("endpoint must be a valid URL")
	}

	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed, nil
}

func cloneHTTPClient(httpClient *http.Client) *http.Client {
	cloned := *httpClient

	if cloned.Timeout == 0 {
		cloned.Timeout = 30 * time.Second
	}

	if cloned.Jar == nil {
		jar, err := cookiejar.New(nil)
		if err == nil {
			cloned.Jar = jar
		}
	}

	return &cloned
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodDelete, http.MethodPatch, http.MethodPost, http.MethodPut:
		return true
	default:
		return false
	}
}
