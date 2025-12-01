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
	// Nota: Si usas .env, idealmente usa os.Getenv("DB_DSN") o similar.
	dsn := "host=127.0.0.1 user=postgres password=postgres dbname=dental_db port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Falló conexión a la Base de Datos:", err)
	} else {
		log.Println("✅ Conectado a la Base de Datos")
	}

	// 3. Inyección de Dependencias

	// --- PACIENTES ---
	patientRepo := repository.NewGormPatientRepo(db)
	patientSrv := services.NewPatientService(patientRepo)
	patientHdl := handler.NewPatientHandler(patientSrv)

	// --- CITAS (AGENDA) ---
	appointRepo := repository.NewGormAppointmentRepo(db)
	appointSrv := services.NewAppointmentService(appointRepo)
	appointHdl := handler.NewAppointmentHandler(appointSrv)

	// 4. Configurar Router
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	v1 := r.Group("/api/v1")
	{
		// Rutas Pacientes
		v1.POST("/patients", patientHdl.Create)
		v1.GET("/patients", patientHdl.GetAll)

		// Rutas Citas
		// AQUI ESTABA EL ERROR: Solo debe haber UNA línea por método HTTP
		v1.POST("/appointments", appointHdl.Create)             // Agendar
		v1.PUT("/appointments/:id", appointHdl.Modify)          // Modificar (Hora/Fecha)
		v1.PATCH("/appointments/:id/cancel", appointHdl.Cancel) // Cancelar

		v1.GET("/appointments", appointHdl.GetAll)
	}

	// 5. Correr Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Servidor corriendo en puerto " + port)
	r.Run(":" + port)
}
