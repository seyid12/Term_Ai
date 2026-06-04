package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// systemPrompt, modele verilen davranış kurallarıdır.
const systemPrompt = `Sen yalnızca Linux terminal asistanısın.
Kurallar:
- SADECE Linux komutları ve Linux'a özgü bilgi ver.
- Windows veya macOS hakkında hiçbir şey söyleme.
- Cevapların kısa ve doğrudan olsun.
- Komutları her zaman kod bloğu içinde göster.
- Kullanıcının sistemi zaten Linux olduğunu varsay, bunu açıklama.
- "Ben bir yapay zeka olduğum için erişimim yok" deme; doğrudan komutu ver.`

// AIProvider, tüm yapay zeka servislerinin uyması gereken kural setidir.
type AIProvider interface {
	GenerateStream(prompt string, model string, tokenChan chan<- string, errChan chan<- error)
}

// ============================================================================
// OLLAMA SAĞLAYICISI
// ============================================================================

type OllamaProvider struct {
	BaseURL string
}

type ollamaChatReq struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
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
		errChan <- fmt.Errorf("Ollama servisine bağlanılamadı: %v", err)
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
// OPENAI UYUMLU SAĞLAYICI (OpenAI, vLLM, Azure)
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
