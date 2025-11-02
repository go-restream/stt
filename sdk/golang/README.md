# ASR SDK (OpenAI Realtime API) 使用说明

## 概述

本SDK是基于OpenAI Realtime API标准的全新实现，提供实时语音识别功能。该版本完全重写了原有协议，与更新的后端服务完全兼容。

**重要变更**: 此版本不再兼容旧协议，请参考[迁移指南](#迁移指南)。

## 功能特性

- ✅ **OpenAI Realtime API兼容**: 完全支持OpenAI实时API事件系统
- 🎯 **实时音频流处理**: 支持WebSocket实时音频传输
- 🔄 **智能会话管理**: 完整的会话生命周期管理
- 🎵 **多音频格式支持**: 16kHz/48kHz采样率，单声道/立体声
- 💬 **Base64音频编码**: 标准Base64音频数据编码/解码
- 📊 **详细事件回调**: 支持所有OpenAI事件类型的回调
- 💗 **心跳机制**: 自动连接健康监控和恢复
- 🛡️ **错误处理**: 完善的错误处理和恢复机制

## 快速开始

### 简单使用方式

```go
package main

import (
	"fmt"
	"log"
	"time"

	asr "streamASR/sdk/golang/client"
)

func main() {
	// 1. 创建语音识别器
	recognizer, err := asr.CreateRecognizer("ws://localhost:8088/ws", "zh-CN")
	if err != nil {
		log.Fatal(err)
	}

	// 2. 启动识别会话
	if err := recognizer.Start(); err != nil {
		log.Fatal(err)
	}
	defer recognizer.Stop()

	// 3. 发送音频数据进行识别
	audioData := []byte{/* PCM音频数据 */}
	if err := recognizer.Write(audioData); err != nil {
		log.Printf("发送音频失败: %v", err)
	}

	// 4. 等待识别结果
	time.Sleep(5 * time.Second)
}
```

## 实时音频流处理
```go
// 持续写入音频数据
func streamAudio(recognizer *asr.Recognizer) {
	for {
		select {
		case data := <-audioSource.Chan():
			if err := recognizer.Write(data); err != nil {
				// 50ms超时自动跳过，持续错误需处理
				if err != asr.ErrWriteTimeout {
					log.Printf("写入失败: %v", err)
					return
				}
			}
		case <-doneChan:
			return
		}
	}
}
```

## 配置参数
| 参数 | 类型 | 说明 | 默认值 |
|------|------|------|-------|
| AudioFormat | int | 音频格式(1=PCM) | 1 |
| SampleRate | int | 采样率(Hz) | 16000 |
| Language | string | 识别语言 | "zh-CN" |
| MaxRetries | int | 最大重试次数 | 3 |
| RetryDelay | time.Duration | 重试间隔 | 2s |

## 事件回调
实现`RecognitionListener`接口处理识别事件：
```go
type RecognitionListener struct{}

func (l *RecognitionListener) OnRecognitionStart(resp *asr.RecognitionResponse) {
	// 识别开始
}

func (l *RecognitionListener) OnSentenceBegin(resp *asr.RecognitionResponse) {
	// 句子开始
}

func (l *RecognitionListener) OnRecognitionResultChange(resp *asr.RecognitionResponse) {
	// 识别结果更新
	log.Printf("Partial: %s", resp.Result.Text)
}

func (l *RecognitionListener) OnSentenceEnd(resp *asr.RecognitionResponse) {
	// 句子结束
	log.Printf("Final: %s", resp.Result.Text)
}

func (l *RecognitionListener) OnRecognitionComplete(resp *asr.RecognitionResponse) {
	// 识别完成
}

func (l *RecognitionListener) OnFail(resp *asr.RecognitionResponse, err error) {
	// 识别失败
}
```

## 音频要求
- 实时流格式: PCM原始数据([]byte)
- 文件测试格式: WAV(PCM)
- 采样率: 16kHz或48kHz(自动重采样)
- 位深: 16bit
- 声道: 单声道或立体声(自动转换)

## 注意事项
1. Write()方法设计用于实时流，50ms超时自动跳过
2. processAudioFile仅用于本地文件测试
3. 网络连接稳定以保证实时识别
4. 大流量场景建议控制写入频率
5. 48kHz音频会自动重采样为16kHz

## 示例代码
参考`cmd/main.go`和`cmd/listener.go`中的完整实现