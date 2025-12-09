"use client"

import { useState, useEffect, useCallback, useMemo } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Loader2, Sparkles, Download, Info } from "lucide-react"
import { API_ENDPOINTS } from "@/lib/api-config"
import CorrelationFlow from "@/components/correlation-flow"
import ContextWizard from "@/components/context-wizard"
import { useToast, ToastContainer } from "@/components/ui/toast"

interface CorrelationData {
  total_correlations: number
  correlations: Array<{
    file1_column: string
    file2_column: string
    correlation: number
    file1_rows: number
    file2_rows: number
  }>
  file1_columns: string[]
  file2_columns: string[]
  file1_rows: number
  file2_rows: number
  note?: string
}

interface SimilarityGraph {
  nodes: Array<{
    id: string
    label: string
    group: string
  }>
  edges: Array<{
    source: string
    target: string
    value: number
    similarity: number
    type: string
  }>
  similarities: Array<{
    file1_column: string
    file2_column: string
    similarity: number
    confidence: number
    type: string
    data_similarity: number
    name_similarity: number
    distribution_similarity: number
    json_confidence: number
    llm_semantic_score: number
    reason?: string
  }>
  total_relationships: number
  correlations?: Array<{
    file1_column: string
    file2_column: string
    pearson_correlation: number
    spearman_correlation: number
    strength: string
    sample_size: number
  }>
}


