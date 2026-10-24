package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/stretchr/testify/require"
)

func newAntigravityAccountWithMapping(mapping map[string]string) *Account {
	mappingAny := make(map[string]any, len(mapping))
	for k, v := range mapping {
		mappingAny[k] = v
	}
	return &Account{
		ID:       1,
		Name:     "ag",
		Platform: PlatformAntigravity,
		Credentials: map[string]any{
			"model_mapping": mappingAny,
		},
	}
}

func TestGeminiThinkingLevelFromBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty body", ``, "high"},
		{"no thinkingConfig", `{"contents":[]}`, "high"},
		{"budget -1 dynamic", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":-1}}}`, "high"},
		{"budget 0 off", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":0}}}`, "low"},
		{"budget 1000 (agy low)", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":1000}}}`, "low"},
		{"budget 4000 (agy medium)", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":4000}}}`, "medium"},
		{"budget 24576", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":24576}}}`, "high"},
		{"thinkingLevel wins over budget", `{"generationConfig":{"thinkingConfig":{"thinkingBudget":24576,"thinkingLevel":"low"}}}`, "low"},
		{"thinkingLevel medium", `{"generationConfig":{"thinkingConfig":{"thinkingLevel":"MEDIUM"}}}`, "medium"},
		{"invalid json", `{"generationConfig":`, "high"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, geminiThinkingLevelFromBody([]byte(tt.body)))
		})
	}
}

func TestResolveGeminiThinkingVariant(t *testing.T) {
	mapping := map[string]string{
		"gemini-3.8-flash-low":    "gemini-3.8-flash-low",
		"gemini-3.8-flash-medium": "gemini-3.8-flash-medium",
		"gemini-3.8-flash-high":   "gemini-3.8-flash-high",
		"gemini-3.8-flash-tiered": "gemini-3.8-flash-tiered",
		"gemini-3.6-flash-high":   "gemini-3.6-flash-high",
		"gemini-3.1-pro-high":     "gemini-3.1-pro-high",
		"gemini-2.5-flash":        "gemini-2.5-flash",
	}
	catalog := []string{
		"gemini-9.9-flash-low",
		"gemini-9.9-flash-medium",
		"gemini-9.9-flash-high",
		"gemini-9.9-flash-tiered",
	}
	budget := func(v string) []byte {
		return []byte(`{"contents":[],"generationConfig":{"thinkingConfig":{"thinkingBudget":` + v + `}}}`)
	}

	tests := []struct {
		name      string
		mapping   any
		model     string
		body      []byte
		want      string
		wantMatch bool
		catalog   []string
	}{
		{"agy high (-1) -> -high", mapping, "gemini-3.8-flash", budget("-1"), "gemini-3.8-flash-high", true, nil},
		{"agy medium (4000) -> -medium", mapping, "gemini-3.8-flash", budget("4000"), "gemini-3.8-flash-medium", true, nil},
		{"agy low (1000) -> -low", mapping, "gemini-3.8-flash", budget("1000"), "gemini-3.8-flash-low", true, nil},
		{"models/ prefix and whitespace stripped", mapping, " models/gemini-3.8-flash ", budget("1000"), "gemini-3.8-flash-low", true, nil},
		{"no thinkingConfig -> -high", mapping, "gemini-3.8-flash", []byte(`{"contents":[]}`), "gemini-3.8-flash-high", true, nil},
		{"allowed catalog bare model degrades to high", nil, "gemini-9.9-flash", budget("1000"), "gemini-9.9-flash-high", true, []string{
			"gemini-9.9-flash", "gemini-9.9-flash-high",
		}},
		{"sparse explicit mapping does not admit a bare request", mapping, "gemini-3.6-flash", budget("1000"), "", false, nil},
		{"already suffixed: untouched", mapping, "gemini-3.8-flash-low", budget("-1"), "", false, nil},
		{"bare name explicitly mapped: untouched", mapping, "gemini-2.5-flash", budget("-1"), "", false, nil},
		{"no variants in mapping: untouched", mapping, "gemini-9.9-flash", budget("-1"), "", false, nil},
		{"non-gemini model: untouched", mapping, "claude-sonnet-4-6", budget("-1"), "", false, nil},
		{"explicit bare alias wins over catalog variants", map[string]any{
			"gemini-9.9-flash": "custom-upstream",
		}, "gemini-9.9-flash", budget("-1"), "", false, catalog},
		{"variant mapping target does not admit an unconfigured bare request", map[string]string{
			"gemini-3.8-flash-high": "gemini-3.8-flash-tiered",
		}, "gemini-3.8-flash", budget("-1"), "", false, nil},
		{"explicit bare identity wins over configured variants", map[string]string{
			"gemini-3.8-flash":      "gemini-3.8-flash",
			"gemini-3.8-flash-high": "gemini-3.8-flash-high",
		}, "gemini-3.8-flash", budget("1000"), "", false, nil},
		{"wildcard mapping does not fabricate variants for non-tiered bare model", map[string]string{
			"gemini-2.5*": "gemini-2.5-flash",
		}, "gemini-2.5-flash", budget("-1"), "", false, nil},
		{"bare request absent from explicit restriction is rejected despite a live catalog", map[string]string{
			"gemini-9.9-flash-medium": "gemini-9.9-flash-medium",
		}, "gemini-9.9-flash", budget("4000"), "", false, catalog},
		{"empty mapping without a live catalog admits no default models", map[string]string{}, "gemini-3.8-flash", budget("-1"), "", false, nil},
		{"live catalog low overrides the derived medium alias", nil, "gemini-9.9-flash", []byte(`{"generationConfig":{"thinkingConfig":{"thinkingLevel":"low"}}}`), "gemini-9.9-flash-low", true, catalog},
		{"live catalog high overrides the derived medium alias", nil, "gemini-9.9-flash", []byte(`{"generationConfig":{"thinkingConfig":{"thinkingLevel":"high"}}}`), "gemini-9.9-flash-high", true, catalog},
		{"explicit wildcard target wins over live catalog variants", map[string]string{
			"gemini-9.9*": "custom-upstream",
		}, "gemini-9.9-flash", budget("1000"), "", false, catalog},
		{"single live variant does not admit an unlisted bare request", nil, "gemini-9.9-flash", budget("-1"), "", false, []string{"gemini-9.9-flash-high"}},
		{"allowed catalog bare model degrades to medium", nil, "gemini-9.9-flash", budget("-1"), "gemini-9.9-flash-medium", true, []string{
			"gemini-9.9-flash", "gemini-9.9-flash-medium",
		}},
		{"allowed catalog bare model degrades to low", nil, "gemini-9.9-flash", budget("-1"), "gemini-9.9-flash-low", true, []string{
			"gemini-9.9-flash", "gemini-9.9-flash-low",
		}},
		{"allowed catalog bare model degrades to tiered", nil, "gemini-9.9-flash", budget("-1"), "gemini-9.9-flash-tiered", true, []string{
			"gemini-9.9-flash", "gemini-9.9-flash-tiered",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := newAntigravityAccountWithMapping(nil)
			account.Credentials["model_mapping"] = tt.mapping
			if tt.catalog != nil {
				account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{Models: tt.catalog})
			}
			got, matched := resolveGeminiThinkingVariant(account, tt.model, tt.body)
			require.Equal(t, tt.wantMatch, matched)
			require.Equal(t, tt.want, got)
		})
	}

	t.Run("nil account", func(t *testing.T) {
		got, matched := resolveGeminiThinkingVariant(nil, "gemini-3.8-flash", budget("-1"))
		require.False(t, matched)
		require.Empty(t, got)
	})
}

