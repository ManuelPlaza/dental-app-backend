package handler

import (
	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"
	"dental-app/internal/core/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MedicalHistoryHandler struct {
	service ports.MedicalHistoryService
}

func NewMedicalHistoryHandler(service ports.MedicalHistoryService) *MedicalHistoryHandler {
	return &MedicalHistoryHandler{service: service}
}

// POST: El doctor guarda lo que hizo en la cita
func (h *MedicalHistoryHandler) Create(c *gin.Context) {
	var history domain.MedicalHistory
	if err := c.ShouldBindJSON(&history); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido: " + err.Error()})
		return
	}

	if err := h.service.Create(&history); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, history)
}

// GET: Obtener toda la historia para generar el PDF o ver en pantalla
func (h *MedicalHistoryHandler) GetByPatient(c *gin.Context) {
	idStr := c.Param("patientId")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de paciente inválido"})
		return
	}
	patientID := uint(id64)

	histories, err := h.service.GetHistoryByPatient(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, histories)
}

// ... (al final del archivo)

// GET PDF: Descargar historia clínica
func (h *MedicalHistoryHandler) DownloadPDF(c *gin.Context) {
	// 1. Obtener ID del paciente desde la URL
	patientIDStr := c.Param("patientId")
	id64, _ := strconv.ParseUint(patientIDStr, 10, 32)
	patientID := uint(id64)

	// 2. Buscar la historia en Base de Datos (Usamos el servicio que ya tienes)
	histories, err := h.service.GetHistoryByPatient(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error buscando datos: " + err.Error()})
		return
	}

	if len(histories) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "El paciente no tiene historia clínica"})
		return
	}

	// NOTA: Por simplicidad para el MVP, generamos el PDF del registro MÁS RECIENTE (el primero de la lista)
	latestHistory := histories[0]

	// 3. Generar el PDF usando la función que creamos en el Paso 2
	// OJO: Aquí llamamos a la función del paquete services directamente, o podrías meterla en la interfaz
	// Para hacerlo rápido, asumiremos que está en el paquete services.
	// Asegúrate de importar "dental-app/internal/core/services"
	pdfBytes, err := services.GenerateHistoryPDF(latestHistory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generando PDF: " + err.Error()})
		return
	}

	// 4. Enviar el archivo al navegador
	// Esto le dice al navegador: "Oye, esto es un PDF, ábrelo o descárgalo"
	c.Header("Content-Disposition", "attachment; filename=historia_clinica.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
