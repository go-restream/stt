# ASR SDK 最佳实践指南

本文档提供了使用ASR SDK (OpenAI Realtime API)的最佳实践、性能优化建议和生产环境部署指南。

## 🚀 快速开始

### 基础使用模式

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "time"

    asr "streamASR/sdk/golang/client"
)

func main() {
    // 1. 创建带超时的上下文
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 2. 配置超时和信号处理
    config := asr.DefaultConfig()
    config.Timeout = 30 * time.Second

    // 3. 创建识别器
    recognizer, err := asr.NewRecognizer(config)
    if err != nil {
        log.Fatalf("创建识别器失败: %v", err)
    }

    // 4. 启动识别
    if err := recognizer.Start(); err != nil {
        log.Fatalf("启动识别失败: %v", err)
    }
    defer recognizer.Stop()

    fmt.Println("🎤 ASR SDK已启动，按Ctrl+C退出")

    // 5. 信号处理
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // 6. 业务逻辑
    go func() {
        // 你的业务逻辑
        processAudioWithRecognizer(ctx, recognizer)
    }()

    <-sigChan
    fmt.Println("\n👋 正在退出...")
}

func processAudioWithRecognizer(ctx context.Context, recognizer *asr.Recognizer) {
    // 业务逻辑实现
}
```

## 🎯 配置最佳实践

### 1. 连接配置

```go
config := asr.DefaultConfig()

// 基础配置
config.URL = "wss://your-server.com/ws"
config.TranscriptionLanguage = "zh-CN"
config.Timeout = 30 * time.Second

// 高级配置
config.EnableReconnect = true
config.MaxReconnectAttempts = 5
config.ReconnectDelay = 2 * time.Second
config.HeartbeatInterval = 20 * time.Second

// 语音检测配置
config.TurnDetectionThreshold = 0.3
config.TurnDetectionPrefixPaddingMs = 300
config.TurnDetectionSilenceDurationMs = 1000
```

### 2. 音频格式选择

```go
// 根据使用场景选择最佳配置
func getConfigForUseCase(useCase string) *asr.Config {
    config := asr.DefaultConfig()

    switch useCase {
    case "high_quality":
        config.InputSampleRate = 48000  // 高质量
        config.InputChannels = 1
        config.TurnDetectionThreshold = 0.1

    case "low_latency":
        config.InputSampleRate = 16000  // 低延迟
        config.Timeout = 5 * time.Second

    case "voice_command":
        config.TurnDetectionThreshold = 0.5  // 语音命令
        config.TurnDetectionSilenceDurationMs = 500

    case "background_noise":
        config.TurnDetectionThreshold = 0.7  // 背景噪音
        config.TurnDetectionPrefixPaddingMs = 500

    default:
        // 使用默认配置
    }

    return config
}
```

## 📊 性能优化

### 1. 音频数据处理

#### 批量处理

```go
const (
    optimalChunkSize = 1024    // 1KB
    maxBufferDuration = 2 * time.Second
)

type AudioProcessor struct {
    recognizer *asr.Recognizer
    audioBuffer []byte
    lastFlush time.Time
}

func (p *AudioProcessor) ProcessAudio(audioData []byte) error {
    p.audioBuffer = append(p.audioBuffer, audioData...)
    duration := time.Since(p.lastFlush)

    if duration >= maxBufferDuration {
        if err := p.recognizer.Write(p.audioBuffer); err != nil {
            return err
        }

        // 提交音频进行识别
        if err := p.recognizer.CommitAudio(); err != nil {
            return err
        }

        // 清空缓冲区
        p.audioBuffer = p.audioBuffer[:0]
        p.lastFlush = time.Now()
    }

    return nil
}
```

#### 音频重采样策略

```go
// 使用SDK内置的重采样功能
func setupAudioProcessor(recognizer *asr.Recognizer, inputRate int) *AudioProcessor {
    config := recognizer.GetConfig()

    if inputRate == 48000 {
        // 48kHz音频，使用高质量重采样
        fmt.Println("🎵 启用48kHz高质量重采样")
    } else {
        // 16kHz音频，直接处理
        fmt.Println("🎵 使用16kHz直接处理")
    }

    return &AudioProcessor{
        recognizer: recognizer,
    }
}
```

### 2. 内存管理

#### 对象池化

```go
var audioBufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024*10) // 10KB缓冲池
    },
}

