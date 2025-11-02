# OpenAI Realtime API 文档

## 概述

StreamASR 现在支持 OpenAI Realtime API 标准，提供与 OpenAI 兼容的实时语音识别 WebSocket 接口。

## 协议版本

- **当前版本:** v1.0.0
- **兼容性:** 与 OpenAI Realtime API 规范完全兼容
- **协议:** WebSocket with JSON 事件

## 连接端点

```
ws://localhost:8080/v1/realtime
wss://your-domain.com/v1/realtime
```

## 认证

使用 Bearer Token 认证：
```javascript
const ws = new WebSocket('ws://localhost:8080/v1/realtime', [], {
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY'
  }
});
```

## 支持的事件类型

### 客户端发送事件

#### 1. session.update
配置或更新会话参数。

```json
{
  "type": "session.update",
  "event_id": "event_1234567890",
  "session": {
    "modality": "audio",
    "instructions": "请转录用户的语音",
    "voice": "alloy",
    "input_audio_format": {
      "type": "pcm16",
      "sample_rate": 16000,
      "channels": 1
    },
    "output_audio_format": {
      "type": "pcm16",
      "sample_rate": 16000,
      "voice": "alloy"
    },
    "input_audio_transcription": {
      "model": "whisper-1",
      "language": "auto"
    },
    "turn_detection": {
      "type": "server_vad",
      "threshold": 0.5,
      "prefix_padding_ms": 300,
      "silence_duration_ms": 200
    }
  }
}
```

#### 2. input_audio_buffer.append
向音频缓冲区添加 Base64 编码的音频数据。

```json
{
  "type": "input_audio_buffer.append",
  "event_id": "event_1234567890",
  "audio": "base64-encoded-audio-data-here"
}
```

#### 3. input_audio_buffer.commit
提交当前音频缓冲区进行识别处理。

```json
{
  "type": "input_audio_buffer.commit",
  "event_id": "event_1234567890"
}
```

#### 4. input_audio_buffer.clear
清空音频缓冲区。

```json
{
  "type": "input_audio_buffer.clear",
  "event_id": "event_1234567890"
}
```

#### 5. heartbeat.ping
发送心跳包保持连接活跃。

```json
{
  "type": "heartbeat.ping",
  "event_id": "event_1234567890",
  "heartbeat_type": 1
}
```

#### 6. conversation.item.deleted
删除对话项。

```json
{
  "type": "conversation.item.deleted",
  "event_id": "event_1234567890",
  "item_id": "item_1234567890"
}
```

### 服务器发送事件

#### 1. session.created
会话创建成功事件。

```json
{
  "type": "session.created",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "session": {
    "id": "sess_1234567890",
    "object": "realtime.session",
    "model": "gpt-4",
    "modalities": ["audio"]
  }
}
```

#### 2. session.updated
会话配置更新成功事件。

```json
{
  "type": "session.updated",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "session": {
    "id": "sess_1234567890",
    "object": "realtime.session",
    "model": "gpt-4",
    "modalities": ["audio"]
  }
}
```

#### 3. conversation.created
对话创建事件。

```json
{
  "type": "conversation.created",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "conversation": {
    "id": "conv_1234567890",
    "object": "realtime.conversation"
  }
}
```

#### 4. conversation.item.created
对话项创建事件。

```json
{
  "type": "conversation.item.created",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "item": {
    "id": "item_1234567890",
    "type": "message",
    "status": "incomplete",
    "audio": {
      "data": "base64-encoded-audio-data",
      "format": "pcm16"
    }
  }
}
```

#### 5. conversation.item.input_audio_transcription.completed
音频转录完成事件。

```json
{
  "type": "conversation.item.input_audio_transcription.completed",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "item": {
    "id": "item_1234567890",
    "type": "message",
    "status": "completed",
    "content": [
      {
        "type": "transcript",
        "transcript": "你好，这是识别的文本内容。"
      }
    ]
  }
}
```

