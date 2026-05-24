package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"mmo/internal/adapter/repository"
	workerhandler "mmo/internal/adapter/worker"
	infradb "mmo/internal/infrastructure/db"
	"mmo/internal/infrastructure/ffmpeg"
	"mmo/internal/infrastructure/queue"
	"mmo/internal/infrastructure/storage"
	"mmo/internal/integration/edgetts"
	"mmo/internal/integration/facebook"
	"mmo/internal/integration/gemini"
	"mmo/internal/integration/googletrends"
	"mmo/internal/integration/pexels"
	"mmo/internal/integration/pixabay"
	"mmo/internal/integration/reddit"
	"mmo/internal/integration/tiktok"
	"mmo/internal/integration/youtube"
	"mmo/pkg/config"
	"mmo/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	videoOnly := flag.Bool("video-only", false, "run only video assembly tasks")
	flag.Parse()

	cfg := config.Load()
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	db, err := infradb.New(cfg.DB)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// ─── Repositories ────────────────────────────────────────────────────────
	channelRepo  := repository.NewChannelRepo(db)
	trendRepo    := repository.NewTrendRepo(db)
	planRepo     := repository.NewContentPlanRepo(db)
	videoJobRepo := repository.NewVideoJobRepo(db)
	publishRepo   := repository.NewPublishJobRepo(db)
	analyticsRepo := repository.NewAnalyticsRepo(db)
	productRepo   := repository.NewProductRepo(db)

	// ─── Infrastructure ───────────────────────────────────────────────────────
	r2, err := storage.NewR2(cfg.R2)
	if err != nil {
		logger.Fatal("failed to init R2 client", zap.Error(err))
	}

	// ─── Integration clients ─────────────────────────────────────────────────
	tiktokClient   := tiktok.New(cfg.TikTok)
	facebookClient := facebook.New(cfg.Facebook)
	geminiClient   := gemini.New(cfg.Gemini)
	pexelsClient   := pexels.New(cfg.Pexels)
	pixabayClient  := pixabay.New(cfg.Pixabay)
	ttsClient      := edgetts.New(cfg.EdgeTTS)
	assembler      := ffmpeg.New(cfg.FFmpeg)
	googleClient   := googletrends.New(cfg.GoogleTrends)
	youtubeClient  := youtube.New(cfg.YouTube)
	redditClient   := reddit.New(cfg.Reddit)

	// ─── Queue client (for task chaining) ────────────────────────────────────
	queueClient := queue.NewClient(cfg.Redis.URL)
	defer queueClient.Close()

	// ─── Task handlers ───────────────────────────────────────────────────────
	refreshHandler  := workerhandler.NewRefreshTokensHandler(channelRepo, tiktokClient, cfg.Auth.EncryptionKey)
	discoverHandler := workerhandler.NewTrendDiscoveryHandler(trendRepo, cfg, googleClient, youtubeClient, redditClient)
	scriptHandler   := workerhandler.NewScriptGenHandler(trendRepo, planRepo, geminiClient, queueClient, cfg.Video.TargetDurationSecs)
	mediaHandler    := workerhandler.NewMediaCollectHandler(planRepo, videoJobRepo, pexelsClient, pixabayClient, r2, queueClient, assembler, cfg.MediaCollect.HTTPTimeout, cfg.Video.MaxClips)
	ttsHandler      := workerhandler.NewTTSHandler(videoJobRepo, ttsClient, r2, queueClient, assembler)
	assemblyHandler := workerhandler.NewVideoAssemblyHandler(videoJobRepo, assembler, r2, queueClient)
	uploadHandler        := workerhandler.NewR2UploadHandler(videoJobRepo, planRepo, r2, assembler)
	publishHandler       := workerhandler.NewPublishHandler(publishRepo, channelRepo, videoJobRepo, productRepo, tiktokClient, facebookClient, cfg.Auth.EncryptionKey)
	checkPublishHandler  := workerhandler.NewCheckPublishHandler(publishRepo, queueClient)
	analyticsSyncHandler := workerhandler.NewAnalyticsSyncHandler(publishRepo, channelRepo, analyticsRepo, tiktokClient, facebookClient, cfg.Auth.EncryptionKey)

	// ─── Asynq server ────────────────────────────────────────────────────────
	srv := queue.NewServer(cfg.Redis.URL, *videoOnly, cfg.Queue)

	mux := asynq.NewServeMux()

	if !*videoOnly {
		mux.HandleFunc(queue.TaskRefreshTokens,  refreshHandler.ProcessTask)
		mux.HandleFunc(queue.TaskDiscoverTrends, discoverHandler.ProcessTask)
		mux.HandleFunc(queue.TaskGenerateScript, scriptHandler.ProcessTask)
	}
	mux.HandleFunc(queue.TaskCollectMedia,   mediaHandler.ProcessTask)
	mux.HandleFunc(queue.TaskGenerateTTS, ttsHandler.ProcessTask)
	mux.HandleFunc(queue.TaskAssembleVideo, assemblyHandler.ProcessTask)
	mux.HandleFunc(queue.TaskUploadToR2,    uploadHandler.ProcessTask)
	mux.HandleFunc(queue.TaskPublishNow,    publishHandler.ProcessTask)
	if !*videoOnly {
		mux.HandleFunc(queue.TaskCheckPublish,   checkPublishHandler.ProcessTask)
		mux.HandleFunc(queue.TaskSyncAnalytics,  analyticsSyncHandler.ProcessTask)
	}

	// ─── Cron scheduler (non-video worker only) ───────────────────────────────
	var scheduler *asynq.Scheduler
	if !*videoOnly {
		scheduler = queue.NewScheduler(cfg.Redis.URL)

		schedules := []struct {
			cron string
			task string
			q    string
		}{
			{cfg.Schedule.CheckPublish,   queue.TaskCheckPublish,   queue.QueueCritical},
			{cfg.Schedule.DiscoverTrends, queue.TaskDiscoverTrends, queue.QueueLow},
			{cfg.Schedule.SyncAnalytics,  queue.TaskSyncAnalytics,  queue.QueueLow},
			{cfg.Schedule.RefreshTokens,  queue.TaskRefreshTokens,  queue.QueueLow},
		}
		for _, s := range schedules {
			if _, err := scheduler.Register(s.cron,
				asynq.NewTask(s.task, nil),
				asynq.Queue(s.q),
			); err != nil {
				logger.Fatal("failed to register cron job", zap.String("task", s.task), zap.Error(err))
			}
		}

		if err := scheduler.Start(); err != nil {
			logger.Fatal("failed to start scheduler", zap.Error(err))
		}
		defer scheduler.Shutdown()
	}

	workerType := "general"
	if *videoOnly {
		workerType = "video-only"
	}
	logger.Info("worker started", zap.String("type", workerType))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Fatal("worker error", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("shutting down worker...")
	srv.Shutdown()
	logger.Info("worker stopped")
}
