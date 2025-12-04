// API Configuration - centralized URL management
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001'

export const API_ENDPOINTS = {
  base: API_BASE_URL,
  upload: `${API_BASE_URL}/upload`,
  query: `${API_BASE_URL}/query`,
  status: `${API_BASE_URL}/status`,
  preview: `${API_BASE_URL}/preview`,
  kpis: `${API_BASE_URL}/kpis`,
  visualizations: `${API_BASE_URL}/visualizations`,
  columnTypes: `${API_BASE_URL}/column-types`,
  filter: `${API_BASE_URL}/filter`,
  correlation: `${API_BASE_URL}/correlation`,
  columnSimilarity: `${API_BASE_URL}/column-similarity`,
  exportMapper: `${API_BASE_URL}/export-mapper`,
} as const
