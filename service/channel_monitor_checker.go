package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/tidwall/gjson"
)

var monitorHTTPClient = newChannelMonitorHTTPClient(monitorRequestTimeout, monitorResponseHeaderTimeout)
var monitorImageHTTPClient = newChannelMonitorHTTPClient(monitorImageRequestTimeout, monitorImageResponseHeaderTimeout)
var monitorPingHTTPClient = newChannelMonitorHTTPClient(monitorPingTimeout, monitorResponseHeaderTimeout)

func newChannelMonitorHTTPClient(timeout, responseHeaderTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           safeMonitorDialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          16,
			IdleConnTimeout:       monitorIdleConnTimeout,
			TLSHandshakeTimeout:   monitorTLSHandshakeTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
		},
	}
}

type monitorProviderAdapter struct {
	buildPath    func(model string) string
	buildBody    func(model, prompt string) ([]byte, error)
	buildHeaders func(apiKey string) map[string]string
	textPath     string
	validator    func(respBytes []byte) string
}

type CheckOptions struct {
	APIMode          string
	ExtraHeaders     map[string]string
	BodyOverrideMode string
	BodyOverride     map[string]any
}

var monitorOpenAIChatAdapter = monitorProviderAdapter{
	buildPath: func(string) string { return providerOpenAIPath },
	buildBody: func(model, prompt string) ([]byte, error) {
		return common.Marshal(map[string]any{
			"model":      model,
			"messages":   []map[string]string{{"role": "user", "content": prompt}},
			"max_tokens": monitorChallengeMaxTokens,
			"stream":     true,
		})
	},
	buildHeaders: func(apiKey string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + apiKey}
	},
	textPath: "choices.0.message.content",
}

var monitorOpenAIResponsesAdapter = monitorProviderAdapter{
	buildPath: func(string) string { return providerOpenAIResponsesPath },
	buildBody: func(model, prompt string) ([]byte, error) {
		return common.Marshal(map[string]any{
			"model":             model,
			"instructions":      "You are a channel health-check endpoint. Answer the arithmetic challenge exactly and briefly.",
			"input":             prompt,
			"max_output_tokens": monitorChallengeMaxTokens,
			"stream":            true,
		})
	},
	buildHeaders: func(apiKey string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + apiKey}
	},
	textPath: "output.0.content.0.text",
}

var monitorOpenAIImageGenerationAdapter = monitorProviderAdapter{
	buildPath: func(string) string { return providerOpenAIImagePath },
	buildBody: func(model, _ string) ([]byte, error) {
		return common.Marshal(map[string]any{
			"model":  model,
			"prompt": monitorImage2Prompt,
			"n":      1,
			"size":   monitorImage2Size1K,
		})
	},
	buildHeaders: func(apiKey string) map[string]string {
		return map[string]string{"Authorization": "Bearer " + apiKey}
	},
	validator: extractMonitorImageGenerationResult,
}

var monitorProviderAdapters = map[string]monitorProviderAdapter{
	MonitorProviderOpenAI: monitorOpenAIChatAdapter,
	MonitorProviderAnthropic: {
		buildPath: func(string) string { return providerAnthropicPath },
		buildBody: func(model, prompt string) ([]byte, error) {
			return common.Marshal(map[string]any{
				"model":      model,
				"messages":   []map[string]string{{"role": "user", "content": prompt}},
				"max_tokens": monitorChallengeMaxTokens,
			})
		},
		buildHeaders: func(apiKey string) map[string]string {
			return map[string]string{
				"x-api-key":         apiKey,
				"anthropic-version": monitorAnthropicAPIVersion,
			}
		},
		textPath: "content.0.text",
	},
	MonitorProviderGemini: {
		buildPath: func(model string) string {
			return fmt.Sprintf(providerGeminiPathTemplate, url.PathEscape(model))
		},
		buildBody: func(_, prompt string) ([]byte, error) {
			return common.Marshal(map[string]any{
				"contents": []map[string]any{
					{"parts": []map[string]any{{"text": prompt}}},
				},
				"generationConfig": map[string]any{"maxOutputTokens": monitorChallengeMaxTokens},
			})
		},
		buildHeaders: func(apiKey string) map[string]string {
			return map[string]string{"x-goog-api-key": apiKey}
		},
		textPath: "candidates.0.content.parts.0.text",
	},
}

