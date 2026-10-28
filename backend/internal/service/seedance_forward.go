package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ForwardSeedance relays one lifecycle step to the account's Seedance channel.
// The request body is passed through; only model is rewritten using the
// account's mapping. Paths, task fields and billing units come from the
// account's seedance_profile so channels with different APIs can coexist.
func (s *OpenAIGatewayService) ForwardSeedance(ctx context.Context, c *gin.Context, account *Account, endpoint GrokMediaEndpoint, taskID string, body []byte) (*OpenAIForwardResult, error) {
	if !account.IsSeedanceChannel() || !endpoint.IsSeedance() {
		return nil, fmt.Errorf("seedance requires an OpenAI API key account with a custom base URL")
	}
	profile, err := account.ResolveSeedanceProfile()
	if err != nil {
		return nil, err
	}
	base, err := s.validateUpstreamBaseURL(account.GetCredential("base_url"))
	if err != nil {
		return nil, err
	}
	target, err := seedanceTargetURL(base, profile, endpoint, strings.TrimPrefix(taskID, "seedance:"))
	if errors.Is(err, errSeedanceEndpointUnsupported) {
		writeGrokMediaErrorResponse(c, http.StatusNotFound, "not_found_error", "This Seedance channel does not support this endpoint")
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	model, upstreamModel := "", ""
	var info GrokMediaRequestInfo
	method := http.MethodGet
	switch endpoint {
	case SeedanceEndpointCreate:
		info, err = ParseSeedanceRequest(body)
		if err != nil {
			return nil, err
		}
		model = info.Model
		upstreamModel = account.GetMappedModel(model)
		body, err = sjson.SetBytes(body, "model", upstreamModel)
		if err != nil {
			return nil, err
		}
		method = http.MethodPost
	case SeedanceEndpointDelete:
		method = http.MethodDelete
	}
	token := strings.TrimSpace(account.GetCredential("api_key"))
	if token == "" {
		return nil, fmt.Errorf("seedance account missing api_key")
	}
	started := time.Now()
	var reqBody *bytes.Reader
	if method == http.MethodPost {
		reqBody = bytes.NewReader(body)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(ctx), method, target, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if endpoint == SeedanceEndpointContent {
		req.Header.Set("Accept", "*/*")
		if rangeHeader := strings.TrimSpace(c.GetHeader("Range")); rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
	} else {
		req.Header.Set("Accept", "application/json")
	}
	account.ApplyHeaderOverrides(req.Header)
	proxy := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if endpoint == SeedanceEndpointContent {
		final, err := s.followSeedanceContentRedirects(ctx, c, resp, req.URL, proxy, account)
		if err != nil {
			return nil, err
		}
		// The deferred close above reads resp, so it closes the final hop.
		resp = final
		return s.finishSeedanceContent(c, resp, taskID, started)
	}
	responseBody, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
	if err != nil {
		return nil, err
	}
	// Do not retry ambiguous asynchronous creates: the upstream may already have
	// accepted a billable job. Preserve native error codes and response bodies.
	if resp.StatusCode >= 300 {
		writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
		return nil, fmt.Errorf("seedance upstream status %d", resp.StatusCode)
	}
	result := &OpenAIForwardResult{Model: model, BillingModel: model, UpstreamModel: upstreamModel, Duration: time.Since(started), ResponseHeaders: resp.Header.Clone()}
	switch endpoint {
	case SeedanceEndpointCreate:
		id := strings.TrimSpace(seedanceLookup(responseBody, profile.TaskIDField).String())
		if id == "" {
			writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
			return nil, fmt.Errorf("seedance create response missing task ID")
		}
		result.ResponseID = SeedanceTaskKey(id)
		// Create-time snapshot for deferred video billing (status may omit them).
		result.VideoResolution = seedanceResolution(seedanceLookup(body, profile.RequestResolutionField).String())
		result.VideoDurationSeconds = seedanceDurationSeconds(seedanceLookup(body, profile.RequestDurationField))
	case SeedanceEndpointStatus:
		result.ResponseID = taskID
		result.UpstreamModel = gjson.GetBytes(responseBody, "model").String()
		applySeedanceStatusBilling(profile, responseBody, result)
	}
	writeGrokMediaResponse(c, resp, responseBody, s.responseHeaderFilter)
	return result, nil
}

// ResolveSeedanceTaskPlatform reports the platform of the account a Seedance
// task was bound to at submission. Status and content lookups carry no model,
// so composite groups use it to tell Seedance tasks from Grok ones; ok is false
// when this user/key never submitted the task to a Seedance channel.
func (s *OpenAIGatewayService) ResolveSeedanceTaskPlatform(ctx context.Context, groupID *int64, taskID string, userID, apiKeyID int64) (string, bool) {
	if s == nil || strings.TrimSpace(taskID) == "" {
		return "", false
	}
	accountID, err := s.ResolveGrokMediaVideoRequestAccount(ctx, groupID, SeedanceTaskKey(taskID), userID, apiKeyID)
	if err != nil || accountID <= 0 {
		return "", false
	}
	var account *Account
	if s.schedulerSnapshot != nil {
		account, err = s.schedulerSnapshot.GetAccount(ctx, accountID)
	} else if s.accountRepo != nil {
		account, err = s.accountRepo.GetByID(ctx, accountID)
	}
	if err != nil || account == nil || !account.IsSeedanceChannel() {
		return "", false
	}
	return account.Platform, true
}

// applySeedanceStatusBilling attaches billable units when the status body
// reports success. Token channels bill reported usage; video channels bill
// per second by resolution, merged with the create-time snapshot by the handler.
func applySeedanceStatusBilling(profile SeedanceProfile, statusBody []byte, result *OpenAIForwardResult) {
	if !profile.IsSuccessStatus(seedanceLookup(statusBody, profile.StatusField).String()) {
		return
	}
	switch profile.BillingMode {
	case SeedanceBillingVideo:
		result.VideoCount = 1
		result.VideoDurationSeconds = seedanceDurationSeconds(seedanceLookup(statusBody, profile.DurationField))
		result.VideoResolution = seedanceResolution(seedanceLookup(statusBody, profile.ResolutionField).String())
	default:
		result.Usage.OutputTokens = max(0, int(seedanceLookup(statusBody, profile.UsageTokensField).Int()))
	}
}

// finishSeedanceContent streams the rendered file. Downloads never bill:
// completion is observed (and claimed once) on status polling.
const seedanceContentMaxRedirects = 5

// followSeedanceContentRedirects follows the download redirects some channels
// return instead of the file (e.g. a 302 to a signed CDN URL). The Location
// comes from the upstream, so hops drop the channel API key and must resolve
// to public hosts. A redirect that cannot be followed is returned as-is and
// rejected by finishSeedanceContent.
func (s *OpenAIGatewayService) followSeedanceContentRedirects(ctx context.Context, c *gin.Context, resp *http.Response, current *url.URL, proxy string, account *Account) (*http.Response, error) {
	for hops := 0; isSeedanceContentRedirect(resp.StatusCode) && hops < seedanceContentMaxRedirects; hops++ {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			return resp, nil
		}
		next, err := current.Parse(location)
		if err != nil || (next.Scheme != "https" && next.Scheme != "http") || next.Hostname() == "" || urlvalidator.IsBlockedHost(next.Hostname()) {
			return resp, nil
		}
		req, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamPublicHostsOnly(ctx)), http.MethodGet, next.String(), nil)
		if err != nil {
			return resp, nil
		}
		req.Header.Set("Accept", "*/*")
		if rangeHeader := strings.TrimSpace(c.GetHeader("Range")); rangeHeader != "" {
			req.Header.Set("Range", rangeHeader)
		}
		hopResp, err := s.httpUpstream.Do(req, proxy, account.ID, account.Concurrency)
		if err != nil {
			writeGrokMediaErrorResponse(c, http.StatusBadGateway, "upstream_error", "Seedance content redirect download failed")
			return nil, fmt.Errorf("seedance content redirect: %w", err)
		}
		_ = resp.Body.Close()
		resp, current = hopResp, next
	}
	return resp, nil
}

func isSeedanceContentRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

func (s *OpenAIGatewayService) finishSeedanceContent(c *gin.Context, resp *http.Response, taskID string, started time.Time) (*OpenAIForwardResult, error) {
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		writeGrokMediaErrorResponse(c, http.StatusBadGateway, "upstream_error", "Seedance content redirect could not be followed")
		return nil, fmt.Errorf("seedance content redirect could not be followed (status %d)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusRequestedRangeNotSatisfiable {
		body, err := ReadUpstreamResponseBody(resp.Body, s.cfg, c, openAITooLargeError)
		if err != nil {
			return nil, err
		}
		writeGrokMediaResponse(c, resp, body, s.responseHeaderFilter)
		return nil, fmt.Errorf("seedance content upstream status %d", resp.StatusCode)
	}
	if err := writeGrokMediaContentResponse(c, resp); err != nil {
		return nil, err
	}
	return &OpenAIForwardResult{
		ResponseID:      taskID,
		Duration:        time.Since(started),
		ResponseHeaders: resp.Header.Clone(),
	}, nil
}
