// OpenAI Realtime API WebSocket connection
let socket;
let sessionId;
let heartbeatInterval;
const HEARTBEAT_INTERVAL = 30000; // 30 second heartbeat interval

// Audio processing related variables
let audioContext;
let processor;
let mediaStream;
let analyser;
let canvasCtx;
let animationId;

// Debug related variables
let isRecording = false;
let recordingStartTime;

// DOM elements
const startBtn = document.getElementById('startBtn');
const stopBtn = document.getElementById('stopBtn');
const transcript = document.getElementById('transcript');
const status = document.getElementById('status');
const saveBtn = document.getElementById('copyBtn');
const sampleRateSelect = document.getElementById('sampleRate');
const vadEnabledCheckbox = document.getElementById('vadEnabled');
const manualCommitBtn = document.getElementById('manualCommitBtn');

// Update status display
function updateStatus(message, isError = false) {
    status.textContent = message;
    status.className = isError ? 'status error' : 'status';
    console.log(message);
}

// Generate unique event ID
function generateEventId() {
    return 'evt_' + Date.now() + '_' + Math.random().toString(36).substring(2, 11);
}

// Send audio buffer commit request
function sendAudioBufferCommit() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket not connected, cannot send commit request');
        return;
    }

    const commitEvent = {
        type: "input_audio_buffer.commit",
        event_id: generateEventId(),
        session_id: sessionId
    };

    try {
        socket.send(JSON.stringify(commitEvent));
        console.log('[Audio Processing] Sent audio buffer commit request');
        updateStatus('Submitting audio, waiting for confirmation...');
    } catch (error) {
        console.error('Failed to send commit request:', error);
        updateStatus('Audio submission failed', true);
    }
}

// Send session configuration
function sendSessionConfiguration() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket not connected, cannot send session configuration');
        return;
    }

    const sessionUpdateEvent = {
        type: "session.update",
        event_id: generateEventId(),
        // Initial session.update does not include session_id, server will provide it in session.updated response
        session: {
            // Initial session.update does not include id field
            modality: "audio",
            input_audio_format: {
                type: "pcm16",
                sample_rate: parseInt(sampleRateSelect.value),
                channels: 1
            },
            output_audio_format: {
                type: "pcm16",
                sample_rate: parseInt(sampleRateSelect.value),
                channels: 1
            },
            input_audio_transcription: {
                model: "FunAudioLLM/SenseVoiceSmall",
                language: "zh"
            },
            turn_detection: {
                type: "server_vad",
                threshold: 0.5,
                prefix_padding_ms: 300,
                silence_duration_ms: 2000
            }
        }
    };

    socket.send(JSON.stringify(sessionUpdateEvent));
    console.log('Session configuration sent');
}

// Update sample rate configuration
function updateSampleRate(newSampleRate) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket not connected, cannot update sample rate configuration');
        updateStatus('WebSocket not connected, cannot update sample rate', true);
        return;
    }

    if (!sessionId) {
        console.warn('Session ID not obtained, cannot update sample rate configuration');
        updateStatus('Session not established, cannot update sample rate', true);
        return;
    }

    const sampleRateUpdateEvent = {
        type: "session.update",
        event_id: generateEventId(),
        session_id: sessionId,
        session: {
            modality: "audio",
            input_audio_format: {
                type: "pcm16",
                sample_rate: newSampleRate,
                channels: 1
            },
            output_audio_format: {
                type: "pcm16",
                sample_rate: newSampleRate,
                channels: 1
            }
        }
    };

    try {
        socket.send(JSON.stringify(sampleRateUpdateEvent));
        console.log(`[Sample Rate Update] Sent sample rate update request: ${newSampleRate}Hz`);
        updateStatus(`Updating sample rate to ${newSampleRate}Hz...`);

        // If recording is in progress, prompt user to restart recording
        if (isRecording) {
            setTimeout(() => {
                updateStatus('Sample rate updated, please restart recording to apply new settings', false);
            }, 1000);
        } else {
            setTimeout(() => {
                updateStatus(`Sample rate updated to ${newSampleRate}Hz`);
            }, 1000);
        }
    } catch (error) {
        console.error('Failed to send sample rate update request:', error);
        updateStatus('Sample rate update failed', true);
    }
}