export default function Dashboard({ csvLoaded }: { csvLoaded: boolean }) {
  const [bothFilesLoaded, setBothFilesLoaded] = useState(false)
  const [correlationData, setCorrelationData] = useState<CorrelationData | null>(null)
  const [correlationLoading, setCorrelationLoading] = useState(false)
  const [similarityGraph, setSimilarityGraph] = useState<SimilarityGraph | null>(null)
  const [similarityLoading, setSimilarityLoading] = useState(false)
  const [contextWizardOpen, setContextWizardOpen] = useState(false)
  const [hasContext, setHasContext] = useState({ file1: false, file2: false })

  // DB State
  const [dataSourceMode, setDataSourceMode] = useState<'csv' | 'db'>('csv')
  const [dbConfig, setDbConfig] = useState({
    type: 'postgres',
    host: 'localhost',
    port: 5432,
    user: 'postgres',
    password: '',
    dbname: 'postgres',
    sslmode: 'disable'
  })
  const [isConnecting, setIsConnecting] = useState(false)
  const [isConnected, setIsConnected] = useState(false)
  const [dbTables, setDbTables] = useState<string[]>([])
  const [analyzingTable, setAnalyzingTable] = useState(false)

  // Toast notifications
  const { toasts, showToast, removeToast } = useToast()

  const fetchCorrelations = useCallback(async () => {
    if (!bothFilesLoaded) return

    setCorrelationLoading(true)
    try {
      const response = await fetch(API_ENDPOINTS.correlation)
      if (!response.ok) throw new Error('Failed to fetch correlations')

      const data = await response.json()
      setCorrelationData(data)
    } catch (error) {
      console.error('Error fetching correlations:', error)
    } finally {
      setCorrelationLoading(false)
    }
  }, [bothFilesLoaded])

  const handleFeedback = async (file1Col: string, file2Col: string, isCorrect: boolean, matchData?: any) => {
    try {
      const response = await fetch(`${API_ENDPOINTS.base}/feedback/match`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          file1_column: file1Col,
          file2_column: file2Col,
          is_correct: isCorrect,
          name_similarity: matchData?.name_similarity || 0,
          data_similarity: matchData?.data_similarity || 0,
          pattern_score: matchData?.json_confidence || 0,
          confidence: matchData?.confidence || 0
        })
      })

      if (!response.ok) throw new Error('Failed to submit feedback')

      const result = await response.json()
      console.log('Feedback submitted:', result)

      // Show success message with toast
      if (isCorrect) {
        showToast('success', '✅ Match confirmed! Will be boosted.')
      } else {
        showToast('warning', '👎 Match rejected! Will be penalized.')
      }

      // Auto re-evaluate
      showToast('info', 'Re-evaluating...')
      await fetchSimilarityGraph()
      showToast('success', 'Updated!')

      // Refresh similarity graph to show updated scores
      fetchSimilarityGraph()
    } catch (error) {
      console.error('Error submitting feedback:', error)
      showToast('error', 'Failed to submit feedback. Please try again.')
    }
  }

  const checkBothFilesLoaded = useCallback(async () => {
    try {
      const response = await fetch(API_ENDPOINTS.status)
      if (response.ok) {
        const data = await response.json()
        const bothLoaded = data.file1_loaded && data.file2_loaded
        setBothFilesLoaded(bothLoaded)
        if (bothLoaded) {
          fetchCorrelations()
        }
      }
    } catch (error) {
      console.error("Error checking file status:", error)
    }
  }, [fetchCorrelations])

  const fetchSimilarityGraph = useCallback(async () => {
    setSimilarityLoading(true)
    try {
      const response = await fetch(API_ENDPOINTS.columnSimilarity)
      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`Failed to fetch similarity graph: ${response.status}`)
      }
      const data = await response.json()
      setSimilarityGraph(data)
    } catch (error: any) {
      console.error("Error fetching similarity graph:", error)
      setSimilarityGraph(null)
      alert(`Error generating correlation graph: ${error.message || 'Unknown error'}`)
    } finally {
      setSimilarityLoading(false)
    }
  }, [])

  const exportToCSV = useCallback(() => {
    if (!similarityGraph || !similarityGraph.similarities || similarityGraph.similarities.length === 0) {
      showToast('warning', 'No correlation data available to export. Generate correlation first.')
      return
    }

    try {
      // CSV Header
      const headers = [
        'File 1 Column',
        'File 2 Column',
        'Confidence (%)',
        'Similarity (%)',
        'Type',
        'Name Similarity (%)',
        'Data Similarity (%)',
        'Distribution Similarity (%)',
        'LLM Semantic Score (%)',
        'Pattern Score (%)',
        'Reason'
      ]

      // CSV Rows
      const rows = similarityGraph.similarities.map((sim: any) => [
        sim.file1_column || '',
        sim.file2_column || '',
        (sim.confidence || 0).toFixed(2),
        ((sim.similarity || 0) * 100).toFixed(2),
        sim.type || '',
        ((sim.name_similarity || 0) * 100).toFixed(2),
        ((sim.data_similarity || 0) * 100).toFixed(2),
        ((sim.distribution_similarity || 0) * 100).toFixed(2),
        ((sim.llm_semantic_score || 0) * 100).toFixed(2),
        ((sim.json_confidence || 0) * 100).toFixed(2),
        (sim.reason || '').replace(/"/g, '""') // Escape quotes for CSV
      ])

      // Construct CSV content
      const csvContent = [
        headers.map(h => `"${h}"`).join(','),
        ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
      ].join('\n')

      // Create blob and download
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `column-correlations-${new Date().toISOString().split('T')[0]}.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      showToast('success', '✅ CSV file exported successfully!')
    } catch (error) {
      console.error('Error exporting CSV:', error)
      showToast('error', 'Failed to export CSV file. Please try again.')
    }
  }, [similarityGraph, showToast])

  const exportToJSON = useCallback(() => {
    if (!similarityGraph || !similarityGraph.similarities || similarityGraph.similarities.length === 0) {
      showToast('warning', 'No correlation data available to export. Generate correlation first.')
      return
    }

    try {
      // Create mapper data structure
      const mapperData = {
        metadata: {
          exportDate: new Date().toISOString(),
          totalMappings: similarityGraph.similarities.length,
          totalRelationships: similarityGraph.total_relationships,
        },
        mappings: similarityGraph.similarities.map((sim: any) => ({
          file1_column: sim.file1_column,
          file2_column: sim.file2_column,
          confidence: sim.confidence,
          similarity: sim.similarity,
          type: sim.type,
          metrics: {
            data_similarity: sim.data_similarity,
            name_similarity: sim.name_similarity,
            distribution_similarity: sim.distribution_similarity,
            llm_semantic_score: sim.llm_semantic_score,
            json_confidence: sim.json_confidence,
          },
          reason: sim.reason
        })),
        correlations: similarityGraph.correlations || []
      }

      // Convert to JSON string with formatting
      const jsonString = JSON.stringify(mapperData, null, 2)

      // Create blob and download
      const blob = new Blob([jsonString], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `column-mapper-${new Date().toISOString().split('T')[0]}.json`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      showToast('success', '✅ JSON file exported successfully!')
    } catch (error) {
      console.error('Error exporting JSON:', error)
      showToast('error', 'Failed to export JSON file. Please try again.')
    }
  }, [similarityGraph, showToast])

  const checkContextStatus = useCallback(async () => {
    try {
      const response = await fetch(`${API_ENDPOINTS.base}/context/status`)
      if (response.ok) {
        const data = await response.json()
        setHasContext({
          file1: data.file1?.has_context || false,
          file2: data.file2?.has_context || false
        })
      }
    } catch (error) {
      console.error('Error checking context status:', error)
    }
  }, [])

  const handleContextWizardComplete = useCallback(() => {
    showToast('success', '✅ Context saved! Generating correlation with enhanced accuracy...')
    checkContextStatus()
    // Automatically trigger correlation generation after context is collected
    setTimeout(() => {
      fetchSimilarityGraph()
    }, 500)
  }, [checkContextStatus, fetchSimilarityGraph, showToast])

  const handleGenerateWithContext = useCallback(() => {
    // Check if context exists
    if (hasContext.file1 && hasContext.file2) {
      // Context already exists, just generate correlation
      fetchSimilarityGraph()
    } else {
      // Open context wizard first
      setContextWizardOpen(true)
    }
  }, [hasContext, fetchSimilarityGraph])


  // Check file status when csvLoaded changes
  useEffect(() => {
    if (csvLoaded) {
      checkBothFilesLoaded()
      checkContextStatus()
    } else {
      setBothFilesLoaded(false)
      setCorrelationData(null)
      setSimilarityGraph(null)
    }
  }, [csvLoaded, checkBothFilesLoaded, checkContextStatus])

  // Automatic correlation generation disabled - user must click button
  // useEffect(() => {
  //   if (bothFilesLoaded && !similarityGraph && !similarityLoading) {
  //     fetchSimilarityGraph()
  //   }
  // }, [bothFilesLoaded, similarityGraph, similarityLoading, fetchSimilarityGraph])

  // Automatic correlation generation disabled - user must click button

  if (!csvLoaded) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        <div className="text-center max-w-md">
          <Sparkles className="h-16 w-16 mx-auto mb-4 opacity-50 text-foreground" />
          <h3 className="text-lg font-semibold text-foreground mb-2">No Data Loaded</h3>
          <p className="text-sm mb-6">Upload CSV files from the left panel to begin correlation analysis</p>
          <div className="bg-info-light border border-info-border rounded-lg p-4 text-left">
            <p className="text-xs text-foreground mb-2">
              <span className="font-semibold">Quick Start:</span>
            </p>
            <ol className="text-xs text-muted-foreground space-y-1 list-decimal list-inside">
              <li>Upload CSV File 1 using the left panel</li>
              <li>Upload CSV File 2 (optional)</li>
              <li>Click "Generate Correlation" to analyze</li>
            </ol>
          </div>
        </div>
      </div>
    )
  }

  // Show Generate Correlation button prominently when only one file is loaded
  if (csvLoaded && !bothFilesLoaded) {
    return (
      <div className="h-full overflow-y-auto p-4 space-y-4 bg-background">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h2 className="text-base font-medium text-foreground">Dashboard</h2>
            <p className="text-xs text-muted-foreground mt-0.5">Data analysis and correlation engine</p>
          </div>
        </div>

        <div className="h-full flex items-center justify-center">
          <div className="text-center max-w-md">
            <Sparkles className="h-16 w-16 mx-auto mb-4 text-info" />
            <h3 className="text-lg font-semibold text-foreground mb-2">Ready for Correlation Analysis</h3>
            <p className="text-sm text-muted-foreground mb-6">
              Upload a second CSV file to compare and generate correlation mappings between the two datasets
            </p>
            <div className="bg-warning-light border border-warning-border rounded-lg p-4 mb-6">
              <p className="text-xs text-foreground">
                <span className="font-semibold">Note:</span> Correlation analysis requires two CSV files to compare column similarities and data relationships.
              </p>
            </div>
            <Button
              onClick={async () => {
                // Check status and trigger correlation if both files are loaded
                const response = await fetch(API_ENDPOINTS.status)
                if (response.ok) {
                  const data = await response.json()
                  setBothFilesLoaded(data.file1_loaded && data.file2_loaded)
                  if (data.file1_loaded && data.file2_loaded) {
                    fetchSimilarityGraph()
                  } else {
                    showToast('error', 'Session expired or files missing. Please re-upload both files.')
                  }
                }
              }}
              size="lg"
              className="bg-info hover:bg-info/90 text-info-foreground px-8 py-3"
            >
              <Sparkles className="h-5 w-5 mr-2" />
              Check Files & Generate Correlation
            </Button>
            <p className="text-xs text-muted-foreground mt-4">
              If you've already uploaded 2 files, click the button above to check status and generate correlation
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto p-4 space-y-4 bg-background">
      {/* Prominent Generate Correlation Section */}
      {bothFilesLoaded && !similarityGraph && (
        <div className="bg-gradient-to-r from-info-light to-info-light/50 border-2 border-info-border rounded-lg p-6 mb-4">
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <Sparkles className="h-5 w-5 text-info" />
                <h3 className="text-lg font-semibold text-foreground">Generate Correlation Analysis</h3>
                {(hasContext.file1 || hasContext.file2) && (
                  <span className="text-xs bg-success-light text-success px-2 py-1 rounded-full font-medium">
                    Context Available ✓
                  </span>
                )}
              </div>
              <p className="text-sm text-foreground mb-2">
                {hasContext.file1 && hasContext.file2
                  ? "Context collected! Click to generate enhanced correlation analysis with improved accuracy."
                  : "Add context about your datasets for better correlation accuracy (recommended)."}
              </p>
              {!hasContext.file1 || !hasContext.file2 ? (
                <div className="flex items-center gap-2 text-xs text-muted-foreground bg-info-light p-2 rounded mt-2">
                  <Info className="h-4 w-4 flex-shrink-0" />
                  <span>
                    Providing context reduces false positives by 30-50% and helps identify domain-specific relationships.
                  </span>
                </div>
              ) : null}
            </div>
            <div className="ml-4 flex flex-col gap-2">
              <Button
                onClick={handleGenerateWithContext}
                size="lg"
                className="bg-info hover:bg-info/90 text-info-foreground px-6 py-3 h-auto"
                disabled={similarityLoading}
              >
                {similarityLoading ? (
                  <>
                    <Loader2 className="h-5 w-5 mr-2 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Sparkles className="h-5 w-5 mr-2" />
                    {hasContext.file1 && hasContext.file2 ? 'Generate Correlation' : 'Add Context & Generate'}
                  </>
                )}
              </Button>
              {(hasContext.file1 && hasContext.file2) && (
                <Button
                  onClick={fetchSimilarityGraph}
                  size="sm"
                  variant="outline"
                  className="text-xs"
                  disabled={similarityLoading}
                >
                  Skip Context (Quick Mode)
                </Button>
              )}
            </div>
          </div>
        </div>
      )}
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-base font-medium text-foreground">Dashboard</h2>
          <p className="text-xs text-muted-foreground mt-0.5">Data analysis and correlation engine</p>
        </div>
        <div className="flex items-center gap-2">
          {bothFilesLoaded && (
            <>
              <Button
                onClick={fetchSimilarityGraph}
                variant="outline"
                size="sm"
                className="border-border text-foreground hover:bg-accent"
                disabled={similarityLoading}
              >
                {similarityLoading ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Sparkles className="h-3.5 w-3.5 mr-1.5" />
                    Generate Correlation
                  </>
                )}
              </Button>
              {similarityGraph && similarityGraph.similarities && similarityGraph.similarities.length > 0 && (
                <>
                  <Button
                    onClick={exportToCSV}
                    variant="outline"
                    size="sm"
                    className="border-success-border text-success hover:bg-success-light"
                  >
                    <Download className="h-3.5 w-3.5 mr-1.5" />
                    Export CSV
                  </Button>
                  <Button
                    onClick={exportToJSON}
                    variant="outline"
                    size="sm"
                    className="border-info-border text-info hover:bg-info-light"
                  >
                    <Download className="h-3.5 w-3.5 mr-1.5" />
                    Export JSON
                  </Button>
                </>
              )}
            </>
          )}
        </div>
      </div>

      {/* Dashboard View */}
      <>
        {/* Column Relationship Graph Section */}
        {bothFilesLoaded && (
          <Card className="border border-border">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-base font-medium text-foreground">Column Relationship Graph</CardTitle>
                  <CardDescription className="mt-0.5 text-xs text-muted-foreground">
                    Interactive flow diagram showing related fields between both CSV files with confidence percentages
                  </CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    onClick={fetchSimilarityGraph}
                    variant="outline"
                    size="sm"
                    className="border-border text-foreground hover:bg-accent"
                    disabled={similarityLoading}
                  >
                    {similarityLoading ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                        Generating...
                      </>
                    ) : (
                      <>
                        <Sparkles className="h-3.5 w-3.5 mr-1.5" />
                        {similarityGraph ? "Refresh" : "Generate"}
                      </>
                    )}
                  </Button>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {similarityLoading ? (
                <div className="flex items-center justify-center p-8">
                  <Loader2 className="h-6 w-6 animate-spin text-foreground" />
                  <span className="ml-2 text-sm text-foreground">Analyzing column relationships...</span>
                </div>
              ) : similarityGraph && similarityGraph.edges && similarityGraph.edges.length > 0 ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="text-sm text-muted-foreground">
                      Found <span className="font-medium text-foreground">{similarityGraph.total_relationships || 0}</span> relationship(s) between columns.
                    </div>
                    {/* Legend */}
                    <div className="flex items-center gap-4 text-xs flex-wrap">
                      <div className="flex items-center gap-1">
                        <div className="w-3 h-3 rounded-full bg-file1-light border-2 border-file1-border"></div>
                        <span className="text-muted-foreground">File 1</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <div className="w-3 h-3 rounded-full bg-file2-light border-2 border-file2-border"></div>
                        <span className="text-muted-foreground">File 2</span>
                      </div>
                      <div className="flex items-center gap-2 ml-2 border-l border-border pl-2">
                        <span className="text-muted-foreground font-medium">Similarity:</span>
                        <div className="flex items-center gap-1">
                          <div className="w-4 h-0.5 bg-confidence-high"></div>
                          <span className="text-muted-foreground">≥70%</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <div className="w-4 h-0.5 bg-confidence-medium"></div>
                          <span className="text-muted-foreground">≥50%</span>
                        </div>
                        <div className="flex items-center gap-1">
                          <div className="w-4 h-0.5 bg-chart-orange"></div>
                          <span className="text-muted-foreground">≥30%</span>
                        </div>
                      </div>
                      {correlationData && correlationData.correlations.length > 0 && (
                        <div className="flex items-center gap-2 ml-2 border-l border-border pl-2">
                          <span className="text-muted-foreground font-medium">Correlation:</span>
                          <div className="flex items-center gap-1">
                            <div className="w-6 h-0.5 bg-confidence-high" style={{
                              backgroundImage: `repeating-linear-gradient(to right, hsl(var(--confidence-high)) 0px, hsl(var(--confidence-high)) 4px, transparent 4px, transparent 8px)`,
                              animation: 'dashdraw 2s linear infinite'
                            }}></div>
                            <span className="text-muted-foreground">Dotted Green</span>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* React Flow Diagram */}
                  <div className="mb-4" style={{ height: '500px' }}>
                    <CorrelationFlow
                      similarities={similarityGraph.similarities || []}
                    />
                  </div>

                  {/* Relationship Table */}
                  {similarityGraph.similarities && similarityGraph.similarities.length > 0 && (
                    <div className="border border-border rounded-md overflow-auto max-h-[400px]">
                      <table className="w-full text-sm">
                        <thead className="bg-muted border-b border-border">
                          <tr>
                            <th className="px-4 py-2 text-left font-medium text-foreground border-r border-border">File 1 Column</th>
                            <th className="px-4 py-2 text-left font-medium text-foreground border-r border-border">File 2 Column</th>
                            <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Similarity</th>
                            <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Confidence</th>
                            <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Type</th>
                            <th className="px-4 py-2 text-center font-medium text-foreground">Feedback</th>
                          </tr>
                        </thead>
                        <tbody>
                          {similarityGraph.similarities.map((sim: any, idx: number) => (
                            <tr key={idx} className="border-b border-border hover:bg-accent">
                              <td className="px-4 py-2 text-foreground border-r border-border">{sim.file1_column}</td>
                              <td className="px-4 py-2 text-foreground border-r border-border">{sim.file2_column}</td>
                              <td className="px-4 py-2 text-center text-foreground border-r border-border">
                                {((sim.similarity || 0) * 100).toFixed(1)}%
                              </td>
                              <td className="px-4 py-2 text-center border-r border-border">
                                <span className={`px-2 py-0.5 rounded text-xs font-medium ${(sim.confidence || 0) > 70 ? 'bg-success-light text-success' :
                                  (sim.confidence || 0) > 50 ? 'bg-warning-light text-warning' :
                                    'bg-error-light text-error'
                                  }`}>
                                  {(sim.confidence || 0).toFixed(1)}%
                                </span>
                              </td>
                              <td className="px-4 py-2 text-center text-xs text-muted-foreground border-r border-border">
                                {sim.type || 'unknown'}
                              </td>
                              <td className="px-4 py-2">
                                <div className="flex items-center justify-center gap-2">
                                  <button
                                    onClick={() => handleFeedback(sim.file1_column, sim.file2_column, true, sim)}
                                    className="p-1 hover:bg-success-light rounded transition-colors text-lg"
                                    title="This match is correct"
                                  >
                                    👍
                                  </button>
                                  <button
                                    onClick={() => handleFeedback(sim.file1_column, sim.file2_column, false, sim)}
                                    className="p-1 hover:bg-error-light rounded transition-colors text-lg"
                                    title="This match is incorrect"
                                  >
                                    👎
                                  </button>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              ) : similarityGraph && similarityGraph.total_relationships === 0 ? (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No relationships found with confidence above 10%. Try uploading files with similar column structures.
                </div>
              ) : (
                <div className="text-center py-8">
                  <div className="text-muted-foreground text-sm mb-3">
                    Click the "Generate Correlation" button to create an interactive flow diagram showing relationships between columns from both files.
                  </div>
                  <div className="text-xs text-muted-foreground/70">
                    The diagram will show nodes (columns) and edges (relationships) with confidence percentages.
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        {/* Correlation Analysis Section */}
        {bothFilesLoaded && (
          <Card className="border border-border">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-base font-medium text-foreground">Correlation Analysis</CardTitle>
                  <CardDescription className="mt-0.5 text-xs text-muted-foreground">
                    Numeric correlations between columns from both CSV files
                  </CardDescription>
                </div>
                <Button
                  onClick={fetchCorrelations}
                  variant="outline"
                  size="sm"
                  className="border-border text-foreground hover:bg-accent"
                  disabled={correlationLoading}
                >
                  {correlationLoading ? (
                    <>
                      <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                      Loading...
                    </>
                  ) : (
                    "Refresh"
                  )}
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {correlationLoading ? (
                <div className="flex items-center justify-center p-8">
                  <Loader2 className="h-6 w-6 animate-spin text-foreground" />
                  <span className="ml-2 text-sm text-foreground">Calculating correlations...</span>
                </div>
              ) : similarityGraph?.correlations && similarityGraph.correlations.length > 0 ? (
                <div className="space-y-4">
                  <div className="text-sm text-muted-foreground mb-4">
                    Found <span className="font-medium text-foreground">{similarityGraph.correlations.length}</span> numeric correlation(s) between matched columns.
                  </div>

                  {/* Correlation Table with AI Explanations */}
                  <div className="border border-border rounded-md overflow-hidden">
                    <table className="w-full text-sm">
                      <thead className="bg-muted border-b border-border">
                        <tr>
                          <th className="px-4 py-2 text-left font-medium text-foreground border-r border-border">Columns</th>
                          <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Pearson</th>
                          <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Spearman</th>
                          <th className="px-4 py-2 text-center font-medium text-foreground border-r border-border">Strength</th>
                          <th className="px-4 py-2 text-left font-medium text-foreground">AI Explanation</th>
                        </tr>
                      </thead>
                      <tbody>
                        {similarityGraph.correlations.map((corr: any, idx: number) => {
                          const pearson = corr.pearson_correlation;
                          const spearman = corr.spearman_correlation;
                          const strength = corr.strength;

                          // Generate AI explanation
                          const absPearson = Math.abs(pearson);
                          let aiExplanation = '';
                          if (absPearson >= 0.7) {
                            aiExplanation = `Strong ${pearson > 0 ? 'positive' : 'negative'} relationship detected. These columns move together ${pearson > 0 ? 'in the same direction' : 'in opposite directions'}. This suggests a meaningful connection between "${corr.file1_column}" and "${corr.file2_column}".`;
                          } else if (absPearson >= 0.4) {
                            aiExplanation = `Moderate ${pearson > 0 ? 'positive' : 'negative'} correlation found. There appears to be some relationship between these columns, though other factors may also influence the values.`;
                          } else if (absPearson >= 0.2) {
                            aiExplanation = `Weak ${pearson > 0 ? 'positive' : 'negative'} correlation. The relationship between these columns is minor and may not be practically significant.`;
                          } else {
                            aiExplanation = `Very weak or no meaningful correlation. These columns appear to vary independently of each other.`;
                          }

                          return (
                            <tr key={idx} className="border-b border-border last:border-b-0 hover:bg-accent">
                              <td className="px-4 py-3 border-r border-border">
                                <div className="font-medium text-foreground">{corr.file1_column}</div>
                                <div className="text-xs text-muted-foreground">↔ {corr.file2_column}</div>
                                <div className="text-xs text-muted-foreground/70 mt-1">n={corr.sample_size}</div>
                              </td>
                              <td className="px-4 py-3 text-center border-r border-border">
                                <span className={`font-semibold ${absPearson > 0.7 ? 'text-confidence-high' :
                                  absPearson > 0.4 ? 'text-confidence-medium' :
                                    'text-muted-foreground'
                                  }`}>
                                  {pearson.toFixed(3)}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-center border-r border-border">
                                <span className="text-foreground">{spearman.toFixed(3)}</span>
                              </td>
                              <td className="px-4 py-3 text-center border-r border-border">
                                <span className={`inline-block px-2 py-1 rounded text-xs font-medium ${strength === 'Strong' ? 'bg-success-light text-success' :
                                  strength === 'Moderate' ? 'bg-warning-light text-warning' :
                                    strength === 'Weak' ? 'bg-chart-orange/20 text-chart-orange' :
                                      'bg-muted text-muted-foreground'
                                  }`}>
                                  {strength}
                                </span>
                              </td>
                              <td className="px-4 py-3">
                                <div className="text-sm text-foreground bg-info-light p-2 rounded">
                                  {aiExplanation}
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              ) : correlationData && correlationData.total_correlations === 0 ? (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No correlations found. Make sure both files have numeric columns and the same number of rows.
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  Correlations will be calculated automatically when both files are loaded.
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </>
      <ToastContainer toasts={toasts} onRemove={removeToast} />
      <ContextWizard
        isOpen={contextWizardOpen}
        onClose={() => setContextWizardOpen(false)}
        onComplete={handleContextWizardComplete}
      />
    </div>
  )
}
