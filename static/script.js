// OpenAI Realtime API WebSocket连接
let socket;
let sessionId;
let heartbeatInterval;
const HEARTBEAT_INTERVAL = 30000; // 30秒心跳间隔

// 音频处理相关变量
let audioContext;
let processor;
let mediaStream;
let analyser;
let canvasCtx;
let animationId;

// 调试相关变量
let isRecording = false;
let recordingStartTime;

// DOM元素
const startBtn = document.getElementById('startBtn');
const stopBtn = document.getElementById('stopBtn');
const transcript = document.getElementById('transcript');
const status = document.getElementById('status');
const saveBtn = document.getElementById('copyBtn');
const sampleRateSelect = document.getElementById('sampleRate');
const vadEnabledCheckbox = document.getElementById('vadEnabled');
const manualCommitBtn = document.getElementById('manualCommitBtn');

// 更新状态显示
function updateStatus(message, isError = false) {
    status.textContent = message;
    status.className = isError ? 'status error' : 'status';
    console.log(message);
}

// 生成唯一事件ID
function generateEventId() {
    return 'evt_' + Date.now() + '_' + Math.random().toString(36).substring(2, 11);
}

// 发送音频缓冲区提交请求
function sendAudioBufferCommit() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket未连接，无法发送commit请求');
        return;
    }

    const commitEvent = {
        type: "input_audio_buffer.commit",
        event_id: generateEventId(),
        session_id: sessionId
    };

    try {
        socket.send(JSON.stringify(commitEvent));
        console.log('[音频处理] 已发送音频缓冲区commit请求');
        updateStatus('音频提交中，等待确认...');
    } catch (error) {
        console.error('发送commit请求失败:', error);
        updateStatus('音频提交失败', true);
    }
}

// 发送会话配置
function sendSessionConfiguration() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket未连接，无法发送会话配置');
        return;
    }

    const sessionUpdateEvent = {
        type: "session.update",
        event_id: generateEventId(),
        // 初始session.update不包含session_id，服务器会在session.updated响应中提供
        session: {
            // 初始session.update不包含id字段
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
    console.log('会话配置已发送');
}

// 更新采样率配置
function updateSampleRate(newSampleRate) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket未连接，无法更新采样率配置');
        updateStatus('WebSocket未连接，无法更新采样率', true);
        return;
    }

    if (!sessionId) {
        console.warn('会话ID未获取，无法更新采样率配置');
        updateStatus('会话未建立，无法更新采样率', true);
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
        console.log(`[采样率更新] 已发送采样率更新请求: ${newSampleRate}Hz`);
        updateStatus(`正在更新采样率到 ${newSampleRate}Hz...`);

        // 如果正在录音，提示用户需要重新开始录音
        if (isRecording) {
            setTimeout(() => {
                updateStatus('采样率已更新，请重新开始录音以应用新设置', false);
            }, 1000);
        } else {
            setTimeout(() => {
                updateStatus(`采样率已更新到 ${newSampleRate}Hz`);
            }, 1000);
        }
    } catch (error) {
        console.error('发送采样率更新请求失败:', error);
        updateStatus('采样率更新失败', true);
    }
}

