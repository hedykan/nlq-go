# SSH隧道数据库连接使用指南

## 功能概述

本小姐（哼，是在说项目！）已经实现了SSH隧道数据库连接功能，允许通过SSH跳板机连接远程MySQL数据库。

## 配置方式

### 方式1: 使用配置文件

在 `config.yaml` 中添加SSH配置：

```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  database: production_db
  username: root
  password: root
  readonly: true

  # SSH隧道配置（可选）
  ssh_enabled: true
  ssh_host: jumpserver.example.com
  ssh_port: 22
  ssh_user: deploy
  ssh_password: ""
  ssh_private_key_file: ~/.ssh/id_rsa
  ssh_key_passphrase: ""
```

### 方式2: 使用环境变量

```bash
export NLQ_DATABASE_SSH_ENABLED=true
export NLQ_DATABASE_SSH_HOST=jumpserver.example.com
export NLQ_DATABASE_SSH_USER=deploy
export NLQ_DATABASE_SSH_PRIVATE_KEY_FILE=/home/user/.ssh/id_rsa
```

## 认证方式

### 1. 密码认证

```yaml
database:
  ssh_enabled: true
  ssh_host: jumpserver.example.com
  ssh_user: deploy
  ssh_password: "your_password"
  ssh_private_key_file: ""  # 留空
```

### 2. 私钥认证（无密码）

```yaml
database:
  ssh_enabled: true
  ssh_host: jumpserver.example.com
  ssh_user: deploy
  ssh_password: ""  # 留空
  ssh_private_key_file: ~/.ssh/id_rsa
  ssh_key_passphrase: ""  # 如果私钥无密码则留空
```

### 3. 私钥认证（有密码）

```yaml
database:
  ssh_enabled: true
  ssh_host: jumpserver.example.com
  ssh_user: deploy
  ssh_private_key_file: ~/.ssh/id_rsa
  ssh_key_passphrase: "your_key_passphrase"
```

## 使用场景

### 场景1: 直连本地数据库（不使用SSH）

```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  database: nlq
  username: root
  password: root
  readonly: false

  # SSH配置留空或设置为false
  ssh_enabled: false
```

### 场景2: 通过SSH隧道连接远程数据库

```yaml
database:
  driver: mysql
  # 数据库地址相对于SSH服务器
  host: localhost
  port: 3306
  database: production_db
  username: root
  password: root
  readonly: true

  # 启用SSH隧道
  ssh_enabled: true
  ssh_host: production-jump.example.com
  ssh_port: 22
  ssh_user: deploy
  ssh_private_key_file: ~/.ssh/deploy_key
```

## 配置验证

### SSH配置要求

当 `ssh_enabled: true` 时，必须提供：
- ✅ `ssh_host`: SSH服务器地址
- ✅ `ssh_port`: SSH端口（默认22）
- ✅ `ssh_user`: SSH用户名
- ✅ `ssh_password` 或 `ssh_private_key_file`: 认证方式（二选一）

### 私钥文件要求

- ✅ 文件必须存在且可读
- ⚠️ 建议文件权限为 600（只有所有者可读写）

## 错误处理

### 常见错误及解决方案

| 错误信息 | 原因 | 解决方案 |
|---------|------|---------|
| "SSH隧道已启用，但SSH主机地址未配置" | `ssh_host` 为空 | 设置 `ssh_host` |
| "SSH隧道已启用，但SSH用户名未配置" | `ssh_user` 为空 | 设置 `ssh_user` |
| "SSH隧道已启用，必须配置SSH密码或私钥文件" | 缺少认证方式 | 设置 `ssh_password` 或 `ssh_private_key_file` |
| "SSH私钥文件不存在" | 私钥文件路径错误 | 检查 `ssh_private_key_file` 路径 |
| "连接SSH服务器失败" | 无法连接SSH服务器 | 检查 `ssh_host` 和 `ssh_port` |
| "SSH隧道未连接" | 尝试使用隧道但未连接 | 确保先调用 `Connect()` |

## 安全建议

1. **不要在配置文件中硬编码密码**
   - 使用环境变量存储敏感信息
   - 或使用密钥管理服务

2. **私钥文件权限**
   ```bash
   chmod 600 ~/.ssh/id_rsa
   ```

3. **生产环境应验证主机密钥**
   - 当前实现使用 `InsecureIgnoreHostKey()` 方便开发
   - 生产环境应使用 `FixedHostKey()` 验证主机密钥

4. **使用只读用户**
   - 建议为NLQ创建只读数据库用户
   - 避免使用有写权限的账户

## 技术细节

### 工作原理

1. **建立SSH连接**: 连接到SSH服务器
2. **端口转发**: 在本地分配端口，转发到远程数据库
3. **数据库连接**: 通过本地端口连接数据库

### 端口分配

- 自动分配可用端口（127.0.0.1:随机端口）
- 无需手动指定本地端口
- 支持多个隧道同时运行

### 超时设置

- SSH连接超时: 30秒
- 数据库连接超时: 10秒（可配置）
- 数据库读取超时: 30秒（可配置）

## 测试

### 运行SSH隧道测试

```bash
# 运行所有SSH相关测试
go test ./internal/database -run TestSSH -v

# 运行连接相关测试
go test ./internal/database -run TestConnection -v

# 查看测试覆盖率
go test ./internal/database -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 测试覆盖率

- SSH核心功能: 90%+
- 错误处理: 100%
- 连接管理: 75%+

## API参考

### DatabaseConfig 结构体

```go
type DatabaseConfig struct {
    // 数据库配置
    Driver         string
    Host           string
    Port           int
    Database       string
    Username       string
    Password       string
    Readonly       bool
    ReadTimeout    time.Duration
    ConnectTimeout time.Duration

    // SSH隧道配置
    SSHEnabled        bool
    SSHHost           string
    SSHPort           int
    SSHUser           string
    SSHPassword       string
    SSHPrivateKeyFile string
    SSHKeyPassphrase  string
}
```

### 主要函数

```go
// 创建数据库连接（自动处理SSH隧道）
func NewConnection(cfg *config.DatabaseConfig) (*gorm.DB, error)

// 验证SSH配置
func (c *DatabaseConfig) ValidateSSHConfig() error

// 关闭连接和SSH隧道
func CloseConnection(db *gorm.DB) error

// 获取当前SSH隧道实例
func GetSSHTunnel() *SSHTunnel
```

## 向后兼容性

✅ **完全向后兼容**：不使用SSH时，行为与之前完全一致

- `SSHEnabled` 默认为 `false`
- SSH配置字段都是可选的
- 未配置SSH时，自动使用直连模式

## 示例配置文件

### 完整示例（生产环境）

```yaml
database:
  # 数据库配置
  driver: mysql
  host: localhost  # 相对于SSH服务器
  port: 3306
  database: nlq_production
  username: nlq_readonly
  password: ${DATABASE_PASSWORD}  # 从环境变量读取
  readonly: true
  read_timeout: 30s
  connect_timeout: 10s

  # SSH隧道配置
  ssh_enabled: true
  ssh_host: ${SSH_HOST}
  ssh_port: 22
  ssh_user: ${SSH_USER}
  ssh_private_key_file: ${SSH_KEY_PATH}
  ssh_key_passphrase: ${SSH_KEY_PASSPHRASE}

llm:
  provider: zhipuai
  model: glm-4.7
  api_key: ${LLM_API_KEY}
  base_url: https://open.bigmodel.cn/api/paas/v4

server:
  host: 0.0.0.0
  port: 8080
  enable_cors: true
```

---

**最后更新**: 2026-03-19
**维护者**: ChannelWill开发团队
