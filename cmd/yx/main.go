package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/wanghongyi/yunxiao-free-cli/internal/cli"
	"github.com/wanghongyi/yunxiao-free-cli/internal/config"
	"github.com/wanghongyi/yunxiao-free-cli/internal/yunxiao"
)

const (
	tokenHelpURL = "https://help.aliyun.com/zh/yunxiao/developer-reference/obtain-personal-access-token?scm=20140722.H_2841293._.OR_help-T_cn~zh-V_1"
	version      = "0.2.0"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printRootUsage()
		return nil
	}

	switch args[0] {
	case "auth":
		return runAuth(ctx)
	case "org":
		return runOrg(ctx, args[1:])
	case "project":
		return runProject(ctx, args[1:])
	case "workitem":
		return runWorkitem(ctx, args[1:])
	case "testcase":
		return runTestCase(ctx, args[1:])
	case "config":
		return runConfig(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printRootUsage()
		return nil
	default:
		printRootUsage()
		return fmt.Errorf("未知命令: %s", args[0])
	}
}

func runAuth(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Domain == "" {
		cfg.Domain = "openapi-rdc.aliyuncs.com"
	}

	fmt.Fprintln(os.Stderr, "请录入云效个人访问令牌(PAT)。获取方式:")
	fmt.Fprintln(os.Stderr, tokenHelpURL)
	line, err := cli.PromptLine("PAT (直接回车可保留当前值): ")
	if err != nil {
		return fmt.Errorf("读取 PAT 失败: %w", err)
	}

	if strings.TrimSpace(line) != "" {
		cfg.Token = strings.TrimSpace(line)
	} else if cfg.Token == "" {
		return errors.New("PAT 不能为空")
	}

	client := yunxiao.NewClient(cfg.Domain, cfg.Token)
	if err := chooseDefaultOrganization(ctx, client, &cfg, true); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	path, _ := config.ConfigFilePath()
	fmt.Printf("认证完成，配置已保存: %s\n", path)
	fmt.Printf("默认组织: %s (%s)\n", cfg.DefaultOrganizationName, cfg.DefaultOrganizationID)
	return nil
}

func runOrg(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx org <list|use|members>")
		return nil
	}
	cfg, client, err := ensureReady(ctx, true)
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("org list", flag.ContinueOnError)
		jsonOut := fs.Bool("json", false, "JSON 输出")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		orgs, err := client.ListOrganizations(ctx)
		if err != nil {
			return err
		}
		if *jsonOut {
			return cli.PrintJSON(orgs)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tDEFAULT")
		for _, org := range orgs {
			mark := ""
			if org.ID == cfg.DefaultOrganizationID {
				mark = "*"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\n", org.ID, org.Name, mark)
		}
		return tw.Flush()
	case "use":
		return runOrgUse(ctx, cfg, client, args[1:])
	case "members":
		return runOrgMembers(ctx, cfg, client, args[1:])
	default:
		return fmt.Errorf("未知 org 子命令: %s", args[0])
	}
}

