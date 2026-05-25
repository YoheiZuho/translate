package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yoheizuho/translate/config"
	"github.com/yoheizuho/translate/handlers"
	"github.com/yoheizuho/translate/llm"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	client := llm.NewClient(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.RequestTimeout, cfg.LLM)
	h := handlers.New(cfg, client)

	r := gin.Default()

	// CORS for Mastodon and other web clients
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/translate", h.Translate)
	r.GET("/languages", h.Languages)
	r.POST("/detect", h.Detect)
	r.GET("/frontend/settings", h.FrontendSettings)
	r.GET("/spec", h.Spec)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":        "LibreTranslate-compatible LLM Translation API",
			"description": "LLM-powered translation server with LibreTranslate API compatibility",
			"endpoints": []string{
				"POST /translate",
				"GET  /languages",
				"POST /detect",
				"GET  /frontend/settings",
			},
		})
	})

	addr := ":" + cfg.Port
	log.Printf("Starting server on %s (model: %s, backend: %s)", addr, cfg.OpenAIModel, cfg.OpenAIBaseURL)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
