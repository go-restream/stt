# ASR SDK (OpenAI Realtime API) API 文档

## 概述

本文档详细描述了基于OpenAI Realtime API标准的ASR SDK的所有公共接口、类型和常量。

## 核心概念

### 事件驱动架构

SDK采用事件驱动架构，通过WebSocket连接与后端进行双向通信。所有操作都通过事件进行，包括：

- **会话管理**：创建、更新、销毁会话
- **音频处理**：发送、提交、清空音频缓冲区
- **转录处理**：接收实时和最终转录结果
- **连接管理**：监控连接状态、处理重连

### 异步处理

SDK使用goroutines进行异步处理，确保高性能和实时响应：

- **消息接收**：独立goroutine处理WebSocket消息
- **事件分发**：独立goroutine分发事件到回调
- **连接监控**：独立goroutine监控连接健康
- **心跳管理**：独立goroutine处理心跳信号

## 主要类型

### Config 配置类型

```go
type Config struct {
    // 连接配置
    URL                   string        `json:"url"`
    Timeout               time.Duration `json:"timeout,omitempty"`
    Headers               map[string]string `json:"headers,omitempty"`

    // 音频配置
    InputSampleRate        int           `json:"input_sample_rate,omitempty"`
    OutputSampleRate       int           `json:"output_sample_rate,omitempty"`
    InputChannels          int           `json:"input_channels,omitempty"`
    OutputChannels         int           `json:"output_channels,omitempty"`

    // 会话配置
    Modality              string        `json:"modality,omitempty"`
    Instructions          string        `json:"instructions,omitempty"`
    Voice                 string        `json:"voice,omitempty"`

    // 转录配置
    TranscriptionModel     string        `json:"transcription_model,omitempty"`
    TranscriptionLanguage  string        `json:"transcription_language,omitempty"`

    // 语音检测配置
    TurnDetectionType               string  `json:"turn_detection_type,omitempty"`
    TurnDetectionThreshold          float32 `json:"turn_detection_threshold,omitempty"`
    TurnDetectionPrefixPaddingMs     int     `json:"turn_detection_prefix_padding_ms,omitempty"`
    TurnDetectionSilenceDurationMs   int     `json:"turn_detection_silence_duration_ms,omitempty"`

    // 工具配置
    Tools                 []interface{} `json:"tools,omitempty"`
    ToolChoice             string        `json:"tool_choice,omitempty"`

    // 重连配置
    EnableReconnect       bool          `json:"enable_reconnect,omitempty"`
    MaxReconnectAttempts  int           `json:"max_reconnect_attempts,omitempty"`
    ReconnectDelay       time.Duration `json:"reconnect_delay,omitempty"`

    // 心跳配置
    HeartbeatInterval     time.Duration `json:"heartbeat_interval,omitempty"`
}
```

### EventHandler 事件处理器接口

```go
type EventHandler interface {
    // 会话生命周期事件
    OnSessionCreated(*SessionCreatedEvent)
    OnSessionUpdated(*SessionUpdatedEvent)

    // 对话管理事件
    OnConversationCreated(*ConversationCreatedEvent)
    OnConversationItemCreated(*ConversationItemCreatedEvent)
    OnConversationItemDeleted(*ConversationItemDeletedEvent)

    // 音频缓冲区事件
    OnAudioBufferAppended(*InputAudioBufferAppendEvent)
    OnAudioBufferCommitted(*InputAudioBufferCommittedEvent)
    OnAudioBufferCleared(*InputAudioBufferClearedEvent)
    OnSpeechStarted(*InputAudioBufferSpeechStartedEvent)
    OnSpeechStopped(*InputAudioBufferSpeechStoppedEvent)

    // 转录结果事件
    OnTranscriptionCompleted(*ConversationItemInputAudioTranscriptionCompletedEvent)
    OnTranscriptionFailed(*ConversationItemInputAudioTranscriptionFailedEvent)

    // 连接状态事件
    OnConnected()
    OnDisconnected()
    OnError(*ErrorEvent)

    // 心跳事件
    OnPing(*HeartbeatPingEvent)
    OnPong(*HeartbeatPongEvent)
}
```

### Recognizer 识别器类型

