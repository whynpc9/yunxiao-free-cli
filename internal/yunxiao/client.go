package yunxiao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func (c *Client) CreateWorkitem(ctx context.Context, organizationID string, req CreateWorkitemRequest) (CreateWorkitemResponse, error) {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitems", url.PathEscape(organizationID))
	var out CreateWorkitemResponse
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return CreateWorkitemResponse{}, err
	}
	return out, nil
}

func (c *Client) UpdateWorkitem(ctx context.Context, organizationID, workitemID string, fields map[string]any) error {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitems/%s", url.PathEscape(organizationID), url.PathEscape(workitemID))
	return c.doJSON(ctx, http.MethodPut, path, nil, fields, nil)
}

func (c *Client) DeleteWorkitem(ctx context.Context, organizationID, workitemID string) error {
	path := fmt.Sprintf("/oapi/v1/projex/organizations/%s/workitems/%s", url.PathEscape(organizationID), url.PathEscape(workitemID))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *Client) GetWorkitemTimeTypeList(ctx context.Context, organizationID string) ([]WorkitemTimeType, error) {
	var resp struct {
		Success   bool               `json:"success"`
		ErrorCode string             `json:"errorCode"`
		ErrorMsg  string             `json:"errorMsg"`
		TimeType  []WorkitemTimeType `json:"timeType"`
	}
	path := fmt.Sprintf("/organization/%s/workitems/type/list", url.PathEscape(organizationID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, explainLegacyAPIError(err)
	}
	if !resp.Success && resp.ErrorCode != "" {
		return nil, fmt.Errorf("legacy api failed: %s %s", resp.ErrorCode, resp.ErrorMsg)
	}
	return resp.TimeType, nil
}

func (c *Client) ListWorkitemTime(ctx context.Context, organizationID, workitemID string) ([]WorkitemTime, error) {
	var resp struct {
		Success      bool           `json:"success"`
		ErrorCode    string         `json:"errorCode"`
		ErrorMsg     string         `json:"errorMsg"`
		WorkitemTime []WorkitemTime `json:"workitemTime"`
	}
	path := fmt.Sprintf("/organization/%s/workitems/%s/time/list", url.PathEscape(organizationID), url.PathEscape(workitemID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, explainLegacyAPIError(err)
	}
	if !resp.Success && resp.ErrorCode != "" {
		return nil, fmt.Errorf("legacy api failed: %s %s", resp.ErrorCode, resp.ErrorMsg)
	}
	return normalizeWorkitemTimes(resp.WorkitemTime), nil
}

func (c *Client) ListWorkitemEstimate(ctx context.Context, organizationID, workitemID string) ([]WorkitemTime, error) {
	var resp struct {
		Success              bool           `json:"success"`
		ErrorCode            string         `json:"errorCode"`
		ErrorMsg             string         `json:"errorMsg"`
		WorkitemTimeEstimate []WorkitemTime `json:"workitemTimeEstimate"`
	}
	path := fmt.Sprintf("/organization/%s/workitems/%s/estimate/list", url.PathEscape(organizationID), url.PathEscape(workitemID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &resp); err != nil {
		return nil, explainLegacyAPIError(err)
	}
	if !resp.Success && resp.ErrorCode != "" {
		return nil, fmt.Errorf("legacy api failed: %s %s", resp.ErrorCode, resp.ErrorMsg)
	}
	return normalizeWorkitemTimes(resp.WorkitemTimeEstimate), nil
}

func (c *Client) CreateWorkitemRecord(ctx context.Context, organizationID string, req CreateWorkitemRecordRequest) (WorkitemTime, error) {
	var resp struct {
		Success      bool         `json:"success"`
		ErrorCode    string       `json:"errorCode"`
		ErrorMsg     string       `json:"errorMsg"`
		WorkitemTime WorkitemTime `json:"WorkitemTime"`
	}
	path := fmt.Sprintf("/organization/%s/workitems/record", url.PathEscape(organizationID))
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &resp); err != nil {
		return WorkitemTime{}, explainLegacyAPIError(err)
	}
	if !resp.Success && resp.ErrorCode != "" {
		return WorkitemTime{}, fmt.Errorf("legacy api failed: %s %s", resp.ErrorCode, resp.ErrorMsg)
	}
	items := normalizeWorkitemTimes([]WorkitemTime{resp.WorkitemTime})
	if len(items) == 0 {
		return WorkitemTime{}, errors.New("legacy api returned empty workitem time")
	}
	return items[0], nil
}

func (c *Client) CreateWorkitemEstimate(ctx context.Context, organizationID string, req CreateWorkitemEstimateRequest) (WorkitemTime, error) {
	var resp struct {
		Success               bool         `json:"success"`
		ErrorCode             string       `json:"errorCode"`
		ErrorMsg              string       `json:"errorMsg"`
		WorkitemTimeEstimate  WorkitemTime `json:"workitemTimeEstimate"`
		WorkitemTimeEstimates WorkitemTime `json:"WorkitemTimeEstimate"`
	}
	path := fmt.Sprintf("/organization/%s/workitems/estimate", url.PathEscape(organizationID))
	if err := c.doJSON(ctx, http.MethodPost, path, nil, req, &resp); err != nil {
		return WorkitemTime{}, explainLegacyAPIError(err)
	}
	if !resp.Success && resp.ErrorCode != "" {
		return WorkitemTime{}, fmt.Errorf("legacy api failed: %s %s", resp.ErrorCode, resp.ErrorMsg)
	}
	item := resp.WorkitemTimeEstimate
	if item.Identifier == "" {
		item = resp.WorkitemTimeEstimates
	}
	items := normalizeWorkitemTimes([]WorkitemTime{item})
	if len(items) == 0 {
		return WorkitemTime{}, errors.New("legacy api returned empty workitem estimate")
	}
	return items[0], nil
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

func normalizeWorkitemTimes(items []WorkitemTime) []WorkitemTime {
	for i := range items {
		switch v := items[i].RecordUser.(type) {
		case map[string]any:
			items[i].RecordUserDetail = WorkitemRecordUser{
				Account:     stringValue(v["account"]),
				Identifier:  stringValue(v["identifier"]),
				RealName:    stringValue(v["realName"]),
				NickName:    stringValue(v["nickName"]),
				DisplayName: stringValue(v["displayName"]),
				Email:       stringValue(v["email"]),
			}
		case string:
			items[i].RecordUserDetail = WorkitemRecordUser{Identifier: v}
		}
	}
	return items
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func explainLegacyAPIError(err error) error {
	if err == nil {
		return nil
	}
	text := err.Error()
	if strings.Contains(text, "invalid character '<'") || strings.Contains(text, "api-reference-standard-proprietary") {
		return fmt.Errorf("%w; 工时接口使用云效旧版 Workitem OpenAPI，当前 domain 很可能不支持该旧接口，请核对服务接入点或账号版本", err)
	}
	return err
}
