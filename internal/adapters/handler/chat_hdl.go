package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	service ports.ChatService
}

func NewChatHandler(service ports.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

// Chat handles POST /api/v1/public/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req domain.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido o mensajes vacíos"})
		return
	}

	reply, err := h.service.Chat(req.Messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al procesar tu consulta"})
		return
	}

	c.JSON(http.StatusOK, domain.ChatResponse{Reply: reply})
}
