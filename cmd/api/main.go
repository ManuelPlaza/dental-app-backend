package main

import (
	"log"
	"os"

	"dental-app/internal/adapters/handler"
	"dental-app/internal/adapters/repository"
	"dental-app/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 1. Cargar variables de entorno
	godotenv.Load()

	// 2. Conexión a Base de Datos
	dsn := "host=127.0.0.1 user=postgres password=postgres dbname=dental_db port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Falló conexión a la Base de Datos:", err)
	} else {
		log.Println("✅ Conectado a la Base de Datos")
	}

	// 3. INYECCIÓN DE DEPENDENCIAS (Wiring)

	// --- MÓDULO PACIENTES ---
	patientRepo := repository.NewGormPatientRepo(db)
	patientSrv := services.NewPatientService(patientRepo)
	patientHdl := handler.NewPatientHandler(patientSrv)

	// --- MÓDULO CITAS (Agenda) ---
	appointRepo := repository.NewGormAppointmentRepo(db)
	appointSrv := services.NewAppointmentService(appointRepo)
	appointHdl := handler.NewAppointmentHandler(appointSrv)

	// --- MÓDULO PAGOS (Caja) ---
	payRepo := repository.NewGormPaymentRepo(db)
	
	// OJO AQUÍ: Le pasamos 'payRepo' Y TAMBIÉN 'appointRepo'
	// Esto permite que el servicio de pagos consulte el precio de la cita
	paySrv := services.NewPaymentService(payRepo, appointRepo)
	
	payHdl := handler.NewPaymentHandler(paySrv)

	// 4. Configurar Router (Gin)
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	v1 := r.Group("/api/v1")
	{
		// Rutas de Pacientes
		v1.POST("/patients", patientHdl.Create)
		v1.GET("/patients", patientHdl.GetAll)

		// Rutas de Citas
		v1.POST("/appointments", appointHdl.Create)
		v1.GET("/appointments", appointHdl.GetAll)
		v1.PUT("/appointments/:id", appointHdl.Modify)
		v1.PATCH("/appointments/:id/cancel", appointHdl.Cancel)

		// Rutas de Pagos
		v1.POST("/payments", payHdl.Create)   // Registrar pago
		v1.GET("/payments", payHdl.GetAll)    // Ver todos los pagos
		
		// === NUEVA RUTA: ESTADO DE CUENTA ===
		// Ejemplo: /api/v1/appointments/1/balance
		v1.GET("/appointments/:id/balance", payHdl.GetBalance)
	}

	// 5. Correr Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Servidor corriendo en puerto " + port)
	r.Run(":" + port)
}