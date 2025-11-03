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
- **🔇 AI Noise Reduction** - Built-in denoiser using GTCRN model for enhanced speech recognition in noisy environments
- **🔄 OpenAI Compatible** - Supports OpenAI-compatible ASR interface with configurable multiple models
- **📊 Structured Logging** - Detailed logging and monitoring based on logrus
- **🐳 Docker Support** - Complete containerized deployment solution
- **🔧 Version Management** - Automated version management and build process
- **🌐 Multi-language SDK** - Provides Go and TypeScript client SDKs

## 🚀 Quick Start

### 📋 Prerequisites

- **Go 1.23+** - Server runtime environment
- **VAD Model File** - Sherpa-ONNX VAD model (silero_vad.onnx)
- **Denoiser Model File** - Sherpa-ONNX GTCRN model (gtcrn_simple.onnx)
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

## 🌐 Web Interface

StreamASR provides a built-in Web UI tool that allows users to perform real-time speech recognition testing directly through their browser.

### 📸 Web UI Preview

![StreamASR Web UI](asrTool.png)

### 🎯 Accessing Web UI

After starting the service, visit in your browser:

```bash
# Main interface
http://localhost:8088/

# Or directly access static files
http://localhost:8088/static/index.html
```

### ✨ Web UI Features

- **🎤 Real-time Audio Visualization** - Dynamic display of audio waveforms and volume levels
- **🔧 Configuration Options** - Support for sample rate selection (16kHz/48kHz) and VAD toggle
- **⚡ Real-time Transcription** - Real-time display of speech recognition results
- **🎨 Theme Switching** - Support for multiple visual themes (Deep Blue Tech, Purple Cyber, Green Matrix)
- **💾 Result Saving** - Support for copying and saving transcription results
- **🤖 AI Summary** - Integrated AI functionality for intelligent summarization of transcription content

### 🎮 Usage Steps

1. **Open Browser** Visit `http://localhost:8088`
2. **Configure Parameters** Select sample rate and VAD detection toggle
3. **Click Start** Launch speech recognition
4. **Authorize Microphone** Browser will request microphone permission
5. **Start Speaking** View real-time transcription results
6. **Save Results** Use save button to copy transcription text

### 🔧 Technical Features

- **WebSocket Connection** - Low-latency communication based on WebSocket
- **Auto Reconnection** - Support for automatic reconnection on disconnection
- **Heartbeat Detection** - 30-second heartbeat to maintain stable connection
- **Error Handling** - Comprehensive error prompts and status display

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

# Denoiser configuration (AI Noise Reduction)
denoiser:
  enable: true                               # Enable/disable denoiser
  model: "./model/gtcrn_simple.onnx"         # GTCRN denoiser model path
  sample_rate: 16000                         # Sample rate
  num_threads: 1                             # Number of threads
  debug: 0                                   # Debug level (0-3)
  bypass_for_testing: false                  # Bypass denoiser for testing
  max_processing_time_ms: 50                 # Maximum processing time (ms)

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
      - ./build/model:/app/model:ro
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
   ls -la model/silero_vad.onnx
   ```

2. **Denoiser Model File Missing**
   ```bash
   # Ensure denoiser model file exists
   ls -la model/gtcrn_simple.onnx
   ```

3. **ASR Service Connection Failed**
   ```bash
   # Check ASR service configuration
   curl -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model":"FireRed-large","file":"..."}' \
        $ASR_BASE_URL/audio/transcriptions
   ```

4. **Port Occupied**
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
- **Denoiser Latency**: < 20ms additional processing time for noise reduction
- **Noise Reduction**: Improved ASR accuracy in noisy environments

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

### v0.1.2 (2025-11-03)

#### ✨ New Features
- **🔇 AI Noise Reduction** - Built-in denoiser using GTCRN model for enhanced speech recognition
- **🏷️ Version Management System** - Complete version management and build process
- **🐳 Docker Support** - Complete containerized deployment solution
- **📋 Makefile Integration** - Automated build and deployment scripts
- **📖 Documentation Enhancement** - Detailed deployment and development documentation

#### 🔧 Technical Improvements
- **🎯 Audio Pipeline Enhancement** - Integrated denoiser between VAD and ASR processing
- **🔧 Project Structure Optimization** - Clearer code organization and module division
- **📝 Logging Enhancement** - Startup logs include version information
- **🛠️ Build Process** - Support for automatic version information injection
- **🧪 Comprehensive Testing** - Unit tests for denoiser functionality

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