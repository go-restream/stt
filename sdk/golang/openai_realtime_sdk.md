# StreamASR OpenAI Realtime API SDK

## 概述

StreamASR OpenAI Realtime API SDK 提供了与 OpenAI Realtime API 标准完全兼容的客户端实现，支持实时语音识别功能。

## 特性

- ✅ 完全兼容 OpenAI Realtime API v1.0
- ✅ 实时音频流处理
- ✅ 自动音频格式转换和重采样
- ✅ 内置 VAD (语音活动检测)
- ✅ 多种语言支持
- ✅ 自动重连机制
- ✅ 详细错误处理
- ✅ TypeScript 类型支持
- ✅ 多平台支持 (浏览器, Node.js, React Native)

## 安装

### NPM
```bash
npm install streamasr-openai-realtime
```

### Yarn
```bash
yarn add streamasr-openai-realtime
```

### CDN
```html
<script src="https://unpkg.com/streamasr-openai-realtime@latest/dist/index.js"></script>
```

## 快速开始

### 基础使用

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

// 创建客户端实例
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  url: 'ws://localhost:8080/v1/realtime'
});

// 监听转录结果
client.on('transcription', (transcript) => {
  console.log('识别结果:', transcript.text);
});

// 监听错误
client.on('error', (error) => {
  console.error('错误:', error.message);
});

// 连接并配置会话
await client.connect();
await client.configureSession({
  modality: 'audio',
  input_audio_format: {
    type: 'pcm16',
    sample_rate: 16000,
    channels: 1
  },
  input_audio_transcription: {
    model: 'whisper-1',
    language: 'zh'
  }
});

// 开始录音
await client.startRecording();
```

### React 组件示例

```jsx
import React, { useState, useEffect } from 'react';
import { StreamASRClient } from 'streamasr-openai-realtime';

function VoiceRecorder() {
  const [isRecording, setIsRecording] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [client, setClient] = useState(null);

  useEffect(() => {
    const asrClient = new StreamASRClient({
      apiKey: 'your-api-key',
      url: 'ws://localhost:8080/v1/realtime'
    });

    asrClient.on('transcription', (data) => {
      setTranscript(data.text);
    });

    asrClient.on('error', (error) => {
      console.error('ASR 错误:', error);
    });

    setClient(asrClient);

    return () => {
      asrClient.disconnect();
    };
  }, []);

  const handleConnect = async () => {
    if (!client) return;

    try {
      await client.connect();
      await client.configureSession({
        modality: 'audio',
        input_audio_format: {
          type: 'pcm16',
          sample_rate: 16000,
          channels: 1
        },
        input_audio_transcription: {
          model: 'whisper-1',
          language: 'zh'
        }
      });
    } catch (error) {
      console.error('连接失败:', error);
    }
  };

  const toggleRecording = async () => {
    if (!client) return;

    if (isRecording) {
      await client.stopRecording();
      setIsRecording(false);
    } else {
      await client.startRecording();
      setIsRecording(true);
    }
  };

  return (
    <div>
      <button onClick={handleConnect}>
        连接 ASR
      </button>
      <button onClick={toggleRecording}>
        {isRecording ? '停止录音' : '开始录音'}
      </button>
      <div>
        <h3>识别结果:</h3>
        <p>{transcript}</p>
      </div>
    </div>
  );
}

