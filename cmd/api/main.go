package main

import (
	"log"
	"os"
	"time"

	"dental-app/internal/adapters/handler"
	"dental-app/internal/adapters/middleware"
	"dental-app/internal/adapters/repository"
	"dental-app/internal/core/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	// 0. Cargar timezone de Colombia ANTES de todo
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		log.Fatal("❌ Error cargando timezone:", err)
	}
	time.Local = loc // Establecer como timezone local del proceso
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

	// --- MÓDULO AUTH ---
	userRepo := repository.NewGormUserRepo(db)
	authSrv := services.NewAuthService(userRepo)
	authHdl := handler.NewAuthHandler(authSrv)

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

	// --- MÓDULO DASHBOARD ---
	dashboardSrv := services.NewDashboardService(appointRepo, payRepo, patientRepo)
	dashboardHdl := handler.NewDashboardHandler(dashboardSrv)

	// 4. Configurar Router (Gin)
	r := gin.Default()

	// --- CONFIGURACIÓN CORS ---
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	// --- RUTAS PÚBLICAS (sin autenticación) ---
	r.POST("/api/v1/auth/login", authHdl.Login)
	r.POST("/api/v1/auth/refresh", authHdl.Refresh)

	// --- RUTAS PROTEGIDAS (con JWT) ---
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(authSrv))
	{
		// Auth
		v1.POST("/auth/logout", authHdl.Logout)
		v1.GET("/auth/me", authHdl.Me)

		// Dashboard
		v1.GET("/dashboard", dashboardHdl.GetStats)

		// Rutas Pacientes
		v1.POST("/patients", patientHdl.Create)
		v1.GET("/patients", patientHdl.GetAll)
		v1.GET("/patients/document/:document_number", patientHdl.FindByDocument)
		v1.PUT("/patients/:id", patientHdl.Update)

		// Rutas Citas
		v1.POST("/appointments", appointHdl.Create)
		v1.GET("/appointments", appointHdl.GetAll)
		v1.GET("/appointments/paginated", appointHdl.GetPaginated)
		v1.GET("/appointments/summary", appointHdl.GetSummary)
		v1.GET("/appointments/cancellation-reasons", appointHdl.GetCancellationReasons)
		v1.PUT("/appointments/:id", appointHdl.Modify)
		v1.PATCH("/appointments/:id/cancel", appointHdl.Cancel)
		v1.PUT("/admin/appointments/:id", appointHdl.AdminUpdate)

		// Rutas Pagos
		v1.POST("/payments", payHdl.Create)
		v1.GET("/payments", payHdl.GetAll)
		v1.PUT("/payments/:id", payHdl.Update)
		v1.GET("/appointments/:id/balance", payHdl.GetBalance)

		// Rutas Historia Clínica
		v1.POST("/medical-history", historyHdl.Create)
		v1.GET("/medical-history", historyHdl.GetAll)
		v1.GET("/patients/:patientId/medical-history", historyHdl.GetByPatient)
		v1.GET("/patients/:patientId/medical-history/pdf", historyHdl.DownloadPDF)

		// Rutas Especialistas
		v1.POST("/specialists", specialistHdl.Create)
		v1.GET("/specialists/without-user", specialistHdl.GetWithoutUser)
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
