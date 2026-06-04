package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Provider      string
	Model         string
	OpenAIKey     string
	GeminiKey     string
	AzureEndpoint string
	AzureKey      string
}

// loadConfig, kullanıcının ev dizinindeki config dosyasını okur.
// Satır formatı: KEY=VALUE
func loadConfig() Config {
	cfg := Config{Provider: "ollama"} // varsayılan
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "term-ai", "config")
	f, err := os.Open(configPath)
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "PROVIDER":
			cfg.Provider = val
		case "MODEL":
			cfg.Model = val
		case "OPENAI_API_KEY":
			cfg.OpenAIKey = val
		case "GEMINI_API_KEY":
			cfg.GeminiKey = val
		case "AZURE_AI_ENDPOINT":
			cfg.AzureEndpoint = val
		case "AZURE_AI_KEY":
			cfg.AzureKey = val
		}
	}
	return cfg
}

// saveConfig, yeni ayarları config dosyasına kaydeder
func saveConfig(cfg Config) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "term-ai")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "config")
	content := fmt.Sprintf("PROVIDER=%s\nMODEL=%s\nOPENAI_API_KEY=%s\nGEMINI_API_KEY=%s\nAZURE_AI_ENDPOINT=%s\nAZURE_AI_KEY=%s\n",
		cfg.Provider, cfg.Model, cfg.OpenAIKey, cfg.GeminiKey, cfg.AzureEndpoint, cfg.AzureKey)
	return os.WriteFile(configPath, []byte(content), 0644)
}
