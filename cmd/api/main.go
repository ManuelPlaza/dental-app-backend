package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"dental-app/internal/adapters/handler"
	"dental-app/internal/adapters/middleware"
	"dental-app/internal/adapters/groq"
	"dental-app/internal/adapters/meta"
	"dental-app/internal/adapters/nequi"
	"dental-app/internal/adapters/repository"
	"dental-app/internal/adapters/sse"
	"dental-app/internal/adapters/worker"
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

	// 2. Conexión a DB totalmente dinámica
	var dsn string
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		// Caso Railway o Docker Compose con URL completa
		dsn = dbURL
	} else {
		// Caso Local manual usando variables individuales
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			os.Getenv("DB_HOST"),
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_NAME"),
			os.Getenv("DB_PORT"),
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Error de conexión: %v", err)
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

	// --- MÓDULO CATEGORÍAS DE SERVICIOS ---
	serviceCategoryRepo := repository.NewGormServiceCategoryRepo(db)
	serviceCategorySrv := services.NewServiceCategoryService(serviceCategoryRepo)
	serviceCategoryHdl := handler.NewServiceCategoryHandler(serviceCategorySrv)

	// --- MÓDULO 2: CITAS (Agenda) ---
	appointRepo := repository.NewGormAppointmentRepo(db)

	// --- MÓDULO NOTIFICACIONES ---
	notificationRepo := repository.NewGormNotificationRepo(db)
	notificationSrv := services.NewNotificationService(notificationRepo, appointRepo, patientRepo)
	notificationHdl := handler.NewNotificationHandler(notificationSrv)

	// --- MÓDULO 5: ESPECIALISTAS (declarado aquí por dependencia en citas) ---
	specialistRepo := repository.NewGormSpecialistRepo(db)

	// --- MÓDULO CONSENTIMIENTOS LEY 1581 ---
	consentRepo := repository.NewGormDataConsentRepo(db)

	// --- MÓDULO META CAPI ---
	metaClient := meta.NewCAPIClient()

	// --- MÓDULO CITAS ---
	appointSrv := services.NewAppointmentService(appointRepo, patientRepo, serviceRepo, specialistRepo, notificationSrv, consentRepo, metaClient)
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
	specialistSrv := services.NewSpecialistService(specialistRepo)
	specialistHdl := handler.NewSpecialistHandler(specialistSrv)

	// --- MÓDULO DASHBOARD ---
	dashboardSrv := services.NewDashboardService(appointRepo, payRepo, patientRepo)
	dashboardHdl := handler.NewDashboardHandler(dashboardSrv)

	// --- MÓDULO BANNERS ---
	bannerHub := sse.NewBannerHub()
	bannerRepo := repository.NewGormBannerRepo(db)
	bannerSrv := services.NewBannerService(bannerRepo)
	bannerHdl := handler.NewBannerHandler(bannerSrv, bannerHub)

	// --- MÓDULO LINKS DE PAGO NEQUI ---
	nequiClient := nequi.NewClient()
	paymentLinkRepo := repository.NewGormPaymentLinkRepo(db)
	paymentLinkSrv := services.NewPaymentLinkService(paymentLinkRepo, payRepo, appointRepo, nequiClient)
	paymentLinkHdl := handler.NewPaymentLinkHandler(paymentLinkSrv)

	// --- MÓDULO CHATBOT IA ---
	groqClient := groq.NewClient()
	chatConfigRepo := repository.NewGormChatConfigRepo(db)
	chatSrv := services.NewChatService(serviceRepo, chatConfigRepo, groqClient)
	chatHdl := handler.NewChatHandler(chatSrv)
	chatConfigHdl := handler.NewChatConfigHandler(chatConfigRepo, chatSrv)

	// --- MÓDULO PÚBLICO (landing page, sin auth) ---
	publicHdl := handler.NewPublicHandler(serviceSrv, patientSrv, appointSrv, consentRepo)

	notifWorker := worker.NewNotificationWorker(notificationSrv)
	notifWorker.Start()

	payLinkWorker := worker.NewPaymentLinkWorker(paymentLinkSrv)
	payLinkWorker.Start()

	appointWorker := worker.NewAppointmentWorker(appointSrv)
	appointWorker.Start()

	// 4. Configurar Router (Gin)
	// V6: Usar modo release en producción para no exponer debug info
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// --- CONFIGURACIÓN CORS ---
	config := cors.DefaultConfig()
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		parts := strings.Split(allowedOrigins, ",")
		for i, o := range parts {
			parts[i] = strings.TrimSpace(o)
		}
		config.AllowOrigins = parts
	} else {
		config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong - El servidor dental está vivo 🦷"})
	})

	// --- RUTAS PÚBLICAS (sin autenticación) ---
	// V6: Rate limiting en endpoints de autenticación (5 intentos por minuto por IP)
	authRateLimit := middleware.RateLimitMiddleware(5, 1*time.Minute)
	r.POST("/api/v1/auth/login", authRateLimit, authHdl.Login)
	r.POST("/api/v1/auth/refresh", authRateLimit, authHdl.Refresh)
	r.GET("/api/v1/service-categories", serviceCategoryHdl.GetAll)
	r.GET("/api/v1/appointments/confirm", notificationHdl.ConfirmAppointment)
	r.GET("/api/v1/public/services", publicHdl.GetServices)
	r.GET("/api/v1/public/patients/document/:document_number", publicHdl.FindPatientByDocument)
	r.POST("/api/v1/public/appointments", publicHdl.RequestAppointment)
	r.GET("/api/v1/public/available-slots", publicHdl.GetAvailableSlots)
	r.GET("/api/v1/public/banners", bannerHdl.GetActive)
	r.GET("/api/v1/public/banners/events", bannerHdl.StreamEvents)

	// Link de pago público (sin auth) — rate limited
	payRateLimit := middleware.RateLimitMiddleware(10, 1*time.Minute)
	r.GET("/api/v1/pay/:token", payRateLimit, paymentLinkHdl.GetStatus)
	r.POST("/api/v1/webhooks/nequi", paymentLinkHdl.HandleNequiWebhook)

	// Chatbot IA — rate limited (20 req/min por IP)
	chatRateLimit := middleware.RateLimitMiddleware(20, 1*time.Minute)
	r.POST("/api/v1/public/chat", chatRateLimit, chatHdl.Chat)

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
		v1.GET("/appointments/:id/payments", payHdl.GetByAppointment)

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
		v1.PATCH("/specialists/:id/set-default", specialistHdl.SetDefault)

		// Rutas Banners (admin)
		v1.POST("/banners", bannerHdl.Create)
		v1.GET("/banners", bannerHdl.GetAll)
		v1.GET("/banners/:id", bannerHdl.GetByID)
		v1.PUT("/banners/:id", bannerHdl.Update)
		v1.DELETE("/banners/:id", bannerHdl.Delete)

		// Rutas Links de Pago Nequi (admin)
		v1.POST("/appointments/:id/payment-links", paymentLinkHdl.Generate)
		v1.GET("/appointments/:id/payment-links", paymentLinkHdl.GetByAppointment)
		v1.DELETE("/payment-links/:token", paymentLinkHdl.Cancel)

		v1.GET("/services", serviceHdl.GetAll)
		v1.GET("/admin/services", serviceHdl.GetAllAdmin)
		v1.POST("/services", serviceHdl.Create)
		v1.PUT("/services/:id", serviceHdl.Update)
		v1.PATCH("/services/:id/activate", serviceHdl.Activate)
		v1.PATCH("/services/:id/inactivate", serviceHdl.Inactivate)

		// Configuración del Chatbot IA
		v1.GET("/admin/chat-config", chatConfigHdl.Get)
		v1.PUT("/admin/chat-config", chatConfigHdl.Save)
	}

	// 5. Correr Servidor
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("❌ La variable de entorno PORT no está definida")
	}

	log.Printf("🚀 Servidor en el puerto %s", port)
	r.Run(":" + port)
}
