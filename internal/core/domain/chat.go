package domain

type ChatMessage struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

type ChatRequest struct {
	Messages []ChatMessage `json:"messages" binding:"required,min=1,max=20"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}
