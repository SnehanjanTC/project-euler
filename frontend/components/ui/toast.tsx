"use client"

import { useState, useEffect } from 'react'
import { CheckCircle, XCircle, AlertCircle, X } from 'lucide-react'

interface Toast {
    id: string
    type: 'success' | 'error' | 'warning' | 'info'
    message: string
}

let toastCounter = 0

export const useToast = () => {
    const [toasts, setToasts] = useState<Toast[]>([])

    const showToast = (type: Toast['type'], message: string) => {
        const id = `toast-${Date.now()}-${toastCounter++}`
        const newToast: Toast = { id, type, message }

        setToasts(prev => [...prev, newToast])

        // Auto-remove after 4 seconds
        setTimeout(() => {
            setToasts(prev => prev.filter(t => t.id !== id))
        }, 4000)
    }

    const removeToast = (id: string) => {
        setToasts(prev => prev.filter(t => t.id !== id))
    }

    return { toasts, showToast, removeToast }
}

export function ToastContainer({ toasts, onRemove }: { toasts: Toast[], onRemove: (id: string) => void }) {
    if (toasts.length === 0) return null

    return (
        <div className="fixed top-4 right-4 z-50 space-y-2">
            {toasts.map(toast => (
                <div
                    key={toast.id}
                    className={`flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg min-w-[300px] max-w-md animate-slide-in ${toast.type === 'success' ? 'bg-green-50 border border-green-200' :
                            toast.type === 'error' ? 'bg-red-50 border border-red-200' :
                                toast.type === 'warning' ? 'bg-yellow-50 border border-yellow-200' :
                                    'bg-blue-50 border border-blue-200'
                        }`}
                >
                    {toast.type === 'success' && <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0" />}
                    {toast.type === 'error' && <XCircle className="w-5 h-5 text-red-600 flex-shrink-0" />}
                    {toast.type === 'warning' && <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0" />}
                    {toast.type === 'info' && <AlertCircle className="w-5 h-5 text-blue-600 flex-shrink-0" />}

                    <p className={`text-sm flex-1 ${toast.type === 'success' ? 'text-green-800' :
                            toast.type === 'error' ? 'text-red-800' :
                                toast.type === 'warning' ? 'text-yellow-800' :
                                    'text-blue-800'
                        }`}>
                        {toast.message}
                    </p>

                    <button
                        onClick={() => onRemove(toast.id)}
                        className="flex-shrink-0 hover:opacity-70 transition-opacity"
                    >
                        <X className="w-4 h-4 text-gray-500" />
                    </button>
                </div>
            ))}
        </div>
    )
}