#### 6. conversation.item.input_audio_transcription.failed
音频转录失败事件。

```json
{
  "type": "conversation.item.input_audio_transcription.failed",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "item_id": "item_1234567890",
  "error": {
    "type": "api_error",
    "code": "recognition_failed",
    "message": "语音识别失败，请重试"
  }
}
```

#### 7. input_audio_buffer.speech_started
语音活动检测开始事件。

```json
{
  "type": "input_audio_buffer.speech_started",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "audio_start_ms": 12345
}
```

#### 8. input_audio_buffer.speech_stopped
语音活动检测停止事件。

```json
{
  "type": "input_audio_buffer.speech_stopped",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "audio_end_ms": 23456
}
```

#### 9. input_audio_buffer.committed
音频缓冲区提交确认事件。

```json
{
  "type": "input_audio_buffer.committed",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890"
}
```

#### 10. input_audio_buffer.cleared
音频缓冲区清空确认事件。

```json
{
  "type": "input_audio_buffer.cleared",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890"
}
```

#### 11. heartbeat.pong
服务器响应心跳包。

```json
{
  "type": "heartbeat.pong",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "heartbeat_type": 1
}
```

#### 12. error
错误事件。

```json
{
  "type": "error",
  "event_id": "event_1234567890",
  "session_id": "sess_1234567890",
  "error": {
    "type": "invalid_request_error",
    "code": "message_processing_error",
    "message": "处理消息时发生错误",
    "param": "audio"
  }
}
```

## 支持的音频格式

### 输入音频
- **格式:** PCM16
- **采样率:** 16kHz 或 48kHz
- **编码:** Base64
- **声道:** 单声道

### 输出音频
- **格式:** PCM16
- **采样率:** 16kHz
- **编码:** Base64
- **声道:** 单声道

## 使用示例

### JavaScript 客户端示例

```javascript
class StreamASRClient {
  constructor(apiKey, url = 'ws://localhost:8080/v1/realtime') {
    this.apiKey = apiKey;
    this.url = url;
    this.ws = null;
    this.sessionId = null;
    this.audioBuffer = [];
  }

  connect() {
    this.ws = new WebSocket(this.url, [], {
      headers: {
        'Authorization': `Bearer ${this.apiKey}`
      }
    });

    this.ws.onopen = () => {
      console.log('Connected to StreamASR');
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('Disconnected from StreamASR');
    };
  }

  handleMessage(message) {
    switch (message.type) {
      case 'session.created':
        this.sessionId = message.session.id;
        console.log('Session created:', this.sessionId);
        break;
      case 'conversation.item.input_audio_transcription.completed':
        console.log('Transcription:', message.item.content[0].transcript);
        break;
      case 'error':
        console.error('Error:', message.error);
        break;
      default:
        console.log('Received message:', message);
    }
  }

  sendEvent(event) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
    }
  }

  updateSession(config) {
    this.sendEvent({
      type: 'session.update',
      event_id: `event_${Date.now()}`,
      session: config
    });
  }

  appendAudio(base64Audio) {
    this.sendEvent({
      type: 'input_audio_buffer.append',
      event_id: `event_${Date.now()}`,
      audio: base64Audio
    });
  }

  commitAudio() {
    this.sendEvent({
      type: 'input_audio_buffer.commit',
      event_id: `event_${Date.now()}`
    });
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// 使用示例
const client = new StreamASRClient('your-api-key');
client.connect();

// 等待连接建立后配置会话
setTimeout(() => {
  client.updateSession({
    modality: 'audio',
    input_audio_format: {
      type: 'pcm16',
      sample_rate: 16000,
      channels: 1
    },
    input_audio_transcription: {
      model: 'whisper-1',
      language: 'zh'
    },
    turn_detection: {
      type: 'server_vad',
      threshold: 0.5,
      silence_duration_ms: 800
    }
  });
}, 1000);
```

### Python 客户端示例

