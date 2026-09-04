export interface GatewayModelListItem {
  id: string
}

function normalizeGatewayBaseUrl(baseUrl: string): string {
  const fallback = typeof window !== 'undefined' ? window.location.origin : ''
  const value = (baseUrl || fallback).trim().replace(/\/+$/, '')
  if (!value) return '/v1'
  return /\/v1$/i.test(value) ? value : `${value}/v1`
}

export function buildGatewayModelsUrl(baseUrl: string): string {
  return `${normalizeGatewayBaseUrl(baseUrl)}/models`
}

function modelIDsFromResponse(value: unknown): string[] {
  if (typeof value !== 'object' || value === null) return []
  const data = (value as { data?: unknown }).data
  if (!Array.isArray(data)) return []

  const seen = new Set<string>()
  const modelIDs: string[] = []
  for (const item of data) {
    if (typeof item !== 'object' || item === null) continue
    const rawModelID = (item as { id?: unknown }).id
    if (typeof rawModelID !== 'string') continue
    const modelID = rawModelID.trim()
    if (!modelID || seen.has(modelID)) continue
    seen.add(modelID)
    modelIDs.push(modelID)
  }
  return modelIDs
}

export async function fetchGatewayModelIDs(
  baseUrl: string,
  apiKey: string,
  signal?: AbortSignal
): Promise<string[]> {
  const response = await fetch(buildGatewayModelsUrl(baseUrl), {
    method: 'GET',
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${apiKey}`
    },
    cache: 'no-store',
    signal
  })

  if (!response.ok) {
    throw new Error(`Gateway models request failed with status ${response.status}`)
  }

  return modelIDsFromResponse(await response.json())
}