func runChannelMonitorCheckForModel(ctx context.Context, provider, apiMode, endpoint, apiKey, model string, options ...*CheckOptions) *CheckResult {
	res := &CheckResult{
		Model:     model,
		Status:    MonitorStatusError,
		CheckedAt: time.Now(),
	}

	challenge := generateMonitorChallenge()
	start := time.Now()
	opts := normalizeCheckOptions(apiMode, options...)
	respText, rawBody, statusCode, err := callChannelMonitorProvider(ctx, provider, endpoint, apiKey, model, challenge.Prompt, opts)
	latency := time.Since(start)
	latencyMs := int(latency / time.Millisecond)
	res.LatencyMs = &latencyMs

	if err != nil {
		res.Message = truncateMonitorMessage(sanitizeMonitorErrorMessage(err.Error()))
		return res
	}
	if statusCode < 200 || statusCode >= 300 {
		res.Message = truncateMonitorMessage(sanitizeMonitorErrorMessage(fmt.Sprintf("upstream HTTP %d: %s", statusCode, truncateMonitorErrorBody(rawBody))))
		return res
	}
	if isMonitorOpenAIImageGeneration(provider, opts.APIMode) {
		if strings.TrimSpace(respText) == "" {
			res.Status = MonitorStatusFailed
			res.Message = truncateMonitorMessage(formatMonitorImageGenerationNoDataMessage(rawBody))
			return res
		}
		if latency > monitorDegradedThresholdFor(provider, opts.APIMode) {
			res.Status = MonitorStatusDegraded
			res.Message = truncateMonitorMessage(fmt.Sprintf("slow response: %dms", latencyMs))
			return res
		}
		res.Status = MonitorStatusOperational
		return res
	}
	if defaultBodyMode(opts.BodyOverrideMode) == MonitorBodyOverrideModeReplace {
		if strings.TrimSpace(respText) == "" {
			res.Status = MonitorStatusFailed
			res.Message = truncateMonitorMessage("replace-mode: upstream returned 2xx with empty text")
			return res
		}
		if latency >= monitorDegradedThresholdFor(provider, opts.APIMode) {
			res.Status = MonitorStatusDegraded
			res.Message = truncateMonitorMessage(fmt.Sprintf("slow response: %dms", latencyMs))
			return res
		}
		res.Status = MonitorStatusOperational
		return res
	}
	if !validateMonitorChallenge(respText, challenge.Expected) {
		res.Status = MonitorStatusFailed
		res.Message = truncateMonitorMessage(sanitizeMonitorErrorMessage(fmt.Sprintf("challenge mismatch (expected %s, got %q)", challenge.Expected, respText)))
		return res
	}
	if latency >= monitorDegradedThresholdFor(provider, opts.APIMode) {
		res.Status = MonitorStatusDegraded
		res.Message = truncateMonitorMessage(fmt.Sprintf("slow response: %dms", latencyMs))
		return res
	}
	res.Status = MonitorStatusOperational
	return res
}