export default VoiceRecorder;
```

## API 参考

### StreamASRClient

#### 构造函数

```typescript
new StreamASRClient(options: ClientOptions)
```

**ClientOptions:**
```typescript
interface ClientOptions {
  apiKey: string;                    // API 密钥
  url?: string;                      // WebSocket URL，默认: 'ws://localhost:8080/v1/realtime'
  autoReconnect?: boolean;            // 自动重连，默认: true
  reconnectInterval?: number;         // 重连间隔(ms)，默认: 3000
  maxReconnectAttempts?: number;      // 最大重连次数，默认: 5
  enableLogging?: boolean;            // 启用日志，默认: false
}
```

#### 方法

##### connect()
```typescript
async connect(): Promise<void>
```
连接到 StreamASR 服务器。

##### disconnect()
```typescript
disconnect(): void
```
断开连接并清理资源。

##### configureSession()
```typescript
async configureSession(config: SessionConfig): Promise<void>
```

**SessionConfig:**
```typescript
interface SessionConfig {
  modality: 'audio' | 'text' | 'text_and_audio';
  instructions?: string;
  voice?: string;
  input_audio_format?: {
    type: 'pcm16';
    sample_rate: 16000 | 48000;
    channels: 1;
  };
  output_audio_format?: {
    type: 'pcm16';
    sample_rate: 16000;
    voice?: string;
  };
  input_audio_transcription?: {
    model: string;
    language: string;
  };
  turn_detection?: {
    type: 'server_vad';
    threshold: number;        // 0.0 - 1.0
    prefix_padding_ms: number;
    silence_duration_ms: number;
  };
}
```

##### startRecording()
```typescript
async startRecording(): Promise<void>
```
开始录音并自动发送音频数据。

##### stopRecording()
```typescript
async stopRecording(): Promise<void>
```
停止录音。

##### sendAudio()
```typescript
async sendAudio(audioData: ArrayBuffer): Promise<void>
```
手动发送音频数据。

##### commitAudio()
```typescript
async commitAudio(): Promise<void>
```
提交当前音频缓冲区进行处理。

##### clearAudioBuffer()
```typescript
async clearAudioBuffer(): Promise<void>
```
清空音频缓冲区。

#### 事件

##### on(event, listener)
```typescript
on(event: 'transcription', listener: (data: TranscriptionData) => void): void
on(event: 'speech_started', listener: (data: SpeechStartedData) => void): void
on(event: 'speech_stopped', listener: (data: SpeechStoppedData) => void): void
on(event: 'session_created', listener: (data: SessionCreatedData) => void): void
on(event: 'session_updated', listener: (data: SessionUpdatedData) => void): void
on(event: 'error', listener: (error: ErrorData) => void): void
on(event: 'connected', listener: () => void): void
on(event: 'disconnected', listener: (data: DisconnectedData) => void): void
```

**事件数据类型:**
```typescript
interface TranscriptionData {
  text: string;
  language: string;
  confidence: number;
  timestamp: number;
}

interface SpeechStartedData {
  audio_start_ms: number;
  timestamp: number;
}

interface SpeechStoppedData {
  audio_end_ms: number;
  timestamp: number;
}

interface SessionCreatedData {
  sessionId: string;
  model: string;
  modalities: string[];
}

interface SessionUpdatedData {
  sessionId: string;
  config: SessionConfig;
}

interface ErrorData {
  type: string;
  code: string;
  message: string;
  details?: any;
}

interface DisconnectedData {
  code: number;
  reason: string;
}
```

##### off(event, listener)
```typescript
off(event: string, listener: Function): void
```
移除事件监听器。

## 高级用法

### 自定义音频处理

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

const client = new StreamASRClient({
  apiKey: 'your-api-key',
  enableLogging: true
});

// 自定义音频源
async function processCustomAudio() {
  await client.connect();
  await client.configureSession({
    modality: 'audio',
    input_audio_format: {
      type: 'pcm16',
      sample_rate: 16000,
      channels: 1
    }
  });

  // 获取自定义音频流
  const audioStream = getCustomAudioStream();

  // 处理音频流
  const reader = audioStream.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    // 发送音频数据
    await client.sendAudio(value);
  }
}

processCustomAudio();
```

### 批量音频处理

