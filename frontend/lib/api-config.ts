// API Configuration - centralized URL management
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001'

export const API_ENDPOINTS = {
  base: API_BASE_URL,
  upload: `${API_BASE_URL}/api/analyze-file`,
  query: `${API_BASE_URL}/api/query`, // Not implemented yet
  status: `${API_BASE_URL}/api/status`, // Need to implement this
  preview: `${API_BASE_URL}/api/preview`, // Not implemented yet
  kpis: `${API_BASE_URL}/api/kpis`, // Not implemented yet
  visualizations: `${API_BASE_URL}/api/visualizations`, // Not implemented yet
  columnTypes: `${API_BASE_URL}/api/column-types`, // Not implemented yet
  filter: `${API_BASE_URL}/api/filter`, // Not implemented yet
  correlation: `${API_BASE_URL}/api/similarity/graph`, // Mapping correlation to graph for now
  columnSimilarity: `${API_BASE_URL}/api/similarity/graph`,
  exportMapper: `${API_BASE_URL}/api/export-mapper`, // Not implemented yet
} as const
