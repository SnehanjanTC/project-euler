"use client"

import { useState } from 'react'
import Image from 'next/image'
import AIProviderModal from './ai-provider-modal'
import { ThemeToggle } from './theme-toggle'

export default function Header() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  return (
    <>
      <header className="w-full border-b border-border bg-background z-50">
        <div className="h-20 flex items-center justify-between px-6">
          <div className="flex items-center gap-3">
            <Image
              src="/project_euler.png"
              alt="Project Euler"
              width={180}
              height={180}
              className="object-contain"
            />
          </div>

          <div className="flex items-center gap-3">
            <ThemeToggle />
            <button
              onClick={() => setIsModalOpen(true)}
              className="px-4 py-2 border border-border rounded-md hover:bg-accent transition-colors text-sm font-medium text-foreground"
            >
              AI Settings
            </button>
          </div>
        </div>
      </header>

      <AIProviderModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </>
  )
}
