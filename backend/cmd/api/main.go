package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"mmo/internal/adapter/handler"
	"mmo/internal/adapter/repository"
	infradb "mmo/internal/infrastructure/db"
	infraqueue "mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/gemini"
	"mmo/internal/integration/tiktok"
	"mmo/internal/usecase"
	"mmo/pkg/config"
	"mmo/pkg/logger"
	"mmo/pkg/middleware"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	db, err := infradb.New(cfg.DB)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// ─── Repositories ────────────────────────────────────────────────────────
	channelRepo     := repository.NewChannelRepo(db)
	trendRepo       := repository.NewTrendRepo(db)
	contentPlanRepo := repository.NewContentPlanRepo(db)
	videoJobRepo    := repository.NewVideoJobRepo(db)
	videoTemplRepo  := repository.NewVideoTemplateRepo(db)
	publishRepo     := repository.NewPublishJobRepo(db)
	analyticsRepo   := repository.NewAnalyticsRepo(db)
	productRepo     := repository.NewProductRepo(db)
	autoPilotRepo   := repository.NewAutoPilotRepo(db)

	// ─── Infrastructure ───────────────────────────────────────────────────────
	r2, err := storage.NewR2(cfg.R2)
	if err != nil {
		logger.Fatal("failed to init R2 client", zap.Error(err))
	}

	// ─── Queue client ─────────────────────────────────────────────────────────
	queueClient := infraqueue.NewClient(cfg.Redis.URL)
	defer queueClient.Close()

	// ─── Redis client (PKCE store, etc.) ─────────────────────────────────────
	redisOpt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("invalid redis URL", zap.Error(err))
	}
	redisClient := redis.NewClient(redisOpt)
	defer redisClient.Close()

	// ─── Integration clients ─────────────────────────────────────────────────
	tiktokClient   := tiktok.New(cfg.TikTok)
	facebookClient := facebook.New(cfg.Facebook)
	geminiClient   := gemini.New(cfg.Gemini)

	// ─── Use cases ───────────────────────────────────────────────────────────
	channelUC  := usecase.NewChannelUsecase(channelRepo, tiktokClient, facebookClient, cfg.Auth.EncryptionKey, cfg.Channel.FacebookTokenExpiry, redisClient)
	contentUC  := usecase.NewContentUsecase(trendRepo, contentPlanRepo, geminiClient, queueClient, cfg.Video.TargetDurationSecs, cfg.Content.Language)
	videoUC    := usecase.NewVideoUsecase(videoJobRepo, videoTemplRepo, r2, queueClient, cfg.Video.PresignedURLTTL)
	publishUC   := usecase.NewPublishUsecase(publishRepo, videoJobRepo, channelRepo, queueClient, cfg.Publish.MinScheduleBeforeNow)
	analyticsUC := usecase.NewAnalyticsUsecase(analyticsRepo)
	productUC   := usecase.NewProductUsecase(productRepo, channelRepo, tiktokClient, facebookClient, cfg.Auth.EncryptionKey)
	autoPilotUC := usecase.NewAutoPilotUsecase(autoPilotRepo, trendRepo, contentPlanRepo, geminiClient, queueClient, cfg.Video.TargetDurationSecs, cfg.Content.Language)

	// ─── Handlers ────────────────────────────────────────────────────────────
	healthHandler  := handler.NewHealthHandler(db)
	authHandler    := handler.NewAuthHandler(db, cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL)
	channelHandler := handler.NewChannelHandler(channelUC)
	contentHandler := handler.NewContentHandler(contentUC)
	videoHandler   := handler.NewVideoHandler(videoUC)
	publishHandler   := handler.NewPublishHandler(publishUC)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsUC)
	pipelineHandler  := handler.NewPipelineHandler(videoJobRepo, publishRepo)
	productHandler   := handler.NewProductHandler(productUC)
	autoPilotHandler := handler.NewAutoPilotHandler(autoPilotUC)

	// ─── Router ──────────────────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.App.FrontendURL))

	r.GET("/health", healthHandler.Check)

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login",    authHandler.Login)
			auth.POST("/refresh",  authHandler.Refresh)
			auth.GET("/me",              middleware.Auth(cfg.Auth.JWTSecret), authHandler.Me)
			auth.PUT("/profile",         middleware.Auth(cfg.Auth.JWTSecret), authHandler.UpdateProfile)
			auth.PUT("/change-password", middleware.Auth(cfg.Auth.JWTSecret), authHandler.ChangePassword)
		}

		protected := v1.Group("", middleware.Auth(cfg.Auth.JWTSecret))
		{
			// ─── Channels ────────────────────────────────────────────────────
			ch := protected.Group("/channels")
			{
				ch.GET("",                   channelHandler.List)
				ch.GET("/connect/:platform", channelHandler.GetAuthURL)
				ch.GET("/facebook/pages",    channelHandler.ListFacebookPages)
				ch.POST("/oauth/tiktok",     channelHandler.ConnectTikTok)
				ch.POST("/oauth/facebook",   channelHandler.ConnectFacebook)
				ch.DELETE("/:id",            channelHandler.Delete)
				ch.PUT("/:id/toggle",        channelHandler.Toggle)
			}

			// ─── Content ─────────────────────────────────────────────────────
			ct := protected.Group("/content")
			{
				ct.GET("",                      contentHandler.ListPlans)
				ct.POST("",                     contentHandler.CreateFromTrend)
				ct.GET("/:id",                  contentHandler.GetPlan)
				ct.PUT("/:id",                  contentHandler.UpdatePlan)
				ct.POST("/:id/approve",         contentHandler.ApprovePlan)
				ct.POST("/:id/reject",          contentHandler.RejectPlan)
				ct.POST("/:id/generate-script", contentHandler.RegenerateScript)
				ct.DELETE("/:id",               contentHandler.DeletePlan)
			}
			protected.GET("/trends",                 contentHandler.ListTrends)
			protected.POST("/trends/discover",       contentHandler.TriggerDiscover)
			protected.POST("/trends/bulk-reject",    contentHandler.BulkRejectTrends)
			ct.POST("/bulk-action",                  contentHandler.BulkActionPlans)

			// ─── Videos ──────────────────────────────────────────────────────
			vid := protected.Group("/videos")
			{
				vid.GET("",              videoHandler.List)
				vid.GET("/:id",          videoHandler.Get)
				vid.DELETE("/:id",       videoHandler.Delete)
				vid.POST("/:id/retry",   videoHandler.Retry)
				vid.GET("/:id/download", videoHandler.GetDownloadURL)
			}

			// ─── Templates ───────────────────────────────────────────────────
			protected.GET("/templates", videoHandler.ListTemplates)

			// ─── Publish ─────────────────────────────────────────────────────
			pub := protected.Group("/publish")
			{
				pub.GET("",              publishHandler.List)
				pub.POST("",             publishHandler.Create)
				pub.GET("/:id",          publishHandler.Get)
				pub.PUT("/:id",          publishHandler.Update)
				pub.DELETE("/:id",       publishHandler.Cancel)
				pub.POST("/:id/publish-now", publishHandler.PublishNow)
			}
			protected.GET("/calendar", publishHandler.Calendar)

			// ─── Analytics ───────────────────────────────────────────────────
			protected.GET("/analytics/overview",    analyticsHandler.Overview)
			protected.GET("/analytics/posts",       analyticsHandler.ListPosts)
			protected.GET("/analytics/timeseries",  analyticsHandler.Timeseries)

			protected.GET("/pipeline/status", pipelineHandler.Status)

			// SSE — EventSource can't set headers, so token is accepted via ?token= query param
			v1.GET("/pipeline/events", middleware.AuthSSE(cfg.Auth.JWTSecret), pipelineHandler.Events)

			// ─── Products (shop catalog) ──────────────────────────────────────
			prod := protected.Group("/products")
			{
				prod.GET("",        productHandler.List)
				prod.POST("",       productHandler.Create)
				prod.GET("/:id",    productHandler.Get)
				prod.DELETE("/:id", productHandler.Delete)
				prod.POST("/sync",  productHandler.Sync)
			}
			pub.GET("/:id/products",  productHandler.ListByPublishJob)
			pub.POST("/:id/products", productHandler.TagPublishJob)

			// ─── Auto Pilot ──────────────────────────────────────────────────
			ap := protected.Group("/auto-pilot")
			{
				ap.GET("",              autoPilotHandler.List)
				ap.POST("",             autoPilotHandler.Create)
				ap.GET("/:id",          autoPilotHandler.Get)
				ap.PUT("/:id",          autoPilotHandler.Update)
				ap.PUT("/:id/toggle",   autoPilotHandler.Toggle)
				ap.DELETE("/:id",       autoPilotHandler.Delete)
				ap.POST("/:id/run",     autoPilotHandler.RunNow)
			}
		}
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Info("API server started", zap.Int("port", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", zap.Error(err))
	}
	logger.Info("server stopped")
}

