# Security Policy

## Supported versions

Security fixes are provided for the latest released version.

| Version | Supported |
| --- | --- |
| Latest release (`0.1.x`) | ✅ |
| Older releases | ❌ |

## Security posture

`splitdns` is designed to be conservative when modifying macOS system
configuration under `/etc/resolver`:

- **No shell interpretation.** External commands are invoked via `os/exec` with
  separated arguments — there is no `sh -c` and no shell string concatenation.
- **Symlink protection.** Files are verified to be regular files (via `Lstat`)
  before any read, write, or delete; symlinks are refused.
- **Strict path containment.** Target paths are cleaned and must stay strictly
  inside `/etc/resolver`; domains containing `/`, `\`, `..`, NUL, whitespace, or
  control characters are rejected.
- **Never auto-elevates.** `splitdns` never elevates privileges on its own; it
  prints the exact `sudo splitdns …` command for you to run.

See the "Security notes" section of the [README](./README.md) for details.

## Reporting a vulnerability

Please report security issues **privately** — do not open a public issue for a
vulnerability.

- Preferred: open a private
  [GitHub Security Advisory](https://github.com/soulteary/splitdns/security/advisories/new).
- Alternatively: email the maintainer (soulteary).

Please include steps to reproduce, affected version(s), and any relevant
environment details. We will acknowledge your report and work with you on a fix
and coordinated disclosure.

---

## 简体中文

### 支持的版本

我们仅为最新发布版本提供安全修复。

| 版本 | 是否支持 |
| --- | --- |
| 最新发布版本（`0.1.x`） | ✅ |
| 更旧的版本 | ❌ |

### 安全设计

`splitdns` 在修改 macOS 系统的 `/etc/resolver` 配置时采用保守策略：

- **不经过 shell 解释。** 外部命令通过 `os/exec` 以分离参数的方式调用——不使用
  `sh -c`，也没有 shell 字符串拼接。
- **符号链接防护。** 在任何读、写、删除操作前，都会通过 `Lstat` 验证目标是常规
  文件；拒绝处理符号链接。
- **严格的路径限制。** 目标路径会被规范化，并且必须严格位于 `/etc/resolver` 之内；
  拒绝包含 `/`、`\`、`..`、NUL、空白或控制字符的域名。
- **绝不自动提权。** `splitdns` 永远不会自行提升权限，而是打印出需要你执行的完整
  `sudo splitdns …` 命令。

详情请见 [README](./README.md) 中的 “Security notes” 部分。

### 报告漏洞

请**私下**报告安全问题——不要为漏洞公开创建 issue。

- 推荐：创建私有的
  [GitHub 安全公告（Security Advisory）](https://github.com/soulteary/splitdns/security/advisories/new)。
- 或者：通过邮件联系维护者（soulteary）。

请附上复现步骤、受影响的版本以及相关的环境信息。我们会确认你的报告，并与你一起
推进修复与协同披露。
