# splitdns

[![CI](https://github.com/soulteary/splitdns/actions/workflows/ci.yml/badge.svg)](https://github.com/soulteary/splitdns/actions/workflows/ci.yml) [![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go](https://img.shields.io/github/go-mod/go-version/soulteary/splitdns)](https://github.com/soulteary/splitdns/blob/main/go.mod)

![](.github/assets/splitdns-banner.png)

`splitdns` 是一个小巧、安全的命令行工具,用于管理 macOS 的 `/etc/resolver`
配置,以实现**基于域名后缀的 Split DNS(分流 DNS)**:将指定域名后缀
(如 `lab.dev`、`corp.example.com`)的查询转发到专用 DNS 服务器,其余流量
继续走系统默认的解析器。

它仅支持 macOS,以单个静态可执行文件发布,并且从不通过 `sh -c` 执行命令——
所有外部命令都以分离的参数方式调用。

> 其他语言:[English](./README.md)

## 功能特性

- 安全地创建、更新、删除、列出与查看 `/etc/resolver` 条目。
- **原子写入**(同目录临时文件 → `fsync` → `chmod 0644` → `rename`),覆盖前
  自动**备份**。
- **符号链接防护**与严格的路径约束:当目标是符号链接、非普通文件,或会逃逸出
  `/etc/resolver` 时,操作会被拒绝。
- **保序解析器**:注释、空行、顺序、重复的 nameserver 以及未知但合法的指令都会
  被完整保留。
- **诊断**(`check`)与端到端**解析测试**(`test`):使用手写的最小 DNS 报文
  直接探测(UDP 带超时,必要时回退 TCP),绝不以裸 TCP 连接来判断可用性。
- 明确区分“配置已写入”与“缓存已刷新”,部分成功的情况始终可见。
- 每个命令都支持机器可读的 `--json` 输出。

## 环境要求

- macOS(Apple Silicon 或 Intel)。
- `scutil`、`dscacheutil`、`killall`(macOS 自带)。
- 写操作(`add`、`set`、`remove`、`flush`)会修改 `/etc/resolver`,需要 `sudo`。
  `splitdns` **绝不**自动提权;它会打印出需要执行的完整 `sudo splitdns …` 命令。

## 安装

### Homebrew

```bash
brew tap soulteary/tap
brew install soulteary/tap/splitdns
```

验证:

```bash
splitdns version
```

### 从源码安装

```bash
go install github.com/soulteary/splitdns@latest
```

或本地构建:

```bash
git clone https://github.com/soulteary/splitdns.git
cd splitdns
make build      # 生成 ./splitdns
```

### 发布产物

通过 GoReleaser 在 Releases 页面提供预编译的 `darwin/amd64` 与 `darwin/arm64`
压缩包。

## 使用方法

```
splitdns <command> [flags]

命令:
  add         为域名后缀新增一个 resolver 条目
  set         更新已有 resolver 条目的字段
  remove      删除一个 resolver 条目(别名:rm)
  list        列出 resolver 条目(别名:ls)
  show        查看一个 resolver 条目
  check       运行配置与环境诊断
  test        通过分流各层测试主机名解析
  flush       刷新 macOS DNS 缓存
  completion  生成 shell 自动补全脚本
  version     打印版本信息

全局参数:
  --dry-run   仅展示计划改动,不修改系统
  --json      输出机器可读的 JSON
  --quiet     抑制非必要输出
  --no-color  关闭彩色输出
```

### 示例

将 `lab.dev` 路由到本地 53 端口的 DNS 服务器:

```bash
sudo splitdns add lab.dev --nameserver 127.0.0.1 --port 53
```

指定多个 nameserver 与自定义端口:

```bash
sudo splitdns add corp.example.com \
  --nameserver 10.0.0.53 --nameserver 10.0.1.53 --port 5353
```

在不写入任何文件的情况下预览改动:

```bash
splitdns add lab.dev --dry-run
```

只更新已有条目的端口(其余字段与注释保持不变):

```bash
sudo splitdns set lab.dev --port 5353
```

列出与查看:

```bash
splitdns list
splitdns show lab.dev
splitdns show lab.dev --raw       # 原始文件内容
```

删除(需要确认,自动化场景用 `--yes`):

```bash
sudo splitdns remove lab.dev
sudo splitdns remove lab.dev --yes
```

诊断与测试:

```bash
splitdns check                    # 校验所有条目 + 环境
splitdns check lab.dev            # 聚焦某个后缀
splitdns test host.lab.dev        # 三层解析测试
```

手动刷新缓存:

```bash
sudo splitdns flush
```

## 命令参考

### `add <domain>`

创建新的 resolver 文件。若已存在则失败,除非使用 `--force`(会先备份现有文件)。

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--nameserver` | `127.0.0.1` | nameserver IP(可重复) |
| `--port` | `53` | DNS 服务器端口 |
| `--search-order` | | `search_order` 值 |
| `--timeout` | | `timeout` 值(秒) |
| `--force` | | 允许覆盖并放宽 `.local`/单标签警告 |
| `--no-flush` | | 写入后不刷新 DNS 缓存 |
| `--backup-dir` | 系统临时目录 | 覆盖前备份的目录 |

全局参数 `--dry-run` 可预览计划改动(目标路径、计划写入的内容以及缓存刷新
命令),而不写入任何文件。

### `set <domain>`

更新已有条目;文件不存在时失败。只修改你指定的字段,其余指令与注释均保留。
更新前会先备份。

### `remove <domain>`

先展示目标文件,再删除。在 TTY 下会提示确认;自动化场景用 `--yes`。在非交互
会话中,未加 `--yes` 时会拒绝删除。支持全局 `--dry-run` 与 `--no-flush`。

### `list`

打印条目表格(名称、域名、nameserver、端口、是否受管)。`--json` 输出稳定的
数组。

### `show <domain>`

打印结构化视图;`--raw` 原样打印原始文件。

### `check [domain]`

运行诊断:平台、resolver 目录、文件名合法性、普通文件/符号链接检查、权限、
语法、nameserver/端口合法性、`.local` 使用、后缀重叠、相同配置、DNS 可达性
(通过真实 DNS 查询)、`scutil --dns` 加载状态,以及影响该域名的 `/etc/hosts`
记录。任意 `ERROR` 状态都会返回非零退出码。

### `test <hostname>`

三层测试:(1)`dscacheutil -q host`;(2)`scutil --dns` 加载状态;
(3)针对已配置 nameserver 的直接 DNS 查询。报告最长匹配后缀、resolver 文件、
nameserver、规则是否已加载、系统解析与直接查询地址、一致性,以及排障建议。

### `flush`

执行 `dscacheutil -flushcache`,随后 `killall -HUP mDNSResponder`。当
`mDNSResponder` 进程不存在时视为无害的空操作。搭配全局 `--dry-run` 参数时,
只打印计划执行的命令而不实际执行。

## 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `2` | 参数 / 用法错误 |
| `3` | 权限错误(需要 `sudo`) |
| `4` | 配置错误 |
| `5` | 运行时检查失败(如 `check` 出现 ERROR,或配置写入成功但刷新失败) |

## 安全说明

- 域名会被规范化(转小写、去除尾点),若包含 `/`、`\`、`..`、NUL、空白或
  控制字符则会被拒绝。
- 目标路径会被 `Clean`,且必须严格位于 `/etc/resolver` 之内。
- 在任何读、写、删操作前,都会通过 `Lstat` 校验目标为普通文件;拒绝符号链接。
- 不使用 `sh -c`,不做 shell 字符串拼接;命令通过 `os/exec` 以分离参数执行。
- `splitdns` 绝不自动提权。

## 已知限制

- 仅支持 macOS,没有 Linux/Windows 支持。
- `splitdns` 只管理 `/etc/resolver` 文件。它不运行 DNS 服务器、不编辑
  `/etc/hosts`、不修改全局网络/DNS 设置、也不安装 dnsmasq/CoreDNS。
- 其他工具创建的条目可被读取并接管(写入时会加上 `# Managed by splitdns`
  标记),但 `splitdns` 不会重排无关文件的格式。

## 开发

```bash
make fmt-check   # gofmt
make vet         # go vet
make test        # 单元测试(自洽;不执行真实系统命令)
make lint        # golangci-lint
make build       # 构建 ./splitdns

# 只读集成测试(macOS;观测真实 DNS 状态):
make integration
```

单元测试是自洽的:所有文件系统访问都使用临时目录,所有外部命令都经过可注入的
fake runner。集成测试使用 `//go:build integration` 标签,绝不修改
`/etc/resolver`。

## 许可证

[Apache-2.0](./LICENSE) © soulteary
