# StreamASR OpenAI Realtime API SDK (English)

## Overview

StreamASR OpenAI Realtime API SDK provides a fully compatible client implementation with the OpenAI Realtime API standard, supporting real-time speech recognition capabilities.

## Features

- ✅ Full compatibility with OpenAI Realtime API v1.0
- ✅ Real-time audio streaming
- ✅ Automatic audio format conversion and resampling
- ✅ Built-in VAD (Voice Activity Detection)
- ✅ Multi-language support
- ✅ Automatic reconnection
- ✅ Comprehensive error handling
- ✅ TypeScript type support
- ✅ Cross-platform support (Browser, Node.js, React Native)

## Installation

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

## Quick Start

### Basic Usage

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

// Create client instance
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  url: 'ws://localhost:8080/v1/realtime'
});

// Listen for transcription results
client.on('transcription', (transcript) => {
  console.log('Transcription:', transcript.text);
});

// Listen for errors
client.on('error', (error) => {
  console.error('Error:', error.message);
});

// Connect and configure session
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
    language: 'en'
  }
});

// Start recording
await client.startRecording();
```

### React Component Example

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
      console.error('ASR Error:', error);
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
          language: 'en'
        }
      });
    } catch (error) {
      console.error('Connection failed:', error);
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
        Connect ASR
      </button>
      <button onClick={toggleRecording}>
        {isRecording ? 'Stop Recording' : 'Start Recording'}
      </button>
      <div>
        <h3>Transcription:</h3>
        <p>{transcript}</p>
      </div>
    </div>
  );
}

export default VoiceRecorder;
```

## API Reference

### StreamASRClient

#### Constructor

```typescript
new StreamASRClient(options: ClientOptions)
```

**ClientOptions:**
```typescript
interface ClientOptions {
  apiKey: string;                    // API key
  url?: string;                      // WebSocket URL, default: 'ws://localhost:8080/v1/realtime'
  autoReconnect?: boolean;            // Auto reconnection, default: true
  reconnectInterval?: number;         // Reconnection interval(ms), default: 3000
  maxReconnectAttempts?: number;      // Max reconnection attempts, default: 5
  enableLogging?: boolean;            // Enable logging, default: false
}
```

#### Methods

##### connect()
```typescript
async connect(): Promise<void>
```
Connect to StreamASR server.

##### disconnect()
```typescript
disconnect(): void
```
Disconnect and clean up resources.

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
Start recording and automatically send audio data.

##### stopRecording()
```typescript
async stopRecording(): Promise<void>
```
Stop recording.

##### sendAudio()
```typescript
async sendAudio(audioData: ArrayBuffer): Promise<void>
```
Manually send audio data.

##### commitAudio()
```typescript
async commitAudio(): Promise<void>
```
Commit current audio buffer for processing.

##### clearAudioBuffer()
```typescript
async clearAudioBuffer(): Promise<void>
```
Clear audio buffer.

#### Events

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

**Event Data Types:**
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
Remove event listener.

## Advanced Usage

### Custom Audio Processing

```javascript
import { StreamASRClient } from 'streamasr-openai-realtime';

const client = new StreamASRClient({
  apiKey: 'your-api-key',
  enableLogging: true
});

// Custom audio source
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

  // Get custom audio stream
  const audioStream = getCustomAudioStream();

  // Process audio stream
  const reader = audioStream.getReader();
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    // Send audio data
    await client.sendAudio(value);
  }
}

processCustomAudio();
```

### Batch Audio Processing

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
      language: 'en'
    }
  });

  const audioFiles = [
    'audio1.wav',
    'audio2.wav',
    'audio3.wav'
  ];

  for (const file of audioFiles) {
    // Load and convert audio
    const audioBuffer = await AudioUtils.loadWavFile(file);
    const pcm16Data = AudioUtils.convertToPCM16(audioBuffer);

    // Send audio
    await client.sendAudio(pcm16Data);

    // Commit for processing
    await client.commitAudio();

    // Wait for processing completion
    await new Promise(resolve => {
      client.once('transcription', resolve);
    });
  }
}