func runOrgUse(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	orgID := ""
	if len(args) >= 1 {
		orgID = strings.TrimSpace(args[0])
	}
	orgs, err := client.ListOrganizations(ctx)
	if err != nil {
		return err
	}
	if len(orgs) == 0 {
		return errors.New("当前账号未查询到组织")
	}

	if orgID == "" {
		labels := make([]string, 0, len(orgs))
		for _, org := range orgs {
			labels = append(labels, fmt.Sprintf("%s (%s)", org.Name, org.ID))
		}
		idx, err := cli.PromptSelect("选择默认组织序号: ", labels)
		if err != nil {
			return err
		}
		cfg.DefaultOrganizationID = orgs[idx].ID
		cfg.DefaultOrganizationName = orgs[idx].Name
	} else {
		var found *yunxiao.Organization
		for i := range orgs {
			if orgs[i].ID == orgID {
				found = &orgs[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("组织不存在或不可访问: %s", orgID)
		}
		cfg.DefaultOrganizationID = found.ID
		cfg.DefaultOrganizationName = found.Name
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("默认组织已设置为: %s (%s)\n", cfg.DefaultOrganizationName, cfg.DefaultOrganizationID)
	return nil
}

func runOrgMembers(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("org members", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	page := fs.Int("page", 1, "页码")
	perPage := fs.Int("per-page", 20, "每页数量")
	query := fs.String("query", "", "按姓名、邮箱等关键字查询")
	statuses := fs.String("statuses", "", "成员状态，逗号分隔，例如 ENABLED,DISABLED")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}

	org := requireOrgID(cfg, *orgID)
	req := yunxiao.SearchMembersRequest{
		Page:     *page,
		PerPage:  *perPage,
		Query:    strings.TrimSpace(*query),
		Statuses: splitCSV(*statuses),
	}
	items, err := client.SearchOrganizationMembers(ctx, org, req)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "USER_ID\tNAME\tNICKNAME\tEMAIL\tSTATUS")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			emptyDash(firstNonEmpty(item.UserID, item.ID)),
			emptyDash(item.Name),
			emptyDash(item.NickName),
			emptyDash(item.Email),
			emptyDash(item.Status),
		)
	}
	return tw.Flush()
}

func runProject(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx project <search|get>")
		return nil
	}
	cfg, client, err := ensureReady(ctx, true)
	if err != nil {
		return err
	}

	switch args[0] {
	case "search":
		return runProjectSearch(ctx, cfg, client, args[1:])
	case "get":
		return runProjectGet(ctx, cfg, client, args[1:])
	default:
		return fmt.Errorf("未知 project 子命令: %s", args[0])
	}
}

func runProjectSearch(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("project search", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	page := fs.Int("page", 1, "页码")
	perPage := fs.Int("per-page", 20, "每页数量")
	orderBy := fs.String("order-by", "gmtCreate", "排序字段")
	sortValue := fs.String("sort", "desc", "排序方向 asc|desc")
	conditions := fs.String("conditions", "", "条件 JSON 字符串")
	extraConditions := fs.String("extra-conditions", "", "附加条件 JSON 字符串")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}

	org := requireOrgID(cfg, *orgID)
	conditionsValue, err := validateJSONObjectString(*conditions, "--conditions")
	if err != nil {
		return err
	}
	extraConditionsValue, err := validateJSONObjectString(*extraConditions, "--extra-conditions")
	if err != nil {
		return err
	}
	req := yunxiao.SearchProjectsRequest{
		Page:            *page,
		PerPage:         *perPage,
		OrderBy:         *orderBy,
		Sort:            *sortValue,
		Conditions:      conditionsValue,
		ExtraConditions: extraConditionsValue,
	}
	items, err := client.SearchProjects(ctx, org, req)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCODE\tNAME\tSTATUS\tSCOPE\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.ID,
			emptyDash(item.CustomCode),
			trim(item.Name, 36),
			emptyDash(item.Status.Name),
			emptyDash(item.Scope),
			emptyDash(item.GmtModified),
		)
	}
	return tw.Flush()
}

func runProjectGet(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("project get", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	projectID := fs.String("id", "", "项目 ID (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*projectID) == "" {
		return errors.New("--id 必填")
	}

	org := requireOrgID(cfg, *orgID)
	item, err := client.GetProject(ctx, org, *projectID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(item)
	}

	fmt.Printf("ID: %s\n", item.ID)
	fmt.Printf("Code: %s\n", emptyDash(item.CustomCode))
	fmt.Printf("Name: %s\n", emptyDash(item.Name))
	fmt.Printf("Status: %s\n", emptyDash(item.Status.Name))
	fmt.Printf("Scope: %s\n", emptyDash(item.Scope))
	fmt.Printf("LogicalStatus: %s\n", emptyDash(item.LogicalStatus))
	fmt.Printf("Creator: %s\n", emptyDash(item.Creator.Name))
	fmt.Printf("Modifier: %s\n", emptyDash(item.Modifier.Name))
	fmt.Printf("CreatedAt: %s\n", emptyDash(item.GmtCreate))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(item.GmtModified))
	fmt.Printf("Description: %s\n", emptyDash(item.Description))
	return nil
}

