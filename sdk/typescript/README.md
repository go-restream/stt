# StreamASR TypeScript SDK

Official TypeScript client SDK for StreamASR OpenAI Realtime API, providing real-time speech recognition capabilities with full OpenAI Realtime API compatibility.

## Features

- ✅ **Full OpenAI Realtime API Compatibility** - Complete implementation of the OpenAI Realtime API specification
- ✅ **Type-Safe** - Comprehensive TypeScript type definitions for all API events and configurations
- ✅ **Cross-Platform** - Works in both browser and Node.js environments
- ✅ **Real-time Audio Processing** - Built-in audio recording, format conversion, and resampling
- ✅ **Voice Activity Detection** - Automatic speech start/stop detection
- ✅ **Session Management** - Automatic reconnection, heartbeat, and session lifecycle management
- ✅ **React Integration** - React hooks and components for easy integration
- ✅ **Error Handling** - Comprehensive error handling and recovery mechanisms
- ✅ **Audio Utilities** - Built-in audio processing utilities and format conversions

## Installation

```bash
# NPM
npm install @streamasr/openai-realtime-sdk

# Yarn
yarn add @streamasr/openai-realtime-sdk

# CDN
<script src="https://unpkg.com/@streamasr/openai-realtime-sdk@latest/dist/index.js"></script>
```

## Quick Start

### Basic Usage

```typescript
import { StreamASRClient } from '@streamasr/openai-realtime-sdk';

// Create client instance
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  url: 'ws://localhost:8080/v1/realtime',
  enableLogging: true,
  autoReconnect: true,
});

// Set up event listeners
client.on('sessionCreated', (sessionData) => {
  console.log('Session created:', sessionData.id);
});

client.on('transcription', (data) => {
  console.log('Transcription:', data.text);
});

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
    channels: 1,
  },
  input_audio_transcription: {
    model: 'whisper-1',
    language: 'zh',
  },
  turn_detection: {
    type: 'server_vad',
    threshold: 0.5,
    prefix_padding_ms: 300,
    silence_duration_ms: 800,
  },
});

// Start recording
await client.startRecording();
```

### React Integration

```typescript
import React from 'react';
import { useStreamASR } from '@streamasr/openai-realtime-sdk/react';

function VoiceRecorder() {
  const {
    isConnected,
    isRecording,
    transcript,
    connect,
    disconnect,
    startRecording,
    stopRecording,
    lastError,
  } = useStreamASR({
    apiKey: 'your-api-key',
    url: 'ws://localhost:8080/v1/realtime',
    autoReconnect: true,
    enableLogging: true,
  });

  return (
    <div>
      <div>Connected: {isConnected ? 'Yes' : 'No'}</div>
      <div>Recording: {isRecording ? 'Yes' : 'No'}</div>
      <div>Transcript: {transcript}</div>
      {lastError && <div>Error: {lastError.message}</div>}

      <button onClick={connect} disabled={isConnected}>
        Connect
      </button>
      <button onClick={disconnect} disabled={!isConnected}>
        Disconnect
      </button>
      <button onClick={startRecording} disabled={!isConnected || isRecording}>
        Start Recording
      </button>
      <button onClick={stopRecording} disabled={!isRecording}>
        Stop Recording
      </button>
    </div>
  );
}
```

## API Reference

### StreamASRClient

#### Constructor

```typescript
new StreamASRClient(options: ClientOptions)
```

**ClientOptions:**
- `apiKey: string` - Your StreamASR API key
- `url?: string` - WebSocket server URL (default: `'ws://localhost:8080/v1/realtime'`)
- `autoReconnect?: boolean` - Enable automatic reconnection (default: `true`)
- `reconnectInterval?: number` - Reconnection interval in ms (default: `3000`)
- `maxReconnectAttempts?: number` - Maximum reconnection attempts (default: `5`)
- `enableLogging?: boolean` - Enable debug logging (default: `false`)
- `heartbeatInterval?: number` - Heartbeat interval in ms (default: `30000`)
- `sessionTimeout?: number` - Session timeout in ms (default: `1800000`)

#### Methods

##### `connect(): Promise<void>`
Connect to the StreamASR server.

##### `disconnect(): void`
Disconnect from the server and clean up resources.

##### `configureSession(config: SessionConfig): Promise<void>`
Configure session parameters.

