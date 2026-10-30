package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Seedance channels share the async task lifecycle (create → poll → optional
// download) but differ in paths, field names and billing units. Each account
// picks a preset and may override individual fields in credentials.seedance_profile.
const (
	SeedanceProfileCredentialKey = "seedance_profile"

	SeedancePresetArk          = "ark"
	SeedancePresetOpenAIVideos = "openai_videos"

	// SeedanceBillingTokens bills Ark-style usage tokens reported on success.
	SeedanceBillingTokens = "tokens"
	// SeedanceBillingVideo bills via the video pricing path (per second by
	// resolution, or channel per-request pricing).
	SeedanceBillingVideo = "video"

	seedanceTaskIDPlaceholder = "{task_id}"
)

type SeedanceProfile struct {
	Preset string `json:"preset"`

	CreatePath  string `json:"create_path"`
	StatusPath  string `json:"status_path"`
	ContentPath string `json:"content_path"`
	DeletePath  string `json:"delete_path"`

	// Response/request fields are gjson paths; comma-separated lists are tried in order.
	TaskIDField            string   `json:"task_id_field"`
	StatusField            string   `json:"status_field"`
	SuccessStatuses        []string `json:"success_statuses"`
	BillingMode            string   `json:"billing_mode"`
	UsageTokensField       string   `json:"usage_tokens_field"`
	DurationField          string   `json:"duration_field"`
	ResolutionField        string   `json:"resolution_field"`
	RequestDurationField   string   `json:"request_duration_field"`
	RequestResolutionField string   `json:"request_resolution_field"`
	// FailureStatuses lists terminal task statuses that indicate a non-recoverable
	// failure. When matched on a status poll, any prepaid charge is refunded.
	FailureStatuses []string `json:"failure_statuses"`
	// RequestFormat rewrites the client create body for the channel; empty passes it through.
	RequestFormat string `json:"request_format"`
}

func seedancePresetProfile(preset string) SeedanceProfile {
	switch preset {
	case SeedancePresetOpenAIVideos:
		return SeedanceProfile{
			Preset:                 SeedancePresetOpenAIVideos,
			CreatePath:             "/v1/videos",
			StatusPath:             "/v1/videos/{task_id}",
			ContentPath:            "/v1/videos/{task_id}/content",
			TaskIDField:            "id",
			StatusField:            "status",
			SuccessStatuses:        []string{"completed"},
			FailureStatuses:        []string{"failed", "cancelled"},
			BillingMode:            SeedanceBillingVideo,
			DurationField:          "seconds,duration",
			ResolutionField:        "resolution,size",
			RequestDurationField:   "seconds,duration",
			RequestResolutionField: "resolution,size",
		}
	default:
		// Empty paths keep the legacy Ark URL builder (origin, proxy prefix or /api/v3 base).
		return SeedanceProfile{
			Preset:                 SeedancePresetArk,
			TaskIDField:            "id",
			StatusField:            "status",
			SuccessStatuses:        []string{"succeeded"},
			FailureStatuses:        []string{"failed", "expired", "cancelled"},
			BillingMode:            SeedanceBillingTokens,
			UsageTokensField:       "usage.completion_tokens",
			DurationField:          "duration",
			ResolutionField:        "resolution",
			RequestDurationField:   "duration",
			RequestResolutionField: "resolution",
		}
	}
}

// ResolveSeedanceProfile merges the account's overrides onto its preset.
func (a *Account) ResolveSeedanceProfile() (SeedanceProfile, error) {
	var override SeedanceProfile
	if a != nil && a.Credentials != nil {
		if raw, ok := a.Credentials[SeedanceProfileCredentialKey]; ok && raw != nil {
			encoded, err := json.Marshal(raw)
			if err != nil {
				return SeedanceProfile{}, fmt.Errorf("invalid seedance_profile: %w", err)
			}
			if err := json.Unmarshal(encoded, &override); err != nil {
				return SeedanceProfile{}, fmt.Errorf("invalid seedance_profile: %w", err)
			}
		}
	}
	preset := strings.ToLower(strings.TrimSpace(override.Preset))
	if preset == "" {
		preset = SeedancePresetArk
	}
	if preset != SeedancePresetArk && preset != SeedancePresetOpenAIVideos {
		return SeedanceProfile{}, fmt.Errorf("unsupported seedance preset %q", override.Preset)
	}
	profile := seedancePresetProfile(preset)
	pick := func(dst *string, value string) {
		if v := strings.TrimSpace(value); v != "" {
			*dst = v
		}
	}
	pick(&profile.CreatePath, override.CreatePath)
	pick(&profile.StatusPath, override.StatusPath)
	pick(&profile.ContentPath, override.ContentPath)
	pick(&profile.DeletePath, override.DeletePath)
	pick(&profile.TaskIDField, override.TaskIDField)
	pick(&profile.StatusField, override.StatusField)
	pick(&profile.UsageTokensField, override.UsageTokensField)
	pick(&profile.DurationField, override.DurationField)
	pick(&profile.ResolutionField, override.ResolutionField)
	pick(&profile.RequestDurationField, override.RequestDurationField)
	pick(&profile.RequestResolutionField, override.RequestResolutionField)
	profile.RequestFormat = strings.ToLower(strings.TrimSpace(override.RequestFormat))
	if !isSeedanceRequestFormat(profile.RequestFormat) {
		return SeedanceProfile{}, fmt.Errorf("unsupported seedance request_format %q", override.RequestFormat)
	}
	if mode := strings.ToLower(strings.TrimSpace(override.BillingMode)); mode != "" {
		if mode != SeedanceBillingTokens && mode != SeedanceBillingVideo {
			return SeedanceProfile{}, fmt.Errorf("unsupported seedance billing_mode %q", override.BillingMode)
		}
		profile.BillingMode = mode
	}
	if statuses := normalizeSeedanceStatuses(override.SuccessStatuses); len(statuses) > 0 {
		profile.SuccessStatuses = statuses
	}
	if statuses := normalizeSeedanceStatuses(override.FailureStatuses); len(statuses) > 0 {
		profile.FailureStatuses = statuses
	}
	for _, path := range []string{profile.CreatePath, profile.StatusPath, profile.ContentPath, profile.DeletePath} {
		if path != "" && !strings.HasPrefix(path, "/") {
			return SeedanceProfile{}, fmt.Errorf("seedance path %q must start with /", path)
		}
	}
	// Task templates without the placeholder would make every task resolve to
	// the same upstream resource (and leak other users' results).
	for _, path := range []string{profile.StatusPath, profile.ContentPath, profile.DeletePath} {
		if path != "" && !strings.Contains(path, seedanceTaskIDPlaceholder) {
			return SeedanceProfile{}, fmt.Errorf("seedance path %q must contain %s", path, seedanceTaskIDPlaceholder)
		}
	}
	return profile, nil
}

func normalizeSeedanceStatuses(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (p SeedanceProfile) IsSuccessStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	for _, s := range p.SuccessStatuses {
		if s == status {
			return true
		}
	}
	return false
}

func (p SeedanceProfile) IsFailureStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	for _, s := range p.FailureStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// seedanceLookup returns the first present value among comma-separated gjson paths.
func seedanceLookup(body []byte, paths string) gjson.Result {
	for _, path := range strings.Split(paths, ",") {
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		if v := gjson.GetBytes(body, path); v.Exists() && v.Type != gjson.Null {
			return v
		}
	}
	return gjson.Result{}
}

// seedanceDurationSeconds accepts numbers and numeric strings ("5", "5s").
func seedanceDurationSeconds(value gjson.Result) int {
	switch value.Type {
	case gjson.Number:
		return int(value.Float())
	case gjson.String:
		raw := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value.String())), "s")
		if f, err := strconv.ParseFloat(raw, 64); err == nil && f > 0 {
			return int(f)
		}
	}
	return 0
}

