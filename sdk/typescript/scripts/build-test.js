#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

console.log('🚀 Building test files...');

// 确保test-build目录存在
const testBuildDir = path.join(__dirname, '../test-build');
const distDir = path.join(__dirname, '../dist');

if (!fs.existsSync(testBuildDir)) {
    fs.mkdirSync(testBuildDir, { recursive: true });
    console.log('✅ Created test-build directory');
}

// 创建README文件用于测试说明
const readmeContent = `# StreamASR SDK Build Test

这个目录包含了构建后的SDK文件和测试页面。

## 文件说明

- \`basic-usage.html\` - 使用构建后JS文件的测试页面
- \`../dist/\` - 构建后的JavaScript文件目录

## 使用方法

1. 确保已运行 \`npm run build\` 构建SDK
2. 运行 \`npm run serve:test\` 启动测试服务器
3. 在浏览器中访问 http://localhost:3001/basic-usage.html
4. 或者直接在浏览器中打开 basic-usage.html 文件

## 测试功能

- SDK加载测试
- WebSocket连接测试
- 音频录制和转录测试
- 实时状态监控
- 错误处理测试

## 服务器要求

确保StreamASR服务器正在运行并监听 \`ws://localhost:8080/v1/realtime\`

如果使用其他地址，请在页面中修改WebSocket URL。
`;

fs.writeFileSync(path.join(testBuildDir, 'README.md'), readmeContent);
console.log('✅ Created README.md');

// 复制示例音频文件（如果存在）
const examplesDir = path.join(__dirname, '../examples');
const samplesDir = path.join(testBuildDir, 'samples');

if (fs.existsSync(examplesDir)) {
    if (!fs.existsSync(samplesDir)) {
        fs.mkdirSync(samplesDir, { recursive: true });
    }

    // 查找音频文件
    const audioFiles = fs.readdirSync(examplesDir).filter(file =>
        file.endsWith('.wav') || file.endsWith('.mp3') || file.endsWith('.ogg')
    );

    audioFiles.forEach(file => {
        const srcPath = path.join(examplesDir, file);
        const destPath = path.join(samplesDir, file);
        fs.copyFileSync(srcPath, destPath);
        console.log(`✅ Copied audio file: ${file}`);
    });
}

// 创建一个简单的Node.js测试脚本
const nodeTestScript = `#!/usr/bin/env node

// Node.js测试脚本 - 测试构建后的模块
try {
    // 测试CommonJS模块
    console.log('Testing CommonJS module...');
    const StreamASR = require('../dist/index.js');

    if (StreamASR && StreamASR.StreamASRClient) {
        console.log('✅ StreamASRClient class loaded successfully');

        // 测试静态方法
        if (typeof StreamASR.StreamASRClient.isSupported === 'function') {
            console.log('✅ isSupported method available');
        } else {
            console.log('❌ isSupported method not found');
        }

        // 尝试创建实例
        try {
            const client = new StreamASR.StreamASRClient({
                url: 'ws://localhost:8080/v1/realtime',
                apiKey: 'test-key',
                enableLogging: true
            });
            console.log('✅ Client instance created successfully');
        } catch (error) {
            console.log('❌ Failed to create client instance:', error.message);
        }
    } else {
        console.log('❌ StreamASRClient class not found');
    }

} catch (error) {
    console.log('❌ Failed to load CommonJS module:', error.message);
}

console.log('\\nNode.js test completed.');
`;

fs.writeFileSync(path.join(testBuildDir, 'test-node.js'), nodeTestScript);
console.log('✅ Created Node.js test script');

