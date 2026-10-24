package antigravity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaudeModelsFromIDs_UsesLiveCatalog(t *testing.T) {
	t.Parallel()

	models := ClaudeModelsFromIDs([]string{"claude-future", "gemini-9.9-flash-medium", ""})

	require.Len(t, models, 2)
	require.Equal(t, "claude-future", models[0].ID)
	require.Equal(t, "model", models[0].Type)
	require.Equal(t, "gemini-9.9-flash-medium", models[1].DisplayName)
}

func TestGeminiModelsFromIDs_UsesLiveCatalog(t *testing.T) {
	t.Parallel()

	models := GeminiModelsFromIDs([]string{
		"gemini-9.9-flash-medium",
		"models/gemini-9.9-flash-high",
		"claude-future",
		"",
	})

	require.Len(t, models, 2)
	require.Equal(t, "models/gemini-9.9-flash-medium", models[0].Name)
	require.Equal(t, "models/gemini-9.9-flash-high", models[1].Name)
	require.Equal(t, []string{"generateContent", "streamGenerateContent"}, models[0].SupportedGenerationMethods)
}

func TestGeminiModelFromIDUsesCatalogID(t *testing.T) {
	t.Parallel()

	model := GeminiModelFromID("models/gemini-9.9-flash-medium")
	require.Equal(t, "models/gemini-9.9-flash-medium", model.Name)
	require.Equal(t, "gemini-9.9-flash-medium", model.DisplayName)
	require.Equal(t, []string{"generateContent", "streamGenerateContent"}, model.SupportedGenerationMethods)
}

func TestIsGeminiReasoningModel(t *testing.T) {
	t.Parallel()

	for _, model := range []string{
		"gemini-9.9-flash-high",
		"gemini-9.9-flash-medium",
		"gemini-9.9-flash-low",
		"gemini-9.9-flash-tiered",
		"models/gemini-9.9-flash-medium",
		"gemini-3.1-pro-high",
		"gemini-pro-agent",
		"gemini-4-argon",
		"gemini-4-argon-high",
		"models/gemini-9.9-argon-medium",
	} {
		require.True(t, IsGeminiReasoningModel(model), model)
	}
	for _, model := range []string{"gemini-9.9-flash", "gemini-2.5-flash", "gpt-oss-120b-medium"} {
		require.False(t, IsGeminiReasoningModel(model), model)
	}
}

func TestClaudeContentItem_ThinkingAlwaysHasSignature(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		item ClaudeContentItem
		want string
	}{
		{
			name: "thinking with empty signature",
			item: ClaudeContentItem{Type: "thinking", Thinking: "plan"},
			want: `{"type":"thinking","thinking":"plan","signature":""}`,
		},
		{
			name: "empty thinking",
			item: ClaudeContentItem{Type: "thinking"},
			want: `{"type":"thinking","thinking":"","signature":""}`,
		},
		{
			name: "text without thinking fields",
			item: ClaudeContentItem{Type: "text", Text: "hi"},
			want: `{"type":"text","text":"hi"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.item)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))
		})
	}
}
