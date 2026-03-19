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

const defaultTimeout = 20 * time.Second

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

func (c *Client) SearchOrganizationMembers(ctx context.Context, organizationID string, req SearchMembersRequest) ([]OrganizationMember, error) {
	path := fmt.Sprintf("/oapi/v1/platform/organizations/%s/members:search", url.PathEscape(organizationID))
	var out []OrganizationMember
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) SearchProjects(ctx context.Context, organizationID string, req SearchProjectsRequest) ([]Project, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/projects:search", url.PathEscape(organizationID))
	var out []Project
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetProject(ctx context.Context, organizationID, projectID string) (Project, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/projects/%s", url.PathEscape(organizationID), url.PathEscape(projectID))
	var out Project
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return Project{}, err
	}
	return out, nil
}

func (c *Client) SearchWorkitems(ctx context.Context, organizationID string, req SearchWorkitemsRequest) ([]WorkItem, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitems:search", url.PathEscape(organizationID))
	var out []WorkItem
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetWorkitem(ctx context.Context, organizationID, workitemID string) (WorkItem, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitems/%s", url.PathEscape(organizationID), url.PathEscape(workitemID))
	var out WorkItem
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return WorkItem{}, err
	}
	return out, nil
}

func (c *Client) ListAllWorkitemTypes(ctx context.Context, organizationID, categories string) ([]WorkItemType, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitemTypes", url.PathEscape(organizationID))
	query := url.Values{}
	if strings.TrimSpace(categories) != "" {
		query.Set("categories", categories)
	}
	var out []WorkItemType
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListWorkitemTypes(ctx context.Context, organizationID, projectID, category string) ([]WorkItemType, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/projects/%s/workitemTypes", url.PathEscape(organizationID), url.PathEscape(projectID))
	query := url.Values{}
	query.Set("category", category)
	var out []WorkItemType
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetWorkitemTypeFieldConfig(ctx context.Context, organizationID, projectID, workitemTypeID string) ([]WorkItemFieldConfig, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/projects/%s/workitemTypes/%s/fields", url.PathEscape(organizationID), url.PathEscape(projectID), url.PathEscape(workitemTypeID))
	var out []WorkItemFieldConfig
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListTestPlans(ctx context.Context, organizationID string) ([]TestPlan, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/testPlan/list", url.PathEscape(organizationID))
	var out []TestPlan
	if err := c.doJSON(ctx, http.MethodPost, path, nil, map[string]any{}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetTestResultList(ctx context.Context, organizationID, testPlanID, directoryID string) ([]TestResultSummary, error) {
	body := map[string]any{}
	projexPath := fmt.Sprintf("/oapi/v1/projex/organizations/%s/%s/result/list/%s", url.PathEscape(organizationID), url.PathEscape(testPlanID), url.PathEscape(directoryID))
	var out []TestResultSummary
	if err := c.doJSON(ctx, http.MethodPost, projexPath, nil, body, &out); err == nil {
		return out, nil
	} else if !strings.Contains(err.Error(), "status=404") {
		return nil, err
	}

	testhubPath := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/%s/result/list/%s", url.PathEscape(organizationID), url.PathEscape(testPlanID), url.PathEscape(directoryID))
	out = nil
	if err := c.doJSON(ctx, http.MethodPost, testhubPath, nil, body, &out); err != nil {
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
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/testcases:search", url.PathEscape(organizationID), url.PathEscape(testRepoID))
	var out []TestCase
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetTestCase(ctx context.Context, organizationID, testRepoID, testCaseID string) (TestCase, error) {
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/testcases/%s", url.PathEscape(organizationID), url.PathEscape(testRepoID), url.PathEscape(testCaseID))
	var out TestCase
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return TestCase{}, err
	}
	return out, nil
}

func (c *Client) GetTestcaseFieldConfig(ctx context.Context, organizationID, testRepoID string) ([]TestCaseFieldConfig, error) {
	path := fmt.Sprintf("/oapi/v1/testhub/organizations/%s/testRepos/%s/testcases/fields", url.PathEscape(organizationID), url.PathEscape(testRepoID))
	var out []TestCaseFieldConfig
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
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