func callChannelMonitorProvider(ctx context.Context, provider, endpoint, apiKey, model, prompt string, opts *CheckOptions) (extractedText, rawBody string, status int, err error) {
	apiMode := checkMonitorAPIMode(opts)
	if err := validateMonitorAPIMode(provider, apiMode); err != nil {
		return "", "", 0, err
	}
	adapter, resolvedMode, ok := monitorProviderAdapterFor(provider, apiMode)
	if !ok {
		return "", "", 0, ErrChannelMonitorInvalidProvider
	}
	body, err := buildMonitorRequestBody(adapter, provider, resolvedMode, model, prompt, opts)
	if err != nil {
		return "", "", 0, err
	}
	fullURL := joinMonitorURL(endpoint, adapter.buildPath(model))
	client := monitorHTTPClient
	if provider == MonitorProviderOpenAI && resolvedMode == MonitorAPIModeImageGeneration {
		client = monitorImageHTTPClient
	}
	respBytes, status, err := postMonitorRawJSON(ctx, client, fullURL, body, mergeMonitorHeaders(adapter.buildHeaders(apiKey), opts))
	if err != nil {
		return "", "", status, err
	}
	if provider == MonitorProviderOpenAI && resolvedMode == MonitorAPIModeImageGeneration {
		return extractMonitorImageGenerationResult(respBytes), string(respBytes), status, nil
	}
	if provider == MonitorProviderOpenAI && resolvedMode == MonitorAPIModeResponses {
		return extractMonitorOpenAIResponsesText(respBytes), string(respBytes), status, nil
	}
	if provider == MonitorProviderOpenAI {
		return extractMonitorOpenAIChatText(respBytes), string(respBytes), status, nil
	}
	if adapter.validator != nil {
		return adapter.validator(respBytes), string(respBytes), status, nil
	}
	return gjson.GetBytes(respBytes, adapter.textPath).String(), string(respBytes), status, nil
}

func monitorDegradedThresholdFor(provider, apiMode string) time.Duration {
	if isMonitorOpenAIImageGeneration(provider, apiMode) {
		return monitorImageDegradedThreshold
	}
	return monitorDegradedThreshold
}

func monitorProviderAdapterFor(provider, apiMode string) (monitorProviderAdapter, string, bool) {
	if isMonitorOpenAIImageGeneration(provider, apiMode) {
		return monitorOpenAIImageGenerationAdapter, MonitorAPIModeImageGeneration, true
	}
	if provider == MonitorProviderOpenAI && defaultMonitorAPIMode(apiMode) == MonitorAPIModeResponses {
		return monitorOpenAIResponsesAdapter, MonitorAPIModeResponses, true
	}
	adapter, ok := monitorProviderAdapters[provider]
	return adapter, MonitorAPIModeChatCompletions, ok
}

func extractMonitorOpenAIResponsesText(respBytes []byte) string {
	if text := extractMonitorOpenAIResponsesStreamText(respBytes); strings.TrimSpace(text) != "" {
		return text
	}
	if text := gjson.GetBytes(respBytes, "output_text").String(); strings.TrimSpace(text) != "" {
		return text
	}
	var texts []string
	outputs := gjson.GetBytes(respBytes, "output")
	if outputs.IsArray() {
		outputs.ForEach(func(_, output gjson.Result) bool {
			outputType := output.Get("type").String()
			if outputType != "" && outputType != "message" {
				return true
			}
			content := output.Get("content")
			if !content.IsArray() {
				return true
			}
			content.ForEach(func(_, block gjson.Result) bool {
				blockType := block.Get("type").String()
				if blockType != "" && blockType != "output_text" {
					return true
				}
				if text := block.Get("text").String(); strings.TrimSpace(text) != "" {
					texts = append(texts, text)
				}
				return true
			})
			return true
		})
	}
	if len(texts) > 0 {
		return strings.Join(texts, "")
	}
	return gjson.GetBytes(respBytes, monitorOpenAIResponsesAdapter.textPath).String()
}

func extractMonitorOpenAIChatText(respBytes []byte) string {
	if text := extractMonitorOpenAIChatStreamText(respBytes); strings.TrimSpace(text) != "" {
		return text
	}
	return gjson.GetBytes(respBytes, monitorOpenAIChatAdapter.textPath).String()
}

