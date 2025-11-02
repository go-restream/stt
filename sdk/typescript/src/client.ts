import { EventEmitter } from 'eventemitter3';
import {
  ClientEvent,
  AnyEvent,
  SessionCreatedEvent,
  SessionUpdatedEvent,
  SessionUpdateEvent,
  ConversationItemInputAudioTranscriptionCompletedEvent,
  ConversationItemInputAudioTranscriptionFailedEvent,
  InputAudioBufferSpeechStartedEvent,
  InputAudioBufferSpeechStoppedEvent,
  ErrorEvent
} from './types/events';
import {
  ClientOptions,
  SessionConfig,
  ConnectionState,
  TranscriptionData,
  SpeechDetectionData,
  ErrorData,
  SessionData,
  TurnDetection
} from './types/config';
import { Logger, LogLevel } from './utils/logger';
import { pcm16ToBase64, resampleAudio } from './utils/audio';
import { AudioRecorder } from './audio/recorder';

export class StreamASRClient extends EventEmitter {
  private ws: WebSocket | null = null;
  private options: Required<ClientOptions>;
  private logger: Logger;
  private connectionState: ConnectionState;
  private sessionId: string | null = null;
  private audioRecorder: AudioRecorder | null = null;
  private heartbeatInterval: NodeJS.Timeout | null = null;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private eventCounter: number = 0;
  private isRecording: boolean = false;
  private currentSampleRate: number = 16000;
  private currentVADConfig: TurnDetection | null = null;

  constructor(options: ClientOptions) {
    super();

    this.logger = Logger.getInstance();

    if (options.enableLogging) {
      this.logger.setLogLevel(LogLevel.DEBUG);
    }

    this.options = {
      apiKey: options.apiKey,
      url: options.url || 'ws://localhost:8080/v1/realtime',
      autoReconnect: options.autoReconnect !== false,
      reconnectInterval: options.reconnectInterval || 3000,
      maxReconnectAttempts: options.maxReconnectAttempts || 5,
      enableLogging: options.enableLogging || false,
      heartbeatInterval: options.heartbeatInterval || 30000,
      sessionTimeout: options.sessionTimeout || 1800000, // 30 minutes
    };

    this.connectionState = {
      connected: false,
      connecting: false,
      reconnecting: false,
      reconnectAttempts: 0,
    };

    this.logger.info('StreamASRClient initialized', {
      url: this.options.url,
      autoReconnect: this.options.autoReconnect
    });
  }

  /**
   * Check if browser supports WebSocket and audio recording
   */
  static isSupported(): boolean {
    return !!(typeof WebSocket !== 'undefined' &&
              navigator &&
              navigator.mediaDevices &&
              navigator.mediaDevices.getUserMedia);
  }

  /**
   * Connect to the WebSocket server
   */
  async connect(): Promise<void> {
    if (this.connectionState.connected || this.connectionState.connecting) {
      this.logger.warn('Client is already connected or connecting');
      return;
    }

    this.connectionState.connecting = true;
    this.emit('connectionStateChanged', this.connectionState);

    try {
      this.logger.info(`Connecting to ${this.options.url}`);

      // Create WebSocket connection
      this.ws = new WebSocket(this.options.url);

      // Set up event handlers
      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
      this.ws.onerror = this.handleError.bind(this);
      this.ws.onclose = this.handleClose.bind(this);

      // Wait for connection to open
      await new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          reject(new Error('Connection timeout'));
        }, 10000);

        this.ws!.addEventListener('open', () => {
          clearTimeout(timeout);
          resolve();
        }, { once: true });

