package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-restream/stt/llm"
	"github.com/go-restream/stt/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var openAIService *OpenAIService

func WsServiceRun(srvPort string, configPath string) {
	gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

	openAIService = NewOpenAIService(DefaultOpenAIConfig(), configPath)

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.File("./static/favicon.ico")
	})

	
		r.GET("/v1/realtime", func(c *gin.Context) {
		openAIService.HandleOpenAIWebSocket(c)
	})

		r.POST("/v1/chat/completions", handleChatCompletion)

		r.GET("/v1/health", func(c *gin.Context) {
		health := gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"service":   "streamASR",
		}

				if openAIService != nil {
			health["openai_service"] = "available"
		} else {
			health["openai_service"] = "unavailable"
		}

		c.JSON(http.StatusOK, health)
	})

		r.GET("/v1/sessions/stats", func(c *gin.Context) {
		if openAIService != nil {
			stats := openAIService.GetSessionStats()
			c.JSON(http.StatusOK, stats)
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OpenAI service not initialized"})
		}
	})

	logger.WithFields(logrus.Fields{
		"component": "ws_engine_core ",
		"action":    "service_running",
		"port":      "🌈"+srvPort,
	}).Info("✔ WebSocket service running")

	logger.WithFields(logrus.Fields{
		"component": "svc_openai_api ",
		"action":    "realtime_api_available",
		"endpoint":  "/v1/realtime",
	}).Info("✔ OpenAI Realtime API available")

	r.Run(":" + srvPort)
}

// handleChatCompletion handles OpenAI-compatible chat completion requests 
func handleChatCompletion(c *gin.Context) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OPENAI_API_KEY not set"})
		return
	}

	var req llm.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isStream := req.Stream
	if isStream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
	}

	client := llm.NewClient(apiKey)
	if isStream {
		flusher, _ := c.Writer.(http.Flusher)
		req.Stream = true
		
		respChan := make(chan string)
		go func() {
			_, err := client.CreateChatCompletion(context.Background(), req)
			if err != nil {
				c.SSEvent("error", gin.H{"error": err.Error()})
				return
			}
				for i := 0; i < 5; i++ {
				respChan <- fmt.Sprintf("AI总结内容片段 %d", i+1)
				time.Sleep(500 * time.Millisecond)
			}
			close(respChan)
		}()

		for chunk := range respChan {
			c.SSEvent("message", gin.H{"content": chunk})
			flusher.Flush()
		}
	} else {
				resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}