func runWorkitem(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx workitem <search|get|types|all-types|fields>")
		return nil
	}
	cfg, client, err := ensureReady(ctx, true)
	if err != nil {
		return err
	}

	switch args[0] {
	case "search":
		return runWorkitemSearch(ctx, cfg, client, args[1:])
	case "get":
		return runWorkitemGet(ctx, cfg, client, args[1:])
	case "types":
		return runWorkitemTypes(ctx, cfg, client, args[1:])
	case "all-types":
		return runWorkitemAllTypes(ctx, cfg, client, args[1:])
	case "fields":
		return runWorkitemFields(ctx, cfg, client, args[1:])
	default:
		return fmt.Errorf("未知 workitem 子命令: %s", args[0])
	}
}

func runWorkitemSearch(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem search", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	category := fs.String("category", "", "工作项分类，例如 Req/Bug/Task")
	spaceID := fs.String("space-id", "", "空间 ID，通常为项目 ID")
	spaceType := fs.String("space-type", "Project", "空间类型，默认 Project")
	page := fs.Int("page", 1, "页码")
	perPage := fs.Int("per-page", 20, "每页数量")
	orderBy := fs.String("order-by", "gmtCreate", "排序字段")
	sortValue := fs.String("sort", "desc", "排序方向 asc|desc")
	conditions := fs.String("conditions", "", "条件 JSON 字符串")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*category) == "" {
		return errors.New("--category 必填")
	}

	org := requireOrgID(cfg, *orgID)
	conditionsValue, err := validateJSONObjectString(*conditions, "--conditions")
	if err != nil {
		return err
	}
	req := yunxiao.SearchWorkitemsRequest{
		Category:   *category,
		Conditions: conditionsValue,
		OrderBy:    *orderBy,
		Page:       *page,
		PerPage:    *perPage,
		Sort:       *sortValue,
		SpaceID:    strings.TrimSpace(*spaceID),
		SpaceType:  strings.TrimSpace(*spaceType),
	}
	items, err := client.SearchWorkitems(ctx, org, req)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tIDENTIFIER\tTITLE\tSTATUS\tASSIGNEE\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.IDString(),
			emptyDash(item.Identifier),
			trim(item.Title(), 36),
			emptyDash(item.Status.Name),
			emptyDash(item.AssignedTo.Name),
			emptyDash(item.GmtModified),
		)
	}
	return tw.Flush()
}

func runWorkitemGet(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem get", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	workitemID := fs.String("id", "", "工作项 ID (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*workitemID) == "" {
		return errors.New("--id 必填")
	}

	org := requireOrgID(cfg, *orgID)
	item, err := client.GetWorkitem(ctx, org, *workitemID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(item)
	}

	fmt.Printf("ID: %s\n", item.IDString())
	fmt.Printf("Identifier: %s\n", emptyDash(item.Identifier))
	fmt.Printf("Title: %s\n", emptyDash(item.Title()))
	fmt.Printf("Category: %s\n", emptyDash(item.CategoryID))
	fmt.Printf("Status: %s\n", emptyDash(item.Status.Name))
	fmt.Printf("LogicalStatus: %s\n", emptyDash(item.LogicalStatus))
	fmt.Printf("Assignee: %s\n", emptyDash(item.AssignedTo.Name))
	fmt.Printf("Creator: %s\n", emptyDash(item.Creator.Name))
	fmt.Printf("Modifier: %s\n", emptyDash(item.Modifier.Name))
	fmt.Printf("Space: %s\n", emptyDash(item.Space.Name))
	fmt.Printf("CreatedAt: %s\n", emptyDash(item.GmtCreate))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(item.GmtModified))
	fmt.Printf("Description: %s\n", emptyDash(item.Description))
	return nil
}