```python
import websocket
import json
import base64
import threading
import time

class StreamASRClient:
    def __init__(self, api_key, url="ws://localhost:8080/v1/realtime"):
        self.api_key = api_key
        self.url = url
        self.ws = None
        self.session_id = None
        self.event_counter = 0

    def generate_event_id(self):
        self.event_counter += 1
        return f"event_{int(time.time() * 1000000)}_{self.event_counter}"

    def on_open(self, ws):
        print("Connected to StreamASR")

    def on_message(self, ws, message):
        data = json.loads(message)
        self.handle_message(data)

    def on_error(self, ws, error):
        print(f"WebSocket error: {error}")

    def on_close(self, ws, close_status_code, close_msg):
        print("Disconnected from StreamASR")

    def handle_message(self, message):
        msg_type = message.get('type')

        if msg_type == 'session.created':
            self.session_id = message['session']['id']
            print(f"Session created: {self.session_id}")
        elif msg_type == 'conversation.item.input_audio_transcription.completed':
            transcript = message['item']['content'][0]['transcript']
            print(f"Transcription: {transcript}")
        elif msg_type == 'error':
            error = message['error']
            print(f"Error: {error['message']}")
        else:
            print(f"Received message: {message}")

    def connect(self):
        headers = {'Authorization': f'Bearer {self.api_key}'}
        self.ws = websocket.WebSocketApp(
            self.url,
            header=headers,
            on_open=self.on_open,
            on_message=self.on_message,
            on_error=self.on_error,
            on_close=self.on_close
        )

    def send_event(self, event):
        if self.ws:
            self.ws.send(json.dumps(event))

    def update_session(self, config):
        event = {
            'type': 'session.update',
            'event_id': self.generate_event_id(),
            'session': config
        }
        self.send_event(event)

    def append_audio(self, base64_audio):
        event = {
            'type': 'input_audio_buffer.append',
            'event_id': self.generate_event_id(),
            'audio': base64_audio
        }
        self.send_event(event)

    def commit_audio(self):
        event = {
            'type': 'input_audio_buffer.commit',
            'event_id': self.generate_event_id()
        }
        self.send_event(event)

    def run_forever(self):
        self.ws.run_forever()

# 使用示例
if __name__ == "__main__":
    client = StreamASRClient("your-api-key")
    client.connect()

    # 在新线程中运行 WebSocket
    ws_thread = threading.Thread(target=client.run_forever)
    ws_thread.daemon = True
    ws_thread.start()

    # 等待连接建立后配置会话
    time.sleep(1)
    client.update_session({
        'modality': 'audio',
        'input_audio_format': {
            'type': 'pcm16',
            'sample_rate': 16000,
            'channels': 1
        },
        'input_audio_transcription': {
            'model': 'whisper-1',
            'language': 'zh'
        }
    })

    # 保持运行
    ws_thread.join()
```

## 错误处理

### 常见错误代码

| 错误代码 | 描述 | 解决方案 |
|---------|------|----------|
| `invalid_request_error` | 请求格式错误 | 检查 JSON 格式和必需字段 |
| `message_processing_error` | 消息处理失败 | 检查事件类型和参数 |
| `audio_conversion_error` | 音频转换失败 | 检查音频格式和编码 |
| `recognition_error` | 语音识别失败 | 检查音频质量和网络连接 |
| `session_expired` | 会话过期 | 重新建立连接 |
| `rate_limit_exceeded` | 请求频率超限 | 降低请求频率 |

## 性能优化建议

1. **音频缓冲区管理**
   - 建议每 100-200ms 发送一次音频数据
   - 缓冲区大小建议在 1024-4096 样本之间

2. **网络优化**
   - 使用压缩算法减少传输数据量
   - 启用 WebSocket 心跳保持连接

3. **实时性优化**
   - 使用较小的音频块以减少延迟
   - 配置合适的 VAD 参数

## 故障排除

详见 [故障排除指南](./troubleshooting.md)