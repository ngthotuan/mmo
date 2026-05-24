package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"mmo/internal/adapter/repository"
	"mmo/internal/domain/video"
	"mmo/pkg/util"
)

type PipelineHandler struct {
	videoRepo   *repository.VideoJobRepo
	publishRepo *repository.PublishJobRepo
}

func NewPipelineHandler(videoRepo *repository.VideoJobRepo, publishRepo *repository.PublishJobRepo) *PipelineHandler {
	return &PipelineHandler{videoRepo: videoRepo, publishRepo: publishRepo}
}

// Events streams real-time pipeline job status over Server-Sent Events.
// The JWT is accepted via Authorization header OR ?token= query param (EventSource limitation).
func (h *PipelineHandler) Events(c *gin.Context) {
	userID := mustParseUserID(c)
	ctx := c.Request.Context()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	p := util.Pagination{Page: 1, PerPage: 50}

	send := func() bool {
		videoJobs, _, err := h.videoRepo.List(ctx, &userID, "", p)
		if err != nil {
			return false
		}

		type jobEvent struct {
			ID            string `json:"id"`
			ContentPlanID string `json:"content_plan_id"`
			Status        string `json:"status"`
			OutputURL     string `json:"output_video_url,omitempty"`
		}

		events := make([]jobEvent, 0, len(videoJobs))
		for _, j := range videoJobs {
			events = append(events, jobEvent{
				ID:            j.ID.String(),
				ContentPlanID: j.ContentPlanID.String(),
				Status:        string(j.Status),
				OutputURL:     j.OutputVideoURL,
			})
		}

		data, _ := json.Marshal(events)
		_, writeErr := c.Writer.Write([]byte("event: jobs\ndata: " + string(data) + "\n\n"))
		c.Writer.Flush()
		return writeErr == nil
	}

	// send initial snapshot immediately
	if !send() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}

// Status returns a snapshot of recent pipeline activity: counts by status.
func (h *PipelineHandler) Status(c *gin.Context) {
	userID := mustParseUserID(c)
	ctx := c.Request.Context()

	// Count video jobs by status (last 7 days via pagination trick — just get 200 rows)
	p := util.Pagination{Page: 1, PerPage: 200}
	videoJobs, _, _ := h.videoRepo.List(ctx, &userID, "", p)

	statusCounts := map[string]int{}
	for _, j := range videoJobs {
		statusCounts[string(j.Status)]++
	}

	// Active jobs
	activeJobs := make([]map[string]any, 0)
	for _, j := range videoJobs {
		if j.Status != video.JobStatusDone && j.Status != video.JobStatusFailed {
			activeJobs = append(activeJobs, map[string]any{
				"id":      j.ID,
				"status":  j.Status,
				"created": j.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"video_status_counts": statusCounts,
		"active_jobs":         activeJobs,
		"total_videos":        len(videoJobs),
	})
}