func runWorkitemTypes(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem types", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	projectID := fs.String("project", "", "项目 ID (必填)")
	category := fs.String("category", "", "工作项分类，例如 Req/Bug/Task (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*projectID) == "" {
		return errors.New("--project 必填")
	}
	if strings.TrimSpace(*category) == "" {
		return errors.New("--category 必填")
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.ListWorkitemTypes(ctx, org, *projectID, *category)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}
	return printWorkitemTypes(items)
}

func runWorkitemAllTypes(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem all-types", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	categories := fs.String("categories", "", "按分类过滤，逗号分隔，例如 Req,Bug,Task")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.ListAllWorkitemTypes(ctx, org, strings.TrimSpace(*categories))
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}
	return printWorkitemTypes(items)
}

func runWorkitemFields(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem fields", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	projectID := fs.String("project", "", "项目 ID (必填)")
	workitemTypeID := fs.String("type", "", "工作项类型 ID (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*projectID) == "" {
		return errors.New("--project 必填")
	}
	if strings.TrimSpace(*workitemTypeID) == "" {
		return errors.New("--type 必填")
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.GetWorkitemTypeFieldConfig(ctx, org, *projectID, *workitemTypeID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}
	return printFieldConfigs(items)
}

func runTestCase(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx testcase <search|get|dirs|fields>")
		return nil
	}
	cfg, client, err := ensureReady(ctx, true)
	if err != nil {
		return err
	}

	switch args[0] {
	case "search":
		return runTestCaseSearch(ctx, cfg, client, args[1:])
	case "get":
		return runTestCaseGet(ctx, cfg, client, args[1:])
	case "dirs":
		return runTestCaseDirs(ctx, cfg, client, args[1:])
	case "fields":
		return runTestCaseFields(ctx, cfg, client, args[1:])
	default:
		return fmt.Errorf("未知 testcase 子命令: %s", args[0])
	}
}

func runTestCaseSearch(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testcase search", flag.ContinueOnError)
	repoID := fs.String("repo", "", "测试用例库 ID (必填)")
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	page := fs.Int("page", 1, "页码")
	perPage := fs.Int("per-page", 20, "每页数量")
	orderBy := fs.String("order-by", "gmtModified", "排序字段")
	sortValue := fs.String("sort", "desc", "排序方向 asc|desc")
	conditions := fs.String("conditions", "", "条件 JSON 字符串")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repoID) == "" {
		return errors.New("--repo 必填")
	}

	org := requireOrgID(cfg, *orgID)
	conditionsValue, err := validateJSONObjectString(*conditions, "--conditions")
	if err != nil {
		return err
	}
	req := yunxiao.SearchTestCasesRequest{
		Page:       *page,
		PerPage:    *perPage,
		OrderBy:    *orderBy,
		Sort:       *sortValue,
		Conditions: conditionsValue,
	}
	items, err := client.SearchTestCases(ctx, org, *repoID, req)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tIDENTIFIER\tTITLE\tSTATE\tPRIORITY\tASSIGNEE\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.IDString(),
			emptyDash(item.Identifier),
			trim(item.Title(), 40),
			emptyDash(item.State),
			emptyDash(item.Priority),
			emptyDash(item.AssignedTo.Name),
			emptyDash(item.GmtModified),
		)
	}
	return tw.Flush()
}