func getAudioBuffer() []byte {
    return audioBufferPool.Get().([]byte)
}

func putAudioBuffer(buffer []byte) {
    if cap(buffer) == 1024*10 { // 只有标准大小的缓冲区才回收到池
        audioBufferPool.Put(buffer)
    }
}
```

#### 内存监控

```go
type MemoryMonitor struct {
    maxMemoryUsage int64
    alertThreshold  int64
}

func (m *MemoryMonitor) Start() {
    m.alertThreshold = 100 * 1024 * 1024 // 100MB

    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            var ms runtime.MemStats
            runtime.ReadMemStats(&ms)

            if int64(ms.Alloc) > m.maxMemoryUsage {
                m.maxMemoryUsage = int64(ms.Alloc)
            }

            if int64(ms.Alloc) > m.alertThreshold {
                log.Printf("⚠️ 内存使用过高: %d MB", ms.Alloc/1024/1024)
            }
        }
    }()
}

func (m *MemoryMonitor) GetStats() map[string]interface{} {
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    return map[string]interface{}{
        "current_memory_mb":     ms.Alloc / 1024 / 1024,
        "max_memory_mb":        m.maxMemoryUsage / 1024 / 1024,
        "gc_pause_count":       ms.NumGC,
    }
}
```

### 3. 网络优化

#### 连接复用

```go
type ConnectionPool struct {
    connections chan *asr.Recognizer
    maxSize       int
    mu           sync.Mutex
    size          int
}

func NewConnectionPool(maxSize int) *ConnectionPool {
    return &ConnectionPool{
        connections: make(chan *asr.Recognizer, maxSize),
        maxSize:     maxSize,
    }
}

func (p *ConnectionPool) Get() *asr.Recognizer {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.size > 0 {
        p.size--
        return <-p.connections
    }

    return nil
}

func (p *ConnectionPool) Put(conn *asr.Recognizer) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.size < p.maxSize {
        p.connections <- conn
        p.size++
    }
}
```

#### 请求优化

```go
// 使用智能音频分段
func optimizeAudioSending(recognizer *asr.Recognizer, audioData []byte) error {
    // VAD检测（如果可用）
    // 分段发送，减少网络开销
    chunkSize := 512 // 较小的块大小

    for i := 0; i < len(audioData); i += chunkSize {
        end := i + chunkSize
        if end > len(audioData) {
            end = len(audioData)
        }

        chunk := audioData[i:end]
        if err := recognizer.Write(chunk); err != nil {
            return err
        }

        // 智能延迟
        time.Sleep(20 * time.Millisecond)
    }

    // 最后提交
    return recognizer.CommitAudio()
}
```

## 🛡️ 错误处理与恢复

### 1. 分层错误处理

```go
type ErrorHandler struct {
    recognizer *asr.Recognizer
    retryCount  int
    maxRetries  int
    retryDelay  time.Duration
}

func NewErrorHandler(recognizer *asr.Recognizer) *ErrorHandler {
    return &ErrorHandler{
        recognizer: recognizer,
        maxRetries: 3,
        retryDelay: 1 * time.Second,
    }
}

func (h *ErrorHandler) HandleWithRetry(fn func() error) error {
    for {
        err := fn()
        if err == nil {
            h.retryCount = 0
            return nil
        }

        h.retryCount++
        if h.retryCount > h.maxRetries {
            return fmt.Errorf("重试次数超限: %w", err)
        }

        // 记录错误
        log.Printf("🔄 重试 %d/%d: %v", h.retryCount, h.maxRetries, err)
        time.Sleep(h.retryDelay)
    }
}
```

### 2. 断路恢复

```go
type RecoveryManager struct {
    recognizer    *asr.Recognizer
    backupURL    string
    maxFailures  int
    failCount     int
    mu           sync.Mutex
}

