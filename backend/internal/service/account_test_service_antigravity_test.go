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
	require.EqualError(t, err, "no synced Antigravity models available")

	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{Models: []string{"gemini-live"}})
	_, _, err = antigravityConnectionTestModel(account, "gemini-stale")
	require.EqualError(t, err, "model gemini-stale is not available for this account")
}
