package handlers

import (
	"encoding/json"
	"fmt"
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
	texts, source, target, apiKey, err := parseTranslateRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !h.isValidAPIKey(apiKey) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
		return
	}

	if !h.isValidTarget(target) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "target language '" + target + "' is not supported",
		})
		return
	}

	sourceName := languageCodeToName(source, h.cfg.Languages)
	targetName := languageCodeToName(target, h.cfg.Languages)

	translatedTexts := make([]string, 0, len(texts))
	var detectedLanguages []gin.H

	for _, text := range texts {
		translated, err := h.client.Translate(c.Request.Context(), text, sourceName, targetName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Translation failed: " + err.Error()})
			return
		}
		translatedTexts = append(translatedTexts, translated)

		if source == "auto" {
			code, confidence, err := h.client.DetectLanguage(c.Request.Context(), text)
			if err != nil {
				code = ""
				confidence = 0
			}
			detectedLanguages = append(detectedLanguages, gin.H{
				"language":   code,
				"confidence": confidence,
			})
		}
	}

	resp := gin.H{"translatedText": translatedTexts}
	if detectedLanguages != nil {
		resp["detectedLanguage"] = detectedLanguages
	}
	c.JSON(http.StatusOK, resp)
}

func parseTranslateRequest(c *gin.Context) (texts []string, source, target, apiKey string, err error) {
	if strings.Contains(c.ContentType(), "application/json") {
		var body struct {
			Q      json.RawMessage `json:"q"`
			Source string          `json:"source"`
			Target string          `json:"target"`
			APIKey string          `json:"api_key"`
		}
		if err = c.ShouldBindJSON(&body); err != nil {
			return
		}
		source, target, apiKey = body.Source, body.Target, body.APIKey
		texts, err = parseQField(body.Q)
	} else {
		var form struct {
			Q      string `form:"q" binding:"required"`
			Source string `form:"source" binding:"required"`
			Target string `form:"target" binding:"required"`
			APIKey string `form:"api_key"`
		}
		if err = c.ShouldBind(&form); err != nil {
			return
		}
		texts = []string{form.Q}
		source, target, apiKey = form.Source, form.Target, form.APIKey
	}
	if err == nil && source == "" {
		err = fmt.Errorf("source is required")
	}
	if err == nil && target == "" {
		err = fmt.Errorf("target is required")
	}
	return
}

func parseQField(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("q is required")
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == "" {
			return nil, fmt.Errorf("q cannot be empty")
		}
		return []string{s}, nil
	}
	var arr []string
	if json.Unmarshal(raw, &arr) == nil {
		if len(arr) == 0 {
			return nil, fmt.Errorf("q cannot be empty")
		}
		return arr, nil
	}
	return nil, fmt.Errorf("q must be a string or array of strings")
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
