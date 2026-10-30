package handler

import (
	"context"
	"mime"
	"net/http"
	"strings"
	"time"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ctxKeySeedanceVideosProtocol marks requests that entered through the OpenAI
// /v1/videos surface so scheduling only picks openai_videos Seedance channels.
const ctxKeySeedanceVideosProtocol = "_seedance_videos_protocol"

// ctxKeySeedanceTaskPlatform carries the platform of the account bound to the
// looked-up task, so composite-group lookups schedule from the right pool.
const ctxKeySeedanceTaskPlatform = "_seedance_task_platform"

// CompositeVideoStatus serves GET /v1/videos/{id} on composite groups. The
// request has no model, so the task's submission binding decides the provider:
// Seedance tasks go to the Seedance channel, everything else to Grok.
func (h *OpenAIGatewayHandler) CompositeVideoStatus(c *gin.Context) {
	if h.resolveSeedanceTaskLookup(c, c.Param("request_id")) {
		h.SeedanceVideoStatus(c)
		return
	}
	h.GrokVideoStatus(c)
}

// CompositeVideoContent serves GET /v1/videos/{id}/content on composite groups.
func (h *OpenAIGatewayHandler) CompositeVideoContent(c *gin.Context) {
	if h.resolveSeedanceTaskLookup(c, c.Param("request_id")) {
		h.SeedanceVideoContent(c)
		return
	}
	h.GrokVideoContent(c)
}

// resolveSeedanceTaskLookup reports whether the caller submitted taskID to a
// Seedance channel and records that channel's platform for scheduling.
func (h *OpenAIGatewayHandler) resolveSeedanceTaskLookup(c *gin.Context, taskID string) bool {
	if platform := c.GetString(ctxKeySeedanceTaskPlatform); platform != "" {
		return true
	}
	if h == nil || h.gatewayService == nil {
		return false
	}
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key.Group == nil {
		return false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		return false
	}
	platform, found := h.gatewayService.ResolveSeedanceTaskPlatform(c.Request.Context(), key.GroupID, taskID, subject.UserID, key.ID)
	if !found {
		return false
	}
	c.Set(ctxKeySeedanceTaskPlatform, platform)
	return true
}

// SeedanceTasks exposes Ark's native asynchronous video task protocol.
func (h *OpenAIGatewayHandler) SeedanceTasks(c *gin.Context) {
	endpoint := service.SeedanceEndpointCreate
	taskID := ""
	if c.Request.Method != http.MethodPost {
		taskID = service.SeedanceTaskKey(c.Param("task_id"))
		endpoint = service.SeedanceEndpointStatus
		if c.Request.Method == http.MethodDelete {
			endpoint = service.SeedanceEndpointDelete
		}
	}
	h.handleSeedance(c, endpoint, taskID, EndpointSeedanceTasks, false)
}

// SeedanceVideoCreate serves POST /v1/videos for Seedance channels on OpenAI groups.
func (h *OpenAIGatewayHandler) SeedanceVideoCreate(c *gin.Context) {
	h.handleSeedance(c, service.SeedanceEndpointCreate, "", EndpointVideos, true)
}

// SeedanceVideoStatus serves GET /v1/videos/{id}.
func (h *OpenAIGatewayHandler) SeedanceVideoStatus(c *gin.Context) {
	h.handleSeedance(c, service.SeedanceEndpointStatus, service.SeedanceTaskKey(c.Param("request_id")), EndpointVideos, true)
}

// SeedanceVideoContent serves GET /v1/videos/{id}/content (file download).
func (h *OpenAIGatewayHandler) SeedanceVideoContent(c *gin.Context) {
	h.handleSeedance(c, service.SeedanceEndpointContent, service.SeedanceTaskKey(c.Param("request_id")), EndpointVideos, true)
}

func (h *OpenAIGatewayHandler) handleSeedance(c *gin.Context, endpoint service.GrokMediaEndpoint, taskID, upstreamEndpoint string, videosProtocol bool) {
	if c.Request.Method == http.MethodPost && c.GetHeader("Content-Type") != "" {
		mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || mediaType != "application/json" {
			h.errorResponse(c, http.StatusUnsupportedMediaType, "invalid_request_error", "Seedance requires application/json")
			return
		}
	}
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key.Group == nil || !seedanceGroupPlatformAllowed(key.Group.Platform) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", "Seedance requires a Seedance, OpenAI or composite group")
		return
	}
	if endpoint != service.SeedanceEndpointCreate && strings.TrimSpace(strings.TrimPrefix(taskID, "seedance:")) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "task id is required")
		return
	}
	if endpoint != service.SeedanceEndpointCreate && key.Group.Platform == service.PlatformComposite {
		// Ark-native lookups on composite groups also schedule from the bound account's pool.
		h.resolveSeedanceTaskLookup(c, strings.TrimPrefix(taskID, "seedance:"))
	}
	if videosProtocol {
		c.Set(ctxKeySeedanceVideosProtocol, true)
	}
	setActualUpstreamEndpoint(c, upstreamEndpoint)
	h.handleGrokMedia(c, endpoint, taskID)
}

