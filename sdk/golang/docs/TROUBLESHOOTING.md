# ASR SDK 故障排除指南

本文档提供了使用ASR SDK (OpenAI Realtime API)时可能遇到的问题及其解决方案。

## 🔍 快速诊断

### 1. 使用内置调试工具

```go
// 获取详细的调试信息
stats := recognizer.GetStats()
fmt.Printf("识别器状态: %+v\n", stats)

// 检查连接状态
status := recognizer.GetConnectionStatus()
switch status {
case asr.ConnectionStatusConnected:
    fmt.Println("✅ 连接正常")
case asr.ConnectionStatusDisconnected:
    fmt.Println("❌ 连接已断开")
case asr.ConnectionStatusFailed:
    fmt.Println("💥 连接失败")
}
```

### 2. 启用详细日志

```go
// 使用结构化日志
logger := log.New()
logger.WithFields(logrus.Fields{
    "session_id": sessionID,
    "event_type": eventType,
    "error": err.Error(),
}).Error("错误详情", err)
```

## 📡 连接问题

### 连接失败

#### 问题症状
- ❌ `connection failed: dial tcp: lookup host: no such host`
- ❌ `connection timeout`
- ❌ `connection failed: handshake error`
- ❌ 频繁断开连接

#### 诊断步骤

1. **检查URL和服务器状态**
```go
// 验证URL格式
if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
    log.Printf("❌ 无效的WebSocket URL: %s", url)
}

// 检查服务器可访问性
import "net"
conn, err := net.DialTimeout("tcp", "localhost:8088", 5*time.Second)
if err != nil {
    log.Printf("❌ 服务器不可访问: %v", err)
}
conn.Close()
```

2. **检查网络配置**
```bash
# 检查防火墙设置
sudo ufw status

# 检查端口占用
netstat -tlnp | grep :8088

# 测试网络连通性
telnet localhost 8088
ping localhost -c 4
```

3. **验证TLS证书**
```bash
# 检查证书有效性
openssl s_client -connect ws://localhost:8088 -showcerts

# 对于wss连接
openssl s_client -connect wss://localhost:8088 -showcerts
```

#### 解决方案

1. **服务器配置修复**
   - 确保WebSocket服务正在运行
   - 检查防火墙设置
   - 验证端口8088已开放

2. **客户端配置优化**
```go
config := asr.DefaultConfig()
config.Timeout = 30 * time.Second
config.EnableReconnect = true
config.MaxReconnectAttempts = 5
config.ReconnectDelay = 2 * time.Second
```

3. **使用备用连接**
```go
// 主要连接
recognizer1, err := asr.CreateRecognizer(config1)
if err != nil {
    log.Fatal(err)
}

// 备用连接
recognizer2, err := asr.CreateRecognizer(config2)
if err != nil {
    log.Fatal(err)
}

// 实现连接切换逻辑
func switchToBackup() error {
    // 停止主连接，启动备用连接
}
```

## 🎵 音频处理问题

### 音频格式错误

#### 问题症状
- ❌ `invalid audio format: invalid sample rate`
- ❌ `audio buffer full`
- ❌ 音频数据长度不是偶数

#### 诊断步骤

1. **检查音频格式**
```go
// 验证配置
config := recognizer.GetConfig()
fmt.Printf("音频配置: 采样率=%d, 声道=%d\n", config.InputSampleRate, config.InputChannels)

// 检查音频数据
if len(audioData)%2 != 0 {
    log.Printf("⚠️ 警告: PCM数据长度必须是偶数")
}

// 检查音频数据范围
max := 32767
min := -32768
for _, sample := range pcmData {
    if int16(sample) > max || int16(sample) < min {
        log.Printf("⚠️ 警告: PCM采样值超出范围: %d", sample)
    }
}
```