// Initialize OpenAI Realtime API WebSocket connection
function initOpenAIWebSocket() {
    // Ensure no existing connection
    if (socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(socket.readyState)) {
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/v1/realtime`;

    try {
        // Close existing connection
        if (socket) {
            socket.close();
        }

        socket = new WebSocket(wsUrl);
        console.log('Establishing OpenAI Realtime API connection...');
    } catch (e) {
        console.error('Failed to create WebSocket:', e);
        updateStatus('Failed to create connection', true);
        return;
    }

    socket.onopen = () => {
        updateStatus('OpenAI Realtime API connection established');

        // Immediately send session configuration (OpenAI Realtime API standard procedure)
        sendSessionConfiguration();

        // Start heartbeat detection
        startHeartbeat();
    };

    socket.onclose = (event) => {
        let message = `Connection closed, code=${event.code}`;
        if (event.reason) {
            message += `, reason=${event.reason}`;
        }
        updateStatus(message, !event.wasClean);
        console.log(message);

        stopRecording();
        stopHeartbeat();

        // If abnormal closure, display error message
        if (event.code === 1006) {
            updateStatus('Connection interrupted unexpectedly, please check server status', true);
        }
    };

    socket.onerror = (error) => {
        console.error('WebSocket error:', error);
        updateStatus('Connection error, please check server status', true);
        stopRecording();
    };

    socket.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            console.log('Received OpenAI event:', data);

            // Handle different types of events
            switch (data.type) {
                case "session.created":
                    // Get session ID from session.created event
                    if (data.session && data.session.id) {
                        sessionId = data.session.id;
                        console.log('[Session] Session created, ID:', sessionId);
                        updateStatus('Session created');
                    }
                    break;

                case "session.updated":
                    // Get session ID from session.updated response
                    if (data.session && data.session.id) {
                        sessionId = data.session.id;
                        console.log('[Session] Session configuration updated, ID:', sessionId);
                    }
                    updateStatus('Session configuration updated');
                    break;

                case "heartbeat.ping":
                    // Respond to heartbeat ping
                    const pongEvent = {
                        type: "heartbeat.pong",
                        event_id: generateEventId(),
                        session_id: sessionId
                    };
                    socket.send(JSON.stringify(pongEvent));
                    break;

                case "conversation.item.created":
                    console.log('[Conversation] Conversation item created:', data.item.id);
                    break;

                case "conversation.item.input_audio_transcription.completed":
                    if (data.item && data.item.content && data.item.content.length > 0) {
                        const transcription = data.item.content[0].transcript;
                        if (transcription && transcription.trim()) {
                            // Add timestamp and format display
                            const timestamp = new Date().toLocaleTimeString();
                            const formattedText = `[${timestamp}] ${transcription.trim()}`;

                            transcript.textContent += formattedText + '\n\n';
                            transcript.scrollTop = transcript.scrollHeight;

                            console.log('[Transcription] Recognition completed:', transcription);
                            updateStatus('Speech recognition completed, continue speaking...');
                        } else {
                            console.log('[Transcription] Recognition result is empty');
                            updateStatus('No valid speech detected, please try again');
                        }
                    }
                    break;

                case "conversation.item.input_audio_transcription.failed":
                    console.error('[Transcription] Recognition failed:', data.error);
                    updateStatus(`Speech recognition failed: ${data.error?.message || 'Unknown error'}`, true);
                    break;

                case "input_audio_buffer.speech_started":
                    updateStatus('Speech start detected');
                    console.log('[Speech Detection] Start recording');
                    break;

                case "input_audio_buffer.speech_stopped":
                    updateStatus('Speech end detected, submitting audio...');
                    console.log('[Speech Detection] Speech stopped, sending commit request');
                    // Client actively sends commit after receiving speech_stopped
                    sendAudioBufferCommit();
                    break;

                case "input_audio_buffer.committed":
                    updateStatus('Audio submitted successfully, recognizing...');
                    console.log('[Audio Processing] Audio submission confirmed');
                    break;

                case "heartbeat.pong":
                    console.log('[Heartbeat] Received pong response');
                    break;

                case "error":
                    console.error('[Error] Server error:', data.error);
                    updateStatus(`Error: ${data.error?.message || 'Unknown error'}`, true);
                    break;

                default:
                    console.log('[Event] Unhandled event type:', data.type, data);
            }
        } catch (e) {
            console.error('Failed to parse message:', e);
            updateStatus('Failed to parse message', true);
        }
    };
}

// Start heartbeat detection
function startHeartbeat() {
    stopHeartbeat();

    heartbeatInterval = setInterval(() => {
        if (socket && socket.readyState === WebSocket.OPEN) {
            try {
                const pingEvent = {
                    type: "heartbeat.ping",
                    event_id: generateEventId(),
                    session_id: sessionId
                };
                socket.send(JSON.stringify(pingEvent));
                console.log('[Heartbeat] Sending ping');
            } catch (e) {
                console.error('Failed to send heartbeat:', e);
            }
        }
    }, HEARTBEAT_INTERVAL);
}

// Stop heartbeat detection
function stopHeartbeat() {
    if (heartbeatInterval) {
        clearInterval(heartbeatInterval);
        heartbeatInterval = null;
    }
}

// Start recording
async function startRecording() {
    try {
        updateStatus('Requesting microphone permission...');

        // Get microphone permission
        mediaStream = await navigator.mediaDevices.getUserMedia({
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                autoGainControl: false,
                sampleRate: parseInt(sampleRateSelect.value)
            }
        });

        // Create audio context
        audioContext = new (window.AudioContext || window.webkitAudioContext)({
            sampleRate: parseInt(sampleRateSelect.value)
        });

        // Wait for audio context to resume
        if (audioContext.state === 'suspended') {
            await audioContext.resume();
        }

        const source = audioContext.createMediaStreamSource(mediaStream);

        // Create analyzer node for visualization
        analyser = audioContext.createAnalyser();
        analyser.fftSize = 256;
        source.connect(analyser);

        // Initialize Canvas context
        const canvas = document.getElementById('audioVisualizer');
        canvasCtx = canvas.getContext('2d');

        // Start animation loop
        visualizeAudio();

        // Create audio processor
        processor = audioContext.createScriptProcessor(4096, 1, 1);

        processor.onaudioprocess = (e) => {
            const inputBuffer = e.inputBuffer;
            const channelData = inputBuffer.getChannelData(0);

            // Convert to 16-bit PCM
            const pcmData = new Int16Array(channelData.length);
            for (let i = 0; i < channelData.length; i++) {
                const sample = Math.max(-1, Math.min(1, channelData[i]));
                pcmData[i] = sample < 0 ? sample * 32768 : sample * 32767;
            }

            // Convert to Base64
            const base64Audio = btoa(String.fromCharCode.apply(null, new Uint8Array(pcmData.buffer)));

            // Send audio data to OpenAI API
            if (socket && socket.readyState === WebSocket.OPEN) {
                try {
                    const audioEvent = {
                        type: "input_audio_buffer.append",
                        event_id: generateEventId(),
                        session_id: sessionId,
                        audio: base64Audio
                    };
                    socket.send(JSON.stringify(audioEvent));
                } catch (e) {
                    console.error('Failed to send audio data:', e);
                    stopRecording();
                }
            }
        };

        // Connect nodes
        source.connect(processor);
        processor.connect(audioContext.destination);

        // Initialize recording state
        isRecording = true;
        recordingStartTime = Date.now();

        updateStatus('Performing speech recognition...');
        startBtn.disabled = true;
        stopBtn.disabled = false;
        if (manualCommitBtn) manualCommitBtn.disabled = false;

    } catch (error) {
        updateStatus(`Recognition failed: ${error.message}`, true);
        console.error('Recognition error:', error);
    }
}

function stopRecording() {
    if (mediaStream) {
        mediaStream.getTracks().forEach(track => track.stop());
    }
    if (audioContext) {
        audioContext.close().catch(e => console.error('Failed to close audio context:', e));
    }
    if (processor) {
        processor.disconnect();
    }
    if (animationId) {
        cancelAnimationFrame(animationId);
    }

    // Stop recording
    isRecording = false;

    updateStatus('Speech recognition stopped');
    startBtn.disabled = false;
    stopBtn.disabled = true;
    if (manualCommitBtn) manualCommitBtn.disabled = true;
}

// Audio visualization function
function visualizeAudio() {
    if (!analyser) return;

    const canvas = document.getElementById('audioVisualizer');
    const WIDTH = canvas.width;
    const HEIGHT = canvas.height;

    const bufferLength = analyser.frequencyBinCount;
    const dataArray = new Uint8Array(bufferLength);
    analyser.getByteFrequencyData(dataArray);

    canvasCtx.clearRect(0, 0, WIDTH, HEIGHT);

    const barWidth = (WIDTH / bufferLength) * 2.5;
    let x = 0;

    for (let i = 0; i < bufferLength; i++) {
        const barHeight = (dataArray[i] / 255) * HEIGHT;

        const gradient = canvasCtx.createLinearGradient(0, HEIGHT - barHeight, 0, HEIGHT);
        gradient.addColorStop(0, '#00d4ff');
        gradient.addColorStop(0.7, '#0066ff');
        gradient.addColorStop(1, '#090979');

        canvasCtx.fillStyle = gradient;
        canvasCtx.fillRect(x, HEIGHT - barHeight, barWidth, barHeight);

        x += barWidth + 1;
    }

    animationId = requestAnimationFrame(visualizeAudio);
}

// Save functionality
function setupSaveButton() {
    saveBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(transcript.textContent);
            // Display save success feedback
            const originalText = saveBtn.querySelector('span').textContent;
            saveBtn.querySelector('span').textContent = 'Saved';
            saveBtn.classList.add('saved');

            setTimeout(() => {
                saveBtn.querySelector('span').textContent = originalText;
                saveBtn.classList.remove('saved');
            }, 2000);
        } catch (err) {
            console.error('Save failed:', err);
            const originalText = saveBtn.querySelector('span').textContent;
            saveBtn.querySelector('span').textContent = 'Save failed';
            setTimeout(() => {
                saveBtn.querySelector('span').textContent = originalText;
            }, 2000);
        }
    });
}

// AI summary functionality
function setupAISummaryButton() {
    const aiSummaryBtn = document.getElementById('aiSummaryBtn');
    const aiSummary = document.getElementById('aiSummary');

    aiSummaryBtn.addEventListener('click', async () => {
        const isVisible = aiSummary.style.display !== 'none';
        aiSummary.style.display = isVisible ? 'none' : 'block';

        if (!isVisible) {
            const transcriptText = transcript.textContent.trim();
            if (!transcriptText) {
                aiSummary.textContent = "No content to summarize";
                return;
            }

            aiSummary.textContent = "Generating AI summary...";

            try {
                // Call backend API
                const response = await fetch('/v1/chat/completions', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        model: "deepseek-coder",
                        messages: [{
                            role: "user",
                            content: `Please summarize the following text:\n${transcriptText}`
                        }],
                        stream: true
                    })
                });

                if (!response.ok) {
                    throw new Error(`API request failed: ${response.status}`);
                }

                // Handle streaming response
                const reader = response.body.getReader();
                const decoder = new TextDecoder();
                let result = '';

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value);
                    const lines = chunk.split('\n').filter(line => line.trim());

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            const data = line.replace('data: ', '');
                            if (data === '[DONE]') continue;

                            try {
                                const json = JSON.parse(data);
                                if (json.choices && json.choices[0].delta.content) {
                                    result += json.choices[0].delta.content;
                                    aiSummary.textContent = result;
                                    aiSummary.scrollTop = aiSummary.scrollHeight;
                                }
                            } catch (e) {
                                console.error('Failed to parse JSON:', e);
                            }
                        }
                    }
                }
            } catch (error) {
                console.error('AI summary failed:', error);
                aiSummary.textContent = `AI summary failed: ${error.message}`;
            }
        }
    });
}

// Initialize
function initializeApp() {
    // Ensure DOM elements exist
    if (!startBtn || !stopBtn || !transcript || !status || !saveBtn) {
        console.error('Key DOM elements not found, please check HTML structure');
        return;
    }

    // Remove previous event listeners to avoid duplication
    document.removeEventListener('DOMContentLoaded', initializeApp);
    startBtn.removeEventListener('click', startRecording);
    stopBtn.removeEventListener('click', stopRecording);

    initOpenAIWebSocket();
    setupSaveButton();
    setupAISummaryButton();

    startBtn.addEventListener('click', startRecording);
    stopBtn.addEventListener('click', stopRecording);

    // Add sample rate selector change event listener
    if (sampleRateSelect) {
        sampleRateSelect.addEventListener('change', (event) => {
            const newSampleRate = parseInt(event.target.value);
            console.log(`[Sample Rate Change] User selected sample rate: ${newSampleRate}Hz`);
            updateSampleRate(newSampleRate);
        });
        console.log('[Initialization] Sample rate selector event listener added');
    } else {
        console.warn('[Initialization] Sample rate selector not found');
    }

    // Add manual commit button event
    if (manualCommitBtn) {
        manualCommitBtn.addEventListener('click', () => {
            if (isRecording) {
                sendAudioBufferCommit();
            } else {
                updateStatus('Please start recording first');
            }
        });
    }

    // Cleanup before closing
    window.addEventListener('beforeunload', () => {
        if (socket) socket.close();
        if (mediaStream) stopRecording();
    });
}

// Ensure DOM is fully loaded before initialization
function checkDOMAndInitialize() {
    const requiredElements = ['startBtn', 'stopBtn', 'transcript', 'status', 'copyBtn'];
    const allElementsExist = requiredElements.every(id => document.getElementById(id));

    if (allElementsExist) {
        initializeApp();
    } else {
        setTimeout(checkDOMAndInitialize, 100);
    }
}

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', checkDOMAndInitialize);
} else {
    checkDOMAndInitialize();
}

// Theme switching functionality
function switchTheme(themeName) {
    document.documentElement.setAttribute('data-theme', themeName);
    localStorage.setItem('theme', themeName);
}

// Check saved theme on initialization
document.addEventListener('DOMContentLoaded', () => {
    const savedTheme = localStorage.getItem('theme') || 'default';
    document.documentElement.setAttribute('data-theme', savedTheme);

    startBtn.addEventListener('click', startRecording);
    stopBtn.addEventListener('click', stopRecording);
});

let collapseTimer = null;

// Theme switcher expand/collapse functionality
function toggleThemeSwitcher(event) {
    const switcher = event.currentTarget;
    const clickedButton = event.target.closest('button');

    // If clicking a theme button, don't toggle expanded state
    if (clickedButton) {
        startAutoCollapse(switcher);
        return;
    }

    switcher.classList.toggle('expanded');
    if (switcher.classList.contains('expanded')) {
        startAutoCollapse(switcher);
    } else {
        clearAutoCollapse();
    }
}

// Start auto-collapse timer
function startAutoCollapse(switcher) {
    clearAutoCollapse();
    collapseTimer = setTimeout(() => {
        switcher.classList.remove('expanded');
    }, 3000);
}

// Clear auto-collapse timer
function clearAutoCollapse() {
    if (collapseTimer) {
        clearTimeout(collapseTimer);
        collapseTimer = null;
    }
}