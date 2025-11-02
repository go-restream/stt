# Troubleshooting Guide

This guide covers common issues and solutions when using the StreamASR TypeScript SDK.

## Connection Issues

### WebSocket Connection Failed

**Symptoms:**
- Error: `Connection failed`
- No response from server
- `ERR_CONNECTION_REFUSED` in browser console

**Solutions:**
1. **Check server URL**:
   ```typescript
   const client = new StreamASRClient({
     url: 'ws://localhost:8080/v1/realtime', // Verify this is correct
   });
   ```

2. **Verify server is running**:
   ```bash
   # Check if your StreamASR server is running
   curl -I http://localhost:8080/v1/realtime
   ```

3. **Check network connectivity**:
   - Ensure no firewall is blocking WebSocket connections
   - Verify the server is accessible from your network

4. **Test with WebSocket testing tools**:
   - Use WebSocket client tools to test the connection directly
   - Example: [WebSocket King](https://websocketking.com/)

### Authentication Failed

**Symptoms:**
- Error: `Authentication failed`
- `401 Unauthorized` response

**Solutions:**
1. **Verify API key**:
   ```typescript
   const client = new StreamASRClient({
     apiKey: 'your-actual-api-key', // Ensure this is correct
   });
   ```

2. **Check API key format**:
   - API keys should be strings without extra whitespace
   - Ensure you're using the correct API key for your server

3. **Test API key**:
   ```bash
   # Test your API key with curl
   curl -H "Authorization: Bearer your-api-key" ws://localhost:8080/v1/realtime
   ```

## Audio Recording Issues

### Microphone Permission Denied

**Symptoms:**
- Error: `Microphone permission denied`
- Recording fails to start
- Browser shows permission denied dialog

**Solutions:**
1. **Request permission properly**:
   ```typescript
   // Check if recording is supported
   if (!StreamASRClient.isSupported()) {
     console.error('Audio recording not supported');
     return;
   }

   // The SDK will automatically request permission
   try {
     await client.startRecording();
   } catch (error) {
     console.error('Permission denied:', error);
   }
   ```

2. **HTTPS requirement**:
   - Modern browsers require HTTPS for microphone access
   - Use `localhost` for development (automatically secure)
   - Use SSL certificates for production

3. **Manual permission request**:
   ```typescript
   // Request permission before starting recording
   const hasPermission = await navigator.mediaDevices.getUserMedia({ audio: true });
   ```

### Audio Quality Issues

**Symptoms:**
- Poor transcription accuracy
- Choppy audio
- Background noise

**Solutions:**
1. **Adjust audio settings**:
   ```typescript
   await client.configureSession({
     input_audio_format: {
       type: 'pcm16',
       sample_rate: 16000, // Try 48000 for better quality
       channels: 1,
     },
     turn_detection: {
       type: 'server_vad',
       threshold: 0.3, // Lower threshold for more sensitive detection
       silence_duration_ms: 1000, // Longer silence duration
     },
   });
   ```

2. **Check microphone hardware**:
   - Test with different microphones
   - Check microphone volume settings
   - Ensure no background noise

3. **Audio format optimization**:
   ```typescript
   // Use higher sample rate for better quality
   await client.configureSession({
     input_audio_format: {
       type: 'pcm16',
       sample_rate: 48000, // Higher quality
       channels: 1,
     },
   });
   ```

## Browser Compatibility

### Unsupported Browser

**Symptoms:**
- `Audio recording is not supported in this browser`
- `getUserMedia is not defined`

**Solutions:**
1. **Check browser compatibility**:
   ```typescript
   if (!StreamASRClient.isSupported()) {
     alert('Your browser does not support audio recording. Please use a modern browser.');
   }
   ```

2. **Recommended browsers**:
   - Chrome 60+
   - Firefox 55+
   - Safari 11+
   - Edge 79+

3. **Polyfills** (for older browsers):
   ```html
   <!-- WebSocket polyfill for older browsers -->
   <script src="https://cdn.jsdelivr.net/npm/websocket-polyfill@latest"></script>
   ```

### Mobile Browser Issues

**Symptoms:**
- Audio recording fails on mobile
- Poor performance on mobile devices

**Solutions:**
1. **Mobile-specific configuration**:
   ```typescript
   // Lower sample rate for mobile
   await client.configureSession({
     input_audio_format: {
       type: 'pcm16',
       sample_rate: 16000, // Mobile-friendly
       channels: 1,
     },
   });
   ```

2. **Touch events for mobile**:
   ```html
   <button ontouchstart="startRecording()" ontouchend="stopRecording()">
     Record
   </button>
   ```

3. **iOS Safari specific**:
   - iOS requires user interaction to start audio
   - Use touch events instead of click events on iOS

## Performance Issues

### High Latency

**Symptoms:**
- Delay between speaking and transcription
- Slow response times

**Solutions:**
1. **Optimize audio chunk size**:
   ```typescript
   // The SDK automatically handles optimal chunking
   // But you can manually send smaller chunks if needed
   const chunkSize = 1024; // Smaller chunks for lower latency
   ```

2. **Reduce sample rate**:
   ```typescript
   await client.configureSession({
     input_audio_format: {
       type: 'pcm16',
       sample_rate: 16000, // Lower than 48000 for better performance
       channels: 1,
     },
   });
   ```

3. **Adjust VAD settings**:
   ```typescript
   await client.configureSession({
     turn_detection: {
       type: 'server_vad',
       threshold: 0.7, // Higher threshold for faster response
       silence_duration_ms: 300, // Shorter silence duration
     },
   });
   ```

### Memory Usage

**Symptoms:**
- High memory usage
- Browser crashes after extended use
- Slow performance over time

**Solutions:**
1. **Clear transcriptions periodically**:
   ```typescript
   let transcriptions = [];

   client.on('transcription', (data) => {
     transcriptions.push(data);

     // Keep only last 100 transcriptions
     if (transcriptions.length > 100) {
       transcriptions = transcriptions.slice(-100);
     }
   });
   ```

2. **Dispose client properly**:
   ```typescript
   function cleanup() {
     client.stopRecording();
     client.disconnect();
     // Remove all event listeners
     client.removeAllListeners();
   }

   // Call cleanup when component unmounts or page unloads
   window.addEventListener('beforeunload', cleanup);
   ```

3. **Monitor memory usage**:
   ```typescript
   // Enable logging to monitor performance
   const client = new StreamASRClient({
     enableLogging: true,
   });
   ```

## Error Handling

### Network Errors

**Symptoms:**
- Random disconnections
- `network_error` events
- Connection drops

**Solutions:**
1. **Enable automatic reconnection**:
   ```typescript
   const client = new StreamASRClient({
     autoReconnect: true,
     maxReconnectAttempts: 10, // Increase attempts
     reconnectInterval: 5000, // Increase interval
   });
   ```

2. **Handle reconnection events**:
   ```typescript
   client.on('maxReconnectAttemptsReached', () => {
     console.error('Could not reconnect to server');
     // Show user-friendly message
   });

   client.on('reconnecting', (attempt) => {
     console.log(`Reconnection attempt ${attempt}`);
     // Show reconnection status
   });
   ```

3. **Network status monitoring**:
   ```typescript
   window.addEventListener('online', () => {
     console.log('Network restored');
     if (!client.isConnected()) {
       client.connect();
     }
   });

   window.addEventListener('offline', () => {
     console.log('Network lost');
   });
   ```

### Server Errors

**Symptoms:**
- `server_error` events
- Unexpected behavior
- Transcription failures

**Solutions:**
1. **Enable debug logging**:
   ```typescript
   const client = new StreamASRClient({
     enableLogging: true, // Shows detailed logs
   });

   // Or use logger directly
   import { Logger, LogLevel } from '@streamasr/openai-realtime-sdk';
   const logger = Logger.getInstance();
   logger.setLogLevel(LogLevel.DEBUG);
   ```

2. **Monitor server logs**:
   - Check StreamASR server logs for errors
   - Verify server configuration
   - Monitor server performance

3. **Error recovery**:
   ```typescript
   client.on('error', (error) => {
     switch (error.code) {
       case 'session_expired':
         // Reconnect with new session
         client.disconnect();
         setTimeout(() => client.connect(), 1000);
         break;

       case 'rate_limit_exceeded':
         // Wait and retry
         setTimeout(() => {
           // Retry the failed operation
         }, 5000);
         break;

       default:
         console.error('Unhandled error:', error);
     }
   });
   ```

## Debug Mode

### Enable Comprehensive Logging

```typescript
import { Logger, LogLevel } from '@streamasr/openai-realtime-sdk';

// Enable debug logging
const logger = Logger.getInstance();
logger.setLogLevel(LogLevel.DEBUG);

// Get recent logs for debugging
const recentLogs = logger.getRecentLogs(100);
console.log('Recent logs:', recentLogs);

// Get error logs only
const errorLogs = logger.getLogsByLevel(LogLevel.ERROR);
console.log('Error logs:', errorLogs);
```

### Browser DevTools

1. **Console Logging**:
   - Open browser DevTools (F12)
   - Check Console tab for SDK logs
   - Filter by `StreamASR` to see SDK-specific logs

2. **Network Tab**:
   - Monitor WebSocket connections
   - Check connection status and messages
   - Verify headers and authentication

3. **Performance Tab**:
   - Monitor memory usage
   - Check CPU usage during recording
   - Identify performance bottlenecks

### Server-Side Debugging

```javascript
// Add debugging to your StreamASR server
const client = new StreamASRClient({
  enableLogging: true,
  url: 'ws://localhost:8080/v1/realtime?debug=true', // If supported
});

// Monitor all events
client.on('event', (event) => {
  console.log('Server event:', event);
});
```

## Common Error Codes

| Error Code | Description | Solution |
|------------|-------------|----------|
| `connection_error` | WebSocket connection failed | Check network, server URL, and firewall |
| `authentication_error` | Invalid API key | Verify API key is correct and active |
| `session_error` | Session configuration invalid | Check session configuration parameters |
| `audio_format_error` | Audio format not supported | Use PCM16 format with supported sample rates |
| `recognition_error` | Speech recognition failed | Check audio quality and server status |
| `network_error` | Network connectivity issue | Check internet connection and server status |
| `recording_error` | Audio recording failed | Check microphone permissions and hardware |
| `rate_limit_error` | Too many requests | Reduce request frequency |
| `session_expired` | Session timeout | Reconnect with new session |

## Getting Help

If you're still experiencing issues:

1. **Check the logs**:
   ```typescript
   // Enable logging and check console output
   const client = new StreamASRClient({ enableLogging: true });
   ```

2. **Create a minimal reproduction**:
   - Simplify your code to isolate the issue
   - Test with different configurations
   - Try different browsers or environments

3. **Check for known issues**:
   - [GitHub Issues](https://github.com/streamasr/openai-realtime-sdk/issues)
   - [Documentation](https://docs.streamasr.com)

4. **Contact support**:
   - Email: support@streamasr.com
   - Include SDK version, browser info, and error logs
   - Provide steps to reproduce the issue

## Version Compatibility

| SDK Version | StreamASR Server | Browser Support | Node.js Support |
|-------------|------------------|-----------------|-----------------|
| 1.0.0 | v1.0.0+ | Chrome 60+, Firefox 55+, Safari 11+, Edge 79+ | 14.0.0+ |

## Performance Tips

1. **Use appropriate sample rates**:
   - 16kHz for most use cases (better performance)
   - 48kHz for high-quality requirements (higher bandwidth)

2. **Optimize VAD settings**:
   ```typescript
   // Balanced settings
   turn_detection: {
     type: 'server_vad',
     threshold: 0.5,
     silence_duration_ms: 800,
   }
   ```

3. **Manage transcription history**:
   ```typescript
   // Keep transcription history manageable
   const MAX_TRANSCRIPTIONS = 50;

   client.on('transcription', (data) => {
     if (transcriptions.length >= MAX_TRANSCRIPTIONS) {
       transcriptions.shift(); // Remove oldest
     }
     transcriptions.push(data);
   });
   ```

4. **Use React hooks for React applications**:
   - Built-in state management
   - Automatic cleanup
   - Optimized re-rendering