2. **优化音频处理**
```go
// 使用合适的块大小
const optimalChunkSize = 1024 // 1KB

// 控制发送频率
const sendInterval = 20 * time.Millisecond

// 使用缓冲区管理
audioBuffer := make([]byte, 0, optimalChunkSize)
for {
    // 发送音频
    if err := recognizer.Write(audioData); err != nil {
        log.Printf("发送音频失败: %v", err)
        break
    }

    time.Sleep(sendInterval)
}
```

3. **监控音频缓冲区使用**
```go
stats := recognizer.GetStats()
if bufferUsage, ok := stats["audio_buffer_size"].(int); ok {
    usagePercent := (bufferUsage * 100) / (1024*100) // 假设1MB缓冲区
    if usagePercent > 80 {
        log.Printf("⚠️ 音频缓冲区使用率过高: %.1f%%", usagePercent)
    }
}
```

#### 解决方案

1. **调整音频配置**
```go
config := asr.DefaultConfig()
config.InputSampleRate = 16000  // 除非需要48kHz
config.InputChannels = 1       // 除非需要立体声
```

2. **实现流量控制**
```go
const maxChunksPerSecond = 50

type RateController struct {
    ticker       *time.Ticker
    chunkCount   int
    lastReset    time.Time
}

func (rc *RateController) Allow() bool {
    now := time.Now()
    if now.Sub(rc.lastReset).Seconds() >= 1 {
        rc.chunkCount = 0
        rc.lastReset = now
        return true
    }

    if rc.chunkCount >= maxChunksPerSecond {
        return false
    }

    rc.chunkCount++
    return true
}
```

## 📝 事件处理问题

### 事件未触发

#### 问题症状
- ✅ 连接建立但未收到`session.created`事件
- ✅ 发送音频但未收到`conversation.item.created`事件
- ❌ 转录结果为空或不完整

#### 诊断步骤

1. **检查会话状态**
```go
session := sessionManager.GetSession()
if session == nil {
    log.Printf("❌ 会话未创建")
}

fmt.Printf("会话信息: %+v\n", session)
```

2. **验证事件处理**
```go
// 启用事件统计
stats := eventDispatcher.GetStats()
log.Printf("事件统计: %+v\n", stats)

// 检查会话是否已初始化
if !session.IsInitialized {
    log.Printf("⚠️ 会话未初始化，等待session.updated事件")
}
```

3. **检查事件响应**
```go
// 手动发送事件验证
event := &asr.InputAudioBufferAppendEvent{
    BaseEvent: asr.BaseEvent{
        Type:    asr.EventTypeInputAudioBufferAppend,
        EventID: asr.GenerateEventID(),
    },
    Audio: "dGVzdHVhAQEAAAAEl...", // 示例Base64数据
}

if err := eventDispatcher.Dispatch(eventJSON); err != nil {
    log.Printf("事件发送失败: %v", err)
}
```

#### 解决方案

1. **确保事件处理器完整实现**
```go
type CompleteEventHandler struct {}

func (h *CompleteEventHandler) OnSessionCreated(event *asr.SessionCreatedEvent) {
    log.Printf("✅ 会话创建: %s", event.Session.ID)
    // 配置会话参数
}

func (h *CompleteEventHandler) OnSessionUpdated(event *asr.SessionUpdatedEvent) {
    log.Printf("✅ 会话更新: %s", event.Session.ID)
    // 开始处理音频
}

// 实现所有必需的回调方法...
```

2. **添加事件处理延迟**
```go
// 在事件处理器中添加延迟
func (h *CompleteEventHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
    // 模拟处理延迟
    time.Sleep(100 * time.Millisecond)

    if len(event.Item.Content) > 0 {
        for _, content := range event.Item.Content {
            if content.Type == "transcript" {
                fmt.Printf("✅ 转录结果: %s", content.Transcript)
            }
        }
    }
}
```

## 💥 错误处理问题

### 错误分类和处理

#### 识别错误类型

