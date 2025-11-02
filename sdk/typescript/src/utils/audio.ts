/**
 * Audio utilities for processing and converting audio data
 */

export interface AudioProcessorOptions {
  targetSampleRate: 16000 | 48000;
  channels: number;
}

/**
 * Convert Float32Array audio data to PCM16 Int16Array
 */
export function floatTo16BitPCM(input: Float32Array): Int16Array {
  const output = new Int16Array(input.length);
  for (let i = 0; i < input.length; i++) {
    const s = Math.max(-1, Math.min(1, input[i]));
    output[i] = s < 0 ? s * 0x8000 : s * 0x7fff;
  }
  return output;
}

/**
 * Convert PCM16 Int16Array audio data to Float32Array
 */
export function pcm16ToFloat(input: Int16Array): Float32Array {
  const output = new Float32Array(input.length);
  for (let i = 0; i < input.length; i++) {
    output[i] = input[i] / 0x8000;
  }
  return output;
}

/**
 * Convert PCM16 data to Base64 string
 */
export function pcm16ToBase64(pcmData: Int16Array): string {
  const buffer = new ArrayBuffer(pcmData.length * 2);
  const view = new DataView(buffer);

  for (let i = 0; i < pcmData.length; i++) {
    view.setInt16(i * 2, pcmData[i], true); // little-endian
  }

  return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

/**
 * Convert Base64 string to PCM16 data
 */
export function base64ToPcm16(base64: string): Int16Array {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);

  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  const buffer = bytes.buffer;
  const view = new DataView(buffer);
  const pcmData = new Int16Array(buffer.byteLength / 2);

  for (let i = 0; i < pcmData.length; i++) {
    pcmData[i] = view.getInt16(i * 2, true); // little-endian
  }

  return pcmData;
}

/**
 * Simple resampling using linear interpolation
 * Note: For production, consider using a more sophisticated resampling library
 */
export function resampleAudio(
  audioData: Int16Array,
  originalSampleRate: number,
  targetSampleRate: number
): Int16Array {
  if (originalSampleRate === targetSampleRate) {
    return audioData;
  }

  const ratio = originalSampleRate / targetSampleRate;
  const outputLength = Math.floor(audioData.length / ratio);
  const output = new Int16Array(outputLength);

  for (let i = 0; i < outputLength; i++) {
    const sourceIndex = i * ratio;
    const index = Math.floor(sourceIndex);
    const fraction = sourceIndex - index;

    if (index < audioData.length - 1) {
      const sample1 = audioData[index];
      const sample2 = audioData[index + 1];
      output[i] = Math.round(sample1 + fraction * (sample2 - sample1));
    } else {
      output[i] = audioData[audioData.length - 1];
    }
  }

  return output;
}

/**
 * Create WAV file header for PCM16 data
 */
export function createWAVHeader(sampleRate: number, channels: number, dataLength: number): ArrayBuffer {
  const headerLength = 44;
  const buffer = new ArrayBuffer(headerLength);
  const view = new DataView(buffer);

  // RIFF identifier
  view.setUint32(0, 0x46464952, false); // "RIFF"
  // file length
  view.setUint32(4, headerLength + dataLength - 8, true);
  // WAVE identifier
  view.setUint32(8, 0x45564157, false); // "WAVE"
  // fmt chunk identifier
  view.setUint32(12, 0x20746d66, false); // "fmt "
  // chunk length
  view.setUint32(16, 16, true);
  // sample format (PCM)
  view.setUint16(20, 1, true);
  // channel count
  view.setUint16(22, channels, true);
  // sample rate
  view.setUint32(24, sampleRate, true);
  // byte rate
  view.setUint32(28, sampleRate * channels * 2, true);
  // block align
  view.setUint16(32, channels * 2, true);
  // bits per sample
  view.setUint16(34, 16, true);
  // data chunk identifier
  view.setUint32(36, 0x61746164, false); // "data"
  // data length
  view.setUint32(40, dataLength, true);

  return buffer;
}

/**
 * Convert PCM16 data to WAV format
 */
export function pcm16ToWAV(audioData: Int16Array, sampleRate: number = 16000): ArrayBuffer {
  const wavHeader = createWAVHeader(sampleRate, 1, audioData.length * 2);
  const wavFile = new ArrayBuffer(wavHeader.byteLength + audioData.length * 2);
  const wavView = new DataView(wavFile);

  // Copy header
  const headerView = new DataView(wavHeader);
  for (let i = 0; i < wavHeader.byteLength; i++) {
    wavView.setUint8(i, headerView.getUint8(i));
  }

  // Copy PCM data
  const dataOffset = wavHeader.byteLength;
  for (let i = 0; i < audioData.length; i++) {
    wavView.setInt16(dataOffset + i * 2, audioData[i], true);
  }

  return wavFile;
}

/**
 * Calculate audio duration from samples and sample rate
 */
export function calculateDuration(samples: number, sampleRate: number): number {
  return samples / sampleRate;
}

/**
 * Convert milliseconds to samples
 */
export function msToSamples(ms: number, sampleRate: number): number {
  return Math.floor((ms / 1000) * sampleRate);
}

/**
 * Convert samples to milliseconds
 */
export function samplesToMs(samples: number, sampleRate: number): number {
  return (samples / sampleRate) * 1000;
}

/**
 * Apply a simple window function to audio data
 */
export function applyWindowFunction(audioData: Float32Array, windowType: 'hann' | 'hamming' = 'hann'): Float32Array {
  const output = new Float32Array(audioData.length);

  for (let i = 0; i < audioData.length; i++) {
    let window = 1;

    if (windowType === 'hann') {
      window = 0.5 * (1 - Math.cos((2 * Math.PI * i) / (audioData.length - 1)));
    } else if (windowType === 'hamming') {
      window = 0.54 - 0.46 * Math.cos((2 * Math.PI * i) / (audioData.length - 1));
    }

    output[i] = audioData[i] * window;
  }

  return output;
}