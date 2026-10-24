//go:build unit

package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAntigravityConnectionTestModelUsesAccountAvailability(t *testing.T) {
	t.Parallel()

	account := &Account{Platform: PlatformAntigravity}
	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{
		Models: []string{
			"gemini-9.9-flash-high",
			"gemini-9.9-flash-medium",
			"gemini-9.9-flash-low",
		},
	})

	requested, mapped, err := antigravityConnectionTestModel(account, "")
	require.NoError(t, err)
	require.Equal(t, "gemini-9.9-flash", requested)
	require.Equal(t, "gemini-9.9-flash-medium", mapped)

	requested, mapped, err = antigravityConnectionTestModel(account, "models/gemini-9.9-flash-high")
	require.NoError(t, err)
	require.Equal(t, "gemini-9.9-flash-high", requested)
	require.Equal(t, "gemini-9.9-flash-high", mapped)
}

func TestAntigravityConnectionTestModelResolvesExplicitMappingOnce(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformAntigravity,
		Credentials: map[string]any{"model_mapping": map[string]any{
			"public-model":   "upstream-model",
			"upstream-model": "different-model",
		}},
	}

	requested, mapped, err := antigravityConnectionTestModel(account, "public-model")
	require.NoError(t, err)
	require.Equal(t, "public-model", requested)
	require.Equal(t, "upstream-model", mapped)
}

func TestRouteAntigravityTestUsesUpstreamRelayAndResolvedModelOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/1/test", nil)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: [DONE]\n\n")),
	}}
	svc := &AccountTestService{httpUpstream: upstream, cfg: upstreamModelSyncTestConfig()}
	account := &Account{
		ID:          1,
		Name:        "relay",
		Platform:    PlatformAntigravity,
		Type:        AccountTypeUpstream,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "relay-key",
			"base_url": "https://relay.example.com",
			"model_mapping": map[string]any{
				"public-model":   "upstream-model",
				"upstream-model": "different-model",
			},
		},
	}

	err := svc.routeAntigravityTest(c, account, "public-model", "hello")

	require.NoError(t, err)
	require.Equal(t, "https://relay.example.com/v1/messages?beta=true", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer relay-key", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "relay-key", upstream.lastReq.Header.Get("x-api-key"))
	body, err := io.ReadAll(upstream.lastReq.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "upstream-model", payload["model"])
}

func TestAntigravityConnectionTestModelRejectsMissingOrUnsupportedCatalogModel(t *testing.T) {
	t.Parallel()

	account := &Account{Platform: PlatformAntigravity}
	_, _, err := antigravityConnectionTestModel(account, "")
	require.Error(t, err)

	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{Models: []string{"gemini-live"}})
	_, _, err = antigravityConnectionTestModel(account, "gemini-stale")
	require.Error(t, err)
}

func TestAntigravityBuildGeminiTestRequest_IncludesValidatedToolConfig(t *testing.T) {
	t.Parallel()

	svc := &AntigravityGatewayService{}
	raw, err := svc.buildGeminiTestRequestWithPrompt("proj-123", "gemini-3.8-flash-high", "")
	require.NoError(t, err)

	var v1Req map[string]any
	require.NoError(t, json.Unmarshal(raw, &v1Req))
	require.Equal(t, "proj-123", v1Req["project"])
	require.Equal(t, "gemini-3.8-flash-high", v1Req["model"])

	inner, ok := v1Req["request"].(map[string]any)
	require.True(t, ok)
	toolCfg, ok := inner["toolConfig"].(map[string]any)
	require.True(t, ok, "expected toolConfig to be present in Gemini test request")
	fcCfg, ok := toolCfg["functionCallingConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "VALIDATED", fcCfg["mode"])
}

func TestAntigravityBuildGeminiTestRequestWithPrompt_ImageModel(t *testing.T) {
	t.Parallel()

	svc := &AntigravityGatewayService{}
	raw, err := svc.buildGeminiTestRequestWithPrompt("proj-123", "gemini-3.1-flash-image", "a cute cat")
	require.NoError(t, err)

	var v1Req map[string]any
	require.NoError(t, json.Unmarshal(raw, &v1Req))
	inner, ok := v1Req["request"].(map[string]any)
	require.True(t, ok)

	genCfg, ok := inner["generationConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []any{"TEXT", "IMAGE"}, genCfg["responseModalities"])

	contents, ok := inner["contents"].([]any)
	require.True(t, ok)
	require.Len(t, contents, 1)
	firstContent := contents[0].(map[string]any)
	parts := firstContent["parts"].([]any)
	require.Equal(t, "a cute cat", parts[0].(map[string]any)["text"])
}

func TestExtractContentFromSSEResponse_ExtractsTextAndInlineImages(t *testing.T) {
	t.Parallel()

	sse := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Here is your image\"},{\"inlineData\":{\"mimeType\":\"image/png\",\"data\":\"iVBORw0KGgo=\"}}]}}]}}\n\n"
	text, images := extractContentFromSSEResponse([]byte(sse))
	require.Equal(t, "Here is your image", text)
	require.Len(t, images, 1)
	require.Equal(t, "image/png", images[0].MimeType)
	require.Equal(t, "data:image/png;base64,iVBORw0KGgo=", images[0].ImageURL)
}

func TestAntigravityShouldUseGeminiTestBuilder_UsesMappedModel(t *testing.T) {
	t.Parallel()

	require.True(t, shouldUseAntigravityGeminiTestRequest("custom-alias", "gemini-2.5-flash"))
	require.True(t, shouldUseAntigravityGeminiTestRequest("gemini-3.1-pro", "gemini-pro-agent"))
	require.False(t, shouldUseAntigravityGeminiTestRequest("gemini-alias", "claude-sonnet-4-6"))
	require.False(t, shouldUseAntigravityGeminiTestRequest("claude-sonnet-4-6", "claude-sonnet-4-6"))
}
