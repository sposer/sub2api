import { afterEach, describe, expect, it, vi } from 'vitest'
import { buildGatewayModelsUrl, fetchGatewayModelIDs } from '../gatewayModels'

describe('gatewayModels', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('builds the standard model-list URL', () => {
    expect(buildGatewayModelsUrl('https://example.com')).toBe('https://example.com/v1/models')
    expect(buildGatewayModelsUrl('https://example.com/v1/')).toBe('https://example.com/v1/models')
  })

  it('returns unique live model IDs', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        object: 'list',
        data: [
          { id: 'gemini-9.9-flash-medium' },
          { id: 'gemini-9.9-flash-medium' },
          { id: 'claude-future' },
          { id: 42 }
        ]
      })
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(fetchGatewayModelIDs('https://example.com/v1', 'sk-test')).resolves.toEqual([
      'gemini-9.9-flash-medium',
      'claude-future'
    ])
    expect(fetchMock).toHaveBeenCalledWith('https://example.com/v1/models', expect.objectContaining({
      headers: expect.objectContaining({ Authorization: 'Bearer sk-test' })
    }))
  })
})
