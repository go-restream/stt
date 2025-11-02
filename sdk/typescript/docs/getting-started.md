# Getting Started Guide

This guide will help you get up and running with the StreamASR TypeScript SDK quickly.

## Prerequisites

- Node.js 14.0.0 or higher
- A StreamASR API key
- For browser usage: a modern browser with WebSocket and microphone support

## Installation

### NPM
```bash
npm install @streamasr/openai-realtime-sdk
```

### Yarn
```bash
yarn add @streamasr/openai-realtime-sdk
```

### CDN
```html
<script src="https://unpkg.com/@streamasr/openai-realtime-sdk@latest/dist/index.js"></script>
```

## Basic Setup

### 1. Import the SDK

```typescript
// ES6 Modules
import { StreamASRClient } from '@streamasr/openai-realtime-sdk';

// CommonJS (Node.js)
const { StreamASRClient } = require('@streamasr/openai-realtime-sdk');

// Browser (global)
const { StreamASRClient } = window.StreamASR;
```

### 2. Create a Client Instance

```typescript
const client = new StreamASRClient({
  apiKey: 'your-api-key-here',
  url: 'ws://localhost:8080/v1/realtime', // Optional, default shown
  enableLogging: true, // Enable debug logging
  autoReconnect: true, // Enable automatic reconnection
});
```

### 3. Set Up Event Listeners

```typescript
// Connection events
client.on('connectionStateChanged', (state) => {
  console.log('Connection state:', state.connected ? 'Connected' : 'Disconnected');
});

client.on('disconnected', () => {
  console.log('Disconnected from server');
});

// Session events
client.on('sessionCreated', (sessionData) => {
  console.log('Session created with ID:', sessionData.id);
});

// Transcription events
client.on('transcription', (data) => {
  console.log('Transcription:', data.text);
});

// Speech detection events
client.on('speechStarted', () => {
  console.log('Speech started');
});

client.on('speechStopped', () => {
  console.log('Speech stopped');
});

// Error events
client.on('error', (error) => {
  console.error('Error:', error.code, error.message);
});
```

### 4. Connect and Configure

```typescript
async function initializeClient() {
  try {
    // Connect to the server
    await client.connect();
    console.log('Connected to StreamASR server');

    // Configure the session
    await client.configureSession({
      modality: 'audio',
      input_audio_format: {
        type: 'pcm16',
        sample_rate: 16000,
        channels: 1,
      },
      input_audio_transcription: {
        model: 'whisper-1',
        language: 'zh', // or 'en', 'ja', 'ko', 'auto'
      },
      turn_detection: {
        type: 'server_vad',
        threshold: 0.5, // 0.0 - 1.0
        prefix_padding_ms: 300,
        silence_duration_ms: 800,
      },
    });

    console.log('Session configured successfully');

  } catch (error) {
    console.error('Failed to initialize client:', error);
  }
}

initializeClient();
```

### 5. Start Recording (Browser)

```typescript
async function startRecording() {
  try {
    // Check if audio recording is supported
    if (!StreamASRClient.isSupported()) {
      throw new Error('Audio recording is not supported in this browser');
    }

    // Start recording from microphone
    await client.startRecording();
    console.log('Recording started');

  } catch (error) {
    console.error('Failed to start recording:', error);
  }
}

// Call this after successful connection
client.on('sessionCreated', () => {
  startRecording();
});
```

### 6. Manual Audio Input (Advanced)

```typescript
// Send audio data manually
const audioData = new Int16Array(1600); // 100ms of 16kHz audio
await client.sendAudio(audioData.buffer);

// Commit audio for processing
await client.commitAudio();
```

### 7. Cleanup

```typescript
function cleanup() {
  if (client.isRecordingActive()) {
    client.stopRecording();
  }

  if (client.isConnected()) {
    client.disconnect();
  }

  console.log('Client cleaned up');
}

// Call cleanup when your application closes
// window.addEventListener('beforeunload', cleanup);
```

## Complete Example

Here's a complete working example for browser usage:

