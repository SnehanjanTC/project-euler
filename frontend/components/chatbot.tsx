"use client"

import { useState, useRef, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Send, Upload, Loader2, FileText, X, CheckCircle2, Database, Server } from "lucide-react"
import { API_ENDPOINTS } from "@/lib/api-config"

interface Message {
  id?: string
  role: "user" | "assistant"
  content: string
  resultData?: any
  resultType?: string
  error?: string
}

interface ChatbotProps {
  onCsvLoadedChange?: (loaded: boolean) => void
}

export default function Chatbot({ onCsvLoadedChange }: ChatbotProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState("")
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [csvLoaded, setCsvLoaded] = useState(false)
  const [csvInfo, setCsvInfo] = useState<{ rows: number; columns: number; column_names: string[] } | null>(null)
  const [csvInfo2, setCsvInfo2] = useState<{ rows: number; columns: number; column_names: string[] } | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const [fileName2, setFileName2] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const fileInputRef2 = useRef<HTMLInputElement>(null)
  const messageIdCounter = useRef(0)

  const generateMessageId = () => {
    messageIdCounter.current += 1
    return `msg-${Date.now()}-${messageIdCounter.current}`
  }


  const [dataSource, setDataSource] = useState<'csv' | 'db'>('csv')
  const [dbConfig, setDbConfig] = useState({
    type: 'postgres',
    host: 'localhost',
    port: 5432,
    user: 'postgres',
    password: '',
    dbname: 'postgres',
    sslmode: 'disable'
  })
  const [dbTables, setDbTables] = useState<string[]>([])
  const [connecting, setConnecting] = useState(false)
  const [connected, setConnected] = useState(false)

  const handleConnect = async () => {
    setConnecting(true)
    try {
      const res = await fetch(`${API_ENDPOINTS.upload.replace('/analyze-file', '')}/db/connect`, {
        method: 'POST',
        body: JSON.stringify(dbConfig)
      })
      if (res.ok) {
        setConnected(true)
        const tables = await fetch(`${API_ENDPOINTS.upload.replace('/analyze-file', '')}/db/tables`).then(r => r.json())
        setDbTables(tables.tables || [])
        setMessages(prev => [...prev, { id: generateMessageId(), role: "assistant", content: "✅ Connected to Database! Select tables to analyze." }])
      } else {
        throw new Error("Connection failed")
      }
    } catch (e) {
      setMessages(prev => [...prev, { id: generateMessageId(), role: "assistant", content: "❌ Connection failed. Check credentials." }])
    } finally {
      setConnecting(false)
    }
  }

  const handleSelectTable = async (tableName: string, fileIndex: number) => {
    try {
      const res = await fetch(`${API_ENDPOINTS.upload.replace('/analyze-file', '')}/db/analyze`, {
        method: 'POST',
        body: JSON.stringify({ table_name: tableName, file_index: fileIndex })
      })
      if (res.ok) {
        const data = await res.json()
        setCsvLoaded(true)
        onCsvLoadedChange?.(true)
        if (fileIndex === 1) {
          setFileName(`DB: ${tableName}`)
          setCsvInfo({ rows: data.rows, columns: data.columns, column_names: data.column_names })
        } else {
          setFileName2(`DB: ${tableName}`)
          setCsvInfo2({ rows: data.rows, columns: data.columns, column_names: data.column_names })
        }
        checkStatus() // updates context
      }
    } catch (e) {
      console.error(e)
    }
  }

  const generateExampleQuestions = async (columns: string[]): Promise<string[]> => {
    try {
      // Fetch column types from backend
      const response = await fetch(API_ENDPOINTS.columnTypes)
      if (response.ok) {
        const data = await response.json()
        const numericCols = data.numeric_columns || []
        const categoricalCols = data.categorical_columns || []

        const examples: string[] = []

        // Generate questions based on numeric columns
        if (numericCols.length > 0) {
          const firstNumeric = numericCols[0]
          const colName = firstNumeric.toLowerCase()

          // Check for common numeric column patterns
          if (colName.includes('salary') || colName.includes('price') || colName.includes('amount') || colName.includes('cost')) {
            examples.push(`What is the average ${firstNumeric}?`, `Show me the top 5 highest ${firstNumeric}`)
          } else if (colName.includes('age')) {
            examples.push(`What is the average ${firstNumeric}?`, `What are the statistics for ${firstNumeric}?`)
          } else {
            examples.push(`What is the average ${firstNumeric}?`, `Show me the top 5 highest ${firstNumeric}`)
          }
        }

        // Generate questions based on categorical columns
        if (categoricalCols.length > 0) {
          const firstCategorical = categoricalCols[0]
          const colName = firstCategorical.toLowerCase()

          if (colName.includes('city') || colName.includes('location') || colName.includes('place')) {
            examples.push(`How many people are in each ${firstCategorical}?`)
          } else {
            examples.push(`How many records are in each ${firstCategorical}?`)
          }
        }

        // Add a statistics question if we have numeric columns and need more examples
        if (numericCols.length > 0 && examples.length < 4) {
          const firstNumeric = numericCols[0]
          examples.push(`What are the statistics for ${firstNumeric}?`)
        }

        // Add a general overview question if we need more examples
        if (examples.length < 4) {
          examples.push(`Give me an overview of the data`)
        }

        return examples.slice(0, 4) // Return max 4 examples
      }
    } catch (error) {
      console.error("Error generating example questions:", error)
    }

    // Fallback to generic examples
    return [
      "What is the average value?",
      "Show me the top 5 records",
      "How many records are in each category?",
      "Give me an overview of the data"
    ]
  }

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  useEffect(() => {
    // Check if CSV is already loaded
    checkStatus()
  }, [])

  const checkStatus = async () => {
    try {
      const response = await fetch(API_ENDPOINTS.status)
      const data = await response.json()
      if (data.loaded) {
        setCsvLoaded(true)
        onCsvLoadedChange?.(true)

        if (data.file1) {
          setCsvInfo({
            rows: data.file1.rows,
            columns: data.file1.columns,
            column_names: data.file1.column_names,
          })
        }

        if (data.file2) {
          setCsvInfo2({
            rows: data.file2.rows,
            columns: data.file2.columns,
            column_names: data.file2.column_names,
          })
        }

        let statusMessage = "✅ CSV file(s) loaded successfully!\n\n"
        if (data.file1) {
          statusMessage += `📊 File 1 Overview:\n• Rows: ${data.file1.rows}\n• Columns: ${data.file1.columns}\n• Column names: ${data.file1.column_names.join(", ")}\n\n`
        }
        if (data.file2) {
          statusMessage += `📊 File 2 Overview:\n• Rows: ${data.file2.rows}\n• Columns: ${data.file2.columns}\n• Column names: ${data.file2.column_names.join(", ")}\n\n`
        }
        if (data.file1 && data.file2) {
          statusMessage += `🔗 Both files loaded! You can now analyze correlations between columns from both files.\n\n`
        }

        setMessages([{
          id: generateMessageId(),
          role: "assistant",
          content: statusMessage
        }])
      }
    } catch (error) {
      console.error("Error checking status:", error)
    }
  }

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>, fileIndex: number = 1) => {
    const file = event.target.files?.[0]
    if (!file) return

    // Validate file type
    if (!file.name.toLowerCase().endsWith('.csv')) {
      setMessages([{
        id: generateMessageId(),
        role: "assistant",
        content: "Error: Please upload a CSV file (.csv extension required)"
      }])
      // Reset file input
      event.target.value = ''
      return
    }

    // Validate file size (50MB max)
    const MAX_FILE_SIZE = 50 * 1024 * 1024 // 50MB
    if (file.size > MAX_FILE_SIZE) {
      setMessages([{
        id: generateMessageId(),
        role: "assistant",
        content: `Error: File is too large. Maximum size is ${MAX_FILE_SIZE / (1024 * 1024)}MB`
      }])
      event.target.value = ''
      return
    }

    if (file.size === 0) {
      setMessages([{
        id: generateMessageId(),
        role: "assistant",
        content: "Error: File is empty"
      }])
      event.target.value = ''
      return
    }

    setUploading(true)
    const formData = new FormData()
    formData.append("file", file)

    try {
      const response = await fetch(`${API_ENDPOINTS.upload}?file_index=${fileIndex}`, {
        method: "POST",
        body: formData,
        // Don't set Content-Type header - browser will set it automatically with boundary for FormData
      })

      if (!response.ok) {
        let errorMessage = "Upload failed"
        try {
          const error = await response.json()
          errorMessage = error.detail || error.message || errorMessage
        } catch {
          errorMessage = `Server returned status ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMessage)
      }

      const data = await response.json()
      setCsvLoaded(true)
      onCsvLoadedChange?.(true)

      if (fileIndex === 1) {
        setFileName(file.name)
        setCsvInfo({
          rows: data.rows,
          columns: data.columns,
          column_names: data.column_names,
        })
      } else {
        setFileName2(file.name)
        setCsvInfo2({
          rows: data.rows,
          columns: data.columns,
          column_names: data.column_names,
        })
      }

      // Check status to see if both files are loaded
      const statusResponse = await fetch(API_ENDPOINTS.status)
      const statusData = await statusResponse.json()

      let successMessage = `✅ CSV file ${fileIndex} "${file.name}" uploaded successfully!\n\n📊 File ${fileIndex} Overview:\n• Rows: ${data.rows}\n• Columns: ${data.columns}\n• Column names: ${data.column_names.join(", ")}\n\n`

      if (statusData.file1 && statusData.file2) {
        successMessage += `🔗 Both files are now loaded! You can analyze correlations between columns from both files.\n\n`
      }

      setMessages([{
        id: generateMessageId(),
        role: "assistant",
        content: successMessage
      }])
    } catch (error: any) {
      console.error("Upload error:", error)
      let errorMessage = error.message || "Unknown error occurred"

      // Provide more helpful error messages
      if (error.message.includes("Failed to fetch") || error.message.includes("NetworkError") || error.name === "TypeError") {
        errorMessage = "Cannot connect to backend server. Please make sure the backend is running on port 8001. Start it with: cd backend && python main.py"
      }

      setMessages([{
        id: generateMessageId(),
        role: "assistant",
        content: `Error uploading file: ${errorMessage}`
      }])
    } finally {
      setUploading(false)
      // Reset file input to allow re-uploading the same file
      event.target.value = ''
    }
  }

  const handleSend = async () => {
    if (!input.trim() || loading) return

    if (!csvLoaded) {
      const userMsgId = generateMessageId()
      const assistantMsgId = generateMessageId()
      setMessages(prev => [...prev, {
        id: userMsgId,
        role: "user",
        content: input
      }, {
        id: assistantMsgId,
        role: "assistant",
        content: "Please upload a CSV file first before asking questions."
      }])
      setInput("")
      return
    }

    const userMessage = input.trim()
    setInput("")
    setMessages(prev => [...prev, { id: generateMessageId(), role: "user", content: userMessage }])
    setLoading(true)

    try {
      const response = await fetch(API_ENDPOINTS.query, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ question: userMessage }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.detail || "Query failed")
      }

      const data = await response.json()

      // Handle new response format
      let assistantMessage: Message = {
        id: generateMessageId(),
        role: "assistant",
        content: data.explanation || data.raw_response || "I've processed your query.",
      }

      if (data.result) {
        assistantMessage.content += `\n\n📊 Result:\n\`\`\`\n${data.result}\n\`\`\``
        if (data.result_data) {
          assistantMessage.resultData = data.result_data
          assistantMessage.resultType = data.result_type
        }
      }

      // Add column similarity information if available
      if (data.column_similarities && data.column_similarities.length > 0) {
        assistantMessage.content += `\n\n🔗 Column Similarities Found:\n`
        data.column_similarities.forEach((sim: any) => {
          assistantMessage.content += `• ${sim.file1_column} ↔ ${sim.file2_column}: ${(sim.similarity * 100).toFixed(1)}% similarity (${sim.confidence.toFixed(1)}% confidence, ${sim.type})\n`
        })
      }

      if (data.error) {
        assistantMessage.error = data.error
        assistantMessage.content += `\n\n⚠️ Error: ${data.error}`
      }

      setMessages(prev => [...prev, assistantMessage])
    } catch (error: any) {
      setMessages(prev => [...prev, {
        id: generateMessageId(),
        role: "assistant",
        content: `Error: ${error.message}`
      }])
    } finally {
      setLoading(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleRemoveFile = (fileIndex: number = 1) => {
    if (fileIndex === 1) {
      setCsvInfo(null)
      setFileName(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    } else {
      setCsvInfo2(null)
      setFileName2(null)
      if (fileInputRef2.current) {
        fileInputRef2.current.value = ''
      }
    }

    // Check if both files are removed
    if ((fileIndex === 1 && !csvInfo2) || (fileIndex === 2 && !csvInfo)) {
      setCsvLoaded(false)
      onCsvLoadedChange?.(false)
      setMessages([])
    }
  }

  return (
    <div className="flex flex-col h-full p-4 bg-background overflow-hidden">
      <Card className="flex-1 flex flex-col border border-border shadow-none overflow-hidden">
        <CardHeader className="border-b border-border bg-card pb-3">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base font-medium text-foreground">Data Analysis</CardTitle>
              <CardDescription className="mt-0.5 text-xs text-muted-foreground">
                Upload a CSV file and ask questions about your data using natural language
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex-1 flex flex-col overflow-hidden p-6 min-h-0">
          {/* Data Source Toggle */}
          <div className="flex gap-2 mb-4 pb-2 border-b border-border">
            <Button
              variant={dataSource === 'csv' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setDataSource('csv')}
              className={dataSource === 'csv' ? "bg-primary text-primary-foreground" : "text-muted-foreground"}
            >
              <Upload className="h-4 w-4 mr-2" />
              CSV Upload
            </Button>
            <Button
              variant={dataSource === 'db' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setDataSource('db')}
              className={dataSource === 'db' ? "bg-primary text-primary-foreground" : "text-muted-foreground"}
            >
              <Database className="h-4 w-4 mr-2" />
              Database
            </Button>
          </div>

          {/* Data Source UI */}
          <div className="mb-4 pb-4 border-b border-border flex-shrink-0">
            {dataSource === 'db' ? (
              <div className="space-y-4">
                {!connected ? (
                  <div className="space-y-3">
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-1">
                        <label className="text-xs font-medium">Host</label>
                        <Input value={dbConfig.host} onChange={e => setDbConfig({ ...dbConfig, host: e.target.value })} className="h-8" />
                      </div>
                      <div className="space-y-1">
                        <label className="text-xs font-medium">Port</label>
                        <Input value={dbConfig.port} onChange={e => setDbConfig({ ...dbConfig, port: parseInt(e.target.value) })} className="h-8" />
                      </div>
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                      <div className="space-y-1">
                        <label className="text-xs font-medium">User</label>
                        <Input value={dbConfig.user} onChange={e => setDbConfig({ ...dbConfig, user: e.target.value })} className="h-8" />
                      </div>
                      <div className="space-y-1">
                        <label className="text-xs font-medium">DB Name</label>
                        <Input value={dbConfig.dbname} onChange={e => setDbConfig({ ...dbConfig, dbname: e.target.value })} className="h-8" />
                      </div>
                    </div>
                    <div className="space-y-1">
                      <label className="text-xs font-medium">Password</label>
                      <Input type="password" value={dbConfig.password} onChange={e => setDbConfig({ ...dbConfig, password: e.target.value })} className="h-8" />
                    </div>
                    <Button onClick={handleConnect} disabled={connecting} className="w-full h-8 bg-info hover:bg-info/90 text-info-foreground">
                      {connecting ? <Loader2 className="h-3 w-3 animate-spin mr-2" /> : <Server className="h-3 w-3 mr-2" />}
                      Connect
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-sm font-medium text-success">Connected to {dbConfig.dbname}</span>
                      <Button variant="ghost" size="sm" onClick={() => setConnected(false)} className="h-6 text-xs text-error">Disconnect</Button>
                    </div>
                    <div className="text-xs text-muted-foreground mb-2">Select a table to analyze:</div>
                    <div className="max-h-40 overflow-y-auto space-y-1 border border-border rounded p-2">
                      {dbTables.map(table => (
                        <div key={table} className="flex items-center justify-between p-1 hover:bg-accent rounded">
                          <span className="text-sm font-mono truncate max-w-[120px]" title={table}>{table}</span>
                          <div className="flex gap-1">
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-6 text-[10px] px-2"
                              onClick={() => handleSelectTable(table, 1)}
                            >
                              File 1
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              className="h-6 text-[10px] px-2"
                              onClick={() => handleSelectTable(table, 2)}
                            >
                              File 2
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Show Selected Status even in DB mode */}
                {(csvInfo || csvInfo2) && (
                  <div className="mt-3 pt-3 border-t border-border">
                    {csvInfo && (
                      <div className="flex items-center gap-2 text-sm text-file1 mb-1">
                        <CheckCircle2 className="h-3 w-3" />
                        <span className="truncate">File 1: {fileName} ({csvInfo.rows} rows)</span>
                      </div>
                    )}
                    {csvInfo2 && (
                      <div className="flex items-center gap-2 text-sm text-file2">
                        <CheckCircle2 className="h-3 w-3" />
                        <span className="truncate">File 2: {fileName2} ({csvInfo2.rows} rows)</span>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ) : (
              // CSV Mode (Existing)
              <div className="space-y-3">
                {/* File 1 Upload */}
                <div className="flex items-center gap-3 flex-wrap">
                  <label htmlFor="file-upload-1" className="cursor-pointer">
                    <Button variant="outline" asChild disabled={uploading} className="gap-2 border-border text-foreground hover:bg-accent">
                      <span>
                        {uploading ? (
                          <>
                            <Loader2 className="h-4 w-4 animate-spin" />
                            Uploading...
                          </>
                        ) : (
                          <>
                            <Upload className="h-4 w-4" />
                            {csvInfo ? "Replace CSV 1" : "Upload CSV 1"}
                          </>
                        )}
                      </span>
                    </Button>
                  </label>
                  <input
                    ref={fileInputRef}
                    id="file-upload-1"
                    type="file"
                    accept=".csv"
                    onChange={(e) => handleFileUpload(e, 1)}
                    className="hidden"
                  />
                  {csvInfo && (
                    <div className="flex items-center gap-2 px-3 py-1.5 bg-file1-light rounded-md border border-file1-border">
                      <CheckCircle2 className="h-4 w-4 text-file1" />
                      <div className="flex items-center gap-3">
                        <div className="text-sm">
                          <span className="font-medium text-file1">{fileName}</span>
                          <span className="text-muted-foreground ml-2">
                            {csvInfo.rows.toLocaleString()} rows × {csvInfo.columns} columns
                          </span>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveFile(1)}
                          className="h-6 w-6 p-0 hover:bg-file1-light text-file1"
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  )}
                </div>

                {/* File 2 Upload */}
                <div className="flex items-center gap-3 flex-wrap">
                  <label htmlFor="file-upload-2" className="cursor-pointer">
                    <Button variant="outline" asChild disabled={uploading} className="gap-2 border-border text-foreground hover:bg-accent">
                      <span>
                        {uploading ? (
                          <>
                            <Loader2 className="h-4 w-4 animate-spin" />
                            Uploading...
                          </>
                        ) : (
                          <>
                            <Upload className="h-4 w-4" />
                            {csvInfo2 ? "Replace CSV 2" : "Upload CSV 2"}
                          </>
                        )}
                      </span>
                    </Button>
                  </label>
                  <input
                    ref={fileInputRef2}
                    id="file-upload-2"
                    type="file"
                    accept=".csv"
                    onChange={(e) => handleFileUpload(e, 2)}
                    className="hidden"
                  />
                  {csvInfo2 && (
                    <div className="flex items-center gap-2 px-3 py-1.5 bg-file2-light rounded-md border border-file2-border">
                      <CheckCircle2 className="h-4 w-4 text-file2" />
                      <div className="flex items-center gap-3">
                        <div className="text-sm">
                          <span className="font-medium text-file2">{fileName2}</span>
                          <span className="text-muted-foreground ml-2">
                            {csvInfo2.rows.toLocaleString()} rows × {csvInfo2.columns} columns
                          </span>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveFile(2)}
                          className="h-6 w-6 p-0 hover:bg-file2-light text-file2"
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
                {csvInfo && (
                  <div className="mt-3 text-xs text-muted-foreground">
                    <span className="font-medium">File 1 Columns:</span> {csvInfo.column_names.join(", ")}
                  </div>
                )}
                {csvInfo2 && (
                  <div className="mt-2 text-xs text-muted-foreground">
                    <span className="font-medium">File 2 Columns:</span> {csvInfo2.column_names.join(", ")}
                  </div>
                )}
                {csvInfo && csvInfo2 && (
                  <div className="mt-2 text-xs text-info font-medium">
                    🔗 Both files loaded! You can analyze correlations between columns from both files.
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Messages Area */}
          <div className="flex-1 overflow-y-auto mb-4 space-y-4 pr-2 min-h-0">
            {messages.length === 0 ? (
              <div className="text-center text-muted-foreground py-12">
                <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p className="text-lg font-medium mb-2">
                  {csvLoaded
                    ? "Ready to analyze your data! 🚀"
                    : "Welcome! 👋"}
                </p>
                <p className="text-sm">
                  {csvLoaded
                    ? "Ask a question about your CSV data to get started. Try asking about averages, counts, filters, or summaries."
                    : "Upload a CSV file to start asking questions about your data using natural language."}
                </p>
              </div>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id || `msg-${message.role}-${message.content.slice(0, 20)}`}
                  className={`flex ${message.role === "user" ? "justify-end" : "justify-start"
                    }`}
                >
                  <div
                    className={`max-w-[85%] rounded-lg px-4 py-3 ${message.role === "user"
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted border border-border text-foreground"
                      }`}
                  >
                    <p className="text-sm whitespace-pre-wrap leading-relaxed">{message.content}</p>
                    {message.error && (
                      <div className="mt-2 p-2 bg-muted rounded text-xs text-foreground border border-border">
                        {message.error}
                      </div>
                    )}
                    {message.resultData && message.resultType === "dataframe" && (
                      <div className="mt-3 p-2 bg-card rounded text-xs border border-border">
                        <div className="font-medium mb-1 text-foreground">Data Preview:</div>
                        <pre className="overflow-x-auto text-xs text-foreground">
                          {JSON.stringify(message.resultData.slice(0, 5), null, 2)}
                          {message.resultData.length > 5 && `\n... (${message.resultData.length - 5} more rows)`}
                        </pre>
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
            {loading && (
              <div className="flex justify-start">
                <div className="bg-muted rounded-lg px-4 py-3 border border-border">
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin text-foreground" />
                    <span className="text-sm text-foreground">Analyzing your data...</span>
                  </div>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          {/* Input Area */}
          <div className="flex gap-2 pt-2 border-t border-border flex-shrink-0">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder={
                csvLoaded
                  ? "Ask a question about your data..."
                  : "Upload a CSV file first..."
              }
              disabled={loading || !csvLoaded}
              className="flex-1 border-border text-foreground placeholder:text-muted-foreground focus:border-ring"
            />
            <Button
              onClick={handleSend}
              disabled={loading || !csvLoaded || !input.trim()}
              size="default"
              className="bg-primary text-primary-foreground hover:bg-primary/90 border border-primary"
            >
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

