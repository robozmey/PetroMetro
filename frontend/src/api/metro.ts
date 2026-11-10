import type { City, MetroSchema } from '../types/metro'

export type ImportPreview = {
  stationsCount: number
  linesCount: number
  schema: Pick<MetroSchema, 'stations' | 'lines'>
}

export async function previewOverpassImport(file: File, signal?: AbortSignal): Promise<ImportPreview> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch('/api/cities/preview-import', {
    method: 'POST',
    body: form,
    signal,
  })
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { error?: string } | null
    throw new Error(body?.error ?? 'Не удалось обработать файл Overpass')
  }
  return response.json() as Promise<ImportPreview>
}

export async function createCity(
  city: { name: string; slug: string },
  schema: ImportPreview['schema'],
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch('/api/cities', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ ...city, ...schema }),
    signal,
  })
  if (!response.ok) {
    const body = await response.json().catch(() => null) as { error?: string } | null
    throw new Error(body?.error ?? 'Не удалось создать город')
  }
}

export async function getMetroSchema(
  citySlug: string,
  signal?: AbortSignal,
): Promise<MetroSchema> {
  const response = await fetch(`/api/cities/${citySlug}/metro/schema`, { signal })

  if (!response.ok) {
    throw new Error('Не удалось загрузить схему метро')
  }

  return response.json() as Promise<MetroSchema>
}

export async function getCities(signal?: AbortSignal): Promise<City[]> {
  const response = await fetch('/api/cities', { signal })

  if (!response.ok) {
    throw new Error('Не удалось загрузить список городов')
  }

  return response.json() as Promise<City[]>
}
