package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ChatConfigHandler struct {
	repo       ports.ChatConfigRepository
	chatSvc    ports.ChatService
}

func NewChatConfigHandler(repo ports.ChatConfigRepository, chatSvc ports.ChatService) *ChatConfigHandler {
	return &ChatConfigHandler{repo: repo, chatSvc: chatSvc}
}

// GET /api/v1/admin/chat-config
func (h *ChatConfigHandler) Get(c *gin.Context) {
	cfg, err := h.repo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error cargando configuración"})
		return
	}
	if cfg == nil {
		cfg = &domain.ChatConfig{} // devuelve vacío si aún no existe
	}
	c.JSON(http.StatusOK, cfg)
}

// PUT /api/v1/admin/chat-config
func (h *ChatConfigHandler) Save(c *gin.Context) {
	var cfg domain.ChatConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido"})
		return
	}

	if err := h.repo.Save(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error guardando configuración"})
		return
	}

	// Invalida el caché del chatbot para que use la nueva info de inmediato
	h.chatSvc.InvalidateCache()

	c.JSON(http.StatusOK, gin.H{"message": "Configuración del chatbot actualizada correctamente"})
}