// seedanceResolution maps "720p", "1280x720" and "adaptive-720p" to billing tiers.
func seedanceResolution(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if tier, ok := LookupVideoBillingResolution(value); ok {
		return tier
	}
	for _, tier := range []string{VideoBillingResolution1080P, VideoBillingResolution720P, VideoBillingResolution480P} {
		if strings.Contains(value, tier) {
			return tier
		}
	}
	if w, h, ok := strings.Cut(value, "x"); ok {
		width, errW := strconv.Atoi(strings.TrimSpace(w))
		height, errH := strconv.Atoi(strings.TrimSpace(h))
		if errW == nil && errH == nil && width > 0 && height > 0 {
			switch short := min(width, height); {
			case short <= 480:
				return VideoBillingResolution480P
			case short <= 720:
				return VideoBillingResolution720P
			default:
				return VideoBillingResolution1080P
			}
		}
	}
	return ""
}

// buildSeedanceProfileURL joins base and a path template, dropping a version
// prefix the base already carries ("/v1" + "/v1/videos" → "/v1/videos").
func buildSeedanceProfileURL(base, template, taskID string) (string, error) {
	path := template
	base = strings.TrimRight(base, "/")
	// Match the overlap on the template, before the task ID is substituted.
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 1; i-- {
		prefix := "/" + strings.Join(segments[:i], "/")
		if strings.HasSuffix(base, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	if strings.Contains(path, seedanceTaskIDPlaceholder) {
		if err := validateUpstreamPathSegment("Seedance task ID", taskID); err != nil || strings.TrimSpace(taskID) == "" {
			return "", fmt.Errorf("invalid Seedance task ID")
		}
		path = strings.ReplaceAll(path, seedanceTaskIDPlaceholder, taskID)
	}
	return base + path, nil
}

func (p SeedanceProfile) pathFor(endpoint GrokMediaEndpoint) string {
	switch endpoint {
	case SeedanceEndpointCreate:
		return p.CreatePath
	case SeedanceEndpointStatus:
		return p.StatusPath
	case SeedanceEndpointContent:
		return p.ContentPath
	case SeedanceEndpointDelete:
		return p.DeletePath
	}
	return ""
}

// errSeedanceEndpointUnsupported marks lifecycle steps the channel does not expose.
var errSeedanceEndpointUnsupported = fmt.Errorf("seedance channel does not support this endpoint")

// seedanceTargetURL resolves a configured template, falling back to the native
// Ark layout only for the Ark preset (content download is never native on Ark).
func seedanceTargetURL(base string, profile SeedanceProfile, endpoint GrokMediaEndpoint, taskID string) (string, error) {
	if template := profile.pathFor(endpoint); template != "" {
		return buildSeedanceProfileURL(base, template, taskID)
	}
	if profile.Preset == SeedancePresetArk && endpoint != SeedanceEndpointContent {
		return buildSeedanceURL(base, endpoint, taskID)
	}
	return "", errSeedanceEndpointUnsupported
}
