package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	// Config dosyasını yükle (varsa)
	cfg := loadConfig()

	// CLI Parametrelerini Tanımla (config'in üzerine yazar)
	providerFlag := flag.String("provider", cfg.Provider, "Yapay zeka sağlayıcı: ollama, openai, vllm, azure")
	modelFlag := flag.String("model", cfg.Model, "Kullanılacak model adı (boş bırakılırsa otomatik algılanır)")
	listFlag := flag.Bool("list", false, "Ollama'daki mevcut modelleri listele")

	flag.Parse()

	// Ollama model listesi göster
	if *listFlag {
		fmt.Println("🔍 Ollama'daki yüklü modeller:")
		resp, err := http.Get("http://localhost:11434/api/tags")
		if err != nil {
			fmt.Println("❌ Ollama çalışmıyor. Başlatmak için: ollama serve")
			os.Exit(1)
		}
		defer resp.Body.Close()
		var tags ollamaTagsResp
		json.NewDecoder(resp.Body).Decode(&tags)
		for i, m := range tags.Models {
			fmt.Printf("  %d) %s\n", i+1, m.Name)
		}
		return
	}

	var provider AIProvider
	var activeModel string

	// Sağlayıcı Seçimi ve Yapılandırması
	switch strings.ToLower(*providerFlag) {
	case "ollama":
		ollamaURL := "http://localhost:11434"
		provider = &OllamaProvider{BaseURL: ollamaURL}

		if *modelFlag != "" {
			activeModel = *modelFlag
		} else {
			// Otomatik model algılama
			activeModel = autoDetectOllamaModel(ollamaURL)
			if activeModel == "" {
				fmt.Println("❌ Ollama'da yüklü model bulunamadı.")
				os.Exit(1)
			}
		}

	case "openai":
		apiKey := cfg.OpenAIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			fmt.Println("⚠️ Uyarı: OpenAI API Key bulunamadı. Arayüz üzerinden (Ayarlar) girebilirsiniz.")
		}
		provider = &OpenAICompatibleProvider{BaseURL: "https://api.openai.com/v1", APIKey: apiKey}
		activeModel = "gpt-4o-mini"
		if *modelFlag != "" {
			activeModel = *modelFlag
		}

	case "gemini":
		apiKey := cfg.GeminiKey
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		if apiKey == "" {
			fmt.Println("⚠️ Uyarı: Gemini API Key bulunamadı. Arayüz üzerinden (Ayarlar) girebilirsiniz.")
		}
		provider = &OpenAICompatibleProvider{BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", APIKey: apiKey}
		activeModel = "gemini-2.5-flash"
		if *modelFlag != "" {
			activeModel = *modelFlag
		}

	case "vllm":
		provider = &OpenAICompatibleProvider{BaseURL: "http://localhost:8000/v1", APIKey: ""}
		activeModel = "mistralai/Mistral-7B-Instruct"
		if *modelFlag != "" {
			activeModel = *modelFlag
		}

	case "azure":
		apiKey := cfg.AzureKey
		if apiKey == "" {
			apiKey = os.Getenv("AZURE_AI_KEY")
		}
		endpoint := cfg.AzureEndpoint
		if endpoint == "" {
			endpoint = os.Getenv("AZURE_AI_ENDPOINT")
		}
		
		if apiKey == "" || endpoint == "" {
			fmt.Println("⚠️ Uyarı: Azure Endpoint veya Key eksik. Arayüz üzerinden (Ayarlar) girebilirsiniz.")
		}
		provider = &OpenAICompatibleProvider{BaseURL: endpoint, APIKey: apiKey}
		activeModel = *modelFlag

	default:
		fmt.Printf("❌ Geçersiz sağlayıcı: '%s'\n   Geçerli seçenekler: ollama, openai, gemini, vllm, azure\n", *providerFlag)
		os.Exit(1)
	}

	// TUI (ui.go) silindi, yerine modern web sunucusunu başlat
	appState := &AppState{
		Provider:     provider,
		ActiveModel:  activeModel,
		ProviderName: *providerFlag,
	}
	startServer(appState)
}