func runTestCaseGet(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testcase get", flag.ContinueOnError)
	repoID := fs.String("repo", "", "测试用例库 ID (必填)")
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	caseID := fs.String("id", "", "测试用例 ID (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repoID) == "" {
		return errors.New("--repo 必填")
	}
	if strings.TrimSpace(*caseID) == "" {
		return errors.New("--id 必填")
	}

	org := requireOrgID(cfg, *orgID)
	item, err := client.GetTestCase(ctx, org, *repoID, *caseID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(item)
	}

	fmt.Printf("ID: %s\n", item.IDString())
	fmt.Printf("Identifier: %s\n", emptyDash(item.Identifier))
	fmt.Printf("Title: %s\n", emptyDash(item.Title()))
	fmt.Printf("State: %s\n", emptyDash(item.State))
	fmt.Printf("Priority: %s\n", emptyDash(item.Priority))
	fmt.Printf("Assignee: %s\n", emptyDash(item.AssignedTo.Name))
	fmt.Printf("Creator: %s\n", emptyDash(item.Creator.Name))
	fmt.Printf("DirectoryID: %s\n", emptyDash(firstNonEmpty(item.DirectoryID, item.Directory.ID)))
	fmt.Printf("Directory: %s\n", emptyDash(item.Directory.Name))
	fmt.Printf("CreatedAt: %s\n", emptyDash(item.GmtCreate))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(item.GmtModified))
	return nil
}

func runTestCaseDirs(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testcase dirs", flag.ContinueOnError)
	repoID := fs.String("repo", "", "测试用例库 ID (必填)")
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repoID) == "" {
		return errors.New("--repo 必填")
	}

	org := requireOrgID(cfg, *orgID)
	dirs, err := client.ListDirectories(ctx, org, *repoID)
	if err != nil {
		return err
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	if *jsonOut {
		return cli.PrintJSON(dirs)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tPARENT_ID")
	for _, d := range dirs {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.ID, d.Name, emptyDash(d.ParentID))
	}
	return tw.Flush()
}

func runTestCaseFields(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testcase fields", flag.ContinueOnError)
	repoID := fs.String("repo", "", "测试用例库 ID (必填)")
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repoID) == "" {
		return errors.New("--repo 必填")
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.GetTestcaseFieldConfig(ctx, org, *repoID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}
	return printTestCaseFieldConfigs(items)
}

func runConfig(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		path, _ := config.ConfigFilePath()
		fmt.Printf("Config: %s\n", path)
		fmt.Printf("Domain: %s\n", cfg.Domain)
		fmt.Printf("Token: %s\n", config.MaskToken(cfg.Token))
		fmt.Printf("Default Organization: %s (%s)\n", cfg.DefaultOrganizationName, cfg.DefaultOrganizationID)
		return nil
	}
	return fmt.Errorf("未知 config 子命令: %s", args[0])
}

func ensureReady(ctx context.Context, ensureOrg bool) (config.Config, *yunxiao.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return config.Config{}, nil, err
	}
	if cfg.Domain == "" {
		cfg.Domain = "openapi-rdc.aliyuncs.com"
	}

	if strings.TrimSpace(cfg.Token) == "" {
		fmt.Fprintln(os.Stderr, "首次使用请先配置 PAT。帮助文档:")
		fmt.Fprintln(os.Stderr, tokenHelpURL)
		line, err := cli.PromptLine("请输入 PAT: ")
		if err != nil {
			return config.Config{}, nil, err
		}
		if strings.TrimSpace(line) == "" {
			return config.Config{}, nil, errors.New("PAT 不能为空")
		}
		cfg.Token = strings.TrimSpace(line)
	}

	client := yunxiao.NewClient(cfg.Domain, cfg.Token)
	if ensureOrg {
		if err := chooseDefaultOrganization(ctx, client, &cfg, false); err != nil {
			return config.Config{}, nil, err
		}
	}

	if err := config.Save(cfg); err != nil {
		return config.Config{}, nil, err
	}
	return cfg, client, nil
}

func chooseDefaultOrganization(ctx context.Context, client *yunxiao.Client, cfg *config.Config, forcePrompt bool) error {
	orgs, err := client.ListOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("查询组织失败，请检查 PAT 是否有效: %w", err)
	}
	if len(orgs) == 0 {
		return errors.New("当前账号未查询到组织")
	}

	if !forcePrompt && cfg.DefaultOrganizationID != "" {
		for _, org := range orgs {
			if org.ID == cfg.DefaultOrganizationID {
				if cfg.DefaultOrganizationName == "" {
					cfg.DefaultOrganizationName = org.Name
				}
				return nil
			}
		}
	}

	if len(orgs) == 1 {
		cfg.DefaultOrganizationID = orgs[0].ID
		cfg.DefaultOrganizationName = orgs[0].Name
		return nil
	}

	labels := make([]string, 0, len(orgs))
	for _, org := range orgs {
		labels = append(labels, fmt.Sprintf("%s (%s)", org.Name, org.ID))
	}
	idx, err := cli.PromptSelect("请选择默认组织序号: ", labels)
	if err != nil {
		return err
	}
	cfg.DefaultOrganizationID = orgs[idx].ID
	cfg.DefaultOrganizationName = orgs[idx].Name
	return nil
}

