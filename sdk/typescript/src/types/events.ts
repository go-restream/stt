import { SessionConfig } from './config';

// Base Event Types
export interface BaseEvent {
  type: string;
  event_id?: string;
  session_id?: string;
}

// Client to Server Events
export interface SessionUpdateEvent extends BaseEvent {
  type: 'session.update';
  session: SessionConfig;
}

export interface InputAudioBufferAppendEvent extends BaseEvent {
  type: 'input_audio_buffer.append';
  audio: string; // Base64 encoded audio data
}

export interface InputAudioBufferCommitEvent extends BaseEvent {
  type: 'input_audio_buffer.commit';
}

export interface InputAudioBufferClearEvent extends BaseEvent {
  type: 'input_audio_buffer.clear';
}

export interface ConversationItemDeletedEvent extends BaseEvent {
  type: 'conversation.item.deleted';
  item_id: string;
}

export interface HeartbeatPingEvent extends BaseEvent {
  type: 'heartbeat.ping';
  heartbeat_type?: number;
}

// Server to Client Events
export interface SessionCreatedEvent extends BaseEvent {
  type: 'session.created';
  session: {
    id: string;
    object: string;
    model: string;
    modalities: string[];
  };
}

export interface SessionUpdatedEvent extends BaseEvent {
  type: 'session.updated';
  session: {
    id: string;
    object: string;
    model: string;
    modalities: string[];
  };
}

export interface ConversationCreatedEvent extends BaseEvent {
  type: 'conversation.created';
  conversation: {
    id: string;
    object: string;
  };
}

export interface ConversationItemCreatedEvent extends BaseEvent {
  type: 'conversation.item.created';
  item: {
    id: string;
    type: string;
    status: string;
    audio?: {
      data: string;
      format: string;
    };
    content?: any[];
  };
}

export interface ConversationItemInputAudioTranscriptionCompletedEvent extends BaseEvent {
  type: 'conversation.item.input_audio_transcription.completed';
  item: {
    id: string;
    type: string;
    status: string;
    content: Array<{
      type: string;
      transcript: string;
    }>;
  };
}

export interface ConversationItemInputAudioTranscriptionFailedEvent extends BaseEvent {
  type: 'conversation.item.input_audio_transcription.failed';
  item_id: string;
  error: {
    type: string;
    code: string;
    message: string;
    param?: string;
  };
}

export interface InputAudioBufferCommittedEvent extends BaseEvent {
  type: 'input_audio_buffer.committed';
}

export interface InputAudioBufferClearedEvent extends BaseEvent {
  type: 'input_audio_buffer.cleared';
}

export interface InputAudioBufferSpeechStartedEvent extends BaseEvent {
  type: 'input_audio_buffer.speech_started';
  audio_start_ms: number;
  item_id?: string;
}

export interface InputAudioBufferSpeechStoppedEvent extends BaseEvent {
  type: 'input_audio_buffer.speech_stopped';
  audio_end_ms: number;
  item_id?: string;
}

export interface HeartbeatPongEvent extends BaseEvent {
  type: 'heartbeat.pong';
  heartbeat_type?: number;
}

export interface ErrorEvent extends BaseEvent {
  type: 'error';
  error: {
    type: string;
    code: string;
    message: string;
    param?: string;
  };
}

// Union types for events
export type ClientEvent =
  | SessionUpdateEvent
  | InputAudioBufferAppendEvent
  | InputAudioBufferCommitEvent
  | InputAudioBufferClearEvent
  | ConversationItemDeletedEvent
  | HeartbeatPingEvent;

export type ServerEvent =
  | SessionCreatedEvent
  | SessionUpdatedEvent
  | ConversationCreatedEvent
  | ConversationItemCreatedEvent
  | ConversationItemInputAudioTranscriptionCompletedEvent
  | ConversationItemInputAudioTranscriptionFailedEvent
  | InputAudioBufferCommittedEvent
  | InputAudioBufferClearedEvent
  | InputAudioBufferSpeechStartedEvent
  | InputAudioBufferSpeechStoppedEvent
  | HeartbeatPongEvent
  | ErrorEvent;

export type AnyEvent = ClientEvent | ServerEvent;