func extractMonitorImageGenerationResult(respBytes []byte) string {
	for _, payload := range monitorSSEDataPayloads(respBytes) {
		if payload == "[DONE]" {
			continue
		}
		if result := extractMonitorImageGenerationResultFromJSON([]byte(payload), true); strings.TrimSpace(result) != "" {
			return result
		}
		if result := extractMonitorImageGenerationResultFromPartialJSON([]byte(payload)); strings.TrimSpace(result) != "" {
			return result
		}
	}

	if result := extractMonitorImageGenerationResultFromJSON(respBytes, false); strings.TrimSpace(result) != "" {
		return result
	}
	return extractMonitorImageGenerationResultFromPartialJSON(respBytes)
}

func extractMonitorImageGenerationResultFromJSON(respBytes []byte, ssePayload bool) string {
	if !gjson.ValidBytes(respBytes) {
		return ""
	}

	root := gjson.ParseBytes(respBytes)
	if ssePayload {
		eventType := strings.TrimSpace(root.Get("type").String())
		if eventType != "" && eventType != "image_generation.completed" {
			return ""
		}
	}

	data := root.Get("data")
	if data.IsArray() {
		if result := extractMonitorImageGenerationResultFromGJSON(data); strings.TrimSpace(result) != "" {
			return result
		}
	}

	return extractMonitorImageGenerationResultFromGJSON(root)
}

func extractMonitorImageGenerationResultFromGJSON(value gjson.Result) string {
	var result string

	value.ForEach(func(key, item gjson.Result) bool {
		keyName := strings.ToLower(strings.TrimSpace(key.String()))
		if keyName == "error" {
			return true
		}
		if isMonitorImageGenerationResultKey(keyName) {
			if (item.IsArray() || item.IsObject()) && strings.TrimSpace(item.Raw) != "" {
				if nested := extractMonitorImageGenerationResultFromGJSON(item); strings.TrimSpace(nested) != "" {
					result = nested
					return false
				}
				return true
			}
			if imageValue := strings.TrimSpace(item.String()); imageValue != "" {
				result = imageValue
				return false
			}
		}
		if item.IsArray() || item.IsObject() {
			if nested := extractMonitorImageGenerationResultFromGJSON(item); strings.TrimSpace(nested) != "" {
				result = nested
				return false
			}
		}
		return true
	})
	return result
}

func isMonitorImageGenerationResultKey(key string) bool {
	switch key {
	case "url", "b64_json", "image_url", "image", "base64":
		return true
	default:
		return false
	}
}

