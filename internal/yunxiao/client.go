package yunxiao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout = 20 * time.Second
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func NewClient(domain, token string) *Client {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "openapi-rdc.aliyuncs.com"
	}
	return &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		baseURL:    "https://" + domain,
		token:      token,
	}
}

func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var out []Organization
	if err := c.doJSON(ctx, http.MethodGet, "/oapi/v1/platform/organizations", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListDirectories(ctx context.Context, organizationID, testRepoID string) ([]Directory, error) {
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/directories", url.PathEscape(organizationID), url.PathEscape(testRepoID))
	var out []Directory
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchTestCases(ctx context.Context, organizationID, testRepoID string, req SearchTestCasesRequest) ([]TestCase, error) {
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/testCases/search", url.PathEscape(organizationID), url.PathEscape(testRepoID))
	var out []TestCase
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetTestCase(ctx context.Context, organizationID, testRepoID, testCaseID string) (TestCase, error) {
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/testCases/%s", url.PathEscape(organizationID), url.PathEscape(testRepoID), url.PathEscape(testCaseID))
	var out TestCase
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return TestCase{}, err
	}
	return out, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, in any, out any) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("build url: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("x-yunxiao-token", c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		trimmed := strings.TrimSpace(string(data))
		if len(trimmed) > 400 {
			trimmed = trimmed[:400] + "..."
		}
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return fmt.Errorf("yunxiao api %s %s failed: status=%d body=%s", method, path, resp.StatusCode, trimmed)
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}
