import { useState, useEffect, useRef, useCallback } from 'react';
import { StreamASRClient } from '../client';
import {
  ClientOptions,
  SessionConfig,
  ConnectionState,
  TranscriptionData,
  ErrorData,
  SessionData
} from '../types/config';

/**
 * React hook for StreamASR client
 */
export interface UseStreamASROptions extends ClientOptions {
  autoConnect?: boolean;
  autoStartRecording?: boolean;
  sessionConfig?: SessionConfig;
}

export interface UseStreamASRResult {
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

  // Session data
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

  // Client instance (for advanced usage)
  client: StreamASRClient | null;
}

export function useStreamASR(options: UseStreamASROptions): UseStreamASRResult {
  const clientRef = useRef<StreamASRClient | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectionState, setConnectionState] = useState<ConnectionState>({
    connected: false,
    connecting: false,
    reconnecting: false,
    reconnectAttempts: 0,
  });
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [isRecording, setIsRecording] = useState(false);
  const [isRecordingSupported, setIsRecordingSupported] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [transcriptions, setTranscriptions] = useState<TranscriptionData[]>([]);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [lastError, setLastError] = useState<ErrorData | null>(null);
  const [sessionData, setSessionData] = useState<SessionData | null>(null);

  // Initialize client
  useEffect(() => {
    const client = new StreamASRClient({
      apiKey: options.apiKey,
      url: options.url,
      autoReconnect: options.autoReconnect,
      enableLogging: options.enableLogging,
    });

    clientRef.current = client;

    // Check if audio recording is supported
    setIsRecordingSupported(StreamASRClient.isSupported());

    // Set up event listeners
    const handleConnectionStateChanged = (state: ConnectionState) => {
      setConnectionState(state);
      setIsConnected(state.connected);
      setIsConnecting(state.connecting);
    };

    const handleSessionCreated = (data: SessionData) => {
      setSessionId(data.id);
      setSessionData(data);
    };

    const handleTranscription = (data: TranscriptionData) => {
      setTranscript(data.text);
      setTranscriptions(prev => [...prev, data]);
    };

    const handleSpeechStarted = () => {
      setIsSpeaking(true);
    };

    const handleSpeechStopped = () => {
      setIsSpeaking(false);
    };

    const handleRecordingStateChanged = (state: { isRecording: boolean }) => {
      setIsRecording(state.isRecording);
    };

    const handleError = (error: ErrorData) => {
      setLastError(error);
    };

    const handleDisconnected = () => {
      setSessionId(null);
      setSessionData(null);
      setIsSpeaking(false);
      setIsRecording(false);
    };

    // Register event listeners
    client.on('connectionStateChanged', handleConnectionStateChanged);
    client.on('sessionCreated', handleSessionCreated);
    client.on('transcription', handleTranscription);
    client.on('speechStarted', handleSpeechStarted);
    client.on('speechStopped', handleSpeechStopped);
    client.on('recordingStateChanged', handleRecordingStateChanged);
    client.on('error', handleError);
    client.on('disconnected', handleDisconnected);

    return () => {
      // Clean up
      client.off('connectionStateChanged', handleConnectionStateChanged);
      client.off('sessionCreated', handleSessionCreated);
      client.off('transcription', handleTranscription);
      client.off('speechStarted', handleSpeechStarted);
      client.off('speechStopped', handleSpeechStopped);
      client.off('recordingStateChanged', handleRecordingStateChanged);
      client.off('error', handleError);
      client.off('disconnected', handleDisconnected);
      client.disconnect();
    };
  }, [options.apiKey, options.url, options.autoReconnect, options.enableLogging]);

  // Auto-connect
  useEffect(() => {
    if (options.autoConnect && clientRef.current && !isConnected && !isConnecting) {
      connect();
    }
  }, [options.autoConnect]);

  // Auto-start recording
  useEffect(() => {
    if (
      options.autoStartRecording &&
      isConnected &&
      !isRecording &&
      isRecordingSupported &&
      options.sessionConfig
    ) {
      startRecording();
    }
  }, [options.autoStartRecording, isConnected, isRecording, isRecordingSupported, options.sessionConfig]);

  // Auto-configure session
  useEffect(() => {
    if (isConnected && options.sessionConfig && clientRef.current) {
      configureSession(options.sessionConfig);
    }
  }, [isConnected, options.sessionConfig]);

  const connect = useCallback(async () => {
    if (!clientRef.current) return;
    try {
      await clientRef.current.connect();
    } catch (error) {
      console.error('Failed to connect:', error);
    }
  }, []);

  const disconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
    }
  }, []);

  const startRecording = useCallback(async () => {
    if (!clientRef.current || !isConnected) return;
    try {
      await clientRef.current.startRecording();
    } catch (error) {
      console.error('Failed to start recording:', error);
    }
  }, [isConnected]);

  const stopRecording = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.stopRecording();
    }
  }, []);

  const commitAudio = useCallback(async () => {
    if (!clientRef.current || !isConnected) return;
    try {
      await clientRef.current.commitAudio();
    } catch (error) {
      console.error('Failed to commit audio:', error);
    }
  }, [isConnected]);

  const clearAudioBuffer = useCallback(async () => {
    if (!clientRef.current || !isConnected) return;
    try {
      await clientRef.current.clearAudioBuffer();
    } catch (error) {
      console.error('Failed to clear audio buffer:', error);
    }
  }, [isConnected]);

  const configureSession = useCallback(async (config: SessionConfig) => {
    if (!clientRef.current || !isConnected) return;
    try {
      await clientRef.current.configureSession(config);
    } catch (error) {
      console.error('Failed to configure session:', error);
    }
  }, [isConnected]);

  const clearError = useCallback(() => {
    setLastError(null);
  }, []);

  return {
    // Connection state
    isConnected,
    isConnecting,
    connectionState,
    sessionId,

    // Recording state
    isRecording,
    isRecordingSupported,

    // Data
    transcript,
    transcriptions,
    isSpeaking,
    lastError,
    sessionData,

    // Actions
    connect,
    disconnect,
    startRecording,
    stopRecording,
    commitAudio,
    clearAudioBuffer,
    configureSession,
    clearError,

    // Client instance
    client: clientRef.current,
  };
}