func (rm *RecoveryManager) HandleConnectionFailure(err error) {
    rm.mu.Lock()
    defer rm.mu.Unlock()

    rm.failCount++
    log.Printf("❌ 连接失败 %d/%d: %v", rm.failCount, rm.maxFailures, err)

    if rm.failCount >= rm.maxFailures {
        log.Printf("🔄 达到最大失败次数，切换到备份服务器: %s", rm.backupURL)

        // 停止当前连接
        if stopErr := rm.recognizer.Stop(); stopErr != nil {
            log.Printf("停止当前连接失败: %v", stopErr)
        }

        // 等待一段时间后重连主服务器
        time.Sleep(30 * time.Second)

        // 重新配置并启动
        config := rm.recognizer.GetConfig()
        config.URL = rm.backupURL

        newRecognizer, err := asr.NewRecognizer(config)
        if err != nil {
            log.Printf("创建备用识别器失败: %v", err)
            return err
        }

        rm.recognizer = newRecognizer
        return rm.recognizer.Start()
    }

    // 简单延迟重试
    time.Sleep(2 * time.Second)
    return rm.recognizer.Start()
}
```

### 3. 优雅降级

```go
func gracefulShutdown(ctx context.Context, recognizer *asr.Recognizer) {
    // 1. 停止接受新音频
    // 2. 完成当前正在处理的音频
    // 3. 停止识别器

    // 使用超时确保清理完成
    shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    done := make(chan struct{})
    go func() {
        if err := recognizer.Stop(); err != nil {
            log.Printf("停止识别器失败: %v", err)
        }
        close(done)
    }()

    select {
    case <-done:
        log.Println("✅ 识别器已优雅停止")
    case <-shutdownCtx.Done():
        log.Println("⚠️ 停止超时，强制停止")
    }
}
```

## 🏭 生产环境部署

### 1. 配置管理

```go
// 从环境变量读取配置
func loadConfigFromEnv() *asr.Config {
    config := asr.DefaultConfig()

    if url := os.Getenv("ASR_SERVER_URL"); url != "" {
        config.URL = url
    }

    if lang := os.Getenv("ASR_LANGUAGE"); lang != "" {
        config.TranscriptionLanguage = lang
    }

    if timeout := os.Getenv("ASR_TIMEOUT"); timeout != "" {
        if duration, err := time.ParseDuration(timeout); err == nil {
            config.Timeout = duration
        }
    }

    if enableReconnect := os.Getenv("ASR_ENABLE_RECONNECT"); enableReconnect != "" {
        config.EnableReconnect = enableReconnect == "true"
    }

    return config
}

// 配置验证
func validateConfig(config *asr.Config) error {
    if config.URL == "" {
        return fmt.Errorf("ASR_SERVER_URL环境变量未设置")
    }

    if !strings.HasPrefix(config.URL, "ws://") && !strings.HasPrefix(config.URL, "wss://") {
        return fmt.Errorf("无效的WebSocket URL格式")
    }

    return nil
}
```

### 2. 健康检查

```go
type HealthChecker struct {
    recognizer   *asr.Recognizer
    interval     time.Duration
    lastCheck    time.Time
}

func NewHealthChecker(recognizer *asr.Recognizer) *HealthChecker {
    return &HealthChecker{
        recognizer: recognizer,
        interval:   30 * time.Second,
    }
}

