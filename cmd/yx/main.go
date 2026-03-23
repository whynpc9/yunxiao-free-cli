package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/wanghongyi/yunxiao-free-cli/internal/cli"
	"github.com/wanghongyi/yunxiao-free-cli/internal/config"
	"github.com/wanghongyi/yunxiao-free-cli/internal/yunxiao"
)

const (
	tokenHelpURL = "https://help.aliyun.com/zh/yunxiao/developer-reference/obtain-personal-access-token?scm=20140722.H_2841293._.OR_help-T_cn~zh-V_1"
	version      = "0.3.0"
)

var (
	htmlBreakRe  = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>`)
	htmlCloseRe  = regexp.MustCompile(`(?i)</\s*(p|div|li|ul|ol|h[1-6]|blockquote|tr)\s*>`)
	htmlTagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
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
	case "testplan":
		return runTestPlan(ctx, args[1:])
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
	if _, envName := config.TokenFromEnv(); envName != "" {
		fmt.Fprintf(os.Stderr, "检测到环境变量 %s，运行时会优先使用它；如需写入本地配置，可继续输入 PAT。\n", envName)
	}
	line, err := cli.PromptLine("PAT (直接回车可保留当前值): ")
	if err != nil {
		return fmt.Errorf("读取 PAT 失败: %w", err)
	}

	runtimeToken := ""
	if strings.TrimSpace(line) != "" {
		cfg.Token = strings.TrimSpace(line)
		runtimeToken = cfg.Token
	} else {
		runtimeToken, _ = config.EffectiveToken(cfg)
		if runtimeToken == "" {
			return fmt.Errorf("PAT 不能为空；也可通过环境变量 %s 指定", config.EnvToken)
		}
	}

	client := yunxiao.NewClient(cfg.Domain, runtimeToken)
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
			emptyDash(valueString(item.GmtModified)),
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
	fmt.Printf("CreatedAt: %s\n", emptyDash(valueString(item.GmtCreate)))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(valueString(item.GmtModified)))
	fmt.Printf("Description: %s\n", emptyDash(item.Description))
	return nil
}

func runWorkitem(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx workitem <search|get|stats|types|all-types|fields>")
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
	case "stats":
		return runWorkitemStats(ctx, cfg, client, args[1:])
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
	fmt.Fprintln(tw, "ID\tSERIAL\tIDENTIFIER\tTITLE\tSTATUS\tASSIGNEE\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.IDString(),
			emptyDash(firstNonEmpty(item.SerialNumber, item.Identifier)),
			emptyDash(item.Identifier),
			trim(item.Title(), 36),
			emptyDash(item.Status.Name),
			emptyDash(item.AssignedTo.Name),
			emptyDash(valueString(item.GmtModified)),
		)
	}
	return tw.Flush()
}

func runWorkitemGet(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem get", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	workitemID := fs.String("id", "", "工作项 ID")
	serialNumber := fs.String("serial", "", "工作项编号，例如 DMRDEV-1364")
	projectID := fs.String("project", "", "项目 ID；使用 --serial 时必填")
	categories := fs.String("categories", "Req,Bug,Task", "使用 --serial 查找时搜索的工作项分类")
	plainDescription := fs.Bool("plain-description", false, "以纯文本展示描述内容")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*workitemID) == "" && strings.TrimSpace(*serialNumber) == "" {
		return errors.New("--id 或 --serial 至少指定一个")
	}
	if strings.TrimSpace(*serialNumber) != "" && strings.TrimSpace(*projectID) == "" {
		return errors.New("使用 --serial 时 --project 必填")
	}

	org := requireOrgID(cfg, *orgID)
	targetID := strings.TrimSpace(*workitemID)
	if targetID == "" {
		found, err := findWorkitemBySerial(ctx, client, org, strings.TrimSpace(*projectID), strings.TrimSpace(*serialNumber), splitCategories(*categories))
		if err != nil {
			return err
		}
		targetID = found.IDString()
	}

	item, err := client.GetWorkitem(ctx, org, targetID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(item)
	}

	parentLabel := ""
	if strings.TrimSpace(item.ParentID) != "" {
		parentLabel = item.ParentID
		if parent, err := client.GetWorkitem(ctx, org, item.ParentID); err == nil {
			parentLabel = fmt.Sprintf("%s / %s", firstNonEmpty(parent.SerialNumber, parent.Identifier, parent.IDString()), parent.Title())
		}
	}

	descriptionText := plainWorkitemDescription(item.Description)
	descriptionLabel := emptyDash(item.Description)
	if *plainDescription {
		descriptionLabel = emptyDash(descriptionText)
	}

	fmt.Printf("ID: %s\n", item.IDString())
	fmt.Printf("Identifier: %s\n", emptyDash(item.Identifier))
	fmt.Printf("SerialNumber: %s\n", emptyDash(firstNonEmpty(item.SerialNumber, item.Identifier)))
	fmt.Printf("Title: %s\n", emptyDash(item.Title()))
	fmt.Printf("Type: %s\n", emptyDash(item.WorkitemType.Name))
	fmt.Printf("Category: %s\n", emptyDash(item.CategoryID))
	fmt.Printf("Status: %s\n", emptyDash(item.Status.Name))
	fmt.Printf("LogicalStatus: %s\n", emptyDash(item.LogicalStatus))
	fmt.Printf("Assignee: %s\n", emptyDash(item.AssignedTo.Name))
	fmt.Printf("Creator: %s\n", emptyDash(item.Creator.Name))
	fmt.Printf("Modifier: %s\n", emptyDash(item.Modifier.Name))
	fmt.Printf("Verifier: %s\n", emptyDash(item.Verifier.Name))
	fmt.Printf("Space: %s\n", emptyDash(item.Space.Name))
	fmt.Printf("Sprint: %s\n", emptyDash(item.Sprint.Name))
	fmt.Printf("Parent: %s\n", emptyDash(parentLabel))
	fmt.Printf("Labels: %s\n", emptyDash(joinLabelNames(item.Labels)))
	fmt.Printf("Participants: %s\n", emptyDash(joinUserNames(item.Participants)))
	fmt.Printf("Trackers: %s\n", emptyDash(joinUserNames(item.Trackers)))
	fmt.Printf("Versions: %s\n", emptyDash(joinNamedRefs(item.Versions)))
	fmt.Printf("FormatType: %s\n", emptyDash(item.FormatType))
	fmt.Printf("CreatedAt: %s\n", emptyDash(valueString(item.GmtCreate)))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(valueString(item.GmtModified)))
	fmt.Printf("DescriptionChars: %d\n", utf8.RuneCountInString(descriptionText))
	fmt.Printf("Description: %s\n", descriptionLabel)
	return nil
}

func runWorkitemStats(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("workitem stats", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	projectID := fs.String("project", "", "项目 ID (必填)")
	creatorName := fs.String("creator", "", "按创建者姓名过滤")
	categories := fs.String("categories", "Req,Bug,Task", "工作项分类，逗号分隔")
	hydrateDetails := fs.Bool("hydrate-details", false, "逐条拉取详情，获取完整描述")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*projectID) == "" {
		return errors.New("--project 必填")
	}
	if strings.TrimSpace(*creatorName) == "" {
		return errors.New("--creator 必填")
	}

	org := requireOrgID(cfg, *orgID)
	items, err := listProjectWorkitems(ctx, client, org, strings.TrimSpace(*projectID), splitCategories(*categories))
	if err != nil {
		return err
	}

	filtered := make([]yunxiao.WorkItem, 0, len(items))
	for _, item := range items {
		if item.Creator.Name == strings.TrimSpace(*creatorName) {
			filtered = append(filtered, item)
		}
	}

	titleChars := 0
	descriptionChars := 0
	itemsWithDescription := 0
	summaries := make([]map[string]any, 0, len(filtered))
	for _, item := range filtered {
		if *hydrateDetails {
			detail, err := client.GetWorkitem(ctx, org, item.IDString())
			if err != nil {
				return fmt.Errorf("查询工作项详情失败 %s: %w", item.IDString(), err)
			}
			item = detail
		}

		titleCount := utf8.RuneCountInString(item.Title())
		descriptionCount := utf8.RuneCountInString(plainWorkitemDescription(item.Description))
		titleChars += titleCount
		descriptionChars += descriptionCount
		if descriptionCount > 0 {
			itemsWithDescription++
		}
		summaries = append(summaries, map[string]any{
			"id":               item.IDString(),
			"serialNumber":     firstNonEmpty(item.SerialNumber, item.Identifier),
			"title":            item.Title(),
			"creator":          item.Creator.Name,
			"titleChars":       titleCount,
			"descriptionChars": descriptionCount,
			"totalChars":       titleCount + descriptionCount,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return fmt.Sprintf("%v", summaries[i]["serialNumber"]) < fmt.Sprintf("%v", summaries[j]["serialNumber"])
	})

	result := map[string]any{
		"projectId":            strings.TrimSpace(*projectID),
		"creator":              strings.TrimSpace(*creatorName),
		"categories":           splitCategories(*categories),
		"hydrateDetails":       *hydrateDetails,
		"count":                len(filtered),
		"titleChars":           titleChars,
		"descriptionChars":     descriptionChars,
		"totalChars":           titleChars + descriptionChars,
		"itemsWithDescription": itemsWithDescription,
		"items":                summaries,
	}
	if *jsonOut {
		return cli.PrintJSON(result)
	}

	fmt.Printf("ProjectID: %s\n", strings.TrimSpace(*projectID))
	fmt.Printf("Creator: %s\n", strings.TrimSpace(*creatorName))
	fmt.Printf("Categories: %s\n", strings.Join(splitCategories(*categories), ","))
	fmt.Printf("HydrateDetails: %t\n", *hydrateDetails)
	fmt.Printf("Count: %d\n", len(filtered))
	fmt.Printf("TitleChars: %d\n", titleChars)
	fmt.Printf("DescriptionChars: %d\n", descriptionChars)
	fmt.Printf("TotalChars: %d\n", titleChars+descriptionChars)
	fmt.Printf("ItemsWithDescription: %d\n", itemsWithDescription)
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

func runTestPlan(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx testplan <list|results>")
		return nil
	}
	cfg, client, err := ensureReady(ctx, true)
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		return runTestPlanList(ctx, cfg, client, args[1:])
	case "results":
		return runTestPlanResults(ctx, cfg, client, args[1:])
	default:
		return fmt.Errorf("未知 testplan 子命令: %s", args[0])
	}
}

func runTestPlanList(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testplan list", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	projectID := fs.String("project", "", "按项目 ID 过滤")
	nameContains := fs.String("name", "", "按测试计划名称包含文本过滤")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.ListTestPlans(ctx, org)
	if err != nil {
		return err
	}

	filtered := make([]yunxiao.TestPlan, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(*projectID) != "" && item.SpaceIdentifier != strings.TrimSpace(*projectID) {
			continue
		}
		if strings.TrimSpace(*nameContains) != "" && !strings.Contains(item.Name, strings.TrimSpace(*nameContains)) {
			continue
		}
		filtered = append(filtered, item)
	}

	if *jsonOut {
		return cli.PrintJSON(filtered)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TESTPLAN_ID\tNAME\tPROJECT_ID\tCREATED")
	for _, item := range filtered {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			item.TestPlanIdentifier,
			trim(item.Name, 36),
			emptyDash(item.SpaceIdentifier),
			emptyDash(valueString(item.GmtCreate)),
		)
	}
	return tw.Flush()
}

func runTestPlanResults(ctx context.Context, cfg config.Config, client *yunxiao.Client, args []string) error {
	fs := flag.NewFlagSet("testplan results", flag.ContinueOnError)
	orgID := fs.String("org", "", "组织 ID，默认取配置中的默认组织")
	testPlanID := fs.String("plan", "", "测试计划 ID (必填)")
	directoryID := fs.String("directory", "", "目录 ID (必填)")
	jsonOut := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*testPlanID) == "" {
		return errors.New("--plan 必填")
	}
	if strings.TrimSpace(*directoryID) == "" {
		return errors.New("--directory 必填")
	}

	org := requireOrgID(cfg, *orgID)
	items, err := client.GetTestResultList(ctx, org, *testPlanID, *directoryID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TESTCASE_ID\tTITLE\tSTATUS\tEXECUTOR\tTESTREPO_ID\tBUGS")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\n",
			emptyDash(item.Identifier),
			trim(item.Subject, 36),
			emptyDash(item.TestResultStatus),
			emptyDash(item.TestResultExecutor.Name),
			emptyDash(item.SpaceIdentifier),
			item.BugCount,
		)
	}
	return tw.Flush()
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
			emptyDash(valueString(item.GmtModified)),
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
	fmt.Printf("CreatedAt: %s\n", emptyDash(valueString(item.GmtCreate)))
	fmt.Printf("UpdatedAt: %s\n", emptyDash(valueString(item.GmtModified)))
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
		effectiveToken, tokenSource := config.EffectiveToken(cfg)
		fmt.Printf("Stored Token: %s\n", config.MaskToken(cfg.Token))
		fmt.Printf("Effective Token: %s\n", config.MaskToken(effectiveToken))
		fmt.Printf("Token Source: %s\n", tokenSource)
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

	runtimeToken, _ := config.EffectiveToken(cfg)
	if strings.TrimSpace(runtimeToken) == "" {
		fmt.Fprintln(os.Stderr, "首次使用请先配置 PAT。帮助文档:")
		fmt.Fprintln(os.Stderr, tokenHelpURL)
		fmt.Fprintf(os.Stderr, "也可通过环境变量 %s 指定。\\n", config.EnvToken)
		line, err := cli.PromptLine("请输入 PAT: ")
		if err != nil {
			return config.Config{}, nil, err
		}
		if strings.TrimSpace(line) == "" {
			return config.Config{}, nil, errors.New("PAT 不能为空")
		}
		cfg.Token = strings.TrimSpace(line)
		runtimeToken = cfg.Token
	}

	client := yunxiao.NewClient(cfg.Domain, runtimeToken)
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

func splitCategories(s string) []string {
	items := splitCSV(s)
	if len(items) == 0 {
		return []string{"Req", "Bug", "Task"}
	}
	return items
}

func listProjectWorkitems(ctx context.Context, client *yunxiao.Client, orgID, projectID string, categories []string) ([]yunxiao.WorkItem, error) {
	all := make([]yunxiao.WorkItem, 0, 256)
	for _, category := range categories {
		for page := 1; ; page++ {
			items, err := client.SearchWorkitems(ctx, orgID, yunxiao.SearchWorkitemsRequest{
				Category:  category,
				OrderBy:   "gmtCreate",
				Page:      page,
				PerPage:   100,
				Sort:      "desc",
				SpaceID:   projectID,
				SpaceType: "Project",
			})
			if err != nil {
				return nil, fmt.Errorf("查询工作项失败 %s page=%d: %w", category, page, err)
			}
			if len(items) == 0 {
				break
			}
			all = append(all, items...)
			if len(items) < 100 {
				break
			}
		}
	}
	return all, nil
}

func findWorkitemBySerial(ctx context.Context, client *yunxiao.Client, orgID, projectID, serial string, categories []string) (yunxiao.WorkItem, error) {
	conditionMap := map[string]any{
		"conditionGroups": [][]map[string]any{
			{
				{
					"fieldIdentifier": "serialNumber",
					"operator":        "CONTAINS",
					"value":           []string{serial},
					"toValue":         nil,
				},
			},
		},
	}
	conditions, err := json.Marshal(conditionMap)
	if err != nil {
		return yunxiao.WorkItem{}, fmt.Errorf("构造 serial 查询条件失败: %w", err)
	}

	for _, category := range categories {
		for page := 1; ; page++ {
			items, err := client.SearchWorkitems(ctx, orgID, yunxiao.SearchWorkitemsRequest{
				Category:   category,
				Conditions: string(conditions),
				OrderBy:    "gmtCreate",
				Page:       page,
				PerPage:    100,
				Sort:       "desc",
				SpaceID:    projectID,
				SpaceType:  "Project",
			})
			if err != nil {
				return yunxiao.WorkItem{}, fmt.Errorf("按编号查询工作项失败 %s: %w", category, err)
			}
			for _, item := range items {
				if firstNonEmpty(item.SerialNumber, item.Identifier) == serial {
					return item, nil
				}
			}
			if len(items) == 0 || len(items) < 100 {
				break
			}
		}
	}
	return yunxiao.WorkItem{}, fmt.Errorf("未找到工作项编号: %s", serial)
}

func plainWorkitemDescription(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var rich struct {
		HTMLValue string `json:"htmlValue"`
	}
	if json.Unmarshal([]byte(raw), &rich) == nil && strings.TrimSpace(rich.HTMLValue) != "" {
		return htmlToPlainText(rich.HTMLValue)
	}
	return normalizeWhitespace(html.UnescapeString(raw))
}

func htmlToPlainText(s string) string {
	s = html.UnescapeString(s)
	s = htmlBreakRe.ReplaceAllString(s, "\n")
	s = htmlCloseRe.ReplaceAllString(s, "\n")
	s = htmlTagRe.ReplaceAllString(s, " ")
	return normalizeWhitespace(s)
}

func normalizeWhitespace(s string) string {
	return strings.TrimSpace(whitespaceRe.ReplaceAllString(strings.TrimSpace(s), " "))
}

func joinUserNames(items []yunxiao.UserRef) string {
	if len(items) == 0 {
		return ""
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}
	return strings.Join(names, ", ")
}

func joinNamedRefs(items []yunxiao.NamedRef) string {
	if len(items) == 0 {
		return ""
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}
	return strings.Join(names, ", ")
}

func joinLabelNames(items []yunxiao.LabelRef) string {
	if len(items) == 0 {
		return ""
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}
	return strings.Join(names, ", ")
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

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
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
  yx workitem get (--id <workitemId> | --serial <DMRDEV-1364> --project <projectId>) [--plain-description] [--org <organizationId>] [--json]
  yx workitem stats --project <projectId> --creator <name> [--categories Req,Bug,Task] [--hydrate-details] [--org <organizationId>] [--json]
  yx workitem types --project <projectId> --category <Req|Bug|Task> [--org <organizationId>] [--json]
  yx workitem all-types [--categories Req,Bug,Task] [--org <organizationId>] [--json]
  yx workitem fields --project <projectId> --type <workitemTypeId> [--org <organizationId>] [--json]
  yx testplan list [--project <projectId>] [--name <keyword>] [--org <organizationId>] [--json]
  yx testplan results --plan <testPlanId> --directory <directoryId> [--org <organizationId>] [--json]
  yx testcase search --repo <testRepoId> [--org <organizationId>] [--conditions '{"conditionGroups":[[]]}'] [--json]
  yx testcase get --repo <testRepoId> --id <testCaseId> [--org <organizationId>] [--json]
  yx testcase dirs --repo <testRepoId> [--org <organizationId>] [--json]
  yx testcase fields --repo <testRepoId> [--org <organizationId>] [--json]
  yx config show
  yx version
`)
}
