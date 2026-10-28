package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Seedance channels accept the same media inputs under different field names.
// Clients send one documented shape (prompt, duration, ratio, resolution,
// first_frame, last_frame, reference_images/videos/audios); request_format
// rewrites it for the channel. Empty keeps the body as sent.
const (
	SeedanceRequestFormatPassthrough = ""
	SeedanceRequestFormatSD25        = "sd25"
	SeedanceRequestFormatOpaigc      = "opaigc"

	seedanceDefaultRatio      = "16:9"
	seedanceDefaultResolution = "720p"
)

var seedanceReferenceKinds = []struct{ field, key, role string }{
	{"reference_images", "image_url", "reference_image"},
	{"reference_videos", "video_url", "reference_video"},
	{"reference_audios", "audio_url", "reference_audio"},
}

var seedanceFrameKinds = []struct{ field, role string }{
	{"first_frame", "first_frame"},
	{"last_frame", "last_frame"},
}

func isSeedanceRequestFormat(format string) bool {
	switch format {
	case SeedanceRequestFormatPassthrough, SeedanceRequestFormatSD25, SeedanceRequestFormatOpaigc:
		return true
	}
	return false
}

// convertSeedanceRequest rewrites a create body for the channel's format.
// requestModel is the client-facing name, used to read a resolution the model
// name pins (e.g. "...-480P") when the upstream model name no longer carries it.
// Ark-shaped bodies (with content) are already channel-native and pass through.
func convertSeedanceRequest(format string, body []byte, requestModel string) ([]byte, error) {
	if format == SeedanceRequestFormatPassthrough {
		return body, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var req map[string]any
	if err := decoder.Decode(&req); err != nil || req == nil {
		return nil, fmt.Errorf("request body must be a JSON object")
	}
	if _, ok := req["content"]; ok {
		return body, nil
	}
	var err error
	switch format {
	case SeedanceRequestFormatSD25:
		err = convertSeedanceSD25(req, requestModel)
	case SeedanceRequestFormatOpaigc:
		err = convertSeedanceOpaigc(req)
	default:
		err = fmt.Errorf("unsupported seedance request_format %q", format)
	}
	if err != nil {
		return nil, err
	}
	return json.Marshal(req)
}

// convertSeedanceSD25 targets sd25.app /v1/videos: seconds, resolution paired
// with aspect_ratio, and media as {"image_url"|"video_url"|"audio_url": url}.
func convertSeedanceSD25(req map[string]any, requestModel string) error {
	moveSeedanceField(req, "duration", "seconds")
	moveSeedanceField(req, "ratio", "aspect_ratio")
	if _, hasSize := req["size"]; !hasSize {
		// The upstream model is generic, so a resolution pinned by the
		// client-facing model name wins over the request.
		if pinned := seedanceResolution(requestModel); pinned != "" {
			req["resolution"] = pinned
		}
		_, hasResolution := req["resolution"]
		_, hasRatio := req["aspect_ratio"]
		// sd25 rejects resolution or aspect_ratio sent alone.
		if hasResolution && !hasRatio {
			req["aspect_ratio"] = seedanceDefaultRatio
		}
		if hasRatio && !hasResolution {
			req["resolution"] = seedanceDefaultResolution
		}
	}
	for _, kind := range seedanceFrameKinds {
		url, ok, err := seedanceFrameURL(req, kind.field)
		if err != nil {
			return err
		}
		if ok {
			req[kind.field] = map[string]any{"image_url": url}
		}
	}
	for _, kind := range seedanceReferenceKinds {
		urls, ok, err := seedanceReferenceURLs(req, kind.field, kind.key)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		items := make([]any, 0, len(urls))
		for _, url := range urls {
			items = append(items, map[string]any{kind.key: url})
		}
		req[kind.field] = items
	}
	return nil
}

// convertSeedanceOpaigc targets opaigc /openapi/v1/videos: duration, ratio and
// resolution stay top level. Text-only requests keep prompt; requests with
// media switch to the official content array with explicit roles, which that
// endpoint also accepts, so reference images are never read as frames.
func convertSeedanceOpaigc(req map[string]any) error {
	moveSeedanceField(req, "seconds", "duration")
	moveSeedanceField(req, "aspect_ratio", "ratio")
	var media []any
	for _, kind := range seedanceFrameKinds {
		url, ok, err := seedanceFrameURL(req, kind.field)
		if err != nil {
			return err
		}
		if ok {
			media = append(media, seedanceContentItem("image_url", url, kind.role))
		}
	}
	for _, kind := range seedanceReferenceKinds {
		urls, _, err := seedanceReferenceURLs(req, kind.field, kind.key)
		if err != nil {
			return err
		}
		for _, url := range urls {
			media = append(media, seedanceContentItem(kind.key, url, kind.role))
		}
	}
	for _, kind := range seedanceFrameKinds {
		delete(req, kind.field)
	}
	for _, kind := range seedanceReferenceKinds {
		delete(req, kind.field)
	}
	if len(media) == 0 {
		return nil
	}
	content := make([]any, 0, len(media)+1)
	if prompt, _ := req["prompt"].(string); strings.TrimSpace(prompt) != "" {
		content = append(content, map[string]any{"type": "text", "text": prompt})
	}
	delete(req, "prompt")
	req["content"] = append(content, media...)
	return nil
}

func seedanceContentItem(key, url, role string) map[string]any {
	return map[string]any{"type": key, key: map[string]any{"url": url}, "role": role}
}

// moveSeedanceField renames from to to; an explicit to value wins.
func moveSeedanceField(req map[string]any, from, to string) {
	value, ok := req[from]
	if !ok {
		return
	}
	delete(req, from)
	if _, exists := req[to]; !exists && value != nil {
		req[to] = value
	}
}

// seedanceFrameURL reads a frame given as a URL or {"image_url": url | {"url": url}}.
func seedanceFrameURL(req map[string]any, field string) (string, bool, error) {
	value, ok := req[field]
	if !ok || value == nil {
		delete(req, field)
		return "", false, nil
	}
	url := seedanceMediaItemURL(value, "image_url")
	if url == "" {
		return "", false, fmt.Errorf("%s must be an image URL or an object with image_url", field)
	}
	return url, true, nil
}

// seedanceReferenceURLs reads an array whose items are URLs or {key: url | {"url": url}}.
func seedanceReferenceURLs(req map[string]any, field, key string) ([]string, bool, error) {
	value, ok := req[field]
	if !ok || value == nil {
		delete(req, field)
		return nil, false, nil
	}
	items, isArray := value.([]any)
	if !isArray {
		return nil, false, fmt.Errorf("%s must be an array", field)
	}
	urls := make([]string, 0, len(items))
	for i, item := range items {
		url := seedanceMediaItemURL(item, key)
		if url == "" {
			return nil, false, fmt.Errorf("%s[%d] must be a URL or an object with %s", field, i, key)
		}
		urls = append(urls, url)
	}
	return urls, true, nil
}

func seedanceMediaItemURL(item any, key string) string {
	switch v := item.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		switch inner := v[key].(type) {
		case string:
			return strings.TrimSpace(inner)
		case map[string]any:
			url, _ := inner["url"].(string)
			return strings.TrimSpace(url)
		}
	}
	return ""
}