/**
 * React hook for managing recording state
 */
export interface UseRecordingResult {
  isRecording: boolean;
  isSupported: boolean;
  startRecording: () => Promise<void>;
  stopRecording: () => void;
  toggleRecording: () => Promise<void>;
  clearError: () => void;
  error: string | null;
}

export function useRecording(client: StreamASRClient | null): UseRecordingResult {
  const [isRecording, setIsRecording] = useState(false);
  const [isSupported, setIsSupported] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIsSupported(StreamASRClient.isSupported());
  }, []);

  useEffect(() => {
    if (!client) return;

    const handleRecordingStateChanged = (state: { isRecording: boolean }) => {
      setIsRecording(state.isRecording);
      setError(null);
    };

    const handleError = (errorData: ErrorData) => {
      if (errorData.type === 'recording_error') {
        setError(errorData.message);
      }
    };

    client.on('recordingStateChanged', handleRecordingStateChanged);
    client.on('error', handleError);

    return () => {
      client.off('recordingStateChanged', handleRecordingStateChanged);
      client.off('error', handleError);
    };
  }, [client]);

  const startRecording = useCallback(async () => {
    if (!client) {
      setError('Client not available');
      return;
    }

    try {
      setError(null);
      await client.startRecording();
    } catch (error) {
      setError((error as Error).message);
    }
  }, [client]);

  const stopRecording = useCallback(() => {
    if (!client) return;
    client.stopRecording();
  }, [client]);

  const toggleRecording = useCallback(async () => {
    if (isRecording) {
      stopRecording();
    } else {
      await startRecording();
    }
  }, [isRecording, startRecording, stopRecording]);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    isRecording,
    isSupported,
    startRecording,
    stopRecording,
    toggleRecording,
    clearError,
    error,
  };
}

/**
 * React hook for transcription management
 */
export interface UseTranscriptionResult {
  transcript: string;
  transcriptions: TranscriptionData[];
  clearTranscriptions: () => void;
  clearTranscript: () => void;
}

export function useTranscription(client: StreamASRClient | null): UseTranscriptionResult {
  const [transcript, setTranscript] = useState('');
  const [transcriptions, setTranscriptions] = useState<TranscriptionData[]>([]);

  useEffect(() => {
    if (!client) return;

    const handleTranscription = (data: TranscriptionData) => {
      setTranscript(data.text);
      setTranscriptions(prev => [...prev, data]);
    };

    client.on('transcription', handleTranscription);

    return () => {
      client.off('transcription', handleTranscription);
    };
  }, [client]);

  const clearTranscriptions = useCallback(() => {
    setTranscriptions([]);
  }, []);

  const clearTranscript = useCallback(() => {
    setTranscript('');
  }, []);

  return {
    transcript,
    transcriptions,
    clearTranscriptions,
    clearTranscript,
  };
}