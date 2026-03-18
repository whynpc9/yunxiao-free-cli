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
	version      = "0.1.0"
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
		fmt.Println("用法: yx org <list|use>")
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
		orgID := ""
		if len(args) >= 2 {
			orgID = strings.TrimSpace(args[1])
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

	default:
		return fmt.Errorf("未知 org 子命令: %s", args[0])
	}
}

func runTestCase(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println("用法: yx testcase <search|get|dirs>")
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
	sortField := fs.String("sort", "desc", "排序方向 asc|desc")
	conditionsJSON := fs.String("conditions", "", "查询条件 JSON 对象")
	jsonOut := fs.Bool("json", false, "JSON 输出")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repoID) == "" {
		return errors.New("--repo 必填")
	}

	org := pickOrgID(cfg, *orgID)
	conds, err := parseConditions(*conditionsJSON)
	if err != nil {
		return err
	}

	req := yunxiao.SearchTestCasesRequest{
		Page:       *page,
		PerPage:    *perPage,
		OrderBy:    *orderBy,
		SortField:  *sortField,
		Conditions: conds,
	}
	items, err := client.SearchTestCases(ctx, org, *repoID, req)
	if err != nil {
		return err
	}

	if *jsonOut {
		return cli.PrintJSON(items)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tIDENTIFIER\tNAME\tSTATE\tPRIORITY\tOWNER\tUPDATED")
	for _, item := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.IDString(),
			trim(item.Identifier, 18),
			trim(item.Name, 40),
			emptyDash(item.State),
			emptyDash(item.Priority),
			emptyDash(item.Owner),
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

	org := pickOrgID(cfg, *orgID)
	item, err := client.GetTestCase(ctx, org, *repoID, *caseID)
	if err != nil {
		return err
	}
	if *jsonOut {
		return cli.PrintJSON(item)
	}

	fmt.Printf("ID: %s\n", item.IDString())
	fmt.Printf("Identifier: %s\n", emptyDash(item.Identifier))
	fmt.Printf("Name: %s\n", emptyDash(item.Name))
	fmt.Printf("State: %s\n", emptyDash(item.State))
	fmt.Printf("Priority: %s\n", emptyDash(item.Priority))
	fmt.Printf("Owner: %s\n", emptyDash(item.Owner))
	fmt.Printf("Creator: %s\n", emptyDash(item.Creator))
	fmt.Printf("DirectoryID: %s\n", emptyDash(item.DirectoryID))
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

	org := pickOrgID(cfg, *orgID)
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

func parseConditions(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		return nil, fmt.Errorf("--conditions 必须为 JSON 对象: %w", err)
	}
	return m, nil
}

func pickOrgID(cfg config.Config, explicit string) string {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		return explicit
	}
	return cfg.DefaultOrganizationID
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
  yx testcase search --repo <testRepoId> [--org <organizationId>] [--page 1] [--per-page 20] [--order-by gmtModified] [--sort desc] [--conditions '{"name":"登录"}'] [--json]
  yx testcase get --repo <testRepoId> --id <testCaseId> [--org <organizationId>] [--json]
  yx testcase dirs --repo <testRepoId> [--org <organizationId>] [--json]
  yx config show
  yx version
`)
}