processBatch();
```

### Multi-language Support

```javascript
// Dynamic language switching
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

// Supported languages
const languages = {
  'en': 'English',
  'zh': '中文',
  'ja': '日本語',
  'ko': '한국어',
  'es': 'Español',
  'fr': 'Français'
};

// Language selection in UI
function renderLanguageSelector() {
  return Object.entries(languages).map(([code, name]) => (
    <button key={code} onClick={() => switchLanguage(code)}>
      {name}
    </button>
  ));
}
```

## Error Handling

### Error Types

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

### Error Handling Example

```javascript
client.on('error', (error) => {
  switch (error.type) {
    case ErrorType.CONNECTION_ERROR:
      console.log('Connection error, attempting reconnection...');
      break;
    case ErrorType.AUTHENTICATION_ERROR:
      console.log('Authentication failed, check API Key');
      break;
    case ErrorType.RECOGNITION_ERROR:
      console.log('Recognition error, please retry');
      break;
    case ErrorType.RATE_LIMIT_ERROR:
      console.log('Rate limit exceeded, please try again later');
      break;
    default:
      console.log('Unknown error:', error.message);
  }
});
```

### Auto Reconnection

```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectInterval: 3000
});

// Listen for reconnection events
client.on('reconnecting', (attempt) => {
  console.log(`Reconnecting... attempt ${attempt}`);
});

client.on('reconnected', () => {
  console.log('Reconnected successfully');
});

client.on('max_reconnect_attempts_reached', () => {
  console.log('Max reconnection attempts reached, stopping');
});
```

## Performance Optimization

### Audio Buffer Optimization

```javascript
const client = new StreamASRClient({
  apiKey: 'your-api-key'
});

// Configure audio parameters for optimal performance
await client.configureSession({
  input_audio_format: {
    type: 'pcm16',
    sample_rate: 16000,  // Use lower sample rate to reduce data volume
    channels: 1
  },
  turn_detection: {
    type: 'server_vad',
    threshold: 0.6,        // Higher threshold to reduce false triggers
    silence_duration_ms: 500,  // Shorter silence duration
    prefix_padding_ms: 200
  }
});

// Optimize audio sending frequency
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

## Testing

### Unit Testing

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

    // Mock transcription event
    client.emit('transcription', {
      text: 'Test text',
      language: 'en'
    });
  });
});
```

### Integration Testing

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
        language: 'en'
      }
    });

    // Send test audio
    const testAudio = loadTestAudio();
    await client.sendAudio(testAudio);
    await client.commitAudio();

    const result = await transcriptionPromise;
    expect(result.text).toBeTruthy();
    expect(result.language).toBe('en');
  }, 10000);
});
```

## Migration Guide

For migrating from the old custom protocol, see our [Migration Guide](../docs/migration_guide.md).

## Troubleshooting

### Common Issues

1. **Connection Failure**
   - Check network connection
   - Verify API Key
   - Confirm server URL is correct

2. **Audio Quality Issues**
   - Adjust sample rate settings
   - Check audio source quality
   - Optimize VAD parameters

3. **Performance Issues**
   - Reduce audio chunk size
   - Adjust sending frequency
   - Enable audio compression

For more issues, see our [Troubleshooting Guide](../docs/troubleshooting.md).

## Changelog

### v1.0.0 (2024-10-26)
- ✅ Initial release
- ✅ Full OpenAI Realtime API support
- ✅ TypeScript type definitions
- ✅ Auto-reconnection mechanism
- ✅ Multi-language support

## License

MIT License

## Support

- Documentation: [https://docs.streamasr.com](https://docs.streamasr.com)
- Issues: [GitHub Issues](https://github.com/streamasr/client-sdk/issues)
- Email: support@streamasr.com