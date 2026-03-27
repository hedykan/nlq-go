package database

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig SSH配置
type SSHConfig struct {
	Host           string // SSH服务器地址
	Port           int    // SSH服务器端口
	User           string // SSH用户名
	Password       string // SSH密码（与私钥二选一）
	PrivateKeyFile string // SSH私钥文件路径
	KeyPassphrase  string // 私钥密码短语
}

// SSHTunnel SSH隧道管理器
type SSHTunnel struct {
	client    *ssh.Client
	config    *SSHConfig
	localPort int
	mu        sync.RWMutex
	connected bool
}

// NewSSHTunnel 创建SSH隧道
func NewSSHTunnel(cfg *SSHConfig) (*SSHTunnel, error) {
	if cfg == nil {
		return nil, NewSSHError("NewSSHTunnel", nil, "SSH配置不能为nil")
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, NewSSHError("NewSSHTunnel", err, "SSH配置验证失败")
	}

	// 检查认证方式
	if cfg.Password == "" && cfg.PrivateKeyFile == "" {
		return nil, NewSSHError("NewSSHTunnel", nil, "必须提供密码或私钥文件")
	}

	// 如果使用私钥，验证私钥文件
	if cfg.PrivateKeyFile != "" {
		if _, err := os.Stat(cfg.PrivateKeyFile); os.IsNotExist(err) {
			return nil, NewSSHError("NewSSHTunnel", err, fmt.Sprintf("私钥文件不存在: %s", cfg.PrivateKeyFile))
		}
	}

	return &SSHTunnel{
		config: cfg,
	}, nil
}

// Connect 连接SSH服务器
func (t *SSHTunnel) Connect() error {
	return t.ConnectWithTimeout(30 * time.Second)
}

// ConnectWithTimeout 带超时的连接
func (t *SSHTunnel) ConnectWithTimeout(timeout time.Duration) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 构建SSH客户端配置
	sshConfig, err := buildSSHClientConfig(t.config)
	if err != nil {
		return NewSSHError("Connect", err, "构建SSH客户端配置失败")
	}

	// 连接SSH服务器
	address := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return NewSSHError("Connect", err, fmt.Sprintf("连接SSH服务器失败: %s", address))
	}

	t.client = client
	t.connected = true

	return nil
}

// ForwardPort 转发端口
func (t *SSHTunnel) ForwardPort(remoteHost string, remotePort int) (string, error) {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return "", NewSSHError("ForwardPort", nil, "SSH隧道未连接，请先调用Connect()")
	}
	t.mu.RUnlock()

	// 获取可用本地端口
	localPort, err := getAvailablePort()
	if err != nil {
		return "", NewSSHError("ForwardPort", err, "获取可用本地端口失败")
	}

	// 监听本地端口
	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return "", NewSSHError("ForwardPort", err, fmt.Sprintf("监听本地端口失败: %s", localAddr))
	}

	// 启动端口转发
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				// 监听器已关闭
				return
			}

			go t.forwardConnection(localConn, remoteHost, remotePort)
		}
	}()

	t.localPort = localPort
	return localAddr, nil
}

// forwardConnection 转发单个连接
func (t *SSHTunnel) forwardConnection(localConn net.Conn, remoteHost string, remotePort int) {
	defer localConn.Close()

	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return
	}

	// 通过SSH隧道连接远程主机
	remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// 双向数据转发
	done := make(chan struct{})

	go func() {
		copyData(localConn, remoteConn)
		close(done)
	}()

	go func() {
		copyData(remoteConn, localConn)
	}()

	<-done
}

// copyData 复制数据
func copyData(dst net.Conn, src net.Conn) {
	defer dst.Close()
	defer src.Close()

	_, _ = io.Copy(dst, src)
}

// Close 关闭SSH隧道
func (t *SSHTunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		err := t.client.Close()
		t.client = nil
		t.connected = false
		return err
	}

	t.connected = false
	return nil
}

// IsConnected 检查是否已连接
func (t *SSHTunnel) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// GetLocalPort 获取本地端口
func (t *SSHTunnel) GetLocalPort() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.localPort
}

// Validate 验证SSH配置
func (c *SSHConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("SSH主机地址不能为空")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("SSH端口必须在1-65535之间")
	}

	if c.User == "" {
		return fmt.Errorf("SSH用户名不能为空")
	}

	return nil
}

// buildSSHClientConfig 构建SSH客户端配置
func buildSSHClientConfig(cfg *SSHConfig) (*ssh.ClientConfig, error) {
	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该使用已知主机密钥
		Timeout:         30 * time.Second,
	}

	// 添加认证方法
	if cfg.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(cfg.Password))
	}

	if cfg.PrivateKeyFile != "" {
		signer, err := getPrivateKeySigner(cfg.PrivateKeyFile, cfg.KeyPassphrase)
		if err != nil {
			return nil, err
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	return sshConfig, nil
}

// getPrivateKeySigner 获取私钥签名器
func getPrivateKeySigner(keyFile string, passphrase string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	// 如果有密码短语，尝试解析加密的私钥
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
		if err != nil {
			return nil, err
		}
		return signer, nil
	}

	// 尝试解析未加密的私钥
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// parseAddress 解析地址
func parseAddress(address string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}

	port, err := parsePort(portStr)
	if err != nil {
		return "", 0, err
	}

	return host, port, nil
}

// parsePort 解析端口号
func parsePort(portStr string) (int, error) {
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, fmt.Errorf("无效的端口号: %s", portStr)
	}

	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("端口号必须在1-65535之间: %d", port)
	}

	return port, nil
}

// getAvailablePort 获取可用端口
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().String()
	_, port, err := parseAddress(addr)
	if err != nil {
		return 0, err
	}

	return port, nil
}

// generateTestPrivateKey 生成测试用RSA私钥
func generateTestPrivateKey(filename string, passphrase string) error {
	// 生成2048位RSA私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// 编码为PEM格式
	var pemData []byte
	if passphrase != "" {
		// 加密私钥 - 使用PKCS8格式
		bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return err
		}
		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: bytes,
		}
		pemData = pem.EncodeToMemory(block)
	} else {
		// 未加密私钥 - 使用PKCS1格式
		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}
		pemData = pem.EncodeToMemory(block)
	}

	// 确保目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(filename, pemData, 0600)
}

// isPrivateKeyEncrypted 检查私钥是否加密
func isPrivateKeyEncrypted(keyFile string) (bool, error) {
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil {
		return false, err
	}

	// 尝试解析为未加密的私钥
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return false, fmt.Errorf("无法解析PEM块")
	}

	// 尝试解析为PKCS1私钥
	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		// 成功解析，说明未加密
		return false, nil
	}

	// 尝试解析为PKCS8私钥
	_, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		// 成功解析，说明未加密
		return false, nil
	}

	// 如果都失败了，可能是加密的
	return true, nil
}