```go
type Recognizer struct {
    // 私有字段，提供配置
    config *Config

    // 公共方法
    Start() error
    Stop() error
    IsRunning() bool
    Write([]byte) error
    CommitAudio() error
    ClearAudioBuffer() error

    // 状态查询方法
    GetSessionID() string
    GetConnectionStatus() ConnectionStatus
    GetStats() map[string]interface{}
}
```

## 事件类型

### 会话事件

#### SessionCreatedEvent

```go
type SessionCreatedEvent struct {
    BaseEvent
    Session struct {
        ID         string   `json:"id"`
        Object     string   `json:"object"`
        Model      string   `json:"model"`
        Modalities []string `json:"modalities"`
    } `json:"session"`
}
```

**触发时机**：当服务器成功创建会话时发送

#### SessionUpdateEvent

```go
type SessionUpdateEvent struct {
    BaseEvent
    Session struct {
        ID        string `json:"id"`
        Modality  string `json:"modality"`
        Instructions string `json:"instructions,omitempty"`
        Voice     string `json:"voice,omitempty"`
        InputAudioFormat struct {
            Type           string `json:"type"`
            SampleRate     int    `json:"sample_rate"`
            Channels       int    `json:"channels"`
        } `json:"input_audio_format,omitempty"`
        OutputAudioFormat struct {
            Type       string `json:"type"`
            SampleRate int    `json:"sample_rate"`
            Voice      string `json:"voice,omitempty"`
        } `json:"output_audio_format,omitempty"`
        InputAudioTranscription *struct {
            Model    string `json:"model"`
            Language string `json:"language"`
        } `json:"input_audio_transcription,omitempty"`
        TurnDetection *struct {
            Type              string  `json:"type"`
            Threshold         float32 `json:"threshold"`
            PrefixPaddingMs   int     `json:"prefix_padding_ms"`
            SilenceDurationMs int     `json:"silence_duration_ms"`
        } `json:"turn_detection,omitempty"`
        Tools []interface{} `json:"tools,omitempty"`
        ToolChoice string `json:"tool_choice,omitempty"`
    } `json:"session"`
}
```

### 转录事件

#### ConversationItemInputAudioTranscriptionCompletedEvent

```go
type ConversationItemInputAudioTranscriptionCompletedEvent struct {
    BaseEvent
    Item struct {
        ID        string `json:"id"`
        Type      string `json:"type"`
        Status    string `json:"status"`
        Content   []struct {
            Type      string `json:"type"`
            Transcript string `json:"transcript"`
        } `json:"content"`
    } `json:"item"`
}
```

#### ConversationItemInputAudioTranscriptionFailedEvent

```go
type ConversationItemInputAudioTranscriptionFailedEvent struct {
    BaseEvent
    ItemID string `json:"item_id"`
    Error struct {
        Type    string `json:"type"`
        Code    string `json:"code"`
        Message string `json:"message"`
        Param   string `json:"param,omitempty"`
    } `json:"error"`
}
```

### 错误事件

#### ErrorEvent

```go
type ErrorEvent struct {
    BaseEvent
    Error struct {
        Type    string `json:"type"`
        Code    string `json:"code"`
        Message string `json:"message"`
        Param   string `json:"param,omitempty"`
    } `json:"error"`
}
```

### 连接状态

```go
type ConnectionStatus int

const (
    ConnectionStatusDisconnected ConnectionStatus = iota
    ConnectionStatusConnecting
    ConnectionStatusConnected
    ConnectionStatusReconnecting
    ConnectionStatusFailed
)
```

## 使用模式

### 基本使用

```go
// 1. 创建配置
config := asr.DefaultConfig()
config.URL = "ws://your-server.com/ws"
config.TranscriptionLanguage = "en-US"

// 2. 创建识别器
recognizer, err := asr.NewRecognizer(config)
if err != nil {
    log.Fatal(err)
}

// 3. 启动识别
if err := recognizer.Start(); err != nil {
    log.Fatal(err)
}
defer recognizer.Stop()

// 4. 使用识别器
audioData := []byte{/* 你的音频数据 */}
if err := recognizer.Write(audioData); err != nil {
    log.Printf("发送音频失败: %v", err)
}
```

### 高级事件处理