func extractMonitorImageGenerationResultFromPartialJSON(respBytes []byte) string {
	body := string(respBytes)
	for _, key := range []string{"b64_json", "base64", "url", "image_url", "image"} {
		if value := extractMonitorStringValueFromPartialJSON(body, key); strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func extractMonitorStringValueFromPartialJSON(body, key string) string {
	search := `"` + key + `"`
	offset := 0
	for {
		idx := strings.Index(body[offset:], search)
		if idx < 0 {
			return ""
		}
		idx += offset + len(search)
		i := skipMonitorJSONWhitespace(body, idx)
		if i >= len(body) || body[i] != ':' {
			offset = idx
			continue
		}
		i = skipMonitorJSONWhitespace(body, i+1)
		if i >= len(body) || body[i] != '"' {
			offset = idx
			continue
		}
		i++
		if i >= len(body) || body[i] == '"' {
			offset = i
			continue
		}
		end := i
		escaped := false
		for end < len(body) {
			ch := body[end]
			if escaped {
				escaped = false
				end++
				continue
			}
			if ch == '\\' {
				escaped = true
				end++
				continue
			}
			if ch == '"' {
				break
			}
			end++
		}
		return body[i:end]
	}
}

func skipMonitorJSONWhitespace(body string, index int) int {
	for index < len(body) {
		switch body[index] {
		case ' ', '\n', '\r', '\t':
			index++
		default:
			return index
		}
	}
	return index
}

func formatMonitorImageGenerationNoDataMessage(rawBody string) string {
	const prefix = "image generation: upstream returned 2xx without image data"
	summary := summarizeMonitorImageGenerationResponse(rawBody)
	if summary == "" {
		return prefix
	}
	return prefix + "; " + summary
}

func summarizeMonitorImageGenerationResponse(rawBody string) string {
	body := strings.TrimSpace(rawBody)
	if body == "" {
		return "response is empty"
	}

	var parts []string
	if gjson.Valid(body) {
		root := gjson.Parse(body)
		if root.IsObject() {
			if keys := monitorJSONTopLevelKeys(root, 8); len(keys) > 0 {
				parts = append(parts, "keys="+strings.Join(keys, ","))
			}
			if data := root.Get("data"); data.Exists() && data.IsArray() {
				parts = append(parts, fmt.Sprintf("data_len=%d", len(data.Array())))
			}
		}
	} else if payloads := monitorSSEDataPayloads([]byte(body)); len(payloads) > 0 {
		parts = append(parts, fmt.Sprintf("sse_events=%d", len(payloads)))
		if eventTypes := monitorSSEPayloadTypes(payloads, 5); len(eventTypes) > 0 {
			parts = append(parts, "event_types="+strings.Join(eventTypes, ","))
		}
	}

	if snippet := truncateMonitorErrorBody(body); snippet != "" {
		parts = append(parts, "body="+snippet)
	}
	return sanitizeMonitorErrorMessage(strings.Join(parts, "; "))
}

func monitorJSONTopLevelKeys(root gjson.Result, limit int) []string {
	if !root.IsObject() || limit <= 0 {
		return nil
	}
	keys := make([]string, 0)
	root.ForEach(func(key, _ gjson.Result) bool {
		keys = append(keys, key.String())
		return true
	})
	sort.Strings(keys)
	if len(keys) <= limit {
		return keys
	}
	return append(keys[:limit], "...")
}

func monitorSSEPayloadTypes(payloads []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	seen := map[string]bool{}
	types := make([]string, 0, limit)
	for _, payload := range payloads {
		if payload == "[DONE]" || !gjson.Valid(payload) {
			continue
		}
		eventType := strings.TrimSpace(gjson.Get(payload, "type").String())
		if eventType == "" || seen[eventType] {
			continue
		}
		seen[eventType] = true
		types = append(types, eventType)
		if len(types) >= limit {
			break
		}
	}
	return types
}

func normalizeCheckOptions(apiMode string, options ...*CheckOptions) *CheckOptions {
	opts := &CheckOptions{APIMode: apiMode, BodyOverrideMode: MonitorBodyOverrideModeOff}
	if len(options) == 0 || options[0] == nil {
		return opts
	}
	*opts = *options[0]
	if opts.APIMode == "" {
		opts.APIMode = apiMode
	}
	opts.BodyOverrideMode = defaultBodyMode(opts.BodyOverrideMode)
	if opts.ExtraHeaders == nil {
		opts.ExtraHeaders = map[string]string{}
	}
	if opts.BodyOverride == nil {
		opts.BodyOverride = map[string]any{}
	}
	return opts
}

func checkMonitorAPIMode(opts *CheckOptions) string {
	if opts == nil {
		return MonitorAPIModeChatCompletions
	}
	return defaultMonitorAPIMode(opts.APIMode)
}

func buildMonitorRequestBody(adapter monitorProviderAdapter, provider, apiMode, model, prompt string, opts *CheckOptions) ([]byte, error) {
	mode := MonitorBodyOverrideModeOff
	if opts != nil {
		mode = defaultBodyMode(opts.BodyOverrideMode)
	}
	if mode == MonitorBodyOverrideModeReplace {
		if opts == nil || len(opts.BodyOverride) == 0 {
			return nil, ErrChannelMonitorTemplateBodyRequired
		}
		if err := validateImageGenerationRequestBody(provider, apiMode, mode, opts.BodyOverride); err != nil {
			return nil, err
		}
		if err := validateReplaceRequestBody(provider, apiMode, opts.BodyOverride); err != nil {
			return nil, err
		}
		body, err := common.Marshal(opts.BodyOverride)
		if err != nil {
			return nil, fmt.Errorf("marshal body_override (replace): %w", err)
		}
		return body, nil
	}

	defaultBody, err := adapter.buildBody(model, prompt)
	if err != nil {
		return nil, fmt.Errorf("marshal default monitor request body: %w", err)
	}
	if mode != MonitorBodyOverrideModeMerge || opts == nil || len(opts.BodyOverride) == 0 {
		return defaultBody, nil
	}
	if err := validateImageGenerationRequestBody(provider, apiMode, mode, opts.BodyOverride); err != nil {
		return nil, err
	}

	var defaultMap map[string]any
	if err := common.Unmarshal(defaultBody, &defaultMap); err != nil {
		return nil, fmt.Errorf("unmarshal default monitor request body: %w", err)
	}
	deny := monitorBodyMergeKeyDenyList[monitorBodyMergeDenyKey(provider, apiMode)]
	for key, value := range opts.BodyOverride {
		if deny[key] {
			continue
		}
		defaultMap[key] = value
	}
	merged, err := common.Marshal(defaultMap)
	if err != nil {
		return nil, fmt.Errorf("marshal merged monitor request body: %w", err)
	}
	return merged, nil
}

var monitorBodyMergeKeyDenyList = map[string]map[string]bool{
	MonitorProviderOpenAI + ":" + MonitorAPIModeChatCompletions: {"model": true, "messages": true, "stream": true},
	MonitorProviderOpenAI + ":" + MonitorAPIModeResponses:       {"model": true, "instructions": true, "input": true, "stream": true},
	MonitorProviderOpenAI + ":" + MonitorAPIModeImageGeneration: {"model": true, "prompt": true},
	MonitorProviderAnthropic:                                    {"model": true, "messages": true},
	MonitorProviderGemini:                                       {"contents": true},
}

func monitorBodyMergeDenyKey(provider, apiMode string) string {
	if provider == MonitorProviderOpenAI {
		return provider + ":" + defaultMonitorAPIMode(apiMode)
	}
	return provider
}

func validateReplaceRequestBody(provider, apiMode string, body map[string]any) error {
	if provider != MonitorProviderOpenAI {
		return nil
	}
	switch defaultMonitorAPIMode(apiMode) {
	case MonitorAPIModeImageGeneration:
		if strings.TrimSpace(monitorStringFromAny(body["model"])) == "" || strings.TrimSpace(monitorStringFromAny(body["prompt"])) == "" {
			return ErrChannelMonitorInvalidRequestBody
		}
	case MonitorAPIModeResponses:
		if strings.TrimSpace(monitorStringFromAny(body["instructions"])) == "" || !hasNonEmptyMonitorBodyValue(body["input"]) {
			return ErrChannelMonitorInvalidRequestBody
		}
	case MonitorAPIModeChatCompletions:
		if !hasNonEmptyMonitorBodyValue(body["messages"]) {
			return ErrChannelMonitorInvalidRequestBody
		}
	}
	return nil
}

func monitorStringFromAny(value any) string {
	str, _ := value.(string)
	return str
}

func hasNonEmptyMonitorBodyValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case []map[string]any:
		return len(typed) > 0
	case []map[string]string:
		return len(typed) > 0
	default:
		return true
	}
}