func seedanceGroupPlatformAllowed(platform string) bool {
	switch platform {
	case service.PlatformSeedance, service.PlatformOpenAI, service.PlatformComposite:
		return true
	}
	return false
}

// seedanceSchedulingPlatform picks the account pool: Seedance groups (and
// composite groups routed to Seedance) use dedicated Seedance accounts, while
// OpenAI groups keep the legacy openai_capabilities opt-in accounts.
func seedanceSchedulingPlatform(c *gin.Context) string {
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key.Group == nil {
		return service.PlatformSeedance
	}
	switch key.Group.Platform {
	case service.PlatformOpenAI:
		return service.PlatformOpenAI
	case service.PlatformComposite:
		// Lookups carry no model: use the platform of the account bound at submission.
		if platform := c.GetString(ctxKeySeedanceTaskPlatform); platform != "" {
			return platform
		}
		if platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok && platform == service.PlatformOpenAI {
			return service.PlatformOpenAI
		}
	}
	return service.PlatformSeedance
}

// Seedance completion is billed once per task on the first successful status
// poll: token channels use reported completion tokens, video channels bill per
// second by resolution (status values preferred, create-time snapshot as fallback).
// Repeated polls share the durable task dedup key.
func prepareSeedanceCompletionBilling(ctx context.Context, h *OpenAIGatewayHandler, key *service.APIKey, subject middleware.AuthSubject, taskID string, result *service.OpenAIForwardResult) *service.OpenAIForwardResult {
	if result == nil || (result.Usage.OutputTokens <= 0 && result.VideoCount <= 0) {
		return nil
	}
	pending, err := h.gatewayService.LoadGrokVideoPendingBilling(ctx, taskID, subject.UserID, key.ID)
	if err != nil || pending == nil {
		return nil
	}
	claimed, err := h.gatewayService.ClaimGrokVideoBilling(ctx, taskID, subject.UserID, key.ID)
	if err != nil || !claimed {
		return nil
	}
	merged := *result
	merged.Model = pending.Model
	merged.BillingModel = firstNonEmptyString(pending.BillingModel, pending.Model)
	merged.UpstreamModel = firstNonEmptyString(pending.UpstreamModel, result.UpstreamModel)
	merged.RequestID = service.StableGrokVideoBillingRequestID(taskID)
	merged.ResponseID = taskID
	merged.Duration = service.GrokVideoE2EDuration(pending.CreatedAt, time.Now())
	if merged.VideoCount > 0 {
		merged.VideoCount = 1
		merged.Usage = service.OpenAIUsage{}
		merged.VideoResolution = firstNonEmptyString(result.VideoResolution, pending.VideoResolution)
		if merged.VideoDurationSeconds <= 0 {
			merged.VideoDurationSeconds = pending.VideoDurationSeconds
		}
	}
	return &merged
}

// prepareSeedanceCreateBilling builds a billing result for immediate charging
// at Seedance video create time. The claim slot is pre-marked as billed so
// status polls do not double-charge. Returns nil if result carries no billable
// video units or the pending billing cannot be stored.
func prepareSeedanceCreateBilling(
	ctx context.Context,
	h *OpenAIGatewayHandler,
	key *service.APIKey,
	subject middleware.AuthSubject,
	taskID string,
	result *service.OpenAIForwardResult,
	videoCreateStartedAt string,
) *service.OpenAIForwardResult {
	if result == nil || !result.SeedanceVideoBilling || result.VideoCount <= 0 {
		return nil
	}
	// Claim billing immediately so status polls see the slot already taken.
	claimed, err := h.gatewayService.ClaimGrokVideoBilling(ctx, taskID, subject.UserID, key.ID)
	if err != nil || !claimed {
		return nil
	}
	merged := *result
	merged.RequestID = service.StableGrokVideoBillingRequestID(taskID)
	merged.ResponseID = taskID
	merged.VideoCount = 1
	merged.Usage = service.OpenAIUsage{}
	merged.Duration = service.GrokVideoE2EDuration(videoCreateStartedAt, time.Now())
	return &merged
}
