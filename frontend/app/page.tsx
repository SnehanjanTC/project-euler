"use client"

import Chatbot from "@/components/chatbot"
import Dashboard from "@/components/dashboard"
import Header from "@/components/header"
import { useState } from "react"

export default function Home() {
  const [csvLoaded, setCsvLoaded] = useState(false)

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-white">
      {/* Global Header */}
      <Header />
      
      {/* Main Content Area */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Side - Chatbot (40% width) */}
        <div className="w-2/5 border-r border-gray-300 flex flex-col overflow-hidden bg-white">
          <Chatbot onCsvLoadedChange={setCsvLoaded} />
        </div>
        
        {/* Right Side - Dashboard (60% width) */}
        <div className="w-3/5 bg-white flex flex-col overflow-hidden">
          <Dashboard csvLoaded={csvLoaded} />
        </div>
      </div>
    </div>
  )
}

