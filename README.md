# yunxiao-free-cli

基于 Go 的云效 CLI，用于访问云效免费版范围内的组织、项目协作和测试管理数据。

当前版本的能力边界参考了云效 MCP 项目 [aliyun/alibabacloud-devops-mcp-server](https://github.com/aliyun/alibabacloud-devops-mcp-server)，但只实现免费版内最有价值的只读子集，便于后续继续封装成 agent skill。

## API 参考

- 服务接入点域名: <https://help.aliyun.com/zh/yunxiao/developer-reference/service-access-point-domain>
- 组织: <https://help.aliyun.com/zh/yunxiao/developer-reference/listorganizations>
- 项目协作（后续扩展）: <https://help.aliyun.com/zh/yunxiao/developer-reference/createproject>
- 测试管理（后续写入扩展）: <https://help.aliyun.com/zh/yunxiao/developer-reference/createtestcase>

本版本已使用的只读接口：

- `ListOrganizations`
- `SearchMembers`
- `SearchProjects`
- `GetProject`
- `SearchWorkitems`
- `GetWorkitem`
- `ListAllWorkitemTypes`
- `ListWorkitemTypes`
- `GetWorkitemTypeFieldConfig`
- `ListTestPlan`
- `GetTestResultList`
- `SearchTestCases`
- `GetTestCase`
- `ListDirectories`
- `GetTestcaseFieldConfig`

## 安装

### 1. 本地构建

```bash
go build -o yx ./cmd/yx
```

### 2. 远程安装脚本（用于 `curl | bash`）

可直接使用 GitHub 上的安装脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/whynpc9/yunxiao-free-cli/main/install.sh | bash
```

脚本优先下载 GitHub Release 二进制包；如果不可用会回退到 `go install`。

### 3. 作为 skill 安装

如果要把本项目作为 Codex skill 安装，可使用：

```bash
npx skills add whynpc9/yunxiao-free-cli --skill yunxiao-cli
```

skill 目录位于 `yunxiao-cli/`，用于指导 agent 安装并调用本仓库提供的 `yx` CLI。

## 首次使用

```bash
yx auth
```

也可以不写入本地配置，直接通过环境变量指定 token：

```bash
export YUNXIAO_TOKEN="<your-pat>"
```

`auth` 会提示录入 PAT（个人访问令牌）并附带帮助页：

<https://help.aliyun.com/zh/yunxiao/developer-reference/obtain-personal-access-token?scm=20140722.H_2841293._.OR_help-T_cn~zh-V_1>

配置存储位置：

- macOS: `~/Library/Application Support/yunxiao-free-cli/config.json`
- Linux: `~/.config/yunxiao-free-cli/config.json`

存储字段包括：

- `token`
- `domain`（默认 `openapi-rdc.aliyuncs.com`）
- `defaultOrganizationId`
- `defaultOrganizationName`

token 优先级：

- `YUNXIAO_TOKEN`
- `YX_TOKEN`
- 本地 `config.json` 中的 `token`

可通过 `yx config show` 查看当前生效 token 的来源。

## 默认组织选择逻辑

- 账号只有一个组织: 自动设为默认组织
- 账号属于多个组织: 交互式提示选择默认组织
- 可随时切换:

```bash
yx org use
# 或
# yx org use <organizationId>
```

## 命令

### 组织

```bash
yx org list
yx org list --json
yx org use
yx org use <organizationId>
yx org members --query 张三
yx org members --statuses ENABLED --json
```

### 项目

```bash
yx project search
yx project search --conditions '{"conditionGroups":[[{"fieldIdentifier":"name","operator":"BETWEEN","value":["测试项目"]}]]}'
yx project get --id <projectId>
yx project get --id <projectId> --json
```

### 工作项

```bash
# 查询工作项
yx workitem search --category Req --space-id <projectId>
yx workitem search --category Bug --space-id <projectId> --conditions '{"conditionGroups":[[{"fieldIdentifier":"subject","operator":"BETWEEN","value":["登录"]}]]}'
yx workitem search --category Task --space-id <projectId> --json

# 查询单个工作项
yx workitem get --id <workitemId>
yx workitem get --id <workitemId> --json

# 查询工作项类型
yx workitem types --project <projectId> --category Req
yx workitem all-types --categories Req,Bug,Task

# 查询工作项类型字段配置
yx workitem fields --project <projectId> --type <workitemTypeId>
```

### 测试用例（只读）

```bash
# 查询测试计划
yx testplan list
yx testplan list --project <projectId>
yx testplan list --name 测试 --json

# 查询测试计划中的测试结果
yx testplan results --plan <testPlanId> --directory <directoryId>
yx testplan results --plan <testPlanId> --directory <directoryId> --json

# 查询用例
yx testcase search --repo <testRepoId>
yx testcase search --repo <testRepoId> --page 1 --per-page 20
yx testcase search --repo <testRepoId> --conditions '{"name":"登录"}'
yx testcase search --repo <testRepoId> --json

# 查询单个用例
yx testcase get --repo <testRepoId> --id <testCaseId>
yx testcase get --repo <testRepoId> --id <testCaseId> --json

# 查询目录
yx testcase dirs --repo <testRepoId>

# 查询测试用例字段配置
yx testcase fields --repo <testRepoId>
```

组织可通过 `--org` 显式指定；未指定时使用默认组织。

`--conditions` 和 `--extra-conditions` 需要传官方 OpenAPI 使用的 JSON 对象字符串，CLI 只做格式校验，不改写内容。

## 开发

```bash
go test ./...
```
