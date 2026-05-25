package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	OpenAIBaseURL  string
	OpenAIAPIKey   string
	OpenAIModel    string
	APIKey         string
	Languages      []Language
	Debug          bool
	RequestTimeout int
	LLM            LLMParams
}

type LLMParams struct {
	Temperature       *float64
	TopP              *float64
	TopK              *int
	RepetitionPenalty *float64
	MaxTokens         int
}

type Language struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Targets []string `json:"targets"`
}

// Hy-MT2 supported languages (also covers most general-purpose LLMs)
var defaultLanguages = []Language{
	{Code: "ar", Name: "Arabic"},
	{Code: "bn", Name: "Bengali"},
	{Code: "bo", Name: "Tibetan"},
	{Code: "cs", Name: "Czech"},
	{Code: "de", Name: "German"},
	{Code: "en", Name: "English"},
	{Code: "es", Name: "Spanish"},
	{Code: "fa", Name: "Persian"},
	{Code: "fr", Name: "French"},
	{Code: "gu", Name: "Gujarati"},
	{Code: "he", Name: "Hebrew"},
	{Code: "hi", Name: "Hindi"},
	{Code: "id", Name: "Indonesian"},
	{Code: "it", Name: "Italian"},
	{Code: "ja", Name: "Japanese"},
	{Code: "kk", Name: "Kazakh"},
	{Code: "km", Name: "Khmer"},
	{Code: "ko", Name: "Korean"},
	{Code: "mn", Name: "Mongolian"},
	{Code: "mr", Name: "Marathi"},
	{Code: "ms", Name: "Malay"},
	{Code: "my", Name: "Burmese"},
	{Code: "nl", Name: "Dutch"},
	{Code: "pl", Name: "Polish"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "ru", Name: "Russian"},
	{Code: "ta", Name: "Tamil"},
	{Code: "te", Name: "Telugu"},
	{Code: "th", Name: "Thai"},
	{Code: "tl", Name: "Filipino"},
	{Code: "tr", Name: "Turkish"},
	{Code: "ug", Name: "Uyghur"},
	{Code: "uk", Name: "Ukrainian"},
	{Code: "ur", Name: "Urdu"},
	{Code: "vi", Name: "Vietnamese"},
	{Code: "yue", Name: "Cantonese"},
	{Code: "zh", Name: "Chinese"},
	{Code: "zh-Hant", Name: "Traditional Chinese"},
}

func Load() *Config {
	langs := buildLanguages()
	timeout := 60
	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}

	return &Config{
		Port:           getEnv("PORT", "5000"),
		OpenAIBaseURL:  getEnv("OPENAI_BASE_URL", "http://localhost:11434/v1"),
		OpenAIAPIKey:   getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:    getEnv("OPENAI_MODEL", "Hy-MT2-7B"),
		APIKey:         os.Getenv("API_KEY"),
		Languages:      langs,
		Debug:          os.Getenv("DEBUG") == "true",
		RequestTimeout: timeout,
		LLM:            loadLLMParams(),
	}
}

func loadLLMParams() LLMParams {
	p := LLMParams{
		MaxTokens: 4096,
	}
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.MaxTokens = n
		}
	}
	if v := os.Getenv("LLM_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.Temperature = &f
		}
	}
	if v := os.Getenv("LLM_TOP_P"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.TopP = &f
		}
	}
	if v := os.Getenv("LLM_TOP_K"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.TopK = &n
		}
	}
	if v := os.Getenv("LLM_REPETITION_PENALTY"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.RepetitionPenalty = &f
		}
	}
	return p
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func buildLanguages() []Language {
	codes := defaultLanguages
	if v := os.Getenv("LANGUAGES"); v != "" {
		codes = parseLanguageCodes(v)
	}

	allCodes := make([]string, len(codes))
	for i, l := range codes {
		allCodes[i] = l.Code
	}

	result := make([]Language, len(codes))
	for i, l := range codes {
		targets := make([]string, 0, len(allCodes)-1)
		for _, c := range allCodes {
			if c != l.Code {
				targets = append(targets, c)
			}
		}
		result[i] = Language{Code: l.Code, Name: l.Name, Targets: targets}
	}
	return result
}

func parseLanguageCodes(s string) []Language {
	parts := strings.Split(s, ",")
	langs := make([]Language, 0, len(parts))
	for _, p := range parts {
		code := strings.TrimSpace(p)
		if code == "" {
			continue
		}
		name := code
		for _, def := range defaultLanguages {
			if def.Code == code {
				name = def.Name
				break
			}
		}
		langs = append(langs, Language{Code: code, Name: name})
	}
	return langs
}
