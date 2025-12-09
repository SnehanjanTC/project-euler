"use client"

import { useState, useEffect } from 'react'
import { API_BASE_URL } from '@/lib/api-config'

interface AIProvider {
    id: string
    name: string
    placeholder: string
    docLink: string
}

interface OllamaConfig {
    baseUrl: string
    model: string
}

const AI_PROVIDERS: AIProvider[] = [
    {
        id: 'openai',
        name: 'OpenAI',
        placeholder: 'No API key set',
        docLink: 'https://platform.openai.com/api-keys'
    },
    {
        id: 'anthropic',
        name: 'Anthropic',
        placeholder: 'No API key set',
        docLink: 'https://console.anthropic.com/settings/keys'
    },
    {
        id: 'google',
        name: 'Google Gemini',
        placeholder: 'No API key set',
        docLink: 'https://makersuite.google.com/app/apikey'
    }
]

interface AIProviderModalProps {
    isOpen: boolean
    onClose: () => void
}

export default function AIProviderModal({ isOpen, onClose }: AIProviderModalProps) {
    const [apiKeys, setApiKeys] = useState<Record<string, string>>({})
    const [ollamaConfig, setOllamaConfig] = useState<OllamaConfig>({
        baseUrl: 'http://localhost:11434',
        model: 'qwen3-vl:2b'
    })

    useEffect(() => {
        // Load API keys
        const saved = localStorage.getItem('ai_provider_keys')
        if (saved) {
            try {
                setApiKeys(JSON.parse(saved))
            } catch (e) {
                console.error('Failed to load API keys:', e)
            }
        }

        // Load Ollama config from localStorage first
        const savedOllama = localStorage.getItem('ollama_config')
        if (savedOllama) {
            try {
                setOllamaConfig(JSON.parse(savedOllama))
            } catch (e) {
                console.error('Failed to load Ollama config:', e)
            }
        }

        // Fetch current Ollama config from backend
        fetch(`${API_BASE_URL}/api/config/ollama`)
            .then(res => res.json())
            .then(data => {
                setOllamaConfig(data)
                localStorage.setItem('ollama_config', JSON.stringify(data))
            })
            .catch(err => console.error('Failed to fetch Ollama config:', err))
    }, [])

    const handleAddKey = (providerId: string) => {
        const updatedKeys = { ...apiKeys }
        localStorage.setItem('ai_provider_keys', JSON.stringify(updatedKeys))

        fetch(`${API_BASE_URL}/api/config/ai-providers`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                provider: providerId,
                api_key: apiKeys[providerId]
            })
        }).catch(err => console.error('Failed to save to backend:', err))
    }

    const handleSaveOllama = () => {
        localStorage.setItem('ollama_config', JSON.stringify(ollamaConfig))

        fetch(`${API_BASE_URL}/api/config/ollama`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(ollamaConfig)
        }).catch(err => console.error('Failed to save Ollama config:', err))
    }

    if (!isOpen) return null

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
            <div className="bg-card rounded-xl shadow-2xl w-full max-w-2xl" onClick={(e) => e.stopPropagation()}>
                {/* Header */}
                <div className="flex items-start justify-between px-8 py-6 border-b border-border">
                    <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                            <h3 className="text-xl font-semibold text-foreground">API Keys</h3>
                            <span className="px-2 py-0.5 bg-success-light text-success text-xs font-medium rounded">BYOK</span>
                        </div>
                        <p className="text-sm text-muted-foreground">
                            By default, your API Key is stored locally on your browser and never sent anywhere else.
                        </p>
                    </div>

                    <button
                        onClick={onClose}
                        className="text-muted-foreground hover:text-foreground text-2xl leading-none ml-4"
                    >
                        ×
                    </button>
                </div>

                {/* Content */}
                <div className="px-8 py-8">
                    <div className="space-y-8">
                        {AI_PROVIDERS.map((provider) => (
                            <div key={provider.id}>
                                <div className="flex items-center justify-between mb-3">
                                    <label className="text-sm font-medium text-foreground">
                                        {provider.name} API Key:
                                    </label>
                                    <a
                                        href={provider.docLink}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className="text-sm text-info hover:underline"
                                    >
                                        (Get API key here)
                                    </a>
                                </div>

                                <div className="flex gap-3">
                                    <input
                                        type="password"
                                        value={apiKeys[provider.id] || ''}
                                        onChange={(e) =>
                                            setApiKeys({ ...apiKeys, [provider.id]: e.target.value })
                                        }
                                        placeholder={provider.placeholder}
                                        className="flex-1 px-4 py-3 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent bg-background text-foreground"
                                    />

                                    <button
                                        onClick={() => handleAddKey(provider.id)}
                                        disabled={!apiKeys[provider.id]}
                                        className={`px-6 py-3 rounded-lg text-sm font-medium transition-colors whitespace-nowrap ${apiKeys[provider.id]
                                            ? 'bg-card border border-border text-foreground hover:bg-accent'
                                            : 'bg-muted text-muted-foreground cursor-not-allowed'
                                            }`}
                                    >
                                        {apiKeys[provider.id] ? 'Change Key' : 'Add Key'}
                                    </button>
                                </div>
                            </div>
                        ))}

                        {/* Ollama Local Configuration */}
                        <div className="mt-8 pt-8 border-t border-border">
                            <div className="flex items-center gap-2 mb-6">
                                <h4 className="text-lg font-semibold text-foreground">Ollama Local</h4>
                                <span className="px-2 py-0.5 bg-accent text-accent-foreground text-xs font-medium rounded">LOCAL</span>
                            </div>

                            <div className="space-y-6">
                                {/* Base URL */}
                                <div>
                                    <div className="flex items-center justify-between mb-3">
                                        <label className="text-sm font-medium text-foreground">
                                            Ollama Base URL:
                                        </label>
                                        <a
                                            href="https://ollama.ai/download"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-sm text-info hover:underline"
                                        >
                                            (Download Ollama)
                                        </a>
                                    </div>

                                    <input
                                        type="text"
                                        value={ollamaConfig.baseUrl}
                                        onChange={(e) =>
                                            setOllamaConfig({ ...ollamaConfig, baseUrl: e.target.value })
                                        }
                                        placeholder="http://localhost:11434"
                                        className="w-full px-4 py-3 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent bg-background text-foreground"
                                    />
                                </div>

                                {/* Model Name */}
                                <div>
                                    <div className="flex items-center justify-between mb-3">
                                        <label className="text-sm font-medium text-foreground">
                                            Model Name:
                                        </label>
                                        <a
                                            href="https://ollama.ai/library"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="text-sm text-info hover:underline"
                                        >
                                            (Browse models)
                                        </a>
                                    </div>

                                    <input
                                        type="text"
                                        value={ollamaConfig.model}
                                        onChange={(e) =>
                                            setOllamaConfig({ ...ollamaConfig, model: e.target.value })
                                        }
                                        placeholder="e.g., llama3, qwen3-vl:2b, mistral"
                                        className="w-full px-4 py-3 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent bg-background text-foreground"
                                    />
                                </div>

                                {/* Save Button */}
                                <div className="flex justify-end">
                                    <button
                                        onClick={handleSaveOllama}
                                        className="px-6 py-3 bg-primary text-primary-foreground rounded-lg text-sm font-medium hover:bg-primary/90 transition-colors"
                                    >
                                        Save Ollama Config
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}
