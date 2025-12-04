"use client"

import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { API_ENDPOINTS } from "@/lib/api-config"

export default function MLStatsPanel() {
    const [stats, setStats] = useState<any>(null)
    const [loading, setLoading] = useState(false)

    const fetchStats = async () => {
        setLoading(true)
        try {
            const response = await fetch(`${API_ENDPOINTS.base}/ml/stats`)
            if (response.ok) {
                const data = await response.json()
                setStats(data)
            }
        } catch (error) {
            console.error('Error fetching ML stats:', error)
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        fetchStats()
        // Refresh every 30 seconds
        const interval = setInterval(fetchStats, 30000)
        return () => clearInterval(interval)
    }, [])

    if (!stats) return null

    return (
        <Card className="border border-gray-300">
            <CardHeader>
                <div className="flex items-center justify-between">
                    <CardTitle className="text-base font-medium text-black">ML Learning Stats</CardTitle>
                    <Button onClick={fetchStats} variant="outline" size="sm" disabled={loading}>
                        {loading ? 'Loading...' : 'Refresh'}
                    </Button>
                </div>
            </CardHeader>
            <CardContent>
                <div className="grid grid-cols-2 gap-4 text-sm">
                    {/* Feedback Stats */}
                    <div className="space-y-2">
                        <h4 className="font-medium text-gray-700">Feedback</h4>
                        <div className="space-y-1 text-xs">
                            <div className="flex justify-between">
                                <span className="text-gray-600">Total:</span>
                                <span className="font-medium">{stats.feedback_stats?.total_feedback || 0}</span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Accuracy:</span>
                                <span className="font-medium text-green-600">
                                    {(stats.feedback_stats?.accuracy || 0).toFixed(1)}%
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Adaptive Weights */}
                    <div className="space-y-2">
                        <h4 className="font-medium text-gray-700">Learned Weights</h4>
                        <div className="space-y-1 text-xs">
                            <div className="flex justify-between">
                                <span className="text-gray-600">Name:</span>
                                <span className="font-medium">
                                    {((stats.adaptive_weights?.current_weights?.name || 0.45) * 100).toFixed(0)}%
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Data:</span>
                                <span className="font-medium">
                                    {((stats.adaptive_weights?.current_weights?.data || 0.35) * 100).toFixed(0)}%
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Pattern:</span>
                                <span className="font-medium">
                                    {((stats.adaptive_weights?.current_weights?.pattern || 0.20) * 100).toFixed(0)}%
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Pattern Learning */}
                    <div className="space-y-2">
                        <h4 className="font-medium text-gray-700">Pattern Learning</h4>
                        <div className="space-y-1 text-xs">
                            <div className="flex justify-between">
                                <span className="text-gray-600">Positive:</span>
                                <span className="font-medium text-green-600">
                                    {stats.pattern_learning?.positive_patterns_count || 0}
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Negative:</span>
                                <span className="font-medium text-red-600">
                                    {stats.pattern_learning?.negative_patterns_count || 0}
                                </span>
                            </div>
                        </div>
                    </div>

                    {/* Calibration */}
                    <div className="space-y-2">
                        <h4 className="font-medium text-gray-700">Calibration</h4>
                        <div className="space-y-1 text-xs">
                            <div className="flex justify-between">
                                <span className="text-gray-600">Samples:</span>
                                <span className="font-medium">
                                    {stats.confidence_calibration?.total_samples || 0}
                                </span>
                            </div>
                            <div className="flex justify-between">
                                <span className="text-gray-600">Error:</span>
                                <span className="font-medium">
                                    {(stats.confidence_calibration?.mean_calibration_error || 0).toFixed(1)}%
                                </span>
                            </div>
                        </div>
                    </div>
                </div>

                {/* Learning Status */}
                {stats.adaptive_weights?.total_updates > 0 && (
                    <div className="mt-4 pt-4 border-t border-gray-200">
                        <div className="flex items-center gap-2 text-xs">
                            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></div>
                            <span className="text-gray-600">
                                ML Active • {stats.adaptive_weights.total_updates} weight updates •
                                {stats.adaptive_weights.loss_trend === 'improving' ? ' 📈 Improving' : ' ✓ Stable'}
                            </span>
                        </div>
                    </div>
                )}
            </CardContent>
        </Card>
    )
}
