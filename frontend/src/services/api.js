const API = 'http://localhost:8080/api'

async function request(path, options) {
  const response = await fetch(`${API}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  })
  const body = response.status === 204 ? null : await response.json().catch(() => null)
  if (!response.ok) throw new Error(body?.error || `Request failed (${response.status})`)
  return body
}

export const api = {
  usage: () => request('/water-usage'),
  summary: () => request('/water-usage/summary'),
  addUsage: (payload) => request('/water-usage', { method: 'POST', body: JSON.stringify(payload) }),
  predictions: () => request('/predictions'),
  latest: () => request('/predictions/latest'),
  predict: (payload) => request('/predictions', { method: 'POST', body: JSON.stringify(payload) }),
}
