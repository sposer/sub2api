package antigravity

import (
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

func TestIsGeminiReasoningModel_DerivesFutureFlashTiers(t *testing.T) {
	t.Parallel()

	for _, model := range []string{
		"gemini-9.9-flash-high",
		"gemini-9.9-flash-medium",
		"gemini-9.9-flash-low",
		"gemini-9.9-flash-tiered",
		"models/gemini-9.9-flash-medium",
	} {
		require.True(t, IsGeminiReasoningModel(model), model)
	}
	require.True(t, IsGeminiReasoningModel("gemini-3.1-pro-high"))
	require.False(t, IsGeminiReasoningModel("gemini-9.9-flash"))
	require.False(t, IsGeminiReasoningModel("gpt-oss-120b-medium"))
}