```go
func classifyError(err error) string {
    switch {
    case asr.ErrConnectionFailed:
        return "CONNECTION_ERROR"
    case asr.ErrSessionNotFound:
        return "SESSION_ERROR"
    case asr.ErrAudioBufferFull:
        return "BUFFER_ERROR"
    case asr.ErrInvalidAudioFormat:
        return "FORMAT_ERROR"
    default:
        if asr.IsConnectionError(err) {
            return "CONNECTION_ERROR"
        } else if asr.IsAudioError(err) {
            return "AUDIO_ERROR"
        } else if asr.IsEventError(err) {
            return "EVENT_ERROR"
        }
        return "UNKNOWN_ERROR"
    }
}
```

#### 实现重试策略

```go
func retryWithBackoff(operation func() error) error {
    var lastAttempt time.Time
    const maxAttempts = 5
    const baseDelay = 100 * time.Millisecond

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        errorType := classifyError(err)
        switch errorType {
        case "CONNECTION_ERROR":
            lastAttempt = time.Now()
            delay := baseDelay * time.Duration(attempt)
            log.Printf("连接错误，%ds后重试: %v", delay, err)
        case "BUFFER_ERROR":
            delay := baseDelay * time.Duration(attempt)
            log.Printf("缓冲区满，%ds后重试: %v", delay, err)
        default:
            // 不重试其他错误
            return err
        }

        if attempt == maxAttempts {
            return fmt.Errorf("重试次数已达上限: %w", err)
        }

        time.Sleep(delay)
    }
}
```

## 📊 性能问题

### 内存使用过高

#### 诊断工具

```go
type MemoryProfiler struct {
    samples []int64
    mu     sync.Mutex
}

func (mp *MemoryProfiler) RecordSample(size int64) {
    mp.mu.Lock()
    defer mp.mu.Unlock()

    mp.samples = append(mp.samples, size)
    if len(mp.samples) > 1000 {
        mp.samples = mp.samples[len(mp.samples)-1000:] // 保留最近1000个样本
    }
}

func (mp *MemoryProfiler) GetStats() map[string]interface{} {
    mp.mu.Lock()
    defer mp.mu.Unlock()

    var total int64
    for _, sample := range mp.samples {
        total += int64(sample)
    }

    return map[string]interface{}{
        "total_samples": len(mp.samples),
        "total_memory_mb": total * 2 / 1024 / 1024, // int16 = 2 bytes
    }
}
```

### CPU使用优化

```go
type PerformanceMonitor struct {
    startTime time.Time
    operationCount int64
}

func (pm *PerformanceMonitor) StartOperation() {
    pm.startTime = time.Now()
    pm.operationCount = 0
}

func (pm *PerformanceMonitor) EndOperation() {
    duration := time.Since(pm.startTime).Milliseconds()
    pm.operationCount++

    if pm.operationCount%100 == 0 {
        log.Printf("操作完成 #%d: 耗时 %d ms\n", pm.operationCount, duration)
    }
}
```

## 🔧 部署环境问题

### 容器环境调试

#### 1. 检查网络连接

```dockerfile
# 使用相同的网络配置
version: '3.8'
services:
  asr-client:
    image: your-asr-client:latest
    network_mode: host
    depends_on:
      - asr-server
    ports:
      - "8088:8088"
    environment:
      - DEBUG_LEVEL: "info"
      - ASR_SERVER_URL: "ws://asr-server:8088"
```

#### 2. Kubernetes调试

```yaml
# debug-deployment.yaml
apiVersion: v1
kind: Pod
metadata:
  name: asr-client-debug
spec:
  containers:
  - name: asr-client
    image: your-asr-client:latest
    env:
      - DEBUG: "true"
      - ASR_SERVER_URL: "ws://asr-server:8088"
      - LOG_LEVEL: "debug"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
```

#### 3. 日志收集配置

