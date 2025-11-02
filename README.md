# StreamASR - Real-time Speech Recognition Service

<div align="center">

![StreamASR Logo](https://img.shields.io/badge/StreamASR-v0.1.2-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8E.svg)
![License](https://img.shields.io/badge/License-MIT-yellow.svg)
![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)
![Version](https://img.shields.io/badge/Version-Management-green.svg)

**🎯 OpenAI Realtime API Compatible Real-time Speech Recognition Service**

[![GitHub stars](https://img.shields.io/github/stars/go-restream/stt?style=social)](https://github.com/go-restream/stt)

**🌐 Language:** [English](README.md) | [中文](README-zh.md)

</div>

## 📋 Project Overview

**StreamASR** is a high-performance real-time speech recognition service that provides WebSocket interface for real-time audio stream transcription (converting OpenAI standard v1/audio/transcriptions interface to real-time speech recognition service). The project integrates VAD (Voice Activity Detection) and multiple ASR engines, supporting flexible configuration and deployment.

### ✨ Core Features

- **🎤 Real-time Speech Recognition** - Low-latency audio stream processing based on WebSocket
- **🧠 Smart VAD Detection** - Integrated Sherpa-ONNX voice activity detection with automatic audio submission trigger
- **🔄 OpenAI Compatible** - Supports OpenAI-compatible ASR interface with configurable multiple models
- **📊 Structured Logging** - Detailed logging and monitoring based on logrus
- **🐳 Docker Support** - Complete containerized deployment solution
- **🔧 Version Management** - Automated version management and build process
- **🌐 Multi-language SDK** - Provides Go and TypeScript client SDKs

## 🚀 Quick Start

### 📋 Prerequisites

- **Go 1.23+** - Server runtime environment
- **VAD Model File** - Sherpa-ONNX VAD model (silero_vad.onnx)
- **ASR Service** - OpenAI-compatible speech recognition API

### ⚡ Quick Launch

#### Method 1: Using Makefile (Recommended)

```bash
# Clone the project
git clone https://github.com/go-restream/stt.git
cd stt

# Install dependencies and build
make install
make build

# Start the service
make run
```

#### Method 2: Manual Compilation

```bash
# Install dependencies
go mod download

# Build the project
go build -o streamASR main.go

# Start the service
./streamASR -c config.yaml
```

#### Method 3: Docker Deployment

```bash
# Using docker-compose
make docker-deploy

# Or manual build
make docker-build
make docker-compose-up
```

### 🎯 Verify Installation

After the service starts, you can verify it through:

```bash
# View version information
./build/streamASR -v

# Health check
curl http://localhost:8088/health

# Check service status
curl http://localhost:8088/status
```

## 🌐 Web 界面

StreamASR 提供了一个内置的 Web UI 工具，方便用户通过浏览器直接进行实时语音识别测试。

### 🎯 访问 Web UI

启动服务后，在浏览器中访问：

```bash
# 主界面
http://localhost:8088/

# 或者直接访问静态文件
http://localhost:8088/static/index.html
```

### ✨ Web UI 功能特性

- **🎤 实时音频可视化** - 动态显示音频波形和音量级别
- **🔧 配置选项** - 支持采样率选择（16kHz/48kHz）和 VAD 开关
- **⚡ 实时转录** - 实时显示语音识别结果
- **🎨 主题切换** - 支持多种视觉主题（深蓝科技、紫色赛博、绿色矩阵）
- **💾 结果保存** - 支持转录结果的复制和保存
- **🤖 AI 总结** - 集成 AI 功能对转录内容进行智能总结

### 🎮 使用步骤

1. **打开浏览器** 访问 `http://localhost:8088`
2. **配置参数** 选择采样率和 VAD 检测开关
3. **点击开始** 启动语音识别
4. **授权麦克风** 浏览器会请求麦克风权限
5. **开始说话** 实时查看转录结果
6. **保存结果** 使用保存按钮复制转录文本

### 🔧 技术特性

- **WebSocket 连接** - 基于 WebSocket 的低延迟通信
- **自动重连** - 支持断线自动重连机制
- **心跳检测** - 30秒心跳保持连接稳定
- **错误处理** - 完善的错误提示和状态显示

## 🔧 Configuration

### Service Configuration File (config.yaml)

```yaml
# Service port configuration
service_port: "8088"

# OpenAI compatible ASR interface configuration
asr:
  base_url: "http://localhost:3000/v1"        # ASR interface base URL
  api_key: "your-api-key"                    # ASR interface API key
  model: "FireRed-large"                     # ASR model name

# OpenAI compatible LLM interface configuration (optional)
llm:
  base_url: "https://api.deepseek.com/v1"    # LLM interface base URL
  api_key: "your-llm-api-key"                # LLM interface API key
  model: "deepseek-chat"

# Audio configuration
audio:
  enable: true
  save_dir: "./audio"                        # Audio file save directory
  keep_files: 10                             # Keep recent wav file records
  sample_rate: 16000                         # Sample rate (16kHz/48kHz)
  channels: 1                                # Number of channels
  bit_depth: 16                              # Bit depth
  buffer_size: 10                            # 10-second buffer

# VAD configuration
vad:
  enable: true
  model: "./model/silero_vad.onnx"          # VAD model path
  threshold: 0.5                             # Speech detection threshold
  min_silence_duration: 1                    # Minimum silence duration (seconds)
  min_speech_duration: 0.1                   # Minimum speech duration (seconds)
  window_size: 512                           # Window size
  max_speech_duration: 8.0                   # Maximum speech duration (seconds)
  sample_rate: 16000                         # Sample rate
  num_threads: 1                             # Number of threads
  provider: "cpu"                            # Compute provider

# Logging configuration
logging:
  level: "info"                              # Log level
  file: ""                                   # Log file path, empty means output to stderr
  format: "json"                             # Log format: json, text
```

## 🐳 Docker Deployment

### Quick Start with Docker Compose (Recommended)

```bash
# One-command deployment (build and start all services)
make docker-deploy

# Check service status
make docker-ps

# View real-time logs
make docker-compose-logs

# Stop all services
make docker-compose-down
```

### Docker Compose Configuration

The `docker-compose.yml` provides complete service orchestration:

```yaml
version: '3.8'
services:
  streamASR:
    build: .
    ports:
      - "8088:8088"
    volumes:
      - ./config/config.yaml:/app/config/config.yaml:ro
      - ./vad/model:/app/vad/model:ro
      - ./audio:/app/audio
      - ./logs:/app/logs
    environment:
      - VERSION=v0.1.2
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8088/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### Manual Docker Commands

```bash
# Build Docker image
make docker-build

# Run container with volume mounts
make docker-run

# Enter container for debugging
make docker-exec

# View container logs
make docker-logs

# Stop and remove container
make docker-stop
```

### Development Mode

```bash
# Build development image and run interactive container
make docker-debug

# Run in development mode with hot reload
make docker-dev
```

### Dockerfile Features

- **Multi-stage build** - Optimizes image size
- **Version information injection** - Automatically injects version, build time and other information
- **Health check** - Built-in health check mechanism
- **Non-root user** - Secure container runtime environment
- **Production-ready** - Optimized for production deployment

### Container Management

```bash
# List all containers
docker ps -a

# Monitor resource usage
docker stats

# Clean up unused resources
make docker-clean
```

For detailed Docker deployment guide, please refer to: [docs/DOCKER.md](docs/DOCKER.md) | [English Version](docs/DOCKER_EN.md)

## 📋 Command Line Options

```bash
# Display version information
./streamASR -v
./streamASR --version

# Specify configuration file
./streamASR -c config.yaml

# View help information
./streamASR -h
```

## 🧪 Client SDK

### Go SDK

```go
package main

import (
    "streamASR/sdk/golang/client"
)

func main() {
    // Create client
    recognizer := client.NewRecognizer("ws://localhost:8088")

    // Connect and start recognition
    err := recognizer.Connect()
    if err != nil {
        panic(err)
    }
    defer recognizer.Close()

    // Handle audio...
}
```

### TypeScript SDK

```typescript
import { StreamASRClient } from '@streamasr/typescript-sdk';

const client = new StreamASRClient({
    url: 'ws://localhost:8088',
    autoConnect: true
});

// Listen for transcription results
client.on('transcription', (result) => {
    console.log('Recognition result:', result.text);
});
```

## 📊 Monitoring and Logging

### Structured Logging

The service uses logrus for structured logging:

```json
{
  "component": "mont_srv_status",
  "action": "health_check_status",
  "version": "v0.1.2-171f62c",
  "build_time": "2025-11-02T05:24:39Z",
  "git_commit": "171f62c",
  "level": "info",
  "msg": "✔ Starting StreamASR v0.1.2-171f62c with config: config.yaml"
}
```

### Health Check

```bash
# Basic health check
curl http://localhost:8088/health

# Return example
{
  "status": "healthy",
  "version": "v0.1.2-171f62c",
  "uptime": "2h30m15s",
  "asr_engine": "available"
}
```

## 🏗️ Project Structure

```
streamASR_realtime/
├── config/                      # Configuration files
│   ├── config.go               # Configuration structure definition
│   └── config.yaml             # Default configuration file
├── internal/                    # Internal packages
│   ├── service/                # Service layer
│   │   ├── apiserver.go        # HTTP API server
│   │   ├── audio_utils.go      # Audio processing utilities
│   │   ├── openai_events.go    # OpenAI event handlers
│   │   ├── openai_websocket.go # WebSocket handler
│   │   ├── recognizer.go       # Speech recognition core
│   │   ├── session_manager.go  # Session manager
│   │   └── vad_integration.go  # VAD integration
│   └── version/                # Version information
│       └── version.go         # Version management
├── llm/                         # LLM integration
│   ├── asr.go                  # ASR service integration
│   ├── asr_test.go             # ASR service tests
│   ├── openai.go               # OpenAI API client
│   ├── openai_test.go          # OpenAI API tests
│   └── types.go                # Common types
├── pkg/                        # Public packages
│   ├── health/                 # Health check
│   │   └── asr_health.go       # ASR health check implementation
│   ├── logger/                 # Logging utilities
│   │   ├── custom_formatter.go # Custom log formatter
│   │   └── logger.go           # Logger implementation
│   ├── resampler/              # Audio resampling
│   │   └── resampler.go        # Audio resampler implementation
│   └── wav/                    # WAV file processing
│       ├── reader.go           # WAV file reader
│       ├── wav.go              # WAV utilities
│       ├── wav_test.go         # WAV tests
│       └── writer.go           # WAV file writer
├── sdk/                        # Client SDKs
│   ├── golang/                 # Go SDK
│   │   ├── client/             # Client implementation
│   │   ├── cmd/                # Command line tools
│   │   ├── docs/               # Go SDK documentation
│   │   ├── examples/           # Usage examples
│   │   ├── pkg/                # Go SDK packages
│   │   └── README.md           # Go SDK readme
│   └── typescript/             # TypeScript SDK
│       ├── docs/               # TypeScript SDK documentation
│       ├── src/                # TypeScript source code
│       ├── test-build/         # Test build files
│       ├── dist/               # Compiled distribution
│       └── README.md           # TypeScript SDK readme
├── vad/                        # VAD related
│   ├── model/                  # VAD model files
│   │   └── silero_vad.onnx     # Silero VAD model
│   └── vad.go                  # VAD detector implementation
├── docs/                       # Project documentation
│   ├── CHANGELOG.md            # Changelog (Chinese)
│   ├── CHANGELOG_EN.md         # Changelog (English)
│   ├── DOCKER.md               # Docker deployment guide (Chinese)
│   ├── DOCKER_EN.md            # Docker deployment guide (English)
│   ├── openai_realtime_api.md  # OpenAI Realtime API reference
│   ├── realtime_ws_events_reference.md # WebSocket events reference
│   ├── realtime_ws_flow.md     # WebSocket flow documentation
│   ├── troubleshooting.md      # Troubleshooting guide
│   ├── VERSION.md              # Version management documentation (Chinese)
│   └── VERSION_EN.md           # Version management documentation (English)
├── static/                     # Web UI static files
│   ├── favicon.ico             # Favicon
│   ├── index.html              # Main web interface
│   ├── script.js               # Web UI JavaScript
│   └── style.css               # Web UI styles
├── samples/                    # Sample files
│   └── sample.wav              # Sample audio file
├── openspec/                   # OpenSpec change management
│   ├── changes/                # Change specifications
│   ├── specs/                  # Technical specifications
│   └── project.md              # Project configuration
├── build/                      # Build output directory (generated)
├── node_modules/               # Node.js dependencies (generated)
├── main.go                     # Application entry point
├── go.mod                      # Go module definition
├── go.sum                      # Go dependency checksums
├── package.json                # Node.js package configuration
├── package-lock.json           # Node.js dependency lock
├── config.yaml                 # Main configuration file
├── Dockerfile                  # Docker build file
├── docker-compose.yml          # Docker Compose configuration
├── Makefile                    # Build scripts
├── VERSION                     # Version file
├── README.md                   # Project documentation (English)
├── README-zh.md                # Project documentation (Chinese)
├── README-en.md                # Project documentation (English alternative)
├── LICENSE                     # License file
├── .dockerignore               # Docker ignore file
├── .editorconfig               # Editor configuration
├── .gitignore                  # Git ignore file
└── CLAUDE.md                   # Claude AI assistant instructions
```

## 🔧 Version Management

The project adopts semantic version management and supports automated version releases:

```bash
# View current version
make version

# Version upgrade
make version-bump-patch    # v0.1.2 -> v0.1.3
make version-bump-minor    # v0.1.2 -> v0.2.0
make version-bump-major    # v0.1.2 -> v1.0.0

# Create Git tag
make tag

# Build Docker image
make docker-build          # Generate streamasr:latest and streamasr:v0.1.2
```

For detailed version management guide, please refer to: [docs/VERSION.md](docs/VERSION.md)

## 🛠️ Development Guide

### Development Environment Setup

```bash
# Clone project
git clone https://github.com/go-restream/stt.git
cd stt

# Install dependencies
make install

# Run tests
make test

# Build
make build

# Run
make run
```

### Development Mode

```bash
# Docker development mode
make docker-debug

# View logs
make docker-logs

# Enter container for debugging
make docker-exec
```

### Testing

```bash
# Run unit tests
make test

# Run integration tests
go test ./...
```

## 🐛 Troubleshooting

### Common Issues

1. **VAD Model File Missing**
   ```bash
   # Ensure VAD model file exists
   ls -la vad/model/silero_vad.onnx
   ```

2. **ASR Service Connection Failed**
   ```bash
   # Check ASR service configuration
   curl -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model":"FireRed-large","file":"..."}' \
        $ASR_BASE_URL/audio/transcriptions
   ```

3. **Port Occupied**
   ```bash
   # Check port occupation
   lsof -i :8088

   # Modify port in configuration file
   vim config.yaml
   ```

### Debug Mode

Enable verbose logging:

```bash
# Modify configuration file
vim config.yaml
# Set logging.level: "debug"

# Or set environment variable
export LOG_LEVEL=debug
./streamASR
```

## 📊 Performance Metrics

- **Response Latency**: < 500ms end-to-end recognition latency
- **Concurrency Support**: Supports multiple concurrent WebSocket connections
- **Audio Processing**: Supports 16kHz/48kHz sample rates
- **VAD Latency**: < 100ms voice activity detection latency

## 🤝 Contributing

We welcome community contributions! Please follow these steps:

1. Fork the project repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

### Development Guidelines

- Follow Go coding standards
- Add unit tests
- Update relevant documentation
- Pass all CI checks

## 📞 Support & Help

### 📚 Complete Documentation

- **[Docker Deployment Guide](docs/DOCKER.md)** - Complete Docker deployment instructions
- **[Version Management Documentation](docs/VERSION.md)** - Version management specifications and usage
- **[Changelog](docs/CHANGELOG.md)** - Detailed version change records

### 🆘 Getting Help

| Method | Description | Response Time |
|--------|-------------|---------------|
| **GitHub Issues** | Bug reports and feature requests | 24-48 hours |
| **GitHub Discussions** | Technical discussions and Q&A | Community response |

---

## 🏷️ Version Updates

### v0.1.2 (2025-11-02)

#### ✨ New Features
- **🏷️ Version Management System** - Complete version management and build process
- **🐳 Docker Support** - Complete containerized deployment solution
- **📋 Makefile Integration** - Automated build and deployment scripts
- **📖 Documentation Enhancement** - Detailed deployment and development documentation

#### 🔧 Technical Improvements
- **🔧 Project Structure Optimization** - Clearer code organization and module division
- **📝 Logging Enhancement** - Startup logs include version information
- **🛠️ Build Process** - Support for automatic version information injection

### v0.1.1

#### ✨ New Features
- **🎤 Real-time Speech Recognition** - WebSocket-based audio stream processing
- **🧠 VAD Integration** - Sherpa-ONNX voice activity detection
- **🔄 ASR Interface** - OpenAI-compatible speech recognition API
- **📊 Health Check** - Service status monitoring interface

---

## 🎯 Summary

**StreamASR** is a feature-complete, easy-to-deploy real-time speech recognition service. Through Docker containerization, version management system, and comprehensive documentation, it provides a reliable speech recognition solution for production environments.

<div align="center">

**⭐ If this project helps you, please give us a Star!**

🎯 **StreamASR - Making Speech Recognition Simple and Powerful**

</div>