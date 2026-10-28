//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const seedanceUnifiedImageBody = `{
	"model": "seedance-2.5",
	"prompt": "a cat",
	"duration": 5,
	"ratio": "9:16",
	"resolution": "720p",
	"reference_images": ["https://example.com/a.png", {"image_url": "https://example.com/b.png"}],
	"reference_videos": [{"video_url": {"url": "https://example.com/v.mp4"}}],
	"reference_audios": ["https://example.com/a.mp3"]
}`

func TestConvertSeedanceRequestPassthroughKeepsBody(t *testing.T) {
	body := []byte(seedanceUnifiedImageBody)
	out, err := convertSeedanceRequest(SeedanceRequestFormatPassthrough, body, "x")
	require.NoError(t, err)
	require.Equal(t, body, out)
}

func TestConvertSeedanceRequestSD25(t *testing.T) {
	out, err := convertSeedanceRequest(SeedanceRequestFormatSD25, []byte(seedanceUnifiedImageBody), "seedance-2.5")
	require.NoError(t, err)
	require.JSONEq(t, `{
		"model": "seedance-2.5",
		"prompt": "a cat",
		"seconds": 5,
		"aspect_ratio": "9:16",
		"resolution": "720p",
		"reference_images": [{"image_url": "https://example.com/a.png"}, {"image_url": "https://example.com/b.png"}],
		"reference_videos": [{"video_url": "https://example.com/v.mp4"}],
		"reference_audios": [{"audio_url": "https://example.com/a.mp3"}]
	}`, string(out))
}

func TestConvertSeedanceRequestSD25PinsModelResolutionAndPairsRatio(t *testing.T) {
	body := `{"model":"seedance-2.5","prompt":"p","duration":5,"resolution":"1080p","first_frame":"https://example.com/f.png","last_frame":{"image_url":"https://example.com/l.png"}}`
	out, err := convertSeedanceRequest(SeedanceRequestFormatSD25, []byte(body), "官转SD2.5-30图版-480P")
	require.NoError(t, err)
	require.JSONEq(t, `{
		"model": "seedance-2.5",
		"prompt": "p",
		"seconds": 5,
		"resolution": "480p",
		"aspect_ratio": "16:9",
		"first_frame": {"image_url": "https://example.com/f.png"},
		"last_frame": {"image_url": "https://example.com/l.png"}
	}`, string(out))
}

func TestConvertSeedanceRequestSD25KeepsSize(t *testing.T) {
	body := `{"model":"seedance-2.5","prompt":"p","seconds":"5","size":"1280x720"}`
	out, err := convertSeedanceRequest(SeedanceRequestFormatSD25, []byte(body), "seedance-2.5")
	require.NoError(t, err)
	require.JSONEq(t, body, string(out))
}

func TestConvertSeedanceRequestOpaigcTextOnly(t *testing.T) {
	body := `{"model":"m","prompt":"p","seconds":5,"aspect_ratio":"16:9","resolution":"480p"}`
	out, err := convertSeedanceRequest(SeedanceRequestFormatOpaigc, []byte(body), "m")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"m","prompt":"p","duration":5,"ratio":"16:9","resolution":"480p"}`, string(out))
}

func TestConvertSeedanceRequestOpaigcMediaUsesContentRoles(t *testing.T) {
	out, err := convertSeedanceRequest(SeedanceRequestFormatOpaigc, []byte(seedanceUnifiedImageBody), "m")
	require.NoError(t, err)
	require.JSONEq(t, `{
		"model": "seedance-2.5",
		"duration": 5,
		"ratio": "9:16",
		"resolution": "720p",
		"content": [
			{"type": "text", "text": "a cat"},
			{"type": "image_url", "image_url": {"url": "https://example.com/a.png"}, "role": "reference_image"},
			{"type": "image_url", "image_url": {"url": "https://example.com/b.png"}, "role": "reference_image"},
			{"type": "video_url", "video_url": {"url": "https://example.com/v.mp4"}, "role": "reference_video"},
			{"type": "audio_url", "audio_url": {"url": "https://example.com/a.mp3"}, "role": "reference_audio"}
		]
	}`, string(out))
}

func TestConvertSeedanceRequestOpaigcFrames(t *testing.T) {
	body := `{"model":"m","prompt":"p","first_frame":"https://example.com/f.png","last_frame":"https://example.com/l.png"}`
	out, err := convertSeedanceRequest(SeedanceRequestFormatOpaigc, []byte(body), "m")
	require.NoError(t, err)
	require.JSONEq(t, `{"model":"m","content":[
		{"type":"text","text":"p"},
		{"type":"image_url","image_url":{"url":"https://example.com/f.png"},"role":"first_frame"},
		{"type":"image_url","image_url":{"url":"https://example.com/l.png"},"role":"last_frame"}
	]}`, string(out))
}

func TestConvertSeedanceRequestKeepsArkContent(t *testing.T) {
	body := []byte(`{"model":"m","content":[{"type":"text","text":"p"}]}`)
	for _, format := range []string{SeedanceRequestFormatSD25, SeedanceRequestFormatOpaigc} {
		out, err := convertSeedanceRequest(format, body, "m")
		require.NoError(t, err)
		require.Equal(t, body, out)
	}
}

func TestConvertSeedanceRequestRejectsBadMedia(t *testing.T) {
	for _, body := range []string{
		`{"model":"m","prompt":"p","reference_images":"https://example.com/a.png"}`,
		`{"model":"m","prompt":"p","reference_images":[{"url":"https://example.com/a.png"}]}`,
		`{"model":"m","prompt":"p","first_frame":{"video_url":"x"}}`,
	} {
		_, err := convertSeedanceRequest(SeedanceRequestFormatSD25, []byte(body), "m")
		require.Error(t, err, body)
	}
}

func TestResolveSeedanceProfileRequestFormat(t *testing.T) {
	account := &Account{Credentials: map[string]any{SeedanceProfileCredentialKey: map[string]any{"preset": "openai_videos", "request_format": " SD25 "}}}
	profile, err := account.ResolveSeedanceProfile()
	require.NoError(t, err)
	require.Equal(t, SeedanceRequestFormatSD25, profile.RequestFormat)

	account.Credentials[SeedanceProfileCredentialKey] = map[string]any{"request_format": "nope"}
	_, err = account.ResolveSeedanceProfile()
	require.Error(t, err)
}
