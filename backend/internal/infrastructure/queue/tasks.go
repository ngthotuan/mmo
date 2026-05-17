package queue

const (
	// Critical queue — time-sensitive
	TaskPublishNow    = "task:publish_now"
	TaskRetryPublish  = "task:retry_publish"

	// Default queue — pipeline steps
	TaskGenerateScript = "task:generate_script"
	TaskCollectMedia   = "task:collect_media"
	TaskGenerateTTS    = "task:generate_tts"
	TaskAssembleVideo  = "task:assemble_video"
	TaskUploadToR2     = "task:upload_to_r2"

	// Low queue — background maintenance
	TaskDiscoverTrends  = "task:discover_trends"
	TaskSyncAnalytics   = "task:sync_analytics"
	TaskCleanupTemp     = "task:cleanup_temp"
	TaskRefreshTokens   = "task:refresh_tokens"
	TaskCheckPublish    = "task:check_publish"

	// Queue names
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueLow      = "low"
)
