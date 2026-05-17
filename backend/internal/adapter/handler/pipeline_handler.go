package handler

import (
	"net/http"

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
	var activeJobs []map[string]interface{}
	for _, j := range videoJobs {
		if j.Status != video.JobStatusDone && j.Status != video.JobStatusFailed {
			activeJobs = append(activeJobs, map[string]interface{}{
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