func TestGeminiThinkingLevelFromClaudeThinking(t *testing.T) {
	tests := []struct {
		name     string
		thinking *antigravity.ThinkingConfig
		want     string
	}{
		{"nil thinking", nil, "high"},
		{"disabled", &antigravity.ThinkingConfig{Type: "disabled"}, "low"},
		{"enabled without budget", &antigravity.ThinkingConfig{Type: "enabled"}, "high"},
		{"budget 1024 -> low", &antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: 1024}, "low"},
		{"budget 4000 -> medium", &antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: 4000}, "medium"},
		{"budget 20000 -> high", &antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: 20000}, "high"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, geminiThinkingLevelFromClaudeThinking(tt.thinking))
		})
	}
}

// 锁定线上 404 回归：Chat Completions / Messages 兼容层收到裸 Gemini 模型名时，
// 必须和 Gemini 原生路径一样解析到带后缀的上游变体，而不是把裸名透传给上游。
func TestResolveGeminiThinkingVariantForLevel_CompatPathResolvesBareModel(t *testing.T) {
	catalog := map[string]string{
		"gemini-3.8-flash-low":    "gemini-3.8-flash-low",
		"gemini-3.8-flash-medium": "gemini-3.8-flash-medium",
		"gemini-3.8-flash-high":   "gemini-3.8-flash-high",
	}
	account := newAntigravityAccountWithMapping(catalog)

	for _, tc := range []struct{ level, want string }{
		{"low", "gemini-3.8-flash-low"},
		{"medium", "gemini-3.8-flash-medium"},
		{"high", "gemini-3.8-flash-high"},
		{"", "gemini-3.8-flash-high"},
	} {
		t.Run("level="+tc.level, func(t *testing.T) {
			got, matched := resolveGeminiThinkingVariantForLevel(account, "gemini-3.8-flash", tc.level)
			require.True(t, matched)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("already suffixed model is untouched", func(t *testing.T) {
		got, matched := resolveGeminiThinkingVariantForLevel(account, "gemini-3.8-flash-high", "low")
		require.False(t, matched)
		require.Empty(t, got)
	})
}

// 同一请求无论走 Gemini 原生还是兼容层，都必须落到同一个上游变体。
func TestGeminiThinkingLevel_NativeAndCompatAgree(t *testing.T) {
	native := geminiThinkingLevelFromBody([]byte(`{"generationConfig":{"thinkingConfig":{"thinkingBudget":4000}}}`))
	compat := geminiThinkingLevelFromClaudeThinking(&antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: 4000})
	require.Equal(t, native, compat)
}

// 裸名解析在 getMappedModel 层覆盖全部 Antigravity 转发入口
// （/v1beta 原生、/v1/chat/completions、/v1/messages）以及账号测试。
func TestGetMappedModelResolvesBareGeminiModelForAllEntrypoints(t *testing.T) {
	svc := &AntigravityGatewayService{}
	catalog := []string{
		"gemini-9.9-flash-low",
		"gemini-9.9-flash-medium",
		"gemini-9.9-flash-high",
	}
	account := &Account{Platform: PlatformAntigravity}
	account.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{Models: catalog})

	t.Run("default entrypoint falls back to high", func(t *testing.T) {
		require.Equal(t, "gemini-9.9-flash-high", svc.getMappedModel(account, "gemini-9.9-flash"))
	})

	t.Run("caller supplied level wins", func(t *testing.T) {
		require.Equal(t, "gemini-9.9-flash-low",
			svc.getMappedModelForThinkingLevel(account, "gemini-9.9-flash", "low"))
		require.Equal(t, "gemini-9.9-flash-medium",
			svc.getMappedModelForThinkingLevel(account, "gemini-9.9-flash", "medium"))
	})

	t.Run("models/ prefix is handled", func(t *testing.T) {
		require.Equal(t, "gemini-9.9-flash-high", svc.getMappedModel(account, "models/gemini-9.9-flash"))
	})

	t.Run("suffixed model keeps normal mapping", func(t *testing.T) {
		require.Equal(t, "gemini-9.9-flash-low", svc.getMappedModel(account, "gemini-9.9-flash-low"))
	})

	t.Run("unsupported model still reports unsupported", func(t *testing.T) {
		require.Empty(t, svc.getMappedModel(account, "gemini-9.8-flash"))
	})

	for _, tc := range []struct {
		name    string
		mapping any
		want    string
	}{
		{"explicit bare alias", map[string]any{"gemini-9.9-flash": "custom-upstream"}, "custom-upstream"},
		{"explicit wildcard alias", map[string]string{"gemini-9.9*": "custom-upstream"}, "custom-upstream"},
		{"explicit bare identity", map[string]string{"gemini-9.9-flash": "gemini-9.9-flash"}, "gemini-9.9-flash"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			aliasAccount := newAntigravityAccountWithMapping(nil)
			aliasAccount.Credentials["model_mapping"] = tc.mapping
			aliasAccount.SetUpstreamModelCatalogSnapshot(UpstreamModelCatalogSnapshot{Models: catalog})

			require.Equal(t, tc.want, svc.getMappedModelForThinkingLevel(aliasAccount, "gemini-9.9-flash", "low"))
		})
	}
}
