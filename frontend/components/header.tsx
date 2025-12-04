"use client"

import { useState } from 'react'
import Image from 'next/image'
import AIProviderModal from './ai-provider-modal'

export default function Header() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  return (
    <>
      <header className="w-full border-b border-gray-300 bg-white z-50">
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

          <button
            onClick={() => setIsModalOpen(true)}
            className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors text-sm font-medium text-gray-700"
          >
            AI Settings
          </button>
        </div>
      </header>

      <AIProviderModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </>
  )
}
