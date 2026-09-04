import { describe, expect, it } from 'vitest'
import { sortAccountTestModels } from '../modelCatalog'

describe('sortAccountTestModels', () => {
  it('orders future Gemini versions and tiers without a version allowlist', () => {
    const models = [
      { id: 'claude-future' },
      { id: 'gemini-9.9-flash-low' },
      { id: 'gemini-10-flash-medium' },
      { id: 'gemini-9.9-flash' },
      { id: 'gemini-10-flash-high' }
    ]

    expect(sortAccountTestModels(models).map((model) => model.id)).toEqual([
      'gemini-10-flash-high',
      'gemini-10-flash-medium',
      'gemini-9.9-flash',
      'gemini-9.9-flash-low',
      'claude-future'
    ])
  })

  it('preserves the source order for non-Gemini models', () => {
    const models = [{ id: 'claude-b' }, { id: 'gpt-a' }]
    expect(sortAccountTestModels(models)).toEqual(models)
  })
})
