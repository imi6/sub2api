package service

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

const (
	SeedanceEndpointCreate           GrokMediaEndpoint        = "seedance_create"
	SeedanceEndpointStatus           GrokMediaEndpoint        = "seedance_status"
	SeedanceEndpointContent          GrokMediaEndpoint        = "seedance_content"
	SeedanceEndpointDelete           GrokMediaEndpoint        = "seedance_delete"
	OpenAIEndpointCapabilitySeedance OpenAIEndpointCapability = "seedance"
	// OpenAIEndpointCapabilitySeedanceVideos is derived, never stored: a
	// seedance account whose profile speaks the OpenAI /v1/videos protocol.
	OpenAIEndpointCapabilitySeedanceVideos OpenAIEndpointCapability = "seedance_videos"
)

func (e GrokMediaEndpoint) IsSeedance() bool {
	return e == SeedanceEndpointCreate || e == SeedanceEndpointStatus || e == SeedanceEndpointContent || e == SeedanceEndpointDelete
}

// SeedanceTaskKey isolates ownership and billing keys from other video providers.
func SeedanceTaskKey(id string) string { return "seedance:" + strings.TrimSpace(id) }

// IsSeedanceChannel reports whether the account is configured as a Seedance
// channel, independent of which client protocol its profile accepts.
func (a *Account) IsSeedanceChannel() bool {
	if a == nil || a.Type != AccountTypeAPIKey || strings.TrimSpace(a.GetCredential("base_url")) == "" {
		return false
	}
	if a.Platform == PlatformSeedance {
		return true
	}
	// Legacy: OpenAI API key accounts opted in through openai_capabilities.
	configured, _ := a.openAIEndpointCapabilitySet()
	return a.Platform == PlatformOpenAI && configured[string(OpenAIEndpointCapabilitySeedance)]
}

// SeedanceProtocol is the client request shape the account accepts; empty when
// the profile is invalid so the scheduler never picks a misconfigured channel.
func (a *Account) SeedanceProtocol() string {
	profile, err := a.ResolveSeedanceProfile()
	if err != nil {
		return ""
	}
	return profile.Preset
}

// ParseSeedanceRequest accepts both Ark ({content:[...]}) and OpenAI videos
// ({prompt, seconds, size, reference_*}) request shapes.
func ParseSeedanceRequest(body []byte) (GrokMediaRequestInfo, error) {
	var info GrokMediaRequestInfo
	if !gjson.ValidBytes(body) || !gjson.ParseBytes(body).IsObject() {
		return info, fmt.Errorf("request body must be a JSON object")
	}
	model := gjson.GetBytes(body, "model")
	if model.Type != gjson.String || strings.TrimSpace(model.String()) == "" {
		return info, fmt.Errorf("model is required")
	}
	info.Model = strings.TrimSpace(model.String())
	info.DurationSeconds = seedanceDurationSeconds(seedanceLookup(body, "seconds,duration"))
	info.Resolution = seedanceResolution(seedanceLookup(body, "resolution,size").String())
	var texts []string
	if content := gjson.GetBytes(body, "content"); content.Exists() {
		if !content.IsArray() || len(content.Array()) == 0 {
			return info, fmt.Errorf("content must be a non-empty array")
		}
		for _, item := range content.Array() {
			switch item.Get("type").String() {
			case "text":
				texts = append(texts, item.Get("text").String())
			case "image_url":
				info.InputImageURLs = append(info.InputImageURLs, item.Get("image_url.url").String())
			}
		}
		info.Prompt = strings.Join(texts, "\n")
		return info, nil
	}
	info.Prompt = strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
	hasReference := false
	for _, field := range []string{"reference_images", "reference_videos", "reference_audios", "first_frame", "last_frame"} {
		if v := gjson.GetBytes(body, field); v.Exists() && v.Type != gjson.Null {
			hasReference = true
		}
	}
	for _, field := range []string{"first_frame", "last_frame"} {
		if url := seedanceMediaURL(gjson.GetBytes(body, field), "image_url"); url != "" {
			info.InputImageURLs = append(info.InputImageURLs, url)
		}
	}
	for _, item := range gjson.GetBytes(body, "reference_images").Array() {
		if url := seedanceMediaURL(item, "image_url"); url != "" {
			info.InputImageURLs = append(info.InputImageURLs, url)
		}
	}
	if info.Prompt == "" && !hasReference {
		return info, fmt.Errorf("content, prompt or reference media is required")
	}
	return info, nil
}

// seedanceMediaURL reads a plain URL string or {"image_url": "..."} / {"image_url": {"url": "..."}}.
func seedanceMediaURL(value gjson.Result, key string) string {
	switch {
	case value.Type == gjson.String:
		return strings.TrimSpace(value.String())
	case value.IsObject():
		if v := value.Get(key); v.Type == gjson.String {
			return strings.TrimSpace(v.String())
		}
		return strings.TrimSpace(value.Get(key + ".url").String())
	}
	return ""
}

func buildSeedanceURL(base string, endpoint GrokMediaEndpoint, taskID string) (string, error) {
	base = strings.TrimRight(base, "/")
	// Accept an origin, a proxy prefix, or the full Ark API base.
	if !strings.HasSuffix(base, "/api/v3") && !strings.HasSuffix(base, "/v3") {
		base += "/api/v3"
	}
	base += "/contents/generations/tasks"
	if endpoint != SeedanceEndpointCreate {
		if err := validateUpstreamPathSegment("Seedance task ID", taskID); err != nil || strings.TrimSpace(taskID) == "" {
			return "", fmt.Errorf("invalid Seedance task ID")
		}
		base += "/" + taskID
	}
	return base, nil
}
