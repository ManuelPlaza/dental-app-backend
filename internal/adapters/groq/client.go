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

// ── Tipos base ────────────────────────────────────────────────────────────────

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ── Tool calling ──────────────────────────────────────────────────────────────

type Tool struct {
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ── Requests / Responses ──────────────────────────────────────────────────────

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

// ── Client ────────────────────────────────────────────────────────────────────

type Client struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
	sim     bool
}

func NewClient() *Client {
	apiKey := os.Getenv("GROQ_API_KEY")
	baseURL := os.Getenv("GROQ_BASE_URL")
	model := os.Getenv("GROQ_MODEL")

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

// Chat hace una llamada simple sin herramientas.
func (c *Client) Chat(systemPrompt string, messages []Message) (string, error) {
	if c.sim {
		log.Printf("[GroqClient:sim] Chat llamado con %d mensajes", len(messages))
		return "Modo simulación activo. Configure GROQ_API_KEY en las variables de entorno para activar el asistente.", nil
	}
	all := append([]Message{{Role: "system", Content: systemPrompt}}, messages...)
	resp, err := c.doRequest(all, nil)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// ChatWithTools hace una llamada con herramientas y ejecuta el bucle de tool calls
// hasta obtener una respuesta de texto final. execFn recibe (toolName, argsJSON) y
// retorna el resultado como string (puede ser JSON).
func (c *Client) ChatWithTools(
	systemPrompt string,
	messages []Message,
	tools []Tool,
	execFn func(name, args string) string,
) (string, error) {
	if c.sim {
		return c.Chat(systemPrompt, messages)
	}

	all := append([]Message{{Role: "system", Content: systemPrompt}}, messages...)

	for round := 0; round < 6; round++ {
		resp, err := c.doRequest(all, tools)
		if err != nil {
			return "", err
		}
		choice := resp.Choices[0]

		if choice.FinishReason != "tool_calls" || len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		// Adjuntar mensaje del asistente con los tool_calls
		all = append(all, choice.Message)

		// Ejecutar cada herramienta y adjuntar resultado
		for _, tc := range choice.Message.ToolCalls {
			result := execFn(tc.Function.Name, tc.Function.Arguments)
			all = append(all, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "Lo siento, tuve dificultades procesando tu solicitud. Por favor intenta de nuevo con una nueva consulta.", nil
}

// doRequest envía la solicitud a Groq y retorna la respuesta parseada.
func (c *Client) doRequest(messages []Message, tools []Tool) (*chatResponse, error) {
	req := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.4,
		MaxTokens:   600,
	}
	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error llamando a Groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Groq respondió con status %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("respuesta vacía de Groq")
	}
	return &result, nil
}