// 初始化OpenAI Realtime API WebSocket连接
function initOpenAIWebSocket() {
    // 确保没有已存在的连接
    if (socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(socket.readyState)) {
        return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const wsUrl = `${protocol}//${host}/v1/realtime`;

    try {
        // 关闭现有连接
        if (socket) {
            socket.close();
        }

        socket = new WebSocket(wsUrl);
        console.log('正在建立OpenAI Realtime API连接...');
    } catch (e) {
        console.error('创建WebSocket失败:', e);
        updateStatus('创建连接失败', true);
        return;
    }

    socket.onopen = () => {
        updateStatus('OpenAI Realtime API连接已建立');

        // 立即发送会话配置（OpenAI Realtime API标准流程）
        sendSessionConfiguration();

        // 启动心跳检测
        startHeartbeat();
    };

    socket.onclose = (event) => {
        let message = `连接已关闭，代码=${event.code}`;
        if (event.reason) {
            message += `，原因=${event.reason}`;
        }
        updateStatus(message, !event.wasClean);
        console.log(message);

        stopRecording();
        stopHeartbeat();

        // 如果是异常关闭，显示错误提示
        if (event.code === 1006) {
            updateStatus('连接异常中断，请检查服务器状态', true);
        }
    };

    socket.onerror = (error) => {
        console.error('WebSocket错误:', error);
        updateStatus('连接错误，请检查服务器状态', true);
        stopRecording();
    };

    socket.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            console.log('收到OpenAI事件:', data);

            // 处理不同类型的事件
            switch (data.type) {
                case "session.created":
                    // 从session.created事件中获取session ID
                    if (data.session && data.session.id) {
                        sessionId = data.session.id;
                        console.log('[会话] 会话已创建，ID:', sessionId);
                        updateStatus('会话已创建');
                    }
                    break;

                case "session.updated":
                    // 从session.updated响应中获取session ID
                    if (data.session && data.session.id) {
                        sessionId = data.session.id;
                        console.log('[会话] 会话配置已更新，ID:', sessionId);
                    }
                    updateStatus('会话配置已更新');
                    break;

                case "heartbeat.ping":
                    // 响应心跳ping
                    const pongEvent = {
                        type: "heartbeat.pong",
                        event_id: generateEventId(),
                        session_id: sessionId
                    };
                    socket.send(JSON.stringify(pongEvent));
                    break;

                case "conversation.item.created":
                    console.log('[对话] 对话项目已创建:', data.item.id);
                    break;

                case "conversation.item.input_audio_transcription.completed":
                    if (data.item && data.item.content && data.item.content.length > 0) {
                        const transcription = data.item.content[0].transcript;
                        if (transcription && transcription.trim()) {
                            // 添加时间戳和格式化显示
                            const timestamp = new Date().toLocaleTimeString();
                            const formattedText = `[${timestamp}] ${transcription.trim()}`;

                            transcript.textContent += formattedText + '\n\n';
                            transcript.scrollTop = transcript.scrollHeight;

                            console.log('[转写] 识别完成:', transcription);
                            updateStatus('语音识别完成，继续说话...');
                        } else {
                            console.log('[转写] 识别结果为空');
                            updateStatus('未识别到有效语音，请重试');
                        }
                    }
                    break;

                case "conversation.item.input_audio_transcription.failed":
                    console.error('[转写] 识别失败:', data.error);
                    updateStatus(`语音识别失败: ${data.error?.message || '未知错误'}`, true);
                    break;

                case "input_audio_buffer.speech_started":
                    updateStatus('检测到语音开始');
                    console.log('[语音检测] 开始录音');
                    break;

                case "input_audio_buffer.speech_stopped":
                    updateStatus('检测到语音结束，正在提交音频...');
                    console.log('[语音检测] 语音停止，发送 commit 请求');
                    // 客户端收到 speech_stopped 后主动发送 commit
                    sendAudioBufferCommit();
                    break;

                case "input_audio_buffer.committed":
                    updateStatus('音频提交成功，正在识别...');
                    console.log('[音频处理] 音频已提交确认');
                    break;

                case "heartbeat.pong":
                    console.log('[心跳] 收到pong响应');
                    break;

                case "error":
                    console.error('[错误] 服务器错误:', data.error);
                    updateStatus(`错误: ${data.error?.message || '未知错误'}`, true);
                    break;

                default:
                    console.log('[事件] 未处理的事件类型:', data.type, data);
            }
        } catch (e) {
            console.error('解析消息失败:', e);
            updateStatus('解析消息失败', true);
        }
    };
}

// 启动心跳检测
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
                console.log('[心跳] 发送ping');
            } catch (e) {
                console.error('发送心跳失败:', e);
            }
        }
    }, HEARTBEAT_INTERVAL);
}