**SessionConfig:**
```typescript
interface SessionConfig {
  modality: 'audio' | 'text' | 'text_and_audio';
  instructions?: string;
  voice?: string;
  input_audio_format?: AudioFormat;
  output_audio_format?: AudioFormat;
  input_audio_transcription?: InputAudioTranscription;
  turn_detection?: TurnDetection;
  tools?: any[];
  tool_choice?: 'auto' | 'none' | 'required';
  temperature?: number;
  max_output_tokens?: number | 'inf';
}

interface AudioFormat {
  type: 'pcm16';
  sample_rate: 16000 | 48000;
  channels: 1;
}

interface InputAudioTranscription {
  model: string;
  language: string;
  prompt?: string;
}

interface TurnDetection {
  type: 'server_vad';
  threshold: number; // 0.0 - 1.0
  prefix_padding_ms: number;
  silence_duration_ms: number;
}
```

##### `startRecording(): Promise<void>`
Start recording audio from the microphone.

##### `stopRecording(): void`
Stop audio recording.

##### `sendAudio(audioData: ArrayBuffer | Int16Array): Promise<void>`
Manually send audio data to the server.

##### `commitAudio(): Promise<void>`
Commit the current audio buffer for processing.

##### `clearAudioBuffer(): Promise<void>`
Clear the audio buffer.

#### Events

The client extends EventEmitter and supports the following events:

##### Connection Events
- `connectionStateChanged` - Connection state changed
- `disconnected` - Connection closed
- `maxReconnectAttemptsReached` - Maximum reconnection attempts reached

##### Session Events
- `sessionCreated` - Session created successfully
- `sessionUpdated` - Session configuration updated

##### Recording Events
- `recordingStateChanged` - Recording state changed

##### Transcription Events
- `transcription` - Transcription result received
- `speechStarted` - Speech activity detected
- `speechStopped` - Speech activity ended

##### Error Events
- `error` - Error occurred

##### Generic Events
- `event` - Any server event received
- `pong` - Heartbeat pong response

**Example:**
```typescript
client.on('transcription', (data: TranscriptionData) => {
  console.log('Transcription:', data.text);
  console.log('Language:', data.language);
  console.log('Timestamp:', new Date(data.timestamp));
});

client.on('error', (error: ErrorData) => {
  console.error('Error:', error.code, error.message);
});
```

### React Hooks

#### `useStreamASR(options: UseStreamASROptions): UseStreamASRResult`

React hook for managing StreamASR client connection and state.

**UseStreamASROptions extends ClientOptions:**
- `autoConnect?: boolean` - Automatically connect on mount (default: `false`)
- `autoStartRecording?: boolean` - Automatically start recording (default: `false`)
- `sessionConfig?: SessionConfig` - Session configuration

**UseStreamASRResult:**
```typescript
interface UseStreamASRResult {
  // Connection state
  isConnected: boolean;
  isConnecting: boolean;
  connectionState: ConnectionState;
  sessionId: string | null;

  // Recording state
  isRecording: boolean;
  isRecordingSupported: boolean;

  // Data
  transcript: string;
  transcriptions: TranscriptionData[];
  isSpeaking: boolean;
  lastError: ErrorData | null;
  sessionData: SessionData | null;

  // Actions
  connect: () => Promise<void>;
  disconnect: () => void;
  startRecording: () => Promise<void>;
  stopRecording: () => void;
  commitAudio: () => Promise<void>;
  clearAudioBuffer: () => Promise<void>;
  configureSession: (config: SessionConfig) => Promise<void>;
  clearError: () => void;

  // Client instance
  client: StreamASRClient | null;
}
```

#### `useRecording(client: StreamASRClient | null): UseRecordingResult`

Hook for managing recording state separately.

#### `useTranscription(client: StreamASRClient | null): UseTranscriptionResult`

Hook for managing transcription results separately.

## Audio Utilities

The SDK provides comprehensive audio utilities:

```typescript
import {
  floatTo16BitPCM,
  pcm16ToFloat,
  pcm16ToBase64,
  base64ToPcm16,
  resampleAudio,
  createWAVHeader,
  pcm16ToWAV,
  calculateDuration,
  msToSamples,
  samplesToMs,
  applyWindowFunction
} from '@streamasr/openai-realtime-sdk';
```

### Browser Support

