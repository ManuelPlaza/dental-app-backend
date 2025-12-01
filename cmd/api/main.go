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

	// 2. Conexión a Base de Datos (Infraestructura)
	dsn := "host=127.0.0.1 user=postgres password=postgres dbname=dental_db port=5432 sslmode=disable"
	// Nota: Si usas .env, puedes usar os.Getenv("DB_HOST")... aquí lo puse directo para asegurar que te funcione rápido.

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Falló conexión a la Base de Datos:", err)
	} else {
		log.Println("✅ Conectado a la Base de Datos")
	}

	// 3. Inyección de Dependencias (Ensamblando las piezas SOLID)
	// Repo (Datos) -> Service (Lógica) -> Handler (HTTP)

	patientRepo := repository.NewGormPatientRepo(db)
	patientSrv := services.NewPatientService(patientRepo)
	patientHdl := handler.NewPatientHandler(patientSrv)

	// 4. Configurar Router (Gin)
	r := gin.Default()

	// Ruta de prueba para ver si respira
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.POST("/patients", patientHdl.Create)
		v1.GET("/patients", patientHdl.GetAll) // <--- AGREGA ESTA LÍNEA
	}

	// 5. Correr Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Servidor corriendo en puerto " + port)
	r.Run(":" + port)

}