```javascript
import { StreamASRClient, AudioUtils } from 'streamasr-openai-realtime';

const client = new StreamASRClient({
  apiKey: 'your-api-key'
});

async function processBatch() {
  await client.connect();
  await client.configureSession({
    modality: 'audio',
    input_audio_transcription: {
      model: 'whisper-1',
      language: 'zh'
    }
  });

  const audioFiles = [
    'audio1.wav',
    'audio2.wav',
    'audio3.wav'
  ];

  for (const file of audioFiles) {
    // 读取并转换音频
    const audioBuffer = await AudioUtils.loadWavFile(file);
    const pcm16Data = AudioUtils.convertToPCM16(audioBuffer);

    // 发送音频
    await client.sendAudio(pcm16Data);

    // 提交处理
    await client.commitAudio();

    // 等待处理完成
    await new Promise(resolve => {
      client.once('transcription', resolve);
    });
  }
}

processBatch();
```

### 多语言支持

```javascript
// 动态切换识别语言
async function switchLanguage(language) {
  await client.configureSession({
    modality: 'audio',
    input_audio_transcription: {
      model: 'whisper-1',
      language: language
    },
    turn_detection: {
      type: 'server_vad',
      threshold: 0.5,
      silence_duration_ms: 800
    }
  });
}

// 支持的语言
const languages = {
  'zh': '中文',
  'en': 'English',
  'ja': '日本語',
  'ko': '한국어',
  'es': 'Español',
  'fr': 'Français'
};

// 在 UI 中提供语言选择
function renderLanguageSelector() {
  return Object.entries(languages).map(([code, name]) => (
    <button key={code} onClick={() => switchLanguage(code)}>
      {name}
    </button>
  ));
}
```

### 实时音量可视化

```javascript
import { StreamASRClient, AudioAnalyzer } from 'streamasr-openai-realtime';

const client = new StreamASRClient({ apiKey: 'your-api-key' });
const analyzer = new AudioAnalyzer();

// 音量可视化
function updateVolumeBar(volume) {
  const bar = document.getElementById('volume-bar');
  bar.style.width = `${volume * 100}%`;
}

analyzer.on('volume', updateVolumeBar);

// 开始录音时启动分析器
client.on('speech_started', () => {
  analyzer.start();
});

client.on('speech_stopped', () => {
  analyzer.stop();
  updateVolumeBar(0);
});
```

## 错误处理

### 错误类型

```typescript
enum ErrorType {
  CONNECTION_ERROR = 'connection_error',
  AUTHENTICATION_ERROR = 'authentication_error',
  SESSION_ERROR = 'session_error',
  AUDIO_FORMAT_ERROR = 'audio_format_error',
  RECOGNITION_ERROR = 'recognition_error',
  NETWORK_ERROR = 'network_error',
  RATE_LIMIT_ERROR = 'rate_limit_error'
}
```

### 错误处理示例

```javascript
client.on('error', (error) => {
  switch (error.type) {
    case ErrorType.CONNECTION_ERROR:
      console.log('连接错误，尝试重连...');
      break;
    case ErrorType.AUTHENTICATION_ERROR:
      console.log('认证失败，请检查 API Key');
      break;
    case ErrorType.RECOGNITION_ERROR:
      console.log('识别错误，请重试');
      break;
    case ErrorType.RATE_LIMIT_ERROR:
      console.log('请求频率过高，请稍后重试');
      break;
    default:
      console.log('未知错误:', error.message);
  }
});
```

### 自动重连

```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectInterval: 3000
});

// 监听重连事件
client.on('reconnecting', (attempt) => {
  console.log(`正在重连... 第 ${attempt} 次尝试`);
});

client.on('reconnected', () => {
  console.log('重连成功');
});

client.on('max_reconnect_attempts_reached', () => {
  console.log('达到最大重连次数，停止重连');
});
```

## 性能优化

### 音频缓冲优化

```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key'
});

// 配置音频参数以优化性能
await client.configureSession({
  input_audio_format: {
    type: 'pcm16',
    sample_rate: 16000,  // 使用较低的采样率减少数据量
    channels: 1
  },
  turn_detection: {
    type: 'server_vad',
    threshold: 0.6,        // 提高阈值减少误触发
    silence_duration_ms: 500,  // 缩短静音时间
    prefix_padding_ms: 200
  }
});

// 优化音频发送频率
let audioBuffer = new Uint8Array(0);
const TARGET_CHUNK_SIZE = 1024;  // 1KB chunks
const SEND_INTERVAL = 100;       // 100ms

setInterval(() => {
  if (audioBuffer.length >= TARGET_CHUNK_SIZE) {
    const chunk = audioBuffer.slice(0, TARGET_CHUNK_SIZE);
    client.sendAudio(chunk.buffer);
    audioBuffer = audioBuffer.slice(TARGET_CHUNK_SIZE);
  }
}, SEND_INTERVAL);
```