// 停止心跳检测
function stopHeartbeat() {
    if (heartbeatInterval) {
        clearInterval(heartbeatInterval);
        heartbeatInterval = null;
    }
}

// 开始录音
async function startRecording() {
    try {
        updateStatus('正在获取麦克风权限...');

        // 获取麦克风权限
        mediaStream = await navigator.mediaDevices.getUserMedia({
            audio: {
                echoCancellation: true,
                noiseSuppression: true,
                autoGainControl: false,
                sampleRate: parseInt(sampleRateSelect.value)
            }
        });

        // 创建音频上下文
        audioContext = new (window.AudioContext || window.webkitAudioContext)({
            sampleRate: parseInt(sampleRateSelect.value)
        });

        // 等待音频上下文恢复
        if (audioContext.state === 'suspended') {
            await audioContext.resume();
        }

        const source = audioContext.createMediaStreamSource(mediaStream);

        // 创建分析节点用于可视化
        analyser = audioContext.createAnalyser();
        analyser.fftSize = 256;
        source.connect(analyser);

        // 初始化Canvas上下文
        const canvas = document.getElementById('audioVisualizer');
        canvasCtx = canvas.getContext('2d');

        // 开始动画循环
        visualizeAudio();

        // 创建音频处理器
        processor = audioContext.createScriptProcessor(4096, 1, 1);

        processor.onaudioprocess = (e) => {
            const inputBuffer = e.inputBuffer;
            const channelData = inputBuffer.getChannelData(0);

            // 转换为16位PCM
            const pcmData = new Int16Array(channelData.length);
            for (let i = 0; i < channelData.length; i++) {
                const sample = Math.max(-1, Math.min(1, channelData[i]));
                pcmData[i] = sample < 0 ? sample * 32768 : sample * 32767;
            }

            // 转换为Base64
            const base64Audio = btoa(String.fromCharCode.apply(null, new Uint8Array(pcmData.buffer)));

            // 发送音频数据到OpenAI API
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
                    console.error('发送音频数据失败:', e);
                    stopRecording();
                }
            }
        };

        // 连接节点
        source.connect(processor);
        processor.connect(audioContext.destination);

        // 初始化录音状态
        isRecording = true;
        recordingStartTime = Date.now();

        updateStatus('正在语音识别...');
        startBtn.disabled = true;
        stopBtn.disabled = false;
        if (manualCommitBtn) manualCommitBtn.disabled = false;

    } catch (error) {
        updateStatus(`识别失败: ${error.message}`, true);
        console.error('识别错误:', error);
    }
}

function stopRecording() {
    if (mediaStream) {
        mediaStream.getTracks().forEach(track => track.stop());
    }
    if (audioContext) {
        audioContext.close().catch(e => console.error('关闭音频上下文失败:', e));
    }
    if (processor) {
        processor.disconnect();
    }
    if (animationId) {
        cancelAnimationFrame(animationId);
    }

    // 停止录音
    isRecording = false;

    updateStatus('语音识别已停止');
    startBtn.disabled = false;
    stopBtn.disabled = true;
    if (manualCommitBtn) manualCommitBtn.disabled = true;
}

// 音频可视化函数
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

// 保存功能
function setupSaveButton() {
    saveBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(transcript.textContent);
            // 显示保存成功反馈
            const originalText = saveBtn.querySelector('span').textContent;
            saveBtn.querySelector('span').textContent = '已保存';
            saveBtn.classList.add('saved');

            setTimeout(() => {
                saveBtn.querySelector('span').textContent = originalText;
                saveBtn.classList.remove('saved');
            }, 2000);
        } catch (err) {
            console.error('保存失败:', err);
            const originalText = saveBtn.querySelector('span').textContent;
            saveBtn.querySelector('span').textContent = '保存失败';
            setTimeout(() => {
                saveBtn.querySelector('span').textContent = originalText;
            }, 2000);
        }
    });
}

