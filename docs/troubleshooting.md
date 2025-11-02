# StreamASR 故障排除指南

## 概述

本指南提供 StreamASR OpenAI Realtime API 常见问题的诊断和解决方案。

## 目录

1. [连接问题](#连接问题)
2. [认证问题](#认证问题)
3. [音频问题](#音频问题)
4. [转录质量问题](#转录质量问题)
5. [性能问题](#性能问题)
6. [错误代码参考](#错误代码参考)
7. [调试工具](#调试工具)

## 连接问题

### 问题: WebSocket 连接失败

#### 症状
- 连接超时
- 连接被拒绝
- 频繁断开连接

#### 可能原因
1. 网络连接问题
2. 防火墙阻止
3. 服务器未运行
4. 端口被占用

#### 解决方案

##### 1. 检查网络连接
```bash
# 测试基本网络连通性
ping your-server.com

# 测试端口连通性
telnet your-server.com 8080

# 使用 curl 测试 HTTP 端点
curl -I http://your-server.com:8080/health
```

##### 2. 检查服务器状态
```javascript
// 添加连接状态监听
client.on('connecting', () => {
  console.log('正在连接服务器...');
});

client.on('connected', () => {
  console.log('连接成功');
});

client.on('disconnected', (event) => {
  console.log('连接断开:', event.code, event.reason);
});

client.on('error', (error) => {
  console.error('连接错误:', error.message);
});
```

##### 3. 配置自动重连
```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectInterval: 3000
});

// 监听重连事件
client.on('reconnecting', (attempt) => {
  console.log(`重连尝试 ${attempt}/5`);
});

client.on('max_reconnect_attempts_reached', () => {
  console.error('达到最大重连次数，请检查网络连接');
});
```

##### 4. 防火墙配置
```bash
# 检查端口是否开放
nmap -p 8080 your-server.com

# 如果端口被阻止，需要开放端口：
# Ubuntu/Debian:
sudo ufw allow 8080

# CentOS/RHEL:
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

## 认证问题

### 问题: API Key 认证失败

#### 症状
- 401 Unauthorized 错误
- Authentication failed 错误事件

#### 可能原因
1. API Key 错误
2. API Key 过期
3. Header 格式不正确

#### 解决方案

##### 1. 验证 API Key
```javascript
// 检查 API Key 格式
const apiKey = 'your-api-key';

if (!apiKey.startsWith('sk-')) {
  console.error('API Key 格式错误');
}

if (apiKey.length < 20) {
  console.error('API Key 长度不足');
}
```

##### 2. 正确设置 Authorization Header
```javascript
// ✅ 正确的方式
const client = new StreamASRClient({
  apiKey: 'sk-xxxxxxxxxxxxxxxxxxxxxxxx',
  // SDK 会自动添加正确的 Authorization header
});

// ❌ 错误的方式
const ws = new WebSocket('ws://localhost:8080/v1/realtime?token=sk-xxx');
```

##### 3. 环境变量配置
```bash
# 设置环境变量
export STREAMASR_API_KEY="sk-your-api-key"
export STREAMASR_URL="ws://localhost:8080/v1/realtime"

# 在代码中使用
const client = new StreamASRClient({
  apiKey: process.env.STREAMASR_API_KEY,
  url: process.env.STREAMASR_URL
});
```

## 音频问题

### 问题: 音频数据无法正确处理

#### 症状
- 无转录结果
- 音频格式错误
- Base64 解码失败

#### 可能原因
1. 音频格式不正确
2. 采样率不匹配
3. Base64 编码错误

#### 解决方案

##### 1. 验证音频格式
```javascript
// 检查音频配置
const audioConfig = {
  type: 'pcm16',
  sample_rate: 16000, // 必须是 16000 或 48000
  channels: 1          // 必须是 1 (单声道)
};

if (audioConfig.sample_rate !== 16000 && audioConfig.sample_rate !== 48000) {
  console.error('不支持的采样率');
}
```

##### 2. 正确转换音频到 Base64
```javascript
// ✅ 正确的 Base64 转换
function arrayBufferToBase64(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

// 处理音频时
const audioBuffer = getAudioData(); // ArrayBuffer
const base64Audio = arrayBufferToBase64(audioBuffer);

await client.sendAudio(base64Audio);
```

##### 3. 音频重采样
```javascript
import { AudioUtils } from 'streamasr-openai-realtime';

// 如果原始音频不是 16kHz 或 48kHz，需要重采样
async function prepareAudio(audioData, originalSampleRate) {
  let processedAudio = audioData;

  if (originalSampleRate !== 16000 && originalSampleRate !== 48000) {
    // 重采样到 16kHz
    processedAudio = await AudioUtils.resample(
      audioData,
      originalSampleRate,
      16000
    );
  }

  return processedAudio;
}
```

## 转录质量问题

### 问题: 转录结果不准确或不完整

#### 症状
- 识别准确率低
- 丢失部分语音
- 语言识别错误

#### 可能原因
1. 音频质量问题
2. 环境噪音
3. VAD 参数不合适
4. 语言配置错误

#### 解决方案

##### 1. 优化音频质量
```javascript
// 音频配置优化
await client.configureSession({
  input_audio_format: {
    type: 'pcm16',
    sample_rate: 16000,  // 使用标准采样率
    channels: 1           // 确保单声道
  }
});

// 音频处理优化
function preprocessAudio(audioData) {
  // 应用降噪
  const denoisedAudio = applyNoiseReduction(audioData);

  // 应用音量标准化
  const normalizedAudio = normalizeVolume(denoisedAudio);

  return normalizedAudio;
}
```

##### 2. 配置 VAD 参数
```javascript
// 根据环境调整 VAD 参数
const vadConfig = {
  type: 'server_vad',
  threshold: 0.5,        // 灵敏度: 0.1-1.0
  prefix_padding_ms: 200,  // 语音开始前的缓冲时间
  silence_duration_ms: 800 // 语音结束后的静音时间
};

// 噪音环境 - 降低灵敏度
if (isNoisyEnvironment()) {
  vadConfig.threshold = 0.7;
  vadConfig.silence_duration_ms = 1000;
}

// 安静环境 - 提高灵敏度
if (isQuietEnvironment()) {
  vadConfig.threshold = 0.3;
  vadConfig.silence_duration_ms = 500;
}

await client.configureSession({
  turn_detection: vadConfig
});
```

##### 3. 语言和模型配置
```javascript
// 配置正确的识别语言
const languageConfig = {
  input_audio_transcription: {
    model: 'whisper-1',
    language: 'zh'  // 使用正确的语言代码
  }
};

// 支持的语言代码
const supportedLanguages = {
  'zh': '中文',
  'en': 'English',
  'ja': '日本語',
  'ko': '한국어',
  'es': 'Español',
  'fr': 'Français',
  'de': 'Deutsch',
  'ru': 'Русский'
};

await client.configureSession(languageConfig);
```

## 性能问题

### 问题: 高延迟或资源占用过高

#### 症状
- 响应延迟高
- CPU 占用率高
- 内存使用过多
- 音频缓冲区溢出

#### 可能原因
1. 音频数据发送过于频繁
2. 缓冲区配置不当
3. 网络带宽不足

#### 解决方案

##### 1. 优化音频发送策略
```javascript
class OptimizedAudioSender {
  constructor(client) {
    this.client = client;
    this.audioBuffer = [];
    this.isSending = false;
    this.lastSendTime = 0;
    this.MIN_SEND_INTERVAL = 50; // 50ms 最小间隔
    this.MAX_BUFFER_SIZE = 4096; // 4KB 最大缓冲
  }

  async addAudio(audioData) {
    this.audioBuffer.push(new Uint8Array(audioData));

    // 检查是否需要发送
    const now = Date.now();
    if (now - this.lastSendTime >= this.MIN_SEND_INTERVAL ||
        this.getBufferSize() >= this.MAX_BUFFER_SIZE) {
      await this.sendBufferedAudio();
      this.lastSendTime = now;
    }
  }

  async sendBufferedAudio() {
    if (this.isSending || this.audioBuffer.length === 0) return;

    this.isSending = true;

    try {
      // 合并缓冲的音频
      const combinedAudio = this.combineAudioBuffers();

      // 发送合并后的音频
      await this.client.sendAudio(combinedAudio);

      // 清空缓冲区
      this.audioBuffer = [];
    } catch (error) {
      console.error('发送音频失败:', error);
    } finally {
      this.isSending = false;
    }
  }

  getBufferSize() {
    return this.audioBuffer.reduce((total, buffer) => total + buffer.length, 0);
  }

  combineAudioBuffers() {
    const totalSize = this.getBufferSize();
    const combined = new Uint8Array(totalSize);
    let offset = 0;

    for (const buffer of this.audioBuffer) {
      combined.set(buffer, offset);
      offset += buffer.length;
    }

    return combined.buffer;
  }
}
```

##### 2. 内存管理
```javascript
class MemoryManagedClient extends StreamASRClient {
  constructor(options) {
    super(options);
    this.maxMemoryUsage = 100 * 1024 * 1024; // 100MB
    this.currentMemoryUsage = 0;
    this.audioChunks = [];
  }

  async sendAudio(audioData) {
    // 检查内存使用
    const dataSize = audioData.byteLength;
    if (this.currentMemoryUsage + dataSize > this.maxMemoryUsage) {
      console.warn('内存使用过高，清理旧数据');
      this.cleanupOldAudio();
    }

    this.currentMemoryUsage += dataSize;
    this.audioChunks.push({
      data: audioData,
      timestamp: Date.now()
    });

    await super.sendAudio(audioData);
  }

  cleanupOldAudio() {
    const cutoffTime = Date.now() - 30000; // 30秒前
    const initialLength = this.audioChunks.length;

    this.audioChunks = this.audioChunks.filter(
      chunk => chunk.timestamp > cutoffTime
    );

    // 重新计算内存使用
    this.currentMemoryUsage = this.audioChunks.reduce(
      (total, chunk) => total + chunk.data.byteLength, 0
    );

    const cleaned = initialLength - this.audioChunks.length;
    console.log(`清理了 ${cleaned} 个旧音频块`);
  }
}
```

##### 3. 网络优化
```javascript
// 启用数据压缩（如果支持）
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  enableCompression: true,  // 启用 WebSocket 压缩
  compressionLevel: 6      // 压缩级别 1-9
});

// 监控网络状况
client.on('network_stats', (stats) => {
  console.log('网络延迟:', stats.latency, 'ms');
  console.log('数据包丢失:', stats.packetLoss, '%');

  // 根据网络状况调整参数
  if (stats.latency > 1000) {
    console.log('网络延迟过高，增加缓冲区大小');
    // 调整缓冲区配置
  }
});
```

## 错误代码参考

### WebSocket 错误代码

| 代码 | 含义 | 解决方案 |
|------|------|----------|
| 1000 | 正常关闭 | 无需处理 |
| 1001 | 端点离开 | 重新连接 |
| 1002 | 协议错误 | 检查客户端协议实现 |
| 1003 | 不支持的数据类型 | 检查数据格式 |
| 1004 | 保留 | 无需处理 |
| 1005 | 保留状态码 | 无需处理 |
| 1006 | 连接异常关闭 | 检查网络，启用重连 |
| 1007 | 无效的框架数据 | 检查数据帧格式 |
| 1008 | 策略违反 | 检查消息大小限制 |
| 1009 | 消息过大 | 减小消息大小 |
| 1010 | 扩展协商失败 | 检查扩展配置 |
| 1011 | 服务器意外错误 | 联系技术支持 |
| 1015 | TLS 握手失败 | 检查证书配置 |

### API 错误类型

| 错误类型 | 描述 | 解决方案 |
|----------|------|----------|
| `invalid_request_error` | 请求格式错误 | 检查 JSON 格式和必需字段 |
| `authentication_error` | 认证失败 | 检查 API Key 和认证方式 |
| `permission_denied_error` | 权限不足 | 检查 API Key 权限 |
| `not_found_error` | 资源不存在 | 检查端点 URL |
| `rate_limit_error` | 请求频率超限 | 降低请求频率 |
| `api_error` | 服务器内部错误 | 重试或联系支持 |
| `audio_format_error` | 音频格式错误 | 检查音频格式和编码 |
| `session_error` | 会话错误 | 重新创建会话 |
| `recognition_error` | 识别失败 | 检查音频质量和网络 |

## 调试工具

### 1. 启用详细日志

```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  enableLogging: true,
  logLevel: 'debug' // 'debug', 'info', 'warn', 'error'
});

// 自定义日志处理
client.on('log', (level, message, data) => {
  console.log(`[${level.toUpperCase()}] ${message}`, data);
});
```

### 2. 网络监控

```javascript
// 网络状况监控
class NetworkMonitor {
  constructor() {
    this.latencyHistory = [];
    this.packetLossCount = 0;
    this.totalPackets = 0;
  }

  startMonitoring(client) {
    // 发送 ping 测量延迟
    setInterval(() => {
      const startTime = Date.now();
      client.sendPing();

      client.once('pong', () => {
        const latency = Date.now() - startTime;
        this.latencyHistory.push(latency);

        if (this.latencyHistory.length > 10) {
          this.latencyHistory.shift();
        }

        console.log('网络延迟:', latency, 'ms');
      });
    }, 5000);

    // 监控数据包
    client.on('audio_sent', () => {
      this.totalPackets++;
    });

    client.on('audio_ack', () => {
      // 收到确认，说明数据包成功
    });

    client.on('timeout', () => {
      this.packetLossCount++;
      console.log('检测到数据包丢失');
    });
  }

  getStats() {
    const avgLatency = this.latencyHistory.reduce((a, b) => a + b, 0) / this.latencyHistory.length;
    const packetLossRate = (this.packetLossCount / this.totalPackets) * 100;

    return {
      averageLatency: avgLatency,
      packetLossRate: packetLossRate,
      connectionQuality: this.getConnectionQuality(avgLatency, packetLossRate)
    };
  }

  getConnectionQuality(latency, packetLoss) {
    if (latency < 100 && packetLoss < 1) return 'excellent';
    if (latency < 200 && packetLoss < 3) return 'good';
    if (latency < 500 && packetLoss < 5) return 'fair';
    return 'poor';
  }
}

// 使用监控器
const monitor = new NetworkMonitor();
monitor.startMonitoring(client);

setInterval(() => {
  const stats = monitor.getStats();
  console.log('网络状况:', stats);
}, 10000);
```

### 3. 音频质量分析

```javascript
class AudioQualityAnalyzer {
  constructor() {
    this.audioHistory = [];
    this.maxHistorySize = 100;
  }

  analyzeAudio(audioData) {
    // 计算音频统计信息
    const stats = this.calculateAudioStats(audioData);

    // 保存历史记录
    this.audioHistory.push(stats);
    if (this.audioHistory.length > this.maxHistorySize) {
      this.audioHistory.shift();
    }

    // 检测问题
    this.detectIssues(stats);

    return stats;
  }

  calculateAudioStats(audioData) {
    const samples = new Int16Array(audioData);

    let sum = 0;
    let max = 0;
    let min = 0;
    let zeroCount = 0;

    for (let i = 0; i < samples.length; i++) {
      const sample = samples[i];
      sum += sample;
      max = Math.max(max, sample);
      min = Math.min(min, sample);
      if (sample === 0) zeroCount++;
    }

    const average = sum / samples.length;
    const rms = Math.sqrt(samples.reduce((sq, sample) => sq + sample * sample, 0) / samples.length);
    const dynamicRange = max - min;
    const silenceRatio = zeroCount / samples.length;

    return {
      average,
      rms,
      peak: Math.max(Math.abs(max), Math.abs(min)),
      dynamicRange,
      silenceRatio,
      clipping: Math.abs(rms) > 30000 // 检测削波
    };
  }

  detectIssues(stats) {
    if (stats.silenceRatio > 0.8) {
      console.warn('音频静音比例过高，检查麦克风');
    }

    if (stats.clipping) {
      console.warn('检测到音频削波，降低输入音量');
    }

    if (stats.dynamicRange < 1000) {
      console.warn('音频动态范围过小，检查麦克风增益');
    }

    if (Math.abs(stats.average) > 1000) {
      console.warn('音频直流偏移，检查硬件');
    }
  }
}

// 使用音频分析器
const analyzer = new AudioQualityAnalyzer();

client.on('audio_received', (audioData) => {
  const quality = analyzer.analyzeAudio(audioData);
  console.log('音频质量:', quality);
});
```

## 获取支持

### 自助诊断
1. **检查连接:** 使用网络监控工具
2. **验证配置:** 检查 API Key 和端点
3. **测试音频:** 确认音频格式正确
4. **查看日志:** 启用详细日志记录

### 技术支持
如果问题仍然存在：

1. **收集信息:**
   - 错误消息和代码
   - 网络环境描述
   - 音频配置参数
   - 复现步骤

2. **联系方式:**
   - GitHub Issues: [提交问题](https://github.com/streamasr/issues)
   - 邮件支持: support@streamasr.com
   - 技术文档: [StreamASR Docs](https://docs.streamasr.com)

3. **报告模板:**
   ```
   **问题描述:**

   **环境信息:**
   - 操作系统:
   - 浏览器/运行时:
   - 网络环境:

   **配置信息:**
   - API 版本:
   - 音频格式:
   - 采样率:

   **错误信息:**
   - 错误代码:
   - 错误消息:
   - 复现步骤:
   ```

## 常用解决方案

### 快速修复清单

- [ ] 检查网络连接
- [ ] 验证 API Key 有效性
- [ ] 确认端点 URL 正确
- [ ] 检查音频格式 (PCM16, 16kHz/48kHz, 单声道)
- [ ] 验证 Base64 编码
- [ ] 配置合适的语言代码
- [ ] 调整 VAD 参数
- [ ] 启用自动重连
- [ ] 检查防火墙设置

这个故障排除指南应该能帮助您诊断和解决大多数 StreamASR 使用中的问题。如果需要进一步的帮助，请参考获取支持部分。