func mergeMonitorHeaders(base map[string]string, opts *CheckOptions) map[string]string {
	if opts == nil || len(opts.ExtraHeaders) == 0 {
		return base
	}
	out := make(map[string]string, len(base)+len(opts.ExtraHeaders))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range opts.ExtraHeaders {
		if IsForbiddenHeaderName(key) {
			continue
		}
		out[key] = value
	}
	return out
}

func extractMonitorOpenAIChatStreamText(respBytes []byte) string {
	var texts []string
	for _, payload := range monitorSSEDataPayloads(respBytes) {
		if payload == "[DONE]" {
			continue
		}
		if text := gjson.Get(payload, "choices.0.delta.content").String(); strings.TrimSpace(text) != "" {
			texts = append(texts, text)
			continue
		}
		if text := gjson.Get(payload, "choices.0.message.content").String(); strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "")
}

func extractMonitorOpenAIResponsesStreamText(respBytes []byte) string {
	var texts []string
	for _, payload := range monitorSSEDataPayloads(respBytes) {
		if payload == "[DONE]" {
			continue
		}
		if text := gjson.Get(payload, "delta").String(); strings.TrimSpace(text) != "" {
			texts = append(texts, text)
			continue
		}
		if text := gjson.Get(payload, "output_text").String(); strings.TrimSpace(text) != "" {
			texts = append(texts, text)
			continue
		}
		if text := extractMonitorOpenAIResponsesText([]byte(payload)); strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "")
}

