# yunxiao-free-cli

基于 Go 的云效 CLI（首轮只实现测试用例只读能力），用于访问云效免费版范围内的组织与测试管理数据。

## API 参考

- 服务接入点域名: <https://help.aliyun.com/zh/yunxiao/developer-reference/service-access-point-domain>
- 组织: <https://help.aliyun.com/zh/yunxiao/developer-reference/listorganizations>
- 项目协作（后续扩展）: <https://help.aliyun.com/zh/yunxiao/developer-reference/createproject>
- 测试管理（后续写入扩展）: <https://help.aliyun.com/zh/yunxiao/developer-reference/createtestcase>

本版本已使用的只读接口：

- `ListOrganizations`
- `SearchTestCases`
- `GetTestCase`
- `ListDirectories`

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

## 首次使用

```bash
yx auth
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
```

### 测试用例（只读）

```bash
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
```

组织可通过 `--org` 显式指定；未指定时使用默认组织。

## 开发

```bash
go test ./...
```
