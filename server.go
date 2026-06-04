package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type AppState struct {
	Provider     AIProvider
	ActiveModel  string
	ProviderName string
	mu           sync.RWMutex
}

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func startServer(appState *AppState) {
	// Statik dosyaları /static/ adresi üzerinden sun
	http.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// Ayarlar API'si
	http.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		handleSettingsAPI(w, r, appState)
	})

	// Ollama Modellerini Listeleme API'si
	http.HandleFunc("/api/ollama/models", func(w http.ResponseWriter, r *http.Request) {
		resp, err := http.Get("http://localhost:11434/api/tags")
		if err != nil {
			http.Error(w, "Ollama çalışmıyor", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		io.Copy(w, resp.Body)
	})

	// Websocket endpointleri
	http.HandleFunc("/ws/terminal", handleTerminalWebsocket)
	http.HandleFunc("/ws/ai", func(w http.ResponseWriter, r *http.Request) {
		handleAIWebsocket(w, r, appState)
	})

	fmt.Println("=====================================================")
	fmt.Println("🌐 Web Arayüzü Başlatılıyor: http://localhost:8080/static/index.html")
	fmt.Println("=====================================================")
	
	// Varsayılan tarayıcıda otomatik aç
	go exec.Command("xdg-open", "http://localhost:8080/static/index.html").Start()

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Sunucu başlatılamadı (8080 portu dolu olabilir): %v", err)
	}
}

func handleTerminalWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Bash'i PTY (Sanal Terminal) ile başlat
	cmd := exec.Command("bash")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return
	}
	defer func() { _ = ptmx.Close() }()

	// PTY'den (Gerçek Linux'tan) oku -> WebSocket (Tarayıcıya) yaz
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			conn.WriteMessage(websocket.TextMessage, buf[:n])
		}
	}()

	// WebSocket'ten (Tarayıcıdan) oku -> PTY'ye (Gerçek Linux'a) yaz
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		ptmx.Write(msg)
	}
}

func handleAIWebsocket(w http.ResponseWriter, r *http.Request, appState *AppState) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	appState.mu.RLock()
	providerName := appState.ProviderName
	activeModel := appState.ActiveModel
	appState.mu.RUnlock()

	// Başlangıçta frontend'e config bilgisini gönder (UI'da model adını göstermek için)
	conn.WriteJSON(map[string]string{
		"type":     "config",
		"provider": providerName,
		"model":    activeModel,
	})

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}

		tokenChan := make(chan string, 100)
		errChan := make(chan error, 1)

		appState.mu.RLock()
		provider := appState.Provider
		currentModel := appState.ActiveModel
		appState.mu.RUnlock()

		go provider.GenerateStream(req.Prompt, currentModel, tokenChan, errChan)

		// Streaming Loop
	L:
		for {
			select {
			case token, ok := <-tokenChan:
				if !ok {
					conn.WriteJSON(map[string]string{"type": "done"})
					break L
				}
				conn.WriteJSON(map[string]string{
					"type":    "token",
					"content": token,
				})
			case err := <-errChan:
				conn.WriteJSON(map[string]string{
					"type":    "error",
					"content": err.Error(),
				})
				break L
			}
		}
	}
}

func handleSettingsAPI(w http.ResponseWriter, r *http.Request, appState *AppState) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		Provider      string `json:"provider"`
		Model         string `json:"model"`
		OpenAIKey     string `json:"openai_key"`
		GeminiKey     string `json:"gemini_key"`
		AzureEndpoint string `json:"azure_endpoint"`
		AzureKey      string `json:"azure_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Sadece mevcut olanları değiştir (Eski keyleri koru)
	oldCfg := loadConfig()
	
	cfg := Config{
		Provider:      req.Provider,
		Model:         req.Model,
		OpenAIKey:     req.OpenAIKey,
		GeminiKey:     req.GeminiKey,
		AzureEndpoint: req.AzureEndpoint,
		AzureKey:      req.AzureKey,
	}

	if cfg.OpenAIKey == "" { cfg.OpenAIKey = oldCfg.OpenAIKey }
	if cfg.GeminiKey == "" { cfg.GeminiKey = oldCfg.GeminiKey }
	if cfg.AzureEndpoint == "" { cfg.AzureEndpoint = oldCfg.AzureEndpoint }
	if cfg.AzureKey == "" { cfg.AzureKey = oldCfg.AzureKey }

	// Dosyaya kaydet
	if err := saveConfig(cfg); err != nil {
		http.Error(w, "Ayarlar kaydedilemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// AppState'i bellekte (Memory) anında güncelle
	appState.mu.Lock()
	appState.ProviderName = req.Provider
	appState.ActiveModel = req.Model

	switch strings.ToLower(req.Provider) {
	case "ollama":
		appState.Provider = &OllamaProvider{BaseURL: "http://localhost:11434"}
	case "openai":
		appState.Provider = &OpenAICompatibleProvider{BaseURL: "https://api.openai.com/v1", APIKey: cfg.OpenAIKey}
	case "gemini":
		appState.Provider = &OpenAICompatibleProvider{BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai", APIKey: cfg.GeminiKey}
	case "vllm":
		appState.Provider = &OpenAICompatibleProvider{BaseURL: "http://localhost:8000/v1", APIKey: ""}
	case "azure":
		appState.Provider = &OpenAICompatibleProvider{BaseURL: cfg.AzureEndpoint, APIKey: cfg.AzureKey}
	}
	appState.mu.Unlock()

	// Başarılı yanıt
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
