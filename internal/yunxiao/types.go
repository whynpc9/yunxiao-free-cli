package yunxiao

import "fmt"

type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NamedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type OrganizationMember struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	Name     string `json:"name"`
	NickName string `json:"nickName"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

type SearchMembersRequest struct {
	Page     int      `json:"page,omitempty"`
	PerPage  int      `json:"perPage,omitempty"`
	Query    string   `json:"query,omitempty"`
	Statuses []string `json:"statuses,omitempty"`
}

type SearchProjectsRequest struct {
	Page            int    `json:"page,omitempty"`
	PerPage         int    `json:"perPage,omitempty"`
	OrderBy         string `json:"orderBy,omitempty"`
	Sort            string `json:"sort,omitempty"`
	Conditions      string `json:"conditions,omitempty"`
	ExtraConditions string `json:"extraConditions,omitempty"`
}

type Project struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	CustomCode    string   `json:"customCode"`
	Description   string   `json:"description"`
	Scope         string   `json:"scope"`
	LogicalStatus string   `json:"logicalStatus"`
	GmtCreate     string   `json:"gmtCreate"`
	GmtModified   string   `json:"gmtModified"`
	Status        NamedRef `json:"status"`
	Creator       UserRef  `json:"creator"`
	Modifier      UserRef  `json:"modifier"`
}

type SearchWorkitemsRequest struct {
	Category   string `json:"category,omitempty"`
	Conditions string `json:"conditions,omitempty"`
	OrderBy    string `json:"orderBy,omitempty"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"perPage,omitempty"`
	Sort       string `json:"sort,omitempty"`
	SpaceID    string `json:"spaceId,omitempty"`
	SpaceType  string `json:"spaceType,omitempty"`
}

type WorkItem struct {
	ID            any      `json:"id"`
	Identifier    string   `json:"identifier"`
	Subject       string   `json:"subject"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	CategoryID    string   `json:"categoryId"`
	LogicalStatus string   `json:"logicalStatus"`
	GmtCreate     string   `json:"gmtCreate"`
	GmtModified   string   `json:"gmtModified"`
	Status        NamedRef `json:"status"`
	AssignedTo    UserRef  `json:"assignedTo"`
	Creator       UserRef  `json:"creator"`
	Modifier      UserRef  `json:"modifier"`
	Space         NamedRef `json:"space"`
}

func (w WorkItem) IDString() string {
	if w.ID == nil {
		return ""
	}
	return fmt.Sprintf("%v", w.ID)
}

func (w WorkItem) Title() string {
	if w.Subject != "" {
		return w.Subject
	}
	return w.Name
}

type WorkItemType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	NameEn      string `json:"nameEn"`
	CategoryID  string `json:"categoryId"`
	Description string `json:"description"`
	DefaultType bool   `json:"defaultType"`
	Enable      bool   `json:"enable"`
}

type WorkItemFieldConfig struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Required    bool   `json:"required"`
	System      bool   `json:"system"`
	Description string `json:"description"`
}

type Directory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

type SearchTestCasesRequest struct {
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"perPage,omitempty"`
	OrderBy    string `json:"orderBy,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Conditions string `json:"conditions,omitempty"`
}

type TestCase struct {
	ID          any      `json:"id"`
	Identifier  string   `json:"identifier"`
	Subject     string   `json:"subject"`
	Name        string   `json:"name"`
	Priority    string   `json:"priority"`
	State       string   `json:"state"`
	Creator     UserRef  `json:"creator"`
	AssignedTo  UserRef  `json:"assignedTo"`
	GmtCreate   string   `json:"gmtCreate"`
	GmtModified string   `json:"gmtModified"`
	DirectoryID string   `json:"directoryId"`
	Directory   NamedRef `json:"directory"`
}

func (t TestCase) IDString() string {
	if t.ID == nil {
		return ""
	}
	return fmt.Sprintf("%v", t.ID)
}

func (t TestCase) Title() string {
	if t.Subject != "" {
		return t.Subject
	}
	return t.Name
}

type TestCaseFieldConfig struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Required    bool   `json:"required"`
	System      bool   `json:"system"`
	Description string `json:"description"`
}
