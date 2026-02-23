package main

import (
	"log"
	"os"

	"dental-app/internal/adapters/handler"
	"dental-app/internal/adapters/repository"
	"dental-app/internal/core/services"

	"github.com/gin-contrib/cors" // <--- Importado
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

	// --- MÓDULO 1: PACIENTES ---
	patientRepo := repository.NewGormPatientRepo(db)
	patientSrv := services.NewPatientService(patientRepo)
	patientHdl := handler.NewPatientHandler(patientSrv)

	// --- MÓDULO 6: SERVICIOS ---
	serviceRepo := repository.NewGormServiceRepo(db)
	serviceSrv := services.NewServiceService(serviceRepo)
	serviceHdl := handler.NewServiceHandler(serviceSrv)

	// --- MÓDULO 2: CITAS (Agenda) ---
	appointRepo := repository.NewGormAppointmentRepo(db)
	appointSrv := services.NewAppointmentService(appointRepo, patientRepo, serviceRepo)
	appointHdl := handler.NewAppointmentHandler(appointSrv)

	// --- MÓDULO 3: PAGOS (Caja) ---
	payRepo := repository.NewGormPaymentRepo(db)
	paySrv := services.NewPaymentService(payRepo, appointRepo)
	payHdl := handler.NewPaymentHandler(paySrv)

	// --- MÓDULO 4: HISTORIA CLÍNICA ---
	historyRepo := repository.NewGormMedicalHistoryRepo(db)
	historySrv := services.NewMedicalHistoryService(historyRepo, appointRepo)
	historyHdl := handler.NewMedicalHistoryHandler(historySrv)

	// --- MÓDULO 5: ESPECIALISTAS ---
	specialistRepo := repository.NewGormSpecialistRepo(db)
	specialistSrv := services.NewSpecialistService(specialistRepo)
	specialistHdl := handler.NewSpecialistHandler(specialistSrv)

	// 4. Configurar Router (Gin)
	r := gin.Default()

	// --- CONFIGURACIÓN CORS (ESTO ES LO QUE FALTABA USAR) ---
	// Permite que Flutter (puerto 3000 o celular) hable con Go (puerto 8080)
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config)) // <--- Aquí se usa la librería importada
	// --------------------------------------------------------

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	v1 := r.Group("/api/v1")
	{
		// Rutas Pacientes
		v1.POST("/patients", patientHdl.Create)
		v1.GET("/patients", patientHdl.GetAll)
		v1.GET("/patients/document/:document_number", patientHdl.FindByDocument)
		v1.PUT("/patients/:id", patientHdl.Update) // <--- NUEVA

		// Rutas Citas
		// Rutas Citas
		v1.POST("/appointments", appointHdl.Create)
		v1.GET("/appointments", appointHdl.GetAll)
		v1.GET("/appointments/paginated", appointHdl.GetPaginated)
		v1.GET("/appointments/summary", appointHdl.GetSummary) // <--- NUEVO
		v1.PUT("/appointments/:id", appointHdl.Modify)
		v1.PATCH("/appointments/:id/cancel", appointHdl.Cancel)
		v1.PUT("/admin/appointments/:id", appointHdl.AdminUpdate) // <--- Reemplaza AdminUpdateStatus

		// Rutas Pagos
		v1.POST("/payments", payHdl.Create)
		v1.GET("/payments", payHdl.GetAll)
		v1.GET("/appointments/:id/balance", payHdl.GetBalance)

		// Rutas Historia Clínica
		v1.POST("/medical-history", historyHdl.Create)
		v1.GET("/medical-history", historyHdl.GetAll) // <--- NUEVO
		v1.GET("/patients/:patientId/medical-history", historyHdl.GetByPatient)
		v1.GET("/patients/:patientId/medical-history/pdf", historyHdl.DownloadPDF)

		// Rutas Especialistas
		v1.POST("/specialists", specialistHdl.Create)
		v1.GET("/specialists/without-user", specialistHdl.GetWithoutUser) // <--- NUEVO
		v1.GET("/specialists", specialistHdl.GetAll)
		v1.PATCH("/specialists/:id/inactivate", specialistHdl.Inactivate)
		v1.PATCH("/specialists/:id/activate", specialistHdl.Activate)

		// Rutas Servicios
		v1.GET("/services", serviceHdl.GetAll)

	}

	// 5. Correr Servidor
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Servidor corriendo en puerto " + port)
	r.Run(":" + port)
}