// AI总结功能
function setupAISummaryButton() {
    const aiSummaryBtn = document.getElementById('aiSummaryBtn');
    const aiSummary = document.getElementById('aiSummary');

    aiSummaryBtn.addEventListener('click', async () => {
        const isVisible = aiSummary.style.display !== 'none';
        aiSummary.style.display = isVisible ? 'none' : 'block';

        if (!isVisible) {
            const transcriptText = transcript.textContent.trim();
            if (!transcriptText) {
                aiSummary.textContent = "没有可总结的内容";
                return;
            }

            aiSummary.textContent = "AI总结生成中...";

            try {
                // 调用后端API
                const response = await fetch('/v1/chat/completions', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        model: "deepseek-coder",
                        messages: [{
                            role: "user",
                            content: `请总结以下文本：\n${transcriptText}`
                        }],
                        stream: true
                    })
                });

                if (!response.ok) {
                    throw new Error(`API请求失败: ${response.status}`);
                }

                // 处理流式响应
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
                                console.error('解析JSON失败:', e);
                            }
                        }
                    }
                }
            } catch (error) {
                console.error('AI总结失败:', error);
                aiSummary.textContent = `AI总结失败: ${error.message}`;
            }
        }
    });
}

// 初始化
function initializeApp() {
    // 确保DOM元素存在
    if (!startBtn || !stopBtn || !transcript || !status || !saveBtn) {
        console.error('关键DOM元素未找到，请检查HTML结构');
        return;
    }

    // 移除之前的事件监听器避免重复
    document.removeEventListener('DOMContentLoaded', initializeApp);
    startBtn.removeEventListener('click', startRecording);
    stopBtn.removeEventListener('click', stopRecording);

    initOpenAIWebSocket();
    setupSaveButton();
    setupAISummaryButton();

    startBtn.addEventListener('click', startRecording);
    stopBtn.addEventListener('click', stopRecording);

    // 添加采样率选择器变化事件监听
    if (sampleRateSelect) {
        sampleRateSelect.addEventListener('change', (event) => {
            const newSampleRate = parseInt(event.target.value);
            console.log(`[采样率变化] 用户选择采样率: ${newSampleRate}Hz`);
            updateSampleRate(newSampleRate);
        });
        console.log('[初始化] 采样率选择器事件监听器已添加');
    } else {
        console.warn('[初始化] 采样率选择器未找到');
    }

    // 添加手动提交按钮事件
    if (manualCommitBtn) {
        manualCommitBtn.addEventListener('click', () => {
            if (isRecording) {
                sendAudioBufferCommit();
            } else {
                updateStatus('请先开始录音');
            }
        });
    }

    // 关闭前清理
    window.addEventListener('beforeunload', () => {
        if (socket) socket.close();
        if (mediaStream) stopRecording();
    });
}

// 确保DOM完全加载后初始化
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

// 主题切换功能
function switchTheme(themeName) {
    document.documentElement.setAttribute('data-theme', themeName);
    localStorage.setItem('theme', themeName);
}

// 初始化时检查保存的主题
document.addEventListener('DOMContentLoaded', () => {
    const savedTheme = localStorage.getItem('theme') || 'default';
    document.documentElement.setAttribute('data-theme', savedTheme);

    startBtn.addEventListener('click', startRecording);
    stopBtn.addEventListener('click', stopRecording);
});

let collapseTimer = null;

// 主题切换器展开/折叠功能
function toggleThemeSwitcher(event) {
    const switcher = event.currentTarget;
    const clickedButton = event.target.closest('button');

    // 如果点击的是主题按钮，不切换展开状态
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

// 启动自动折叠计时器
function startAutoCollapse(switcher) {
    clearAutoCollapse();
    collapseTimer = setTimeout(() => {
        switcher.classList.remove('expanded');
    }, 3000);
}

// 清除自动折叠计时器
function clearAutoCollapse() {
    if (collapseTimer) {
        clearTimeout(collapseTimer);
        collapseTimer = null;
    }
}