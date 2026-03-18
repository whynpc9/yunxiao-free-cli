package yunxiao

import "fmt"

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Directory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

type SearchTestCasesRequest struct {
	Page       int            `json:"page,omitempty"`
	PerPage    int            `json:"perPage,omitempty"`
	OrderBy    string         `json:"orderBy,omitempty"`
	SortField  string         `json:"sortField,omitempty"`
	Conditions map[string]any `json:"conditions,omitempty"`
}

type TestCase struct {
	ID          any    `json:"id"`
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Priority    string `json:"priority"`
	State       string `json:"state"`
	Creator     string `json:"creator"`
	Owner       string `json:"owner"`
	GmtCreate   string `json:"gmtCreate"`
	GmtModified string `json:"gmtModified"`
	DirectoryID string `json:"directoryId"`
}

func (t TestCase) IDString() string {
	if t.ID == nil {
		return ""
	}
	return fmt.Sprintf("%v", t.ID)
}
