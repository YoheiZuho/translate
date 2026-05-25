package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yoheizuho/translate/config"
	"github.com/yoheizuho/translate/llm"
)

type Handler struct {
	cfg    *config.Config
	client *llm.Client
}

func New(cfg *config.Config, client *llm.Client) *Handler {
	return &Handler{cfg: cfg, client: client}
}

// POST /translate
func (h *Handler) Translate(c *gin.Context) {
	var req struct {
		Q      string `form:"q" json:"q" binding:"required"`
		Source string `form:"source" json:"source" binding:"required"`
		Target string `form:"target" json:"target" binding:"required"`
		Format string `form:"format" json:"format"`
		APIKey string `form:"api_key" json:"api_key"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.isValidAPIKey(req.APIKey) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
		return
	}

	if !h.isValidTarget(req.Target) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "target language '" + req.Target + "' is not supported",
		})
		return
	}

	sourceName := languageCodeToName(req.Source, h.cfg.Languages)
	targetName := languageCodeToName(req.Target, h.cfg.Languages)

	translated, err := h.client.Translate(c.Request.Context(), req.Q, sourceName, targetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Translation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"translatedText": translated})
}

// GET /languages
func (h *Handler) Languages(c *gin.Context) {
	c.JSON(http.StatusOK, h.cfg.Languages)
}

// POST /detect
func (h *Handler) Detect(c *gin.Context) {
	var req struct {
		Q      string `form:"q" json:"q" binding:"required"`
		APIKey string `form:"api_key" json:"api_key"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.isValidAPIKey(req.APIKey) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
		return
	}

	code, confidence, err := h.client.DetectLanguage(c.Request.Context(), req.Q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Detection failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, []gin.H{
		{"confidence": confidence, "language": code},
	})
}

// GET /frontend/settings
func (h *Handler) FrontendSettings(c *gin.Context) {
	apiKeyRequired := h.cfg.APIKey != ""
	c.JSON(http.StatusOK, gin.H{
		"keyRequired":      apiKeyRequired,
		"suggestions":      false,
		"filesTranslation": false,
		"supportedFiles":   []string{},
		"language": gin.H{
			"source": gin.H{"code": "en", "name": "English"},
			"target": gin.H{"code": "ja", "name": "Japanese"},
		},
	})
}

// GET /spec
func (h *Handler) Spec(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"openapi": "3.0.0",
		"info": gin.H{
			"title":   "LibreTranslate-compatible LLM Translation API",
			"version": "1.0.0",
		},
	})
}

func (h *Handler) isValidAPIKey(provided string) bool {
	if h.cfg.APIKey == "" {
		return true
	}
	return provided == h.cfg.APIKey
}

func (h *Handler) isValidTarget(code string) bool {
	for _, l := range h.cfg.Languages {
		if l.Code == code {
			return true
		}
	}
	return false
}

func languageCodeToName(code string, langs []config.Language) string {
	if code == "auto" {
		return "auto"
	}
	code = strings.ToLower(strings.TrimSpace(code))
	for _, l := range langs {
		if l.Code == code {
			return l.Name
		}
	}
	// Fall back to code if not found
	return code
}
