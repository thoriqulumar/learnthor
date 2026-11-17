package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/learnthor/backend/api/handlers"
	"github.com/learnthor/backend/api/middleware"
	"github.com/learnthor/backend/services"
)

// New builds and returns the root Gin engine with all routes registered.
// Pass the live *gorm.DB; all services are constructed here so the router
// is the single composition root for dependencies.
func New(db *gorm.DB) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// ── Health check ────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── API v1 ───────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")

	// Auth (public — no JWT required)
	authSvc := services.NewAuthService(db)
	authH := handlers.NewAuthHandler(authSvc)

	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
	}

	// Preferences — requires JWT
	me := v1.Group("/me")
	me.Use(middleware.RequireAuth())
	{
		me.GET("/preferences", authH.GetPreferences)
		me.PUT("/preferences", authH.UpdatePreferences)
	}

	// Curriculum — requires JWT so we can attach per-user progress
	curriculumH := handlers.NewCurriculumHandler(db)
	v1.GET("/sections", middleware.RequireAuth(), curriculumH.GetSections)

	kanjiSvc := services.NewKanjiService(db)
	kanjiH := handlers.NewKanjiHandler(kanjiSvc)
	kanji := v1.Group("/kanji")
	kanji.Use(middleware.RequireAuth())
	{
		kanji.GET("", kanjiH.List)
		kanji.GET("/word", kanjiH.Word)
		kanji.GET("/:character", kanjiH.Get)
	}

	jlptMaterialSvc := services.NewJLPTMaterialService(db)
	jlptMaterialH := handlers.NewJLPTMaterialHandler(jlptMaterialSvc)
	jlpt := v1.Group("/jlpt")
	jlpt.Use(middleware.RequireAuth())
	{
		jlpt.GET("/lessons", jlptMaterialH.List)
		jlpt.GET("/exam", jlptMaterialH.Exam)
		jlpt.POST("/exam/finish", jlptMaterialH.FinishExam)
		jlpt.GET("/lessons/:lesson_id", jlptMaterialH.Get)
		jlpt.POST("/lessons/:lesson_id/finish", jlptMaterialH.Finish)
	}

	// Sessions — start / resume / submit-answer
	sessionSvc := services.NewSessionService(db)
	sessionH := handlers.NewSessionHandler(sessionSvc)

	sessions := v1.Group("/sessions")
	sessions.Use(middleware.RequireAuth())
	{
		// POST /api/v1/sessions          — start or resume a module study session
		sessions.POST("", sessionH.StartSession)
		// GET  /api/v1/sessions/:session_id — resume an active session
		sessions.GET("/:session_id", sessionH.ResumeSession)
		// POST /api/v1/sessions/submit   — submit a single answer
		sessions.POST("/submit", sessionH.SubmitAnswer)
	}

	// Reviews — daily SRS review queue (cross-module)
	reviewSvc := services.NewReviewService(db)
	reviewH := handlers.NewReviewHandler(reviewSvc)

	reviews := v1.Group("/reviews")
	reviews.Use(middleware.RequireAuth())
	{
		// GET  /api/v1/reviews/due    — all cards due for review
		reviews.GET("/due", reviewH.GetDueCards)
		// POST /api/v1/reviews/submit — grade a due card (rating 1–4)
		reviews.POST("/submit", reviewH.SubmitReview)
		// GET  /api/v1/reviews/stats  — counts for dashboard widget
		reviews.GET("/stats", reviewH.GetStats)
	}

	// Payments — premium purchases via Midtrans
	paymentSvc := services.NewPaymentService(db, services.NewMidtransGateway())
	paymentH := handlers.NewPaymentHandler(paymentSvc)

	// Webhook is PUBLIC (Midtrans servers call it) but signature-verified
	// inside the service layer — no JWT middleware here.
	v1.POST("/payments/notification", paymentH.HandleNotification)

	payments := v1.Group("/payments")
	payments.Use(middleware.RequireAuth())
	{
		// POST /api/v1/payments/create        — start a premium purchase
		payments.POST("/create", paymentH.CreateTransaction)
		// GET  /api/v1/payments/:order_id     — poll own payment status
		payments.GET("/:order_id", paymentH.GetPayment)
	}

	// Modules — protected by JWT auth then level-gating.
	gatingSvc := services.NewGatingService(db)
	vocabSvc := services.NewVocabService(db)
	vocabH := handlers.NewVocabHandler(vocabSvc)

	quizSvc := services.NewQuizService(db)
	quizH := handlers.NewQuizHandler(quizSvc)

	modules := v1.Group("/modules")
	modules.Use(middleware.RequireAuth())
	{
		// GET /api/v1/modules/:module_id/vocab
		modules.GET("/:module_id/vocab",
			middleware.RequireModuleAccess(gatingSvc),
			vocabH.GetVocabByModule,
		)

		// GET /api/v1/modules/:module_id/quiz?type=mixed&limit=10
		modules.GET("/:module_id/quiz",
			middleware.RequireModuleAccess(gatingSvc),
			quizH.GetQuiz,
		)

		// POST /api/v1/modules/:module_id/quiz/finish
		modules.POST("/:module_id/quiz/finish",
			middleware.RequireModuleAccess(gatingSvc),
			quizH.FinishQuiz,
		)
	}

	// Section quiz — available only when all modules in the section are mastered.
	sections := v1.Group("/sections")
	sections.Use(middleware.RequireAuth())
	{
		// GET /api/v1/sections/:section_id/quiz
		sections.GET("/:section_id/quiz", quizH.GetSectionQuiz)
		// POST /api/v1/sections/:section_id/quiz/finish
		sections.POST("/:section_id/quiz/finish", quizH.FinishSectionQuiz)
	}

	return r
}
