"use client"

import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { X, ChevronRight, ChevronLeft, Check, Sparkles } from 'lucide-react'
import { API_ENDPOINTS } from '@/lib/api-config'

interface Question {
    id: string
    type: string
    text: string
    options: string[]
    required: boolean
    metadata?: {
        placeholder?: string
        input_type?: string
        hint?: string
        columns?: string[]
        file1_columns?: string[]
        file2_columns?: string[]
    }
}

interface ContextWizardProps {
    isOpen: boolean
    onClose: () => void
    onComplete: () => void
}

import { createPortal } from 'react-dom'

export default function ContextWizard({ isOpen, onClose, onComplete }: ContextWizardProps) {
    const [currentStep, setCurrentStep] = useState(0)
    const [loading, setLoading] = useState(true)
    const [questions, setQuestions] = useState<{
        file1_questions: Question[]
        file2_questions: Question[]
        relationship_questions: Question[]
    } | null>(null)
    const [answers, setAnswers] = useState<Record<string, any>>({})
    const [submitting, setSubmitting] = useState(false)

    const steps = [
        { title: 'File 1 Context', description: 'Tell us about your first dataset' },
        { title: 'File 2 Context', description: 'Tell us about your second dataset' },
        { title: 'Relationships', description: 'How are these datasets related?' },
        { title: 'Review', description: 'Review and confirm your answers' }
    ]

    useEffect(() => {
        if (isOpen) {
            fetchQuestions()
        }
    }, [isOpen])

    const fetchQuestions = async () => {
        setLoading(true)
        try {
            const response = await fetch(`${API_ENDPOINTS.base}/context/questions`, {
                method: 'POST'
            })

            if (!response.ok) throw new Error('Failed to fetch questions')

            const data = await response.json()
            setQuestions(data.questions)
        } catch (error) {
            console.error('Error fetching questions:', error)
            alert('Failed to load context questions. Please try again.')
            onClose()
        } finally {
            setLoading(false)
        }
    }

    const handleAnswer = (questionId: string, value: any) => {
        setAnswers(prev => ({ ...prev, [questionId]: value }))
    }

    const getCurrentQuestions = (): Question[] => {
        if (!questions) return []

        switch (currentStep) {
            case 0:
                return questions.file1_questions
            case 1:
                return questions.file2_questions
            case 2:
                return questions.relationship_questions
            default:
                return []
        }
    }

    const canProceed = (): boolean => {
        const currentQuestions = getCurrentQuestions()
        const requiredQuestions = currentQuestions.filter(q => q.required)

        return requiredQuestions.every(q => {
            const answer = answers[q.id]
            return answer !== undefined && answer !== null && answer !== ''
        })
    }

    const handleNext = () => {
        if (currentStep < steps.length - 1) {
            setCurrentStep(currentStep + 1)
        }
    }

    const handleBack = () => {
        if (currentStep > 0) {
            setCurrentStep(currentStep - 1)
        }
    }

    const handleSubmit = async () => {
        setSubmitting(true)

        try {
            // Parse answers into context structure for each file
            const file1Context = parseContextFromAnswers(1)
            const file2Context = parseContextFromAnswers(2)

            // Submit File 1 context
            await fetch(`${API_ENDPOINTS.base}/context/submit`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    file_index: 1,
                    context_data: file1Context
                })
            })

            // Submit File 2 context
            await fetch(`${API_ENDPOINTS.base}/context/submit`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    file_index: 2,
                    context_data: file2Context
                })
            })

            onComplete()
            onClose()
        } catch (error) {
            console.error('Error submitting context:', error)
            alert('Failed to save context. Please try again.')
        } finally {
            setSubmitting(false)
        }
    }

    const parseContextFromAnswers = (fileIndex: number): any => {
        const prefix = `f${fileIndex}_`
        const context: any = {
            dataset_purpose: answers[`${prefix}purpose`] || '',
            business_domain: answers[`${prefix}domain`] || '',
            key_entities: [],
            temporal_context: answers[`${prefix}temporal`] || null,
            column_descriptions: {},
            relationships: [],
            custom_mappings: {},
            exclusions: []
        }

        // Parse key entities (comma-separated)
        if (answers[`${prefix}entities`]) {
            context.key_entities = answers[`${prefix}entities`]
                .split(',')
                .map((e: string) => e.trim())
                .filter((e: string) => e.length > 0)
        }

        // Parse column descriptions
        if (answers[`${prefix}column_semantics`]) {
            context.column_descriptions = answers[`${prefix}column_semantics`]
        }

        // Parse exclusions
        if (answers[`${prefix}exclusions`]) {
            context.exclusions = answers[`${prefix}exclusions`]
        }

        // Add relationship info
        if (answers['relationship_type']) {
            context.relationships.push({
                type: answers['relationship_type'],
                description: 'User-defined relationship'
            })
        }

        // Parse custom mappings
        if (answers['custom_mappings']) {
            context.custom_mappings = answers['custom_mappings']
        }

        return context
    }

    const renderQuestion = (question: Question) => {
        const inputType = question.metadata?.input_type || 'text'

        switch (inputType) {
            case 'tags':
                return (
                    <Input
                        type="text"
                        placeholder={question.metadata?.placeholder}
                        value={answers[question.id] || ''}
                        onChange={(e) => handleAnswer(question.id, e.target.value)}
                        className="w-full"
                    />
                )

            case 'multi_select':
                return (
                    <div className="space-y-2 max-h-48 overflow-y-auto border border-border rounded p-3 bg-card">
                        {question.options.map((option) => (
                            <label key={option} className="flex items-center gap-2 cursor-pointer hover:bg-accent p-2 rounded">
                                <input
                                    type="checkbox"
                                    checked={(answers[question.id] || []).includes(option)}
                                    onChange={(e) => {
                                        const current = answers[question.id] || []
                                        const updated = e.target.checked
                                            ? [...current, option]
                                            : current.filter((v: string) => v !== option)
                                        handleAnswer(question.id, updated)
                                    }}
                                    className="rounded"
                                />
                                <span className="text-sm">{option}</span>
                            </label>
                        ))}
                    </div>
                )

            case 'column_descriptions':
                return (
                    <div className="space-y-3">
                        {question.metadata?.columns?.map((col) => (
                            <div key={col} className="flex items-center gap-3">
                                <span className="text-sm font-medium min-w-[150px] text-foreground">{col}:</span>
                                <Input
                                    type="text"
                                    placeholder={`Describe ${col}...`}
                                    value={(answers[question.id] || {})[col] || ''}
                                    onChange={(e) => {
                                        const current = answers[question.id] || {}
                                        handleAnswer(question.id, { ...current, [col]: e.target.value })
                                    }}
                                    className="flex-1"
                                />
                            </div>
                        ))}
                    </div>
                )

            default:
                if (question.options.length > 0) {
                    return (
                        <select
                            value={answers[question.id] || ''}
                            onChange={(e) => handleAnswer(question.id, e.target.value)}
                            className="w-full border border-border rounded-lg px-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-ring bg-background text-foreground"
                        >
                            <option value="">Select an option...</option>
                            {question.options.map((option) => (
                                <option key={option} value={option}>{option}</option>
                            ))}
                        </select>
                    )
                } else {
                    return (
                        <Input
                            type="text"
                            placeholder={question.metadata?.placeholder || 'Enter your answer...'}
                            value={answers[question.id] || ''}
                            onChange={(e) => handleAnswer(question.id, e.target.value)}
                            className="w-full"
                        />
                    )
                }
        }
    }

    const renderReviewStep = () => {
        return (
            <div className="space-y-6">
                <div>
                    <h3 className="font-semibold text-lg mb-3 text-foreground">File 1 Context</h3>
                    <div className="bg-muted rounded-lg p-5 space-y-2.5 text-sm border border-border">
                        <div><span className="font-medium text-foreground">Purpose:</span> <span className="text-foreground">{answers['f1_purpose'] || 'Not provided'}</span></div>
                        <div><span className="font-medium text-foreground">Domain:</span> <span className="text-foreground">{answers['f1_domain'] || 'Not provided'}</span></div>
                        <div><span className="font-medium text-foreground">Entities:</span> <span className="text-foreground">{answers['f1_entities'] || 'Not provided'}</span></div>
                    </div>
                </div>

                <div>
                    <h3 className="font-semibold text-lg mb-3 text-foreground">File 2 Context</h3>
                    <div className="bg-muted rounded-lg p-5 space-y-2.5 text-sm border border-border">
                        <div><span className="font-medium text-foreground">Purpose:</span> <span className="text-foreground">{answers['f2_purpose'] || 'Not provided'}</span></div>
                        <div><span className="font-medium text-foreground">Domain:</span> <span className="text-foreground">{answers['f2_domain'] || 'Not provided'}</span></div>
                        <div><span className="font-medium text-foreground">Entities:</span> <span className="text-foreground">{answers['f2_entities'] || 'Not provided'}</span></div>
                    </div>
                </div>

                <div>
                    <h3 className="font-semibold text-lg mb-3 text-foreground">Relationship</h3>
                    <div className="bg-muted rounded-lg p-5 text-sm border border-border text-foreground">
                        {answers['relationship_type'] || 'Not provided'}
                    </div>
                </div>
            </div>
        )
    }

    // Use portal to render at document body level to ensure full screen coverage
    const [mounted, setMounted] = useState(false)

    useEffect(() => {
        setMounted(true)
        return () => setMounted(false)
    }, [])

    if (!isOpen || !mounted) return null



    return createPortal(
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[9999] p-4" style={{ minHeight: '100vh', minWidth: '100vw', top: 0, left: 0, right: 0, bottom: 0 }}>
            <Card className="w-full max-w-6xl max-h-[90vh] overflow-hidden flex flex-col shadow-2xl bg-card">
                <CardHeader className="border-b border-border px-6 py-4 bg-card">
                    <div className="flex items-center justify-between">
                        <div>
                            <CardTitle className="flex items-center gap-2 text-xl">
                                <Sparkles className="h-6 w-6 text-info" />
                                Data Context Collection
                            </CardTitle>
                            <CardDescription className="mt-1.5 text-sm">
                                Help us understand your data for better correlation analysis
                            </CardDescription>
                        </div>
                        <Button variant="ghost" size="sm" onClick={onClose} className="hover:bg-accent">
                            <X className="h-5 w-5" />
                        </Button>
                    </div>
                </CardHeader>

                {/* Two-panel layout */}
                <div className="flex-1 flex overflow-hidden">
                    {/* Left Panel - Vertical Stepper */}
                    <div className="w-72 border-r border-border bg-gradient-to-b from-muted to-card p-6 flex flex-col">
                        <div className="space-y-2 flex-1">
                            {steps.map((step, index) => (
                                <div
                                    key={index}
                                    className={`flex items-start gap-3 p-4 rounded-lg transition-all duration-200 ${index === currentStep
                                        ? 'bg-info-light border-l-4 border-info shadow-sm'
                                        : index < currentStep
                                            ? 'bg-card border border-border'
                                            : 'opacity-50'
                                        }`}
                                >
                                    <div className={`w-9 h-9 rounded-full flex items-center justify-center text-sm font-semibold flex-shrink-0 transition-colors ${index < currentStep
                                        ? 'bg-success text-success-foreground shadow-md'
                                        : index === currentStep
                                            ? 'bg-info text-info-foreground shadow-md'
                                            : 'bg-muted text-muted-foreground'
                                        }`}>
                                        {index < currentStep ? <Check className="h-5 w-5" /> : index + 1}
                                    </div>
                                    <div className="flex-1 min-w-0 pt-0.5">
                                        <div className={`text-sm font-semibold ${index === currentStep ? 'text-info' : 'text-foreground'
                                            }`}>
                                            {step.title}
                                        </div>
                                        <div className="text-xs text-muted-foreground mt-1 leading-relaxed">
                                            {step.description}
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>

                        {/* Progress indicator */}
                        <div className="mt-6 pt-6 border-t border-border">
                            <div className="text-xs font-medium text-muted-foreground mb-2">Overall Progress</div>
                            <div className="w-full bg-muted rounded-full h-2.5 overflow-hidden">
                                <div
                                    className="bg-info h-2.5 rounded-full transition-all duration-500 ease-out"
                                    style={{ width: `${((currentStep + 1) / steps.length) * 100}%` }}
                                />
                            </div>
                            <div className="text-xs text-muted-foreground mt-2 font-medium">
                                Step {currentStep + 1} of {steps.length}
                            </div>
                        </div>
                    </div>

                    {/* Right Panel - Content */}
                    <div className="flex-1 flex flex-col overflow-hidden bg-card">
                        <CardContent className="flex-1 overflow-y-auto p-8">
                            {loading ? (
                                <div className="flex items-center justify-center py-20">
                                    <div className="text-center">
                                        <div className="animate-spin rounded-full h-16 w-16 border-b-4 border-info mx-auto mb-6"></div>
                                        <p className="text-muted-foreground font-medium">Generating questions...</p>
                                    </div>
                                </div>
                            ) : currentStep === steps.length - 1 ? (
                                renderReviewStep()
                            ) : (
                                <div className="space-y-6 max-w-4xl mx-auto">
                                    <div className="mb-8">
                                        <h3 className="text-2xl font-bold text-foreground">{steps[currentStep].title}</h3>
                                        <p className="text-sm text-muted-foreground mt-2">{steps[currentStep].description}</p>
                                    </div>

                                    {getCurrentQuestions().map((question, idx) => (
                                        <div key={question.id} className="space-y-3 bg-muted p-5 rounded-lg border border-border">
                                            <label className="block text-sm font-semibold text-foreground">
                                                <span className="text-muted-foreground mr-2">{idx + 1}.</span>
                                                {question.text}
                                                {question.required && <span className="text-error ml-1.5">*</span>}
                                            </label>
                                            {question.metadata?.hint && (
                                                <p className="text-xs text-muted-foreground italic">{question.metadata.hint}</p>
                                            )}
                                            <div className="mt-2">
                                                {renderQuestion(question)}
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </CardContent>

                        {/* Footer with navigation buttons */}
                        <div className="border-t border-border p-6 flex items-center justify-between bg-muted">
                            <Button
                                variant="outline"
                                onClick={handleBack}
                                disabled={currentStep === 0}
                                className="px-6"
                            >
                                <ChevronLeft className="h-4 w-4 mr-2" />
                                Back
                            </Button>

                            {currentStep === steps.length - 1 ? (
                                <Button
                                    onClick={handleSubmit}
                                    disabled={submitting}
                                    className="bg-success hover:bg-success/90 text-success-foreground px-8 text-base font-medium"
                                    size="lg"
                                >
                                    {submitting ? 'Submitting...' : 'Complete & Generate Correlation'}
                                    <Check className="h-5 w-5 ml-2" />
                                </Button>
                            ) : (
                                <Button
                                    onClick={handleNext}
                                    disabled={!canProceed()}
                                    className="bg-info hover:bg-info/90 text-info-foreground px-8 text-base font-medium"
                                    size="lg"
                                >
                                    Next Step
                                    <ChevronRight className="h-4 w-4 ml-2" />
                                </Button>
                            )}
                        </div>
                    </div>
                </div>
            </Card>
        </div>,
        document.body
    )
}