### 内存管理

```javascript
class OptimizedASRClient extends StreamASRClient {
  constructor(options) {
    super(options);
    this.audioQueue = [];
    this.isProcessing = false;
  }

  async sendAudioOptimized(audioData) {
    // 使用队列避免音频堆积
    this.audioQueue.push(audioData);

    if (!this.isProcessing) {
      this.isProcessing = true;
      await this.processQueue();
      this.isProcessing = false;
    }
  }

  async processQueue() {
    while (this.audioQueue.length > 0) {
      const audioData = this.audioQueue.shift();
      try {
        await this.sendAudio(audioData);
      } catch (error) {
        console.error('发送音频失败:', error);
        break;
      }
    }
  }
}
```

## 测试

### 单元测试

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

describe('StreamASRClient', () => {
  let client;

  beforeEach(() => {
    client = new StreamASRClient({
      apiKey: 'test-api-key',
      url: 'ws://localhost:8080/v1/realtime'
    });
  });

  afterEach(() => {
    client.disconnect();
  });

  test('should connect successfully', async () => {
    await expect(client.connect()).resolves.not.toThrow();
  });

  test('should configure session', async () => {
    await client.connect();

    const config = {
      modality: 'audio',
      input_audio_format: {
        type: 'pcm16',
        sample_rate: 16000,
        channels: 1
      }
    };

    await expect(client.configureSession(config)).resolves.not.toThrow();
  });

  test('should handle transcription events', (done) => {
    client.on('transcription', (data) => {
      expect(data.text).toBeDefined();
      expect(data.language).toBeDefined();
      done();
    });

    // 模拟转录事件
    client.emit('transcription', {
      text: '测试文本',
      language: 'zh'
    });
  });
});
```

### 集成测试

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

describe('Integration Tests', () => {
  let client;

  beforeAll(async () => {
    client = new StreamASRClient({
      apiKey: process.env.ASR_API_KEY,
      url: process.env.ASR_URL
    });

    await client.connect();
  });

  afterAll(() => {
    client.disconnect();
  });

  test('should transcribe audio correctly', async () => {
    const transcriptionPromise = new Promise((resolve) => {
      client.on('transcription', resolve);
    });

    await client.configureSession({
      modality: 'audio',
      input_audio_transcription: {
        model: 'whisper-1',
        language: 'zh'
      }
    });

    // 发送测试音频
    const testAudio = loadTestAudio();
    await client.sendAudio(testAudio);
    await client.commitAudio();

    const result = await transcriptionPromise;
    expect(result.text).toBeTruthy();
    expect(result.language).toBe('zh');
  }, 10000);
});
```

## 故障排除

### 常见问题

1. **连接失败**
   - 检查网络连接
   - 验证 API Key
   - 确认服务器 URL 正确

2. **音频质量问题**
   - 调整采样率设置
   - 检查音频源质量
   - 优化 VAD 参数

3. **性能问题**
   - 减少音频块大小
   - 调整发送频率
   - 启用音频压缩

更多问题请参考 [故障排除指南](../docs/troubleshooting.md)。

## 更新日志

### v1.0.0 (2024-10-26)
- ✅ 初始版本发布
- ✅ 完整的 OpenAI Realtime API 支持
- ✅ TypeScript 类型定义
- ✅ 自动重连机制
- ✅ 多语言支持

## 许可证

MIT License

## 支持

- 文档: [https://docs.streamasr.com](https://docs.streamasr.com)
- 问题反馈: [GitHub Issues](https://github.com/streamasr/client-sdk/issues)
- 邮件支持: support@streamasr.com