func requireOrgID(cfg config.Config, explicit string) string {
	if orgID := pickOrgID(cfg, explicit); strings.TrimSpace(orgID) != "" {
		return orgID
	}
	return cfg.DefaultOrganizationID
}

func pickOrgID(cfg config.Config, explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	return cfg.DefaultOrganizationID
}

func validateJSONObjectString(text, flagName string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		return "", fmt.Errorf("%s 必须为合法 JSON: %w", flagName, err)
	}
	if _, ok := value.(map[string]any); !ok {
		return "", fmt.Errorf("%s 必须为 JSON 对象", flagName)
	}
	return text, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func printWorkitemTypes(items []yunxiao.WorkItemType) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCATEGORY\tDEFAULT\tENABLED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%t\n", item.ID, item.Name, emptyDash(item.CategoryID), item.DefaultType, item.Enable)
	}
	return tw.Flush()
}

func printFieldConfigs(items []yunxiao.WorkItemFieldConfig) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tIDENTIFIER\tNAME\tTYPE\tFORMAT\tREQUIRED\tSYSTEM")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%t\t%t\n",
			emptyDash(item.ID),
			emptyDash(item.Identifier),
			emptyDash(item.Name),
			emptyDash(item.Type),
			emptyDash(item.Format),
			item.Required,
			item.System,
		)
	}
	return tw.Flush()
}

func printTestCaseFieldConfigs(items []yunxiao.TestCaseFieldConfig) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tIDENTIFIER\tNAME\tTYPE\tFORMAT\tREQUIRED\tSYSTEM")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%t\t%t\n",
			emptyDash(item.ID),
			emptyDash(item.Identifier),
			emptyDash(item.Name),
			emptyDash(item.Type),
			emptyDash(item.Format),
			item.Required,
			item.System,
		)
	}
	return tw.Flush()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func trim(s string, max int) string {
	if len(s) <= max || max <= 0 {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func printRootUsage() {
	fmt.Print(`yunxiao-free-cli (yx)

用法:
  yx auth
  yx org list [--json]
  yx org use [organizationId]
  yx org members [--org <organizationId>] [--query <keyword>] [--statuses ENABLED] [--json]
  yx project search [--org <organizationId>] [--conditions '{"conditionGroups":[[]]}'] [--json]
  yx project get --id <projectId> [--org <organizationId>] [--json]
  yx workitem search --category <Req|Bug|Task> [--space-id <projectId>] [--org <organizationId>] [--conditions '{"conditionGroups":[[]]}'] [--json]
  yx workitem get --id <workitemId> [--org <organizationId>] [--json]
  yx workitem types --project <projectId> --category <Req|Bug|Task> [--org <organizationId>] [--json]
  yx workitem all-types [--categories Req,Bug,Task] [--org <organizationId>] [--json]
  yx workitem fields --project <projectId> --type <workitemTypeId> [--org <organizationId>] [--json]
  yx testcase search --repo <testRepoId> [--org <organizationId>] [--conditions '{"conditionGroups":[[]]}'] [--json]
  yx testcase get --repo <testRepoId> --id <testCaseId> [--org <organizationId>] [--json]
  yx testcase dirs --repo <testRepoId> [--org <organizationId>] [--json]
  yx testcase fields --repo <testRepoId> [--org <organizationId>] [--json]
  yx config show
  yx version
`)
}
