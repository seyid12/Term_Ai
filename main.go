package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ============================================================================
// 1. ORTAK ARAYÜZ (INTERFACE) VE MODELLER
// ============================================================================

// AIProvider, tüm yapay zeka servislerinin uyması gereken kural setidir.
type AIProvider interface {
	GenerateStream(prompt string, model string, tokenChan chan<- string, errChan chan<- error)
}

// ============================================================================
// 2. OLLAMA SAĞLAYICISI (LOCAL AI)
// ============================================================================
type OllamaProvider struct {
	BaseURL string
}

// systemPrompt, modele verilen davranış kurallarıdır.
const systemPrompt = `Sen yalnızca Linux terminal asistanısın.
Kurallar:
- SADECE Linux komutları ve Linux'a özgü bilgi ver.
- Windows veya macOS hakkında hiçbir şey söyleme.
- Cevapların kısa ve doğrudan olsun.
- Komutları her zaman kod bloğu içinde göster.
- Kullanıcının sistemi zaten Linux olduğunu varsay, bunu açıklama.
- "Ben bir yapay zeka olduğum için erişimim yok" deme; doğrudan komutu ver.`

type ollamaChatReq struct {
	Model    string               `json:"model"`
	Messages []ollamaChatMessage  `json:"messages"`
	Stream   bool                 `json:"stream"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Message ollamaChatMessage `json:"message"`
	Done    bool              `json:"done"`
}

type ollamaTagsResp struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// autoDetectOllamaModel, Ollama'da yüklü ilk modeli otomatik algılar.
func autoDetectOllamaModel(baseURL string) string {
	resp, err := http.Get(fmt.Sprintf("%s/api/tags", baseURL))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var tags ollamaTagsResp
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return ""
	}
	if len(tags.Models) > 0 {
		return tags.Models[0].Name
	}
	return ""
}

func (o *OllamaProvider) GenerateStream(prompt string, model string, tokenChan chan<- string, errChan chan<- error) {
	// /api/chat endpoint'i sistem mesajını destekler
	url := fmt.Sprintf("%s/api/chat", o.BaseURL)
	reqBody, _ := json.Marshal(ollamaChatReq{
		Model: model,
		Messages: []ollamaChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		errChan <- fmt.Errorf("Ollama servisine bağlanılamadı (Arka planda çalışıyor mu?): %v", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var l ollamaChatResp
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			continue
		}
		tokenChan <- l.Message.Content
		if l.Done {
			break
		}
	}
	close(tokenChan)
}

// ============================================================================
// 3. OPENAI / vLLM / AZURE (OPENAI UYUMLU) SAĞLAYICI
// ============================================================================
type OpenAICompatibleProvider struct {
	BaseURL string
	APIKey  string
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIReq struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type openAIStreamResp struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (o *OpenAICompatibleProvider) GenerateStream(prompt string, model string, tokenChan chan<- string, errChan chan<- error) {
	url := fmt.Sprintf("%s/chat/completions", o.BaseURL)

	reqBody, _ := json.Marshal(openAIReq{
		Model: model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.APIKey))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("API isteği başarısız oldu: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("API hata kodu döndü: %s", resp.Status)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			break
		}

		var chunk openAIStreamResp
		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			tokenChan <- chunk.Choices[0].Delta.Content
			if chunk.Choices[0].FinishReason != nil {
				break
			}
		}
	}
	close(tokenChan)
}

// ============================================================================
// 4. KONFİGÜRASYON — ~/.config/term-ai/config
// ============================================================================

type Config struct {
	Provider string
	Model    string
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
		}
	}
	return cfg
}

// ============================================================================
// 5. ANA PROGRAM (CLI PARSING & ORCHESTRATION)
// ============================================================================
func main() {
	// Config dosyasını yükle (varsa)
	cfg := loadConfig()

	// CLI Parametrelerini Tanımla (config'in üzerine yazar)
	providerFlag := flag.String("provider", cfg.Provider, "Yapay zeka sağlayıcı: ollama, openai, vllm, azure")
	modelFlag := flag.String("model", cfg.Model, "Kullanılacak model adı (boş bırakılırsa otomatik algılanır)")
	listFlag := flag.Bool("list", false, "Ollama'daki mevcut modelleri listele")
	execFlag := flag.Bool("exec", false, "AI'nın önerdiği komutu onay alarak çalıştır")
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

	// Geriye kalan argümanları (Soruyu) birleştir
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println()
		fmt.Println("  🤖 term-ai — Linux Terminal AI Asistanı")
		fmt.Println()
		fmt.Println("  Kullanım:")
		fmt.Println("    term-ai \"sorunuz\"")
		fmt.Println("    term-ai --provider openai \"sorunuz\"")
		fmt.Println("    term-ai --model gemma4:e4b \"sorunuz\"")
		fmt.Println("    term-ai --list                          ← Yüklü modelleri göster")
		fmt.Println()
		fmt.Println("  Örnekler:")
		fmt.Println("    term-ai \"Docker nasıl kurulur?\"")
		fmt.Println("    term-ai \"nginx config dosyası nerede?\"")
		fmt.Println("    term-ai --provider openai \"Python'da async nedir?\"")
		fmt.Println()
		os.Exit(0)
	}
	prompt := strings.Join(args, " ")

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
				fmt.Println("   Önce bir model yükleyin: ollama pull gemma2:2b")
				os.Exit(1)
			}
		}

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("❌ Hata: OPENAI_API_KEY ortam değişkenini ayarlayın.")
			fmt.Println("   export OPENAI_API_KEY=\"sk-...\"")
			os.Exit(1)
		}
		provider = &OpenAICompatibleProvider{BaseURL: "https://api.openai.com/v1", APIKey: apiKey}
		activeModel = "gpt-4o-mini"
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
		apiKey := os.Getenv("AZURE_AI_KEY")
		endpoint := os.Getenv("AZURE_AI_ENDPOINT")
		if apiKey == "" || endpoint == "" {
			fmt.Println("❌ Hata: AZURE_AI_KEY ve AZURE_AI_ENDPOINT değişkenlerini ayarlayın.")
			os.Exit(1)
		}
		provider = &OpenAICompatibleProvider{BaseURL: endpoint, APIKey: apiKey}
		activeModel = *modelFlag
		if activeModel == "" {
			fmt.Println("❌ Hata: Azure için --model ile deployment adını belirtin.")
			os.Exit(1)
		}

	default:
		fmt.Printf("❌ Geçersiz sağlayıcı: '%s'\n   Geçerli seçenekler: ollama, openai, vllm, azure\n", *providerFlag)
		os.Exit(1)
	}

	// İletişim Kanalları (Channels)
	tokenChan := make(chan string)
	errChan := make(chan error, 1)

	// API İsteğini arka planda (Goroutine) başlat
	go provider.GenerateStream(prompt, activeModel, tokenChan, errChan)

	fmt.Printf("\n🤖 [%s / %s]\n\n", *providerFlag, activeModel)

	// Tüm yanıtı biriktir (--exec için)
	var fullResponse strings.Builder

	// Kanalları Dinle (Select + for döngüsü)
	for {
		select {
		case token, ok := <-tokenChan:
			if !ok {
				fmt.Println()
				// --exec modunda bash bloklarını çalıştır
				if *execFlag {
					execShellBlocks(fullResponse.String())
				}
				return
			}
			fmt.Print(token)
			fullResponse.WriteString(token)
		case err := <-errChan:
			fmt.Printf("\n❌ Hata: %v\n", err)
			return
		}
	}
}

// execShellBlocks, AI yanıtındaki ```bash ... ``` bloklarını bulup
// kullanıcıya göstererek onay aldıktan sonra çalıştırır.
func execShellBlocks(response string) {
	re := regexp.MustCompile("(?s)```(?:bash|sh)?\\n?(.*?)```")
	matches := re.FindAllStringSubmatch(response, -1)

	if len(matches) == 0 {
		return
	}

	for _, match := range matches {
		cmd := strings.TrimSpace(match[1])
		if cmd == "" {
			continue
		}

		fmt.Printf("\n⚡ Çalıştırılacak komut:\n  %s\n", cmd)
		fmt.Print("   Onaylıyor musunuz? [E/h]: ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "" || answer == "e" || answer == "y" || answer == "evet" {
			fmt.Printf("\n📤 Çıktı:\n")
			shell := exec.Command("bash", "-c", cmd)
			shell.Stdout = os.Stdout
			shell.Stderr = os.Stderr
			if err := shell.Run(); err != nil {
				fmt.Printf("❌ Komut hatası: %v\n", err)
			}
		} else {
			fmt.Println("   ⏭️  Atlandı.")
		}
	}
}
