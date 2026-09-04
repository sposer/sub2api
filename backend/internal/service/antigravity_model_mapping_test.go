//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAntigravityGatewayService_GetMappedModel_ExplicitRestriction(t *testing.T) {
	account := &Account{
		Platform: PlatformAntigravity,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"public-model": "gemini-9.9-flash-high",
			},
		},
	}

	require.Equal(t, "gemini-9.9-flash-high", mapAntigravityModel(account, "public-model"))
	require.Empty(t, mapAntigravityModel(account, "gemini-9.9-flash-high"))
}

func TestAntigravityGatewayService_GetMappedModel_UsesLiveCatalog(t *testing.T) {
	account := &Account{Platform: PlatformAntigravity}
	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{
		Source: "upstream",
		Models: []string{
			"gemini-9.9-flash-high",
			"gemini-9.9-flash-low",
			"gemini-9.9-flash-medium",
			"gemini-9.9-flash-tiered",
			"gpt-oss-future",
		},
	})

	require.Equal(t, "gemini-9.9-flash-medium", mapAntigravityModel(account, "gemini-9.9-flash"))
	require.Equal(t, "gemini-9.9-flash-high", mapAntigravityModel(account, "models/gemini-9.9-flash-high"))
	require.Equal(t, "gpt-oss-future", mapAntigravityModel(account, "gpt-oss-future"))
	require.Empty(t, mapAntigravityModel(account, "gemini-not-in-catalog"))
}

func TestAntigravityGatewayService_GetMappedModel_RequiresExplicitMappingOrLiveCatalog(t *testing.T) {
	account := &Account{Platform: PlatformAntigravity}

	require.Empty(t, mapAntigravityModel(account, "future-upstream-model"))
	require.Empty(t, mapAntigravityModel(account, "claude-opus-4-6"))
	require.Empty(t, mapAntigravityModel(account, ""))
}

func TestMapAntigravityModel_WildcardTargetEqualsRequest(t *testing.T) {
	tests := []struct {
		name           string
		modelMapping   map[string]any
		requestedModel string
		expected       string
	}{
		{
			name:           "wildcard target equals request model",
			modelMapping:   map[string]any{"claude-*": "claude-sonnet-4-5"},
			requestedModel: "claude-sonnet-4-5",
			expected:       "claude-sonnet-4-5",
		},
		{
			name:           "wildcard target differs from request model",
			modelMapping:   map[string]any{"claude-*": "claude-sonnet-4-5"},
			requestedModel: "claude-opus-4-6",
			expected:       "claude-sonnet-4-5",
		},
		{
			name:           "wildcard no match",
			modelMapping:   map[string]any{"claude-*": "claude-sonnet-4-5"},
			requestedModel: "gpt-4o",
			expected:       "",
		},
		{
			name:           "explicit passthrough same name",
			modelMapping:   map[string]any{"claude-sonnet-4-5": "claude-sonnet-4-5"},
			requestedModel: "claude-sonnet-4-5",
			expected:       "claude-sonnet-4-5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformAntigravity,
				Credentials: map[string]any{
					"model_mapping": tt.modelMapping,
				},
			}
			require.Equal(t, tt.expected, mapAntigravityModel(account, tt.requestedModel))
		})
	}
}

func TestAccountIsModelSupported_AntigravityUsesExplicitMappingOrLiveCatalog(t *testing.T) {
	account := &Account{Platform: PlatformAntigravity}
	require.False(t, account.IsModelSupported("future-upstream-model"))

	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{
		Models: []string{"gemini-9.9-flash-high", "gemini-9.9-flash-medium", "gemini-9.9-flash-low"},
	})
	require.True(t, account.IsModelSupported("gemini-9.9-flash"))
	require.True(t, account.IsModelSupported("models/gemini-9.9-flash-high"))
	require.False(t, account.IsModelSupported("gemini-stale"))
}