```go
// 使用结构化日志，包含容器信息
func setupStructuredLogger() *logrus.Logger {
    logger := logrus.New()

    // 从环境变量获取日志级别
    logLevel := os.Getenv("LOG_LEVEL")
    switch logLevel {
    case "debug":
        logger.SetLevel(logrus.DebugLevel)
    case "info":
        logger.SetLevel(logrus.InfoLevel)
    case "warn":
        logger.SetLevel(logrus.WarnLevel)
    case "error":
        logger.SetLevel(logrus.ErrorLevel)
    }

    // 添加容器信息
    logger.WithFields(logrus.Fields{
        "pod_name": os.Getenv("POD_NAME"),
        "container_id": os.Getenv("CONTAINER_ID"),
        "namespace": os.Getenv("NAMESPACE"),
    }).SetFormatter(&logrus.JSONFormatter{})

    return logger
}
```

## 🚑 安全问题

### 1. 认证和授权

```go
// 实现JWT令牌验证
func validateJWTToken(token string) error {
    // 解析JWT
    parts := strings.Split(token, ".")
    if len(parts) != 3 {
        return fmt.Errorf("无效的JWT格式")
    }

    // 这里应该解析payload、验证签名和过期时间
    // 实际实现需要使用jwt-go库
    return nil
}

// 实现API密钥管理
type APIKeyManager struct {
    keyID string
    key    string
}

func (akm *APIKeyManager) GetKey() (string, error) {
    // 从安全存储获取API密钥
    // 这里应该实现密钥轮换和管理逻辑
    return akm.key, nil
}
```

### 2. 网络安全配置

```go
// 禁用不安全的TLS版本
import (
    "crypto/tls"
    "log"
)

config := &tls.Config{
    MinVersion: tls.VersionTLS12,
    // 强制推荐的安全套件
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
    },
    PreferServerCipherSuites: true,
    InsecureSkipVerify: false,
}

// 在WebSocket连接中使用TLS配置
dialer := websocket.Dialer{
    HandshakeTimeout: 10 * time.Second,
    TLSClientConfig: config,
}
```

## 📋 协议兼容性问题

### WebSocket版本兼容

#### 问题症状
- 🚫 `websocket: bad handshake status: 426 Upgrade Required`
- 🚫 `websocket: missing or invalid upgrade header`

#### 解决方案

1. **检查并添加升级头**
```go
headers := http.Header{
    "Upgrade":    []string{"websocket"},
    "Connection": []string{"Upgrade"},
    "Sec-WebSocket-Key": "your-websocket-key",
    "Sec-WebSocket-Version": "13",
    "Origin": "https://your-domain.com",
}
```

2. **使用支持的子协议**
```go
// 在连接字符串中指定支持的子协议
// ws://localhost:8088/v2/ws
```

3. **处理服务器响应**
```go
// 检查升级响应
resp, err := http.Post(url, "application/json", bytes.NewReader(data))
if err != nil {
    return err
}

defer resp.Body.Close()

if resp.StatusCode != http.StatusSwitchingProtocols {
    return fmt.Errorf("服务器不支持WebSocket升级")
}

// 使用升级后的连接
conn, _, err := websocket.NewClient(resp, nil)
if err != nil {
    return err
}
```

## 🔄 常见问题总结

### 问题分类和快速解决方案

| 问题类型 | 快速检查 | 常见原因 | 解决方案 |
|---------|----------|----------|----------|
| 连接失败 | `ping`服务器 | 检查网络、服务器状态、防火墙 |
| 音频错误 | `invalid sample rate` | 验证音频配置、检查采样率 |
| 会话问题 | 无`session.created` | 检查会话配置、等待初始化 |
| 性能问题 | 高延迟/高内存 | 优化音频块大小、实现流控制 |
| 网络问题 | 间歇性断开 | 检查网络稳定性、实现重连机制 |

### 使用诊断工具

```bash
# 运行完整的诊断脚本
go run ./cmd/diagnose --server ws://localhost:8088 --timeout 30s

# 检查网络连通性
./scripts/network-check.sh localhost 8088

# 监控实时状态
./scripts/health-monitor.sh ws://localhost:8088
```

这个故障排除指南提供了系统性的问题诊断和解决方案，帮助快速定位和解决ASR SDK使用中的问题。