func monitorSSEDataPayloads(respBytes []byte) []string {
	body := string(respBytes)
	if !strings.Contains(body, "data:") {
		return nil
	}
	events := strings.Split(body, "\n\n")
	payloads := make([]string, 0, len(events))
	for _, event := range events {
		var dataLines []string
		for _, line := range strings.Split(event, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		if len(dataLines) > 0 {
			payloads = append(payloads, strings.Join(dataLines, "\n"))
		}
	}
	return payloads
}

func pingChannelMonitorEndpointOrigin(ctx context.Context, endpoint string) *int {
	origin, err := extractMonitorOrigin(endpoint)
	if err != nil || origin == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, origin, nil)
	if err != nil {
		return nil
	}
	start := time.Now()
	resp, err := monitorPingHTTPClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, monitorPingDiscardMaxBytes))
	ms := int(time.Since(start) / time.Millisecond)
	return &ms
}

func postMonitorRawJSON(ctx context.Context, client *http.Client, fullURL string, payload []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, monitorResponseMaxBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

func joinMonitorURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func extractMonitorOrigin(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.New("endpoint missing scheme or host")
	}
	return u.Scheme + "://" + u.Host, nil
}

var monitorSensitiveQueryParamRegex = regexp.MustCompile(`(?i)([?&](?:key|api[_-]?key|access[_-]?token|token|authorization|x-api-key)=)[^&\s"']+`)

var monitorAPIKeyPatterns = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`), "sk-ant-***REDACTED***"},
	{regexp.MustCompile(`sk-[A-Za-z0-9-]{20,}`), "sk-***REDACTED***"},
	{regexp.MustCompile(`AIza[A-Za-z0-9_-]{35}`), "AIza***REDACTED***"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`), "eyJ***REDACTED.JWT***"},
}

func sanitizeMonitorErrorMessage(msg string) string {
	if msg == "" {
		return msg
	}
	msg = monitorSensitiveQueryParamRegex.ReplaceAllString(msg, `${1}REDACTED`)
	for _, pattern := range monitorAPIKeyPatterns {
		msg = pattern.pattern.ReplaceAllString(msg, pattern.replace)
	}
	return msg
}

func truncateMonitorMessage(msg string) string {
	if len(msg) <= monitorMessageMaxBytes {
		return msg
	}
	const ellipsis = "...(truncated)"
	cutoff := monitorMessageMaxBytes - len(ellipsis)
	if cutoff < 0 {
		cutoff = 0
	}
	return msg[:cutoff] + ellipsis
}

func truncateMonitorErrorBody(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len(body) <= monitorErrorBodySnippetMaxBytes {
		return body
	}
	const ellipsis = "...(body truncated)"
	cutoff := monitorErrorBodySnippetMaxBytes - len(ellipsis)
	if cutoff < 0 {
		cutoff = 0
	}
	return body[:cutoff] + ellipsis
}
