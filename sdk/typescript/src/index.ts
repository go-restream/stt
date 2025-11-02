// Main exports
export { StreamASRClient } from './client';
export { AudioRecorder } from './audio/recorder';

// Type exports
export type {
  ClientEvent,
  ServerEvent,
  AnyEvent,
  SessionCreatedEvent,
  SessionUpdatedEvent,
  ConversationItemInputAudioTranscriptionCompletedEvent,
  InputAudioBufferSpeechStartedEvent,
  InputAudioBufferSpeechStoppedEvent,
  ErrorEvent
} from './types/events';

export type {
  ClientOptions,
  SessionConfig,
  AudioFormat,
  InputAudioTranscription,
  TurnDetection,
  AudioRecorderOptions,
  ConnectionState,
  TranscriptionData,
  SpeechDetectionData,
  ErrorData,
  SessionData,
  EventListener,
  TranscriptionListener,
  SpeechDetectionListener,
  ErrorListener,
  SessionListener,
  ConnectionListener
} from './types/config';

// Utility exports
export {
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
} from './utils/audio';

export { Logger, LogLevel } from './utils/logger';

// Re-export EventEmitter for convenience
export { EventEmitter } from 'eventemitter3';