The SDK requires the following browser APIs:
- `WebSocket` - For server communication
- `MediaRecorder` or `getUserMedia` - For audio recording
- `AudioContext` - For audio processing

### Node.js Support

In Node.js environments, you can use the SDK for audio file processing:

```javascript
const { StreamASRClient } = require('@streamasr/openai-realtime-sdk');
const fs = require('fs');

const client = new StreamASRClient({
  apiKey: 'your-api-key',
  url: 'ws://localhost:8080/v1/realtime',
});

await client.connect();
await client.configureSession({ /* session config */ });

// Load and process audio file
const audioBuffer = fs.readFileSync('audio.wav');
const pcmData = extractPCMFromWAV(audioBuffer);

await client.sendAudio(pcmData.buffer);
await client.commitAudio();
```

## Error Handling

The SDK provides comprehensive error handling:

```typescript
client.on('error', (error) => {
  switch (error.type) {
    case 'connection_error':
      console.log('Connection error, attempting reconnection...');
      break;
    case 'authentication_error':
      console.log('Authentication failed, check API Key');
      break;
    case 'recognition_error':
      console.log('Recognition error, please retry');
      break;
    case 'recording_error':
      console.log('Audio recording failed:', error.message);
      break;
    default:
      console.log('Unknown error:', error.message);
  }
});
```

### Error Types

- `connection_error` - WebSocket connection failed
- `authentication_error` - Invalid API key or authentication failed
- `session_error` - Session configuration or management error
- `audio_format_error` - Audio format conversion failed
- `recognition_error` - Speech recognition failed
- `network_error` - Network connectivity issues
- `recording_error` - Audio recording failed
- `rate_limit_error` - Request rate limit exceeded

## Performance Optimization

### Audio Buffer Management

```typescript
// Configure optimal audio parameters
await client.configureSession({
  input_audio_format: {
    type: 'pcm16',
    sample_rate: 16000, // Use lower sample rate to reduce data volume
    channels: 1,
  },
  turn_detection: {
    type: 'server_vad',
    threshold: 0.6, // Higher threshold to reduce false triggers
    silence_duration_ms: 500, // Shorter silence duration
    prefix_padding_ms: 200,
  },
});
```

### Connection Management

```typescript
// Configure reconnection settings
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  autoReconnect: true,
  maxReconnectAttempts: 5,
  reconnectInterval: 3000,
  enableLogging: false, // Disable logging in production
});
```

## Troubleshooting

### Common Issues

1. **Connection Failed**
   - Check network connectivity
   - Verify API key is correct
   - Ensure server URL is accessible
   - Check browser console for WebSocket errors

2. **Audio Recording Failed**
   - Ensure microphone permissions are granted
   - Check if browser supports audio recording
   - Verify HTTPS connection (required for microphone access)
   - Try different audio input devices

3. **No Transcription Results**
   - Check session configuration
   - Verify audio input levels
   - Adjust VAD parameters
   - Check server logs for processing errors

4. **Performance Issues**
   - Reduce audio chunk size
   - Adjust sample rate settings
   - Enable audio compression
   - Monitor network bandwidth

### Debug Mode

Enable debug logging to troubleshoot issues:

```typescript
const client = new StreamASRClient({
  apiKey: 'your-api-key',
  enableLogging: true, // Enable detailed logging
});

// Or use the logger directly
import { Logger, LogLevel } from '@streamasr/openai-realtime-sdk';

const logger = Logger.getInstance();
logger.setLogLevel(LogLevel.DEBUG);
```

## Examples

See the `examples/` directory for complete working examples:

- `basic-usage.html` - Browser example with HTML/JavaScript
- `react-example.tsx` - React component example
- `nodejs-example.js` - Node.js audio processing example

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Documentation**: [https://docs.streamasr.com](https://docs.streamasr.com)
- **Issues**: [GitHub Issues](https://github.com/streamasr/openai-realtime-sdk/issues)
- **Email**: support@streamasr.com

## Changelog

### v1.0.0 (2024-10-26)

- ✅ Initial release
- ✅ Full OpenAI Realtime API support
- ✅ TypeScript type definitions
- ✅ React hooks integration
- ✅ Browser and Node.js support
- ✅ Audio recording and processing utilities
- ✅ Automatic reconnection mechanism
- ✅ Comprehensive error handling
- ✅ Multi-language support
- ✅ Voice activity detection
- ✅ Session management