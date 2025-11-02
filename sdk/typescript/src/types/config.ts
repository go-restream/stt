export interface AudioFormat {
  type: 'pcm16';
  sample_rate: 16000 | 48000;
  channels: 1;
}

export interface InputAudioTranscription {
  model: string;
  language: string;
  prompt?: string;
}

export interface TurnDetection {
  type: 'server_vad';
  threshold: number; // 0.0 - 1.0
  prefix_padding_ms: number;
  silence_duration_ms: number;
}

export interface SessionConfig {
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

export interface ClientOptions {
  apiKey: string;
  url?: string;
  autoReconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  enableLogging?: boolean;
  heartbeatInterval?: number;
  sessionTimeout?: number;
}

export interface AudioRecorderOptions {
  sampleRate?: 16000 | 48000;
  channelCount?: number;
  bufferSize?: number;
  echoCancellation?: boolean;
  noiseSuppression?: boolean;
  autoGainControl?: boolean;
}

export interface ConnectionState {
  connected: boolean;
  connecting: boolean;
  reconnecting: boolean;
  error?: Error;
  reconnectAttempts: number;
}

export interface TranscriptionData {
  text: string;
  language?: string;
  confidence?: number;
  timestamp: number;
  itemId?: string;
}

export interface SpeechDetectionData {
  started: boolean;
  timestamp: number;
  audioStartMs?: number;
  audioEndMs?: number;
}

export interface ErrorData {
  type: string;
  code: string;
  message: string;
  details?: any;
  eventId?: string;
}

export interface SessionData {
  id: string;
  model: string;
  modalities: string[];
  createdAt: number;
  lastActivity: number;
}

// Event listener types
export type EventListener<T = any> = (data: T) => void;
export type TranscriptionListener = EventListener<TranscriptionData>;
export type SpeechDetectionListener = EventListener<SpeechDetectionData>;
export type ErrorListener = EventListener<ErrorData>;
export type SessionListener = EventListener<SessionData>;
export type ConnectionListener = EventListener<ConnectionState>;