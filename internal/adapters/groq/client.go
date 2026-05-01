package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	sim     bool
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func NewClient() *Client {
	apiKey  := os.Getenv("GROQ_API_KEY")
	baseURL := os.Getenv("GROQ_BASE_URL")
	model   := os.Getenv("GROQ_MODEL")

	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	sim := apiKey == ""
	if sim {
		log.Println("⚠️  [GroqClient] GROQ_API_KEY no configurada — modo simulación activo")
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		http:    &http.Client{Timeout: 30 * time.Second},
		sim:     sim,
	}
}

func (c *Client) Chat(systemPrompt string, messages []Message) (string, error) {
	if c.sim {
		log.Printf("[GroqClient:sim] Chat llamado con %d mensajes", len(messages))
		return "Modo simulación activo. Configure GROQ_API_KEY en las variables de entorno para activar el asistente.", nil
	}

	all := make([]Message, 0, len(messages)+1)
	all = append(all, Message{Role: "system", Content: systemPrompt})
	all = append(all, messages...)

	body, err := json.Marshal(chatRequest{
		Model:       c.model,
		Messages:    all,
		Temperature: 0.7,
		MaxTokens:   512,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("error llamando a Groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq respondió con status %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("respuesta vacía de Groq")
	}

	return result.Choices[0].Message.Content, nil
}
