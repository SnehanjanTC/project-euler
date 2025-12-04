"use client"

import { useState, useEffect } from "react"
import { API_ENDPOINTS } from "@/lib/api-config"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Filter, X, Plus } from "lucide-react"

interface FilterCondition {
  id: string
  column: string
  operator: "equals" | "contains" | "greater_than" | "less_than" | "between" | "in"
  value: string | number
  value2?: string | number
}

interface FilterPanelProps {
  csvLoaded: boolean
  columns?: string[]
  onFilterChange: (filters: FilterCondition[]) => void
}

export default function FilterPanel({ csvLoaded, columns = [], onFilterChange }: FilterPanelProps) {
  const [filters, setFilters] = useState<FilterCondition[]>([])
  const [columnTypes, setColumnTypes] = useState<Record<string, "numeric" | "categorical">>({})

  useEffect(() => {
    if (csvLoaded && columns.length > 0) {
      fetchColumnTypes()
    }
  }, [csvLoaded, columns])

  const fetchColumnTypes = async () => {
    try {
      const response = await fetch(API_ENDPOINTS.columnTypes)
      if (response.ok) {
        const data = await response.json()
        setColumnTypes(data.column_types || {})
      }
    } catch (error) {
      console.error("Error fetching column types:", error)
    }
  }

  const addFilter = () => {
    if (columns.length === 0) return
    
    const newFilter: FilterCondition = {
      id: `filter-${Date.now()}`,
      column: columns[0],
      operator: columnTypes[columns[0]] === "numeric" ? "greater_than" : "equals",
      value: ""
    }
    setFilters([...filters, newFilter])
  }

  const removeFilter = (id: string) => {
    const newFilters = filters.filter(f => f.id !== id)
    setFilters(newFilters)
    onFilterChange(newFilters)
  }

  const updateFilter = (id: string, updates: Partial<FilterCondition>) => {
    const newFilters = filters.map(f => 
      f.id === id ? { ...f, ...updates } : f
    )
    setFilters(newFilters)
    onFilterChange(newFilters)
  }

  const applyFilters = () => {
    onFilterChange(filters)
  }

  const clearAllFilters = () => {
    setFilters([])
    onFilterChange([])
  }

  if (!csvLoaded) {
    return (
      <Card className="border border-gray-300">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm flex items-center gap-2 text-black font-medium">
            <Filter className="h-3.5 w-3.5" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-xs text-gray-500">Upload CSV to enable filtering</p>
        </CardContent>
      </Card>
    )
  }

  const getOperatorsForType = (type: "numeric" | "categorical" | undefined) => {
    if (type === "numeric") {
      return [
        { value: "equals", label: "Equals" },
        { value: "greater_than", label: "Greater Than" },
        { value: "less_than", label: "Less Than" },
        { value: "between", label: "Between" }
      ]
    }
    return [
      { value: "equals", label: "Equals" },
      { value: "contains", label: "Contains" },
      { value: "in", label: "In List" }
    ]
  }

  return (
    <Card className="border border-gray-300">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm flex items-center gap-2 text-black font-medium">
            <Filter className="h-3.5 w-3.5" />
            Filters
          </CardTitle>
          {filters.length > 0 && (
            <Button variant="ghost" size="sm" onClick={clearAllFilters} className="h-6 text-xs text-gray-600 hover:bg-gray-50">
              Clear All
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {filters.length === 0 ? (
          <div className="text-center py-4">
            <p className="text-xs text-gray-500 mb-2">No filters applied</p>
            <Button variant="outline" size="sm" onClick={addFilter} className="w-full border-gray-300 text-black hover:bg-gray-50 text-xs">
              <Plus className="h-3 w-3 mr-1" />
              Add Filter
            </Button>
          </div>
        ) : (
          <>
            {filters.map((filter) => {
              const colType = columnTypes[filter.column] || "categorical"
              const operators = getOperatorsForType(colType)
              
              return (
                <div key={filter.id} className="border border-gray-300 rounded-lg p-3 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-medium text-gray-600">Filter {filters.indexOf(filter) + 1}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeFilter(filter.id)}
                      className="h-5 w-5 p-0 text-gray-600 hover:bg-gray-50"
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                  
                  <div className="grid grid-cols-3 gap-2">
                    <Select
                      value={filter.column}
                      onValueChange={(value) => updateFilter(filter.id, { 
                        column: value,
                        operator: columnTypes[value] === "numeric" ? "greater_than" : "equals"
                      })}
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {columns.map((col) => (
                          <SelectItem key={col} value={col}>
                            {col}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    
                    <Select
                      value={filter.operator}
                      onValueChange={(value) => updateFilter(filter.id, { operator: value as FilterCondition["operator"] })}
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {operators.map((op) => (
                          <SelectItem key={op.value} value={op.value}>
                            {op.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    
                    {filter.operator === "between" ? (
                      <div className="col-span-3 grid grid-cols-2 gap-2">
                        <Input
                          type={colType === "numeric" ? "number" : "text"}
                          placeholder="Min"
                          value={filter.value}
                          onChange={(e) => updateFilter(filter.id, { 
                            value: colType === "numeric" ? parseFloat(e.target.value) || 0 : e.target.value 
                          })}
                          className="h-8 text-xs"
                        />
                        <Input
                          type={colType === "numeric" ? "number" : "text"}
                          placeholder="Max"
                          value={filter.value2 || ""}
                          onChange={(e) => updateFilter(filter.id, { 
                            value2: colType === "numeric" ? parseFloat(e.target.value) || 0 : e.target.value 
                          })}
                          className="h-8 text-xs"
                        />
                      </div>
                    ) : (
                      <Input
                        type={colType === "numeric" && filter.operator !== "contains" ? "number" : "text"}
                        placeholder="Value"
                        value={filter.value}
                        onChange={(e) => updateFilter(filter.id, { 
                          value: colType === "numeric" && filter.operator !== "contains" 
                            ? parseFloat(e.target.value) || 0 
                            : e.target.value 
                        })}
                        className="h-8 text-xs"
                      />
                    )}
                  </div>
                </div>
              )
            })}
            
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={addFilter} className="flex-1 text-xs border-gray-300 text-black hover:bg-gray-50">
                <Plus className="h-3 w-3 mr-1" />
                Add Filter
              </Button>
              <Button size="sm" onClick={applyFilters} className="flex-1 text-xs bg-black text-white hover:bg-gray-800 border border-black">
                Apply Filters
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