```go
// 1. 实现EventHandler接口
type MyEventHandler struct{}

func (h *MyEventHandler) OnTranscriptionCompleted(event *asr.ConversationItemInputAudioTranscriptionCompletedEvent) {
    if len(event.Item.Content) > 0 {
        for _, content := range event.Item.Content {
            if content.Type == "transcript" {
                fmt.Printf("转录结果: %s\n", content.Transcript)
            }
        }
    }
}

// 其他方法实现...

// 2. 创建带事件处理器的识别器
recognizer, err := asr.CreateRecognizerWithEventHandler(config, &MyEventHandler{})
```

### 传统回调兼容

```go
// 1. 实现RecognitionCallback接口
type MyCallback struct{}

func (c *MyCallback) OnRecognitionResult(sessionID, text string) {
    fmt.Printf("识别结果: %s\n", text)
}

// 2. 创建带传统回调的识别器
recognizer, err := asr.CreateRecognizerWithCallbacks(config, &MyCallback{})
```

### 错误处理

```go
// 使用预定义错误类型
if err := recognizer.Write(audioData); err != nil {
    switch {
    case asr.ErrConnectionFailed:
        log.Println("连接失败")
    case asr.ErrAudioBufferFull:
        log.Println("音频缓冲区满")
    default:
        if asr.IsConnectionError(err) {
            log.Printf("连接错误: %v", err)
        } else if asr.IsAudioError(err) {
            log.Printf("音频错误: %v", err)
        }
    }
}
```

## 常量量

### 事件类型常量

```go
const (
    EventTypeSessionCreated                                     = "session.created"
    EventTypeSessionUpdate                                      = "session.update"
    EventTypeSessionUpdated                                     = "session.updated"
    EventTypeConversationCreated                                = "conversation.created"
    EventTypeInputAudioBufferAppend                          = "input_audio_buffer.append"
    EventTypeInputAudioBufferCommit                          = "input_audio_buffer.commit"
    EventTypeInputAudioBufferCommitted                       = "input_audio_buffer.committed"
    EventTypeInputAudioBufferClear                            = "input_audio_buffer.clear"
    EventTypeInputAudioBufferSpeechStarted                    = "input_audio_buffer.speech_started"
    EventTypeInputAudioBufferSpeechStopped                    = "input_audio_buffer.speech_stopped"
    EventTypeConversationItemCreated                         = "conversation.item.created"
    EventTypeConversationItemInputAudioTranscriptionCompleted = "conversation.item.input_audio_transcription.completed"
    EventTypeConversationItemInputAudioTranscriptionFailed = "conversation.item.input_audio_transcription.failed"
    EventTypeConversationItemDeleted                         = "conversation.item.deleted"
    EventTypeInputAudioBufferCleared                         = "input_audio_buffer.cleared"
    EventTypeError                                           = "error"
    EventTypeHeartbeatPing                                   = "heartbeat.ping"
    EventTypeHeartbeatPong                                   = "heartbeat.pong"
)
```

### 错误类型常量

```go
var (
    // 连接错误
    ErrConnectionFailed     = errors.New("connection failed")
    ErrConnectionTimeout   = errors.New("connection timeout")
    ErrNotConnected        = errors.New("not connected")
    ErrAlreadyConnected     = errors.New("already connected")

    // 会话错误
    ErrSessionNotFound      = errors.New("session not found")
    ErrSessionNotReady     = errors.New("session not ready")
    ErrInvalidSessionState = errors.New("invalid session state")

    // 音频错误
    ErrInvalidAudioFormat   = errors.New("invalid audio format")
    ErrInvalidSampleRate    = errors.New("invalid sample rate")
    ErrInvalidChannels      = errors.New("invalid audio channels")
    ErrAudioBufferFull    = errors.New("audio buffer full")

    // 配置错误
    ErrInvalidURL          = errors.New("invalid URL")
    ErrInvalidConfig       = errors.New("invalid configuration")
)
)
```

## 最佳实践

### 1. 资源管理

```go
// 使用defer确保资源清理
recognizer, err := asr.NewRecognizer(config)
if err != nil {
    return err
}
defer recognizer.Stop()

// 错误处理
if err := recognizer.Write(audioData); err != nil {
    // 记录错误并决定是否重试
    if shouldRetry(err) {
        time.Sleep(retryDelay)
        // 重试逻辑
    } else {
        return err
    }
}
```