```html
<!DOCTYPE html>
<html>
<head>
    <title>StreamASR Quick Start</title>
    <script src="https://unpkg.com/@streamasr/openai-realtime-sdk@latest/dist/index.js"></script>
</head>
<body>
    <div>
        <h1>StreamASR Quick Start</h1>

        <div>
            <input type="text" id="apiKey" placeholder="Enter API key" />
            <button id="connectBtn">Connect</button>
            <button id="recordBtn" disabled>Start Recording</button>
            <button id="stopBtn" disabled>Stop Recording</button>
            <button id="disconnectBtn" disabled>Disconnect</button>
        </div>

        <div>
            <h3>Status</h3>
            <div id="status">Not connected</div>
            <div id="transcription">Transcription will appear here...</div>
        </div>
    </div>

    <script>
        const { StreamASRClient } = window.StreamASR;

        let client = null;

        const apiKeyInput = document.getElementById('apiKey');
        const connectBtn = document.getElementById('connectBtn');
        const recordBtn = document.getElementById('recordBtn');
        const stopBtn = document.getElementById('stopBtn');
        const disconnectBtn = document.getElementById('disconnectBtn');
        const statusDiv = document.getElementById('status');
        const transcriptionDiv = document.getElementById('transcription');

        function updateStatus(message) {
            statusDiv.textContent = message;
        }

        function addTranscription(text) {
            const timestamp = new Date().toLocaleTimeString();
            transcriptionDiv.textContent = `[${timestamp}] ${text}\n${transcriptionDiv.textContent}`;
        }

        connectBtn.addEventListener('click', async () => {
            const apiKey = apiKeyInput.value.trim();
            if (!apiKey) {
                alert('Please enter an API key');
                return;
            }

            client = new StreamASRClient({
                apiKey: apiKey,
                enableLogging: true,
                autoReconnect: true
            });

            // Event listeners
            client.on('connectionStateChanged', (state) => {
                if (state.connected) {
                    updateStatus('Connected');
                    connectBtn.disabled = true;
                    recordBtn.disabled = false;
                    disconnectBtn.disabled = false;
                } else if (state.connecting) {
                    updateStatus('Connecting...');
                } else {
                    updateStatus('Disconnected');
                    connectBtn.disabled = false;
                    recordBtn.disabled = true;
                    stopBtn.disabled = true;
                    disconnectBtn.disabled = true;
                }
            });

            client.on('sessionCreated', async () => {
                updateStatus('Configuring session...');
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
                    },
                    turn_detection: {
                        type: 'server_vad',
                        threshold: 0.5,
                        prefix_padding_ms: 300,
                        silence_duration_ms: 800
                    }
                });
                updateStatus('Ready to record');
            });

            client.on('transcription', (data) => {
                addTranscription(data.text);
            });

            client.on('recordingStateChanged', (state) => {
                if (state.isRecording) {
                    recordBtn.disabled = true;
                    stopBtn.disabled = false;
                    updateStatus('Recording...');
                } else {
                    recordBtn.disabled = false;
                    stopBtn.disabled = true;
                    updateStatus('Ready to record');
                }
            });

            client.on('error', (error) => {
                updateStatus(`Error: ${error.message}`);
            });

            try {
                await client.connect();
            } catch (error) {
                updateStatus(`Connection failed: ${error.message}`);
            }
        });

        recordBtn.addEventListener('click', async () => {
            try {
                await client.startRecording();
            } catch (error) {
                updateStatus(`Recording failed: ${error.message}`);
            }
        });

        stopBtn.addEventListener('click', () => {
            client.stopRecording();
        });

        disconnectBtn.addEventListener('click', () => {
            client.disconnect();
        });

        // Check browser support
        if (!StreamASRClient.isSupported()) {
            updateStatus('Audio recording is not supported in this browser');
            recordBtn.disabled = true;
        }
    </script>
</body>
</html>
```

## React Quick Start

For React applications, use the provided hooks:

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
    autoReconnect: true,
    enableLogging: true,
  });

  return (
    <div>
      <div>Status: {isConnected ? 'Connected' : 'Disconnected'}</div>
      <div>Recording: {isRecording ? 'Active' : 'Inactive'}</div>
      <div>Transcript: {transcript}</div>
      {lastError && <div>Error: {lastError.message}</div>}

      <button onClick={connect} disabled={isConnected}>
        Connect
      </button>
      <button onClick={startRecording} disabled={!isConnected || isRecording}>
        Start Recording
      </button>
      <button onClick={stopRecording} disabled={!isRecording}>
        Stop Recording
      </button>
      <button onClick={disconnect} disabled={!isConnected}>
        Disconnect
      </button>
    </div>
  );
}

export default VoiceRecorder;
```

## Node.js Quick Start

For server-side audio processing:

```javascript
const { StreamASRClient } = require('@streamasr/openai-realtime-sdk');
const fs = require('fs');

async function processAudioFile(filePath) {
  const client = new StreamASRClient({
    apiKey: 'your-api-key',
    url: 'ws://localhost:8080/v1/realtime',
    enableLogging: true,
  });

  try {
    // Connect and configure
    await client.connect();
    await client.configureSession({
      modality: 'audio',
      input_audio_format: { type: 'pcm16', sample_rate: 16000, channels: 1 },
      input_audio_transcription: { model: 'whisper-1', language: 'zh' },
    });

    // Listen for transcriptions
    client.on('transcription', (data) => {
      console.log('Transcription:', data.text);
    });

    // Load and send audio file
    const audioBuffer = fs.readFileSync(filePath);
    const pcmData = extractPCMFromWAV(audioBuffer);

    await client.sendAudio(pcmData.buffer);
    await client.commitAudio();

    // Wait for results
    await new Promise(resolve => setTimeout(resolve, 5000));

    client.disconnect();
  } catch (error) {
    console.error('Processing failed:', error);
    client.disconnect();
  }
}

processAudioFile('./audio.wav');
```

## Configuration Options

### Session Configuration

- **modality**: `'audio' | 'text' | 'text_and_audio'` - Input modalities
- **language**: Language code (`'zh'`, `'en'`, `'ja'`, `'ko'`, `'auto'`)
- **sample_rate**: `16000 | 48000` - Audio sample rate
- **VAD settings**: Voice Activity Detection parameters

### Client Configuration

- **autoReconnect**: Enable automatic reconnection (default: `true`)
- **enableLogging**: Enable debug logging (default: `false`)
- **heartbeatInterval**: Heartbeat interval in ms (default: `30000`)

## Next Steps

- Explore the [API Reference](./api-reference.md) for detailed documentation
- Check out the [Examples](../examples/) directory for complete implementations
- Read the [Troubleshooting Guide](./troubleshooting.md) for common issues
- Learn about [Advanced Usage](./advanced-usage.md) for custom implementations

## Support

If you encounter any issues:

1. Check the browser console for error messages
2. Verify your API key and server URL
3. Ensure microphone permissions are granted (browser)
4. Review the [Troubleshooting Guide](./troubleshooting.md)

For additional support:
- **GitHub Issues**: [Create an issue](https://github.com/streamasr/openai-realtime-sdk/issues)
- **Email**: support@streamasr.com
- **Documentation**: [https://docs.streamasr.com](https://docs.streamasr.com)