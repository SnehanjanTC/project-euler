"use client"

import React, { useCallback, useEffect, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  MarkerType,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';

interface ColumnSimilarity {
  file1_column: string;
  file2_column: string;
  similarity: number;
  confidence: number;
  type: string;
  data_similarity: number;
  name_similarity: number;
  distribution_similarity: number;
  json_confidence: number;
  llm_semantic_score: number;
}

interface CorrelationFlowProps {
  similarities: ColumnSimilarity[];
}

export default function CorrelationFlow({ similarities }: CorrelationFlowProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedEdge, setSelectedEdge] = useState<ColumnSimilarity | null>(null);

  useEffect(() => {
    if (!similarities || similarities.length === 0) return;

    // Create nodes for File 1 columns (left side)
    const file1Columns = Array.from(new Set(similarities.map(s => s.file1_column)));
    const file2Columns = Array.from(new Set(similarities.map(s => s.file2_column)));

    const file1Nodes: Node[] = file1Columns.map((col, idx) => ({
      id: `file1_${col}`,
      data: {
        label: (
          <div className="px-3 py-2">
            <div className="font-semibold text-sm">{col}</div>
            <div className="text-xs text-muted-foreground">File 1</div>
          </div>
        )
      },
      position: { x: 50, y: idx * 100 + 50 },
      sourcePosition: Position.Right,
      style: {
        background: 'hsl(var(--card))',
        color: 'hsl(var(--foreground))',
        border: '1.5px solid hsl(var(--file2))',
        borderRadius: '6px',
        padding: '0',
        width: 'auto',
        boxShadow: '0 1px 3px hsl(var(--foreground) / 0.1)',
      },
    }));

    const file2Nodes: Node[] = file2Columns.map((col, idx) => ({
      id: `file2_${col}`,
      data: {
        label: (
          <div className="px-3 py-2">
            <div className="font-semibold text-sm">{col}</div>
            <div className="text-xs text-muted-foreground">File 2</div>
          </div>
        )
      },
      position: { x: 500, y: idx * 100 + 50 },
      targetPosition: Position.Left,
      style: {
        background: 'hsl(var(--card))',
        color: 'hsl(var(--foreground))',
        border: '1.5px solid hsl(var(--file1))',
        borderRadius: '6px',
        padding: '0',
        width: 'auto',
        boxShadow: '0 1px 3px hsl(var(--foreground) / 0.1)',
      },
    }));

    // Create edges with similarity data
    const newEdges: Edge[] = similarities.map((sim, idx) => {
      const confidence = sim.confidence;
      const strokeWidth = 1.5; // Thin edges
      const opacity = Math.max(0.6, confidence / 100);

      return {
        id: `edge_${idx}`,
        source: `file1_${sim.file1_column}`,
        target: `file2_${sim.file2_column}`,
        label: `${confidence.toFixed(0)}%`,
        labelStyle: {
          fill: 'hsl(var(--muted-foreground))',
          fontWeight: 600,
          fontSize: '12px',
        },
        labelBgStyle: {
          fill: 'hsl(var(--card))',
          fillOpacity: 0.9,
        },
        style: {
          strokeWidth,
          opacity,
          stroke: confidence > 70 
            ? 'hsl(var(--confidence-high))' 
            : confidence > 40 
            ? 'hsl(var(--confidence-medium))' 
            : 'hsl(var(--confidence-low))',
          strokeDasharray: '5, 5', // Dotted line
        },
        animated: true, // Animated edges
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: confidence > 70 
            ? 'hsl(var(--confidence-high))' 
            : confidence > 40 
            ? 'hsl(var(--confidence-medium))' 
            : 'hsl(var(--confidence-low))',
        },
        data: sim,
      };
    });

    setNodes([...file1Nodes, ...file2Nodes]);
    setEdges(newEdges);
  }, [similarities, setNodes, setEdges]);

  const onEdgeClick = useCallback((event: React.MouseEvent, edge: Edge) => {
    setSelectedEdge(edge.data as ColumnSimilarity);
  }, []);

  return (
    <div className="h-full flex flex-col bg-muted" style={{ minHeight: '500px' }}>
      <div className="flex-1 relative" style={{ height: '400px' }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onEdgeClick={onEdgeClick}
          fitView
          attributionPosition="bottom-left"
        >
          <Background />
          <Controls />
        </ReactFlow>
      </div>

      {/* Similarity Details Panel */}
      {selectedEdge && (
        <div className="border-t border-border bg-card p-4 max-h-64 overflow-y-auto">
          <div className="flex justify-between items-start mb-3">
            <h3 className="font-semibold text-lg">
              {selectedEdge.file1_column} ↔ {selectedEdge.file2_column}
            </h3>
            <button
              onClick={() => setSelectedEdge(null)}
              className="text-muted-foreground hover:text-foreground"
            >
              ✕
            </button>
          </div>

          <div className="space-y-2">
            <div className="flex justify-between">
              <span className="text-sm font-medium">Overall Confidence:</span>
              <span className="text-sm font-bold text-info">
                {selectedEdge.confidence.toFixed(1)}%
              </span>
            </div>

            <div className="border-t border-border pt-2 space-y-1">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Data Similarity:</span>
                <span>{(selectedEdge.data_similarity * 100).toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Name Similarity:</span>
                <span>{(selectedEdge.name_similarity * 100).toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Distribution Similarity:</span>
                <span>{(selectedEdge.distribution_similarity * 100).toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">LLM Semantic Score:</span>
                <span>{(selectedEdge.llm_semantic_score * 100).toFixed(1)}%</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">JSON Structure:</span>
                <span>{selectedEdge.json_confidence.toFixed(1)}%</span>
              </div>
            </div>

            <div className="border-t border-border pt-2">
              <p className="text-sm text-muted-foreground">
                <span className="font-medium">Type:</span> {selectedEdge.type}
              </p>
            </div>

            <div className="border-t border-border pt-2">
              <p className="text-sm font-medium mb-1">AI Explanation:</p>
              <p className="text-sm text-foreground bg-info-light p-2 rounded">
                {generateExplanation(selectedEdge)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Legend */}
      <div className="border-t border-border bg-card p-3">
        <div className="flex items-center gap-4 text-xs">
          <span className="font-medium">Confidence:</span>
          <div className="flex items-center gap-1">
            <div className="w-4 h-2 bg-confidence-high rounded"></div>
            <span>&gt;70% High</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-4 h-2 bg-confidence-medium rounded"></div>
            <span>40-70% Medium</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-4 h-2 bg-confidence-low rounded"></div>
            <span>&lt;40% Low</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function generateExplanation(sim: ColumnSimilarity): string {
  const { file1_column, file2_column, confidence, type, llm_semantic_score, name_similarity, data_similarity } = sim;

  if (confidence > 80) {
    return `Strong match detected between "${file1_column}" and "${file2_column}". ${llm_semantic_score > 0.7
      ? `The AI model identified these as semantically equivalent fields. `
      : ''
      }${name_similarity > 0.8
        ? `Column names are very similar. `
        : ''
      }${data_similarity > 0.7
        ? `Data patterns show high ${type === 'correlation' ? 'correlation' : 'overlap'}. `
        : ''
      }This is a highly confident mapping.`;
  } else if (confidence > 50) {
    return `Moderate match between "${file1_column}" and "${file2_column}". ${llm_semantic_score > 0.5
      ? `AI suggests these fields may represent similar concepts. `
      : ''
      }${name_similarity > 0.5
        ? `Column names share some similarities. `
        : ''
      }Consider validating this mapping manually.`;
  } else {
    return `Weak match between "${file1_column}" and "${file2_column}". ${name_similarity > data_similarity
      ? `Primarily based on name similarity. `
      : `Primarily based on data patterns. `
      }This mapping should be reviewed carefully before use.`;
  }
}