### 2. 音频处理优化

```go
// 使用合适的块大小
const optimalChunkSize = 1024 // 1KB

// 控制发送频率
const sendInterval = 20 * time.Millisecond

func sendAudioChunked(recognizer *asr.Recognizer, audioData []byte) {
    for i := 0; i < len(audioData); i += optimalChunkSize {
        end := i + optimalChunkSize
        if end > len(audioData) {
            end = len(audioData)
        }

        chunk := audioData[i:end]
        if err := recognizer.Write(chunk); err != nil {
            return err
        }

        time.Sleep(sendInterval)
    }
    return nil
}
```

### 3. 并发安全

```go
// SDK内部已经是线程安全的，但用户代码需要注意
// 不要同时调用Write方法
// 使用通道进行音频数据传输

// 推荐模式
audioChan := make(chan []byte, 100)
go func() {
    recognizer.Start()
    for audioData := range audioChan {
        recognizer.Write(audioData)
    }
    recognizer.Stop()
}()

// 发送音频
audioChan <- audioData
```

## 故障排除

### 常见问题解决

#### 连接问题

**问题**: `connection failed: dial tcp: lookup host: no such host`

**解决方案**:
1. 检查URL格式和服务器地址
2. 确认服务器正在运行
3. 检查网络连接

**问题**: `connection timeout`

**解决方案**:
1. 增加超时时间: `config.Timeout = 30 * time.Second`
2. 检查网络延迟
3. 启用重连: `config.EnableReconnect = true`

#### 音频问题

**问题**: `invalid audio format: invalid sample rate`

**解决方案**:
1. 使用支持的采样率: 16000 或 48000
2. 检查音频数据格式: 必须是16位PCM
3. 确保数据长度是偶数

**问题**: `audio buffer full`

**解决方案**:
1. 降低发送频率
2. 增加缓冲区大小
3. 使用CommitAudio()控制处理时机

#### 事件处理问题

**问题**: 没有收到转录结果

**解决方案**:
1. 检查事件处理器是否正确实现
2. 检查会话是否正确配置
3. 使用GetStats()调试事件统计

### 调试技巧

```go
// 启用详细统计
stats := recognizer.GetStats()
log.Printf("识别器状态: %+v", stats)

// 检查连接状态
status := recognizer.GetConnectionStatus()
switch status {
case asr.ConnectionStatusConnected:
    log.Println("连接正常")
case asr.ConnectionStatusDisconnected:
    log.Println("连接断开")
}

// 监控音频缓冲区
bufferInfo := recognizer.GetStats()["audio_buffer_info"].(map[string]interface{})
log.Printf("音频缓冲区: 大小=%v, 使用率=%.1f%%",
    bufferInfo["size"],
    bufferInfo["usage"])
```

## 版本兼容性

### v2.0.0 变更

- ✅ **完全重写**: 基于OpenAI Realtime API
- ✨ **新事件系统**: 支持所有OpenAI事件类型
- ✨ **向后兼容**: 提供传统回调接口适配器
- ✨ **性能优化**: 异步事件处理和音频缓冲管理
- ✨ **错误处理**: 完善的错误分类和恢复机制

### 迁移指南

从v1.x迁移到v2.0.0：

1. **更新导入路径**:
   ```go
   // 旧版本
   import "asr/client"

   // 新版本
   import "streamASR/sdk/golang/client"
   ```

2. **替换创建代码**:
   ```go
   // 旧版本
   recognizer := asr.NewRecognizer(listener)

   // 新版本
   handler := &NewEventHandler{}
   recognizer := asr.CreateRecognizerWithEventHandler(config, handler)
   ```

3. **更新音频处理**:
   ```go
   // 旧版本
   recognizer.Write(audioData)

   // 新版本 (需要提交)
   recognizer.Write(audioData)
   recognizer.CommitAudio()
   ```

4. **更新回调处理**:
   ```go
   // 旧版本
   type OldListener interface {
       OnRecognitionResult(*asr.RecognitionResponse)
   }

   // 新版本
   type NewHandler interface {
       OnTranscriptionCompleted(*asr.ConversationItemInputAudioTranscriptionCompletedEvent)
   }
   ```