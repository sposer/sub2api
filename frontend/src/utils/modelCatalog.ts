interface ModelWithID {
  id: string
}

interface GeminiSortKey {
  major: number
  minor: number
  family: number
  tier: number
  id: string
}

const geminiTierOrder: Record<string, number> = {
  '': 0,
  high: 1,
  medium: 2,
  low: 3,
  tiered: 4
}

function geminiSortKey(modelID: string): GeminiSortKey | null {
  const id = modelID.trim().toLowerCase()
  if (!id.startsWith('gemini-')) return null

  const version = id.match(/^gemini-(\d+)(?:\.(\d+))?(?:-|$)/)
  const suffix = id.replace(/^gemini-(?:\d+)(?:\.\d+)?-?/, '')
  const tierMatch = suffix.match(/(?:^|-)(high|medium|low|tiered)$/)

  return {
    major: Number(version?.[1] ?? -1),
    minor: Number(version?.[2] ?? 0),
    family: suffix.includes('flash') ? 0 : suffix.includes('pro') ? 1 : 2,
    tier: geminiTierOrder[tierMatch?.[1] ?? ''] ?? 5,
    id
  }
}

// decision: account-test models are ordered by parsed family/version; not a per-version
// allowlist because Antigravity availability is supplied by the live account catalog.
export function sortAccountTestModels<T extends ModelWithID>(models: T[]): T[] {
  return models
    .map((model, index) => ({ model, index, key: geminiSortKey(model.id) }))
    .sort((a, b) => {
      if (a.key === null || b.key === null) {
        if (a.key !== null) return -1
        if (b.key !== null) return 1
        return a.index - b.index
      }
      if (a.key.major !== b.key.major) return b.key.major - a.key.major
      if (a.key.minor !== b.key.minor) return b.key.minor - a.key.minor
      if (a.key.family !== b.key.family) return a.key.family - b.key.family
      if (a.key.tier !== b.key.tier) return a.key.tier - b.key.tier
      return a.key.id.localeCompare(b.key.id)
    })
    .map(({ model }) => model)
}