        this.ws!.addEventListener('error', () => {
          clearTimeout(timeout);
          reject(new Error('Connection failed'));
        }, { once: true });
      });

    } catch (error) {
      this.connectionState.connecting = false;
      this.connectionState.error = error as Error;
      this.emit('connectionStateChanged', this.connectionState);

      this.logger.error('Connection failed', error);

      if (this.options.autoReconnect) {
        this.scheduleReconnect();
      }

      throw error;
    }
  }

  /**
   * Disconnect from the server
   */
  disconnect(): void {
    this.logger.info('Disconnecting from server');

    this.connectionState.reconnecting = false;
    this.connectionState.connecting = false;

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    if (this.audioRecorder) {
      this.audioRecorder.dispose();
      this.audioRecorder = null;
    }

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }

    this.connectionState.connected = false;
    this.connectionState.reconnectAttempts = 0;
    this.emit('connectionStateChanged', this.connectionState);
  }

  /**
   * Configure session parameters
   */
  async configureSession(config: SessionConfig): Promise<void> {
    this.ensureConnected();

    this.logger.info('Configuring session', { config });

    // Log VAD configuration details
    const vadConfig = config.turn_detection || {
      type: 'server_vad',
      threshold: 0.5,
      prefix_padding_ms: 300,
      silence_duration_ms: 2000
    };

    this.logger.info('VAD configuration being applied', {
      type: vadConfig.type,
      threshold: vadConfig.threshold,
      prefix_padding_ms: vadConfig.prefix_padding_ms,
      silence_duration_ms: vadConfig.silence_duration_ms
    });

    const event: SessionUpdateEvent = {
      type: 'session.update',
      event_id: this.generateEventId(),
      session: {
        ...config,
        // Set default audio format if not provided
        input_audio_format: config.input_audio_format || {
          type: 'pcm16',
          sample_rate: 16000,
          channels: 1
        },
        output_audio_format: config.output_audio_format || {
          type: 'pcm16',
          sample_rate: 16000,
          channels: 1
        },
        // Set default VAD configuration if not provided
        turn_detection: vadConfig
      }
    };

    // Update current sample rate
    if (config.input_audio_format) {
      this.currentSampleRate = config.input_audio_format.sample_rate;
    }

    // Store current VAD configuration
    if (config.turn_detection) {
      this.currentVADConfig = config.turn_detection;
    } else if (!this.currentVADConfig) {
      // Store default VAD config if none was provided
      this.currentVADConfig = {
        type: 'server_vad',
        threshold: 0.5,
        prefix_padding_ms: 300,
        silence_duration_ms: 2000
      };
    }

    this.sendEvent(event);

    this.logger.info('Session update event sent with VAD configuration', {
      eventId: event.event_id,
      vadConfigured: vadConfig.type === 'server_vad',
      threshold: vadConfig.threshold,
      silenceDuration: vadConfig.silence_duration_ms
    });
  }

  /**
   * Start recording audio
   */
  async startRecording(): Promise<void> {
    this.ensureConnected();

    if (this.isRecording) {
      this.logger.warn('Recording is already active');
      return;
    }

    if (!AudioRecorder.isSupported()) {
      throw new Error('Audio recording is not supported in this browser');
    }

    // Auto-configure VAD if not already configured
    if (!this.isVADEnabled()) {
      this.logger.info('Auto-configuring VAD for recording');
      const vadConfig = {
        type: 'server_vad' as const,
        threshold: 0.5,
        prefix_padding_ms: 300,
        silence_duration_ms: 2000
      };

      this.logger.info('Applying VAD configuration before recording', {
        type: vadConfig.type,
        threshold: vadConfig.threshold,
        prefix_padding_ms: vadConfig.prefix_padding_ms,
        silence_duration_ms: vadConfig.silence_duration_ms
      });

      await this.configureVAD(vadConfig);

      this.logger.info('VAD configuration applied successfully', {
        configured: this.isVADConfigured(),
        currentConfig: this.currentVADConfig
      });
    } else {
      this.logger.info('VAD already configured', {
        config: this.currentVADConfig,
        isConfigured: this.isVADConfigured()
      });
    }

    this.logger.info('Starting audio recording');

    try {
      // Create audio recorder
      this.audioRecorder = new AudioRecorder({
        sampleRate: this.currentSampleRate === 16000 ? 16000 : 48000,
        channelCount: 1,
        bufferSize: 4096,
      });

      // Request permission first
      const hasPermission = await this.audioRecorder.requestPermission();
      if (!hasPermission) {
        throw new Error('Microphone permission denied');
      }

      // Start recording
      await this.audioRecorder.start(
        // onData callback
        (audioData: Int16Array) => {
          this.handleAudioData(audioData);
        },
        // onError callback
        (error: Error) => {
          this.logger.error('Audio recording error', error);
          this.emit('error', {
            type: 'recording_error',
            code: 'audio_capture_failed',
            message: error.message,
          });
        },
        // onStart callback
        () => {
          this.isRecording = true;
          this.logger.info('Audio recording started');
          this.emit('recordingStateChanged', { isRecording: true });
        },
        // onStop callback
        () => {
          this.isRecording = false;
          this.logger.info('Audio recording stopped');
          this.emit('recordingStateChanged', { isRecording: false });
        }
      );

    } catch (error) {
      this.logger.error('Failed to start audio recording', error);
      throw error;
    }
  }

  /**
   * Stop recording audio
   */
  stopRecording(): void {
    if (!this.isRecording || !this.audioRecorder) {
      this.logger.warn('Recording is not active');
      return;
    }

    this.logger.info('Stopping audio recording');
    this.audioRecorder.stop();
    this.audioRecorder.dispose();
    this.audioRecorder = null;
    this.isRecording = false;

    this.emit('recordingStateChanged', { isRecording: false });
  }

  /**
   * Send audio data manually
   */
  async sendAudio(audioData: ArrayBuffer | Int16Array): Promise<void> {
    this.ensureConnected();

    let pcmData: Int16Array;

    if (audioData instanceof ArrayBuffer) {
      pcmData = new Int16Array(audioData);
    } else {
      pcmData = audioData;
    }

    // Resample if needed
    if (this.currentSampleRate !== 16000) {
      pcmData = resampleAudio(pcmData, this.currentSampleRate, 16000);
    }

    this.handleAudioData(pcmData);
  }

  /**
   * Commit audio buffer for processing
   */
  async commitAudio(): Promise<void> {
    this.ensureConnected();

    this.logger.debug('Committing audio buffer');

    const event = {
      type: 'input_audio_buffer.commit' as const,
      event_id: this.generateEventId(),
    };

    this.sendEvent(event);
  }

  /**
   * Clear audio buffer
   */
  async clearAudioBuffer(): Promise<void> {
    this.ensureConnected();

    this.logger.debug('Clearing audio buffer');

    const event = {
      type: 'input_audio_buffer.clear' as const,
      event_id: this.generateEventId(),
    };

    this.sendEvent(event);
  }

  /**
   * Get current connection state
   */
  getConnectionState(): ConnectionState {
    return { ...this.connectionState };
  }

  /**
   * Get current session ID
   */
  getSessionId(): string | null {
    return this.sessionId;
  }

  /**
   * Check if client is connected
   */
  isConnected(): boolean {
    return this.connectionState.connected;
  }

  /**
   * Check if recording is active
   */
  isRecordingActive(): boolean {
    return this.isRecording;
  }

  // Private methods

  private ensureConnected(): void {
    if (!this.connectionState.connected) {
      throw new Error('Client is not connected');
    }
  }

  private generateEventId(): string {
    return `event_${Date.now()}_${++this.eventCounter}`;
  }

  private sendEvent(event: ClientEvent): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket is not connected');
    }

    const message = JSON.stringify(event);
    this.ws.send(message);

    this.logger.debug('Sent event', { type: event.type, eventId: event.event_id });
  }

  private handleOpen(): void {
    this.logger.info('WebSocket connection opened');

    this.connectionState.connected = true;
    this.connectionState.connecting = false;
    this.connectionState.reconnecting = false;
    this.connectionState.error = undefined;
    this.connectionState.reconnectAttempts = 0;

    this.emit('connectionStateChanged', this.connectionState);

    // Start heartbeat
    this.startHeartbeat();
  }

  private handleMessage(event: MessageEvent): void {
    try {
      const message: AnyEvent = JSON.parse(event.data);
      this.logger.debug('Received event', { type: message.type });

      switch (message.type) {
        case 'session.created':
          this.handleSessionCreated(message as SessionCreatedEvent);
          break;

        case 'session.updated':
          this.handleSessionUpdated(message as SessionUpdatedEvent);
          break;

        case 'conversation.item.input_audio_transcription.completed':
          this.handleTranscriptionCompleted(message as ConversationItemInputAudioTranscriptionCompletedEvent);
          break;

        case 'conversation.item.input_audio_transcription.failed':
          this.handleTranscriptionFailed(message as ConversationItemInputAudioTranscriptionFailedEvent);
          break;

        case 'input_audio_buffer.speech_started':
          this.handleSpeechStarted(message as InputAudioBufferSpeechStartedEvent);
          break;

        case 'input_audio_buffer.speech_stopped':
          this.handleSpeechStopped(message as InputAudioBufferSpeechStoppedEvent);
          break;

        case 'error':
          this.handleErrorEvent(message as ErrorEvent);
          break;

        case 'heartbeat.pong':
          this.handleHeartbeatPong();
          break;

        default:
          this.logger.debug('Unhandled event type', { type: message.type });
          this.emit('event', message);
      }
    } catch (error) {
      this.logger.error('Failed to parse message', { error, data: event.data });
    }
  }

  private handleError(error: Event): void {
    this.logger.error('WebSocket error', error);

    this.connectionState.error = new Error('WebSocket connection error');
    this.emit('connectionStateChanged', this.connectionState);

    const errorData: ErrorData = {
      type: 'connection_error',
      code: 'websocket_error',
      message: 'WebSocket connection error',
    };

    this.emit('error', errorData);
  }

  private handleClose(event: CloseEvent): void {
    this.logger.info('WebSocket connection closed', { code: event.code, reason: event.reason });

    this.connectionState.connected = false;
    this.connectionState.connecting = false;

    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }

    if (this.audioRecorder) {
      this.audioRecorder.dispose();
      this.audioRecorder = null;
      this.isRecording = false;
    }

    this.emit('connectionStateChanged', this.connectionState);
    this.emit('disconnected', { code: event.code, reason: event.reason });

    // Auto-reconnect if enabled and not a normal closure
    if (this.options.autoReconnect && event.code !== 1000) {
      this.scheduleReconnect();
    }
  }

  private handleSessionCreated(event: SessionCreatedEvent): void {
    this.sessionId = event.session.id;

    const sessionData: SessionData = {
      id: event.session.id,
      model: event.session.model,
      modalities: event.session.modalities,
      createdAt: Date.now(),
      lastActivity: Date.now(),
    };

    this.logger.info('Session created', { sessionId: this.sessionId });
    this.emit('sessionCreated', sessionData);
  }

  private handleSessionUpdated(event: SessionUpdatedEvent): void {
    this.logger.info('Session updated', { sessionId: event.session.id });
    this.emit('sessionUpdated', {
      id: event.session.id,
      model: event.session.model,
      modalities: event.session.modalities,
    });
  }

  private handleTranscriptionCompleted(event: ConversationItemInputAudioTranscriptionCompletedEvent): void {
    if (event.item.content && event.item.content.length > 0) {
      const transcriptionData: TranscriptionData = {
        text: event.item.content[0].transcript,
        timestamp: Date.now(),
        itemId: event.item.id,
      };

      this.logger.info('Transcription completed', { text: transcriptionData.text });
      this.emit('transcription', transcriptionData);
    }
  }

  private handleTranscriptionFailed(event: ConversationItemInputAudioTranscriptionFailedEvent): void {
    const errorData: ErrorData = {
      type: event.error.type,
      code: event.error.code,
      message: event.error.message,
      eventId: event.event_id,
    };

    this.logger.error('Transcription failed', errorData);
    this.emit('error', errorData);
  }

  private handleSpeechStarted(event: InputAudioBufferSpeechStartedEvent): void {
    const speechData: SpeechDetectionData = {
      started: true,
      timestamp: Date.now(),
      audioStartMs: event.audio_start_ms,
    };

    this.logger.info('Speech started event received', {
      audioStartMs: event.audio_start_ms,
      timestamp: speechData.timestamp,
      vadConfig: this.currentVADConfig
    });
    this.emit('speechStarted', speechData);
  }

  private handleSpeechStopped(event: InputAudioBufferSpeechStoppedEvent): void {
    const speechData: SpeechDetectionData = {
      started: false,
      timestamp: Date.now(),
      audioEndMs: event.audio_end_ms,
    };

    this.logger.info('Speech stopped event received', {
      audioEndMs: event.audio_end_ms,
      timestamp: speechData.timestamp,
      vadConfig: this.currentVADConfig
    });
    this.emit('speechStopped', speechData);

    // Auto-commit on speech stop
    this.logger.info('Auto-committing audio buffer after speech stop');
    this.commitAudio();
  }

  private handleErrorEvent(event: ErrorEvent): void {
    const errorData: ErrorData = {
      type: event.error.type,
      code: event.error.code,
      message: event.error.message,
      eventId: event.event_id,
    };

    this.logger.error('Server error', errorData);
    this.emit('error', errorData);
  }

  private handleHeartbeatPong(): void {
    this.logger.debug('Heartbeat pong received');
    this.emit('pong');
  }

  private handleAudioData(audioData: Int16Array): void {
    // Convert to Base64 and send
    const base64Audio = pcm16ToBase64(audioData);

    const event = {
      type: 'input_audio_buffer.append' as const,
      event_id: this.generateEventId(),
      audio: base64Audio,
    };

    this.sendEvent(event);
  }

  private startHeartbeat(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
    }

    this.heartbeatInterval = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        const event = {
          type: 'heartbeat.ping' as const,
          event_id: this.generateEventId(),
          heartbeat_type: 1,
        };

        this.sendEvent(event);
        this.logger.debug('Heartbeat ping sent');
      }
    }, this.options.heartbeatInterval);
  }

  private scheduleReconnect(): void {
    if (this.connectionState.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.logger.error('Max reconnection attempts reached');
      this.emit('maxReconnectAttemptsReached');
      return;
    }

    const delay = this.options.reconnectInterval * Math.pow(2, this.connectionState.reconnectAttempts);

    this.logger.info(`Scheduling reconnection in ${delay}ms (attempt ${this.connectionState.reconnectAttempts + 1})`);

    this.connectionState.reconnecting = true;
    this.connectionState.reconnectAttempts++;
    this.emit('connectionStateChanged', this.connectionState);

    this.reconnectTimeout = setTimeout(async () => {
      try {
        await this.connect();
        this.logger.info('Reconnection successful');
      } catch (error) {
        this.logger.error('Reconnection failed', error);
        this.scheduleReconnect();
      }
    }, delay);
  }

  /**
   * Configure VAD (Voice Activity Detection) settings
   */
  async configureVAD(config: Partial<TurnDetection>): Promise<void> {
    this.ensureConnected();

    // Validate VAD configuration
    this.validateVADConfig(config);

    // Create complete VAD configuration
    const vadConfig: TurnDetection = {
      type: 'server_vad',
      threshold: config.threshold ?? 0.5,
      prefix_padding_ms: config.prefix_padding_ms ?? 300,
      silence_duration_ms: config.silence_duration_ms ?? 2000,
      ...config
    };

    this.logger.info('Configuring VAD', { vadConfig });

    // Update session with new VAD configuration
    await this.configureSession({
      modality: 'audio', // Default modality for VAD
      turn_detection: vadConfig
    });
  }

  /**
   * Get current VAD configuration
   */
  getVADConfig(): TurnDetection | null {
    return this.currentVADConfig ? { ...this.currentVADConfig } : null;
  }

  /**
   * Check if VAD is enabled
   */
  isVADEnabled(): boolean {
    return this.currentVADConfig !== null;
  }

  /**
   * Check if VAD is properly configured on the server
   */
  isVADConfigured(): boolean {
    return this.currentVADConfig !== null &&
           this.currentVADConfig.type === 'server_vad' &&
           typeof this.currentVADConfig.threshold === 'number' &&
           typeof this.currentVADConfig.prefix_padding_ms === 'number' &&
           typeof this.currentVADConfig.silence_duration_ms === 'number';
  }

  /**
   * Validate VAD configuration parameters
   */
  private validateVADConfig(config: Partial<TurnDetection>): void {
    if (config.threshold !== undefined) {
      if (typeof config.threshold !== 'number' || config.threshold < 0 || config.threshold > 1) {
        throw new Error('VAD threshold must be a number between 0.0 and 1.0');
      }
    }

    if (config.prefix_padding_ms !== undefined) {
      if (typeof config.prefix_padding_ms !== 'number' || config.prefix_padding_ms < 0 || config.prefix_padding_ms > 3000) {
        throw new Error('VAD prefix_padding_ms must be a number between 0 and 3000ms');
      }
    }

    if (config.silence_duration_ms !== undefined) {
      if (typeof config.silence_duration_ms !== 'number' || config.silence_duration_ms < 100 || config.silence_duration_ms > 10000) {
        throw new Error('VAD silence_duration_ms must be a number between 100 and 10000ms');
      }
    }

    if (config.type !== undefined && config.type !== 'server_vad') {
      throw new Error('VAD type must be "server_vad"');
    }
  }
}