// 创建一个React测试页面
const reactTestPage = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>StreamASR React Hook Test</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 1000px;
            margin: 0 auto;
            padding: 20px;
            line-height: 1.6;
        }
        .container {
            background: #f5f5f5;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        .status {
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 15px;
            font-weight: bold;
        }
        .status.success { background: #d4edda; color: #155724; }
        .status.error { background: #f8d7da; color: #721c24; }
        .status.info { background: #d1ecf1; color: #0c5460; }
        button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            margin: 5px;
        }
        .btn-primary { background: #007bff; color: white; }
        .btn-danger { background: #dc3545; color: white; }
        .transcript {
            background: white;
            padding: 20px;
            border-radius: 4px;
            border: 1px solid #ddd;
            min-height: 200px;
            font-family: 'Courier New', monospace;
            white-space: pre-wrap;
        }
    </style>
</head>
<body>
    <h1>StreamASR React Hook Test</h1>
    <div id="root"></div>

    <!-- React and ReactDOM -->
    <script crossorigin src="https://unpkg.com/react@18/umd/react.development.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@18/umd/react-dom.development.js"></script>

    <!-- Babel for JSX transformation -->
    <script src="https://unpkg.com/@babel/standalone/babel.min.js"></script>

    <!-- 构建后的React hooks -->
    <script src="../dist/react.js"></script>
    <!-- 构建后的主SDK -->
    <script src="../dist/index.js"></script>

    <script type="text/babel">
        const { useState, useEffect } = React;

        // 模拟React Hook的组件
        function StreamASRTest() {
            const [client, setClient] = useState(null);
            const [isConnected, setIsConnected] = useState(false);
            const [isRecording, setIsRecording] = useState(false);
            const [transcript, setTranscript] = useState('');
            const [error, setError] = useState('');
            const [status, setStatus] = useState('Initializing...');

            useEffect(() => {
                // 测试SDK加载
                if (typeof StreamASR !== 'undefined') {
                    setStatus('SDK loaded successfully');

                    try {
                        const newClient = new StreamASR.StreamASRClient({
                            url: 'ws://localhost:8080/v1/realtime',
                            apiKey: 'test-key',
                            enableLogging: true
                        });

                        setClient(newClient);
                        setupEventListeners(newClient);
                        setStatus('Client created successfully');
                    } catch (err) {
                        setError(err.message);
                        setStatus('Failed to create client');
                    }
                } else {
                    setStatus('SDK not loaded');
                }
            }, []);

            const setupEventListeners = (clientInstance) => {
                clientInstance.on('connectionStateChanged', (state) => {
                    setIsConnected(state.connected);
                    setStatus(state.connected ? 'Connected' : 'Disconnected');
                });

                clientInstance.on('recordingStateChanged', (state) => {
                    setIsRecording(state.isRecording);
                });

                clientInstance.on('transcription', (data) => {
                    setTranscript(prev => data.text + '\\n' + prev);
                });

                clientInstance.on('error', (errorData) => {
                    setError(errorData.message);
                });
            };

            const handleConnect = async () => {
                if (client) {
                    try {
                        await client.connect();
                    } catch (err) {
                        setError(err.message);
                    }
                }
            };

            const handleDisconnect = () => {
                if (client) {
                    client.disconnect();
                }
            };

            const handleToggleRecording = async () => {
                if (client) {
                    try {
                        if (isRecording) {
                            client.stopRecording();
                        } else {
                            await client.startRecording();
                        }
                    } catch (err) {
                        setError(err.message);
                    }
                }
            };

            return (
                <div>
                    <div className="container">
                        <h3>Status</h3>
                        <div className={\`status \${error ? 'error' : 'info'}\`}>
                            {error || status}
                        </div>

                        <div>
                            <strong>Connection:</strong> {isConnected ? 'Connected' : 'Disconnected'} |
                            <strong>Recording:</strong> {isRecording ? 'Recording' : 'Not Recording'}
                        </div>
                    </div>

                    <div className="container">
                        <h3>Controls</h3>
                        <button
                            className="btn-primary"
                            onClick={handleConnect}
                            disabled={isConnected}
                        >
                            Connect
                        </button>
                        <button
                            className="btn-danger"
                            onClick={handleDisconnect}
                            disabled={!isConnected}
                        >
                            Disconnect
                        </button>
                        <button
                            className={isRecording ? "btn-danger" : "btn-primary"}
                            onClick={handleToggleRecording}
                            disabled={!isConnected}
                        >
                            {isRecording ? 'Stop Recording' : 'Start Recording'}
                        </button>
                    </div>

                    <div className="container">
                        <h3>Transcript</h3>
                        <div className="transcript">
                            {transcript || 'No transcript yet...'}
                        </div>
                    </div>
                </div>
            );
        }

        // 渲染组件
        ReactDOM.render(<StreamASRTest />, document.getElementById('root'));
    </script>
</body>
</html>`;

fs.writeFileSync(path.join(testBuildDir, 'react-test.html'), reactTestPage);
console.log('✅ Created React test page');

// 创建一个打包信息文件
const buildInfo = {
    buildTime: new Date().toISOString(),
    version: require('../package.json').version,
    files: {
        'index.js': 'UMD bundle for browsers',
        'index.esm.js': 'ES Module bundle',
        'react.js': 'React hooks bundle'
    },
    tests: [
        'basic-usage.html - Vanilla JS test',
        'react-test.html - React hooks test',
        'test-node.js - Node.js CommonJS test'
    ]
};

fs.writeFileSync(
    path.join(testBuildDir, 'build-info.json'),
    JSON.stringify(buildInfo, null, 2)
);
console.log('✅ Created build info file');

console.log('\\n✨ Build test files created successfully!');
console.log('\\n📁 Test files location: test-build/');
console.log('🌐 To run tests:');
console.log('   npm run serve:test    # Start test server');
console.log('   node test-build/test-node.js  # Run Node.js test');
console.log('\\n🚀 Happy testing!');