func (h *HealthChecker) Start() {
    ticker := time.NewTicker(h.interval)
    defer ticker.Stop()

    for range ticker.C {
        status := h.recognizer.GetConnectionStatus()
        stats := h.recognizer.GetStats()

        // 健康指标
        health := map[string]interface{}{
            "status":           status,
            "is_running":       h.recognizer.IsRunning(),
            "session_id":       stats["session_id"],
            "buffer_usage":     stats["audio_buffer_size"],
            "last_event_time":  stats["event_stats"].(map[string]interface{})["last_event_time"],
        }

        // 检查问题
        var issues []string
        if status != asr.ConnectionStatusConnected {
            issues = append(issues, "connection_lost")
        }
        if bufferUsage, ok := stats["audio_buffer_size"].(int); ok && bufferUsage > 80*1024 { // 80KB
            issues = append(issues, "buffer_high_usage")
        }

        if len(issues) > 0 {
            log.Printf("⚠️ 健康检查失败: %v", issues)
        } else {
            h.lastCheck = time.Now()
            log.Printf("✅ 健康检查通过")
        }
    }
}
```

### 3. 监控集成

```go
// Prometheus监控指标
var (
    messagesReceived = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "asr_messages_received_total",
            Help: "Total number of messages received from ASR server",
        },
    )

    messagesProcessed = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "asr_messages_processed_total",
            Help: "Total number of messages processed by ASR client",
        },
    )

    transcriptionRequests = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "asr_transcription_requests_total",
            Help: "Total number of transcription requests",
        },
    )
)

type MetricsCollector struct {
    recognizer *asr.Recognizer
}

func (mc *MetricsCollector) Start() {
    // 监控指标收集中器
    metricsCollector := func() {
        messagesReceived.Inc()
        messagesProcessed.Inc()
        transcriptionRequests.Inc()
    }

    // 设置事件处理器来收集指标
    handler := &MetricHandler{
        collector: metricsCollector,
    }

    mc.recognizer = asr.CreateRecognizerWithEventHandler(
        mc.recognizer.GetConfig(),
        handler,
    )
}

type MetricHandler struct {
    collector func()
}

func (h *MetricHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
    h.collector()
}

// 其他方法实现...
```

## 📈 监控和日志

### 1. 结构化日志

```go
// 使用logrus进行结构化日志
import (
    "github.com/sirupsen/logrus"
    "github.com/google/uuid"
)

func setupLogger() *logrus.Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{
        TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
    FieldMap: logrus.FieldMap{
            logrus.FieldKeyTime:  logrus.FieldKey{
                Key:   "timestamp",
                Type:  logrus.FormattingTimeLayoutType,
            },
            logrus.FieldKeyMsgID: logrus.FieldKey{
                Key:   "msg_id",
                Type:  logrus.FormattingTimeLayoutType,
            },
            logrus.FieldKeySessionID: logrus.FieldKey{
                Key:   "session_id",
                Type:  logrus.FormattingTimeLayoutType,
            },
        },
    })

    return logger
}

func logEvent(eventType, sessionID string, details ...interface{}) {
    logger.WithFields(logrus.Fields{
        "event_type": eventType,
        "session_id": sessionID,
        "msg_id":     uuid.New().String(),
    }).Info("ASR事件", details...)
}
```

### 2. 分布式追踪

```go
// OpenTelemetry追踪
import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout"
    oteltrace "go.opentelemetry.io/otel/trace"
)

func initTracer() oteltrace.Tracer {
    oteltrace.InitTracerProvider(
        "asr-sdk-tracer",
        oteltrace.WithBatcher(oteltrace.NewBatchSpanProcessor(oteltrace.AlwaysSample)),
    )(context.Background(), "asr-sdk-tracer")
}

func traceOperation(ctx context.Context, name string, fn func() error) error {
    ctx, span := oteltrace.Start(ctx, name, oteltrace.WithAttributes(
        oteltrace.StringAttribute("operation.name", name),
    ))
    defer span.End()

    err := fn()
    if err != nil {
        span.SetStatus(oteltrace.StatusCodeError, err.Error())
        span.RecordError(err)
    } else {
        span.SetStatus(oteltrace.StatusCodeOk)
    }

    return err
}
```

### 3. 性能指标

```go
// 自定义性能指标
var (
    audioLatency = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "asr_audio_latency_seconds",
            Help:    "Audio processing latency in seconds",
            Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
        },
    )

    audioThroughput = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "asr_audio_throughput_bytes_per_second",
            Help:    "Audio processing throughput in bytes per second",
            Buckets: []float64{1024, 4096, 16384, 65536, 262144},
        },
    )
)

func recordAudioMetrics(startTime time.Time, byteCount int) {
    latency := time.Since(startTime).Seconds()
    throughput := float64(byteCount) / latency.Seconds()

    audioLatency.Observe(latency)
    audioThroughput.Observe(throughput)
}
```

## 🎛️ 安全最佳实践

### 1. 输入验证

```go
func validateAudioData(audioData []byte) error {
    if len(audioData) == 0 {
        return fmt.Errorf("空的音频数据")
    }

    if len(audioData) > 10*1024*1024 { // 10MB限制
        return fmt.Errorf("音频数据过大，超过10MB")
    }

    // 检查PCM格式
    if len(audioData)%2 != 0 {
        return fmt.Errorf("音频数据长度必须是偶数")
    }

    return nil
}
```

### 2. 速率限制

```go
type RateLimiter struct {
    ticker   *time.Ticker
    requests chan struct{}
    limit    int
    count    int
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    return &RateLimiter{
        ticker:   time.NewTicker(time.Second / time.Duration(requestsPerSecond)),
        requests: make(chan struct{}, requestsPerSecond),
        limit:    requestsPerSecond,
    }
}

func (rl *RateLimiter) Allow() bool {
    select {
    case <-rl.requests:
        rl.count++
        if rl.count < rl.limit {
            return true
        }
        return false
    case <-rl.ticker.C:
        rl.count = 0
    }
}
```

### 3. 认证支持

```go
// 认证头设置
func createAuthenticatedConnection(url, token string) (*asr.Recognizer, error) {
    config := asr.DefaultConfig()
    config.URL = url
    config.Headers = map[string]string{
        "Authorization": "Bearer " + token,
        "User-Agent": "ASR-SDK/2.0.0",
    }

    return asr.NewRecognizer(config)
}

// JWT Token验证
func validateJWTToken(token string) error {
    // 实现JWT验证逻辑
    // 这里应该解析JWT、验证签名和过期时间
    if token == "" {
        return fmt.Errorf("空的认证令牌")
    }

    // 示例验证（实际实现需要JWT库）
    return nil
}
```

## 🔧 测试策略

### 1. 单元测试

```go
func TestAudioProcessing(t *testing.T) {
    recognizer := asr.NewRecognizer(asr.DefaultConfig())

    // 模拟WebSocket连接
    // 在单元测试中需要模拟网络层

    // 测试音频处理
    testData := []byte{0x01, 0x02} // 简单的测试数据
    err := recognizer.Write(testData)
    assert.NoError(t, err)
}

func TestEventHandling(t *testing.T) {
    handler := &TestHandler{}

    // 模拟事件
    event := &asr.SessionCreatedEvent{
        BaseEvent: asr.BaseEvent{
            Type:    asr.EventTypeSessionCreated,
            EventID: "test-event-id",
        },
        Session: struct{
            ID:     "test-session",
            Model:  "test-model",
        },
    }

    handler.OnSessionCreated(event)

    // 验证回调被调用
    // 需要使用通道或其他机制来验证
}
```

### 2. 集成测试

```go
func TestWebSocketConnection(t *testing.T) {
    // 使用httptest模拟WebSocket服务器
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 模拟OpenAI Realtime API响应
        if r.Header.Get("Upgrade") != "websocket" {
            http.Error(w, "需要WebSocket升级", http.StatusBadRequest)
            return
        }

        // WebSocket升级逻辑
        // 这里应该实现完整的WebSocket协议
    }))

    defer server.Close()

    // 测试连接和认证
    config := asr.DefaultConfig()
    config.URL = "ws" + server.Listener.Addr().String()

    recognizer := asr.NewRecognizer(config)
    err := recognizer.Start()
    assert.NoError(t, err)

    // 测试音频发送和事件处理
    // ...
}
```

这个最佳实践指南涵盖了生产环境中使用ASR SDK的关键方面，确保高性能、可靠性和安全性。