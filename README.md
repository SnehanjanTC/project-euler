<div align="center">
  <img src="frontend/public/project_euler.png" alt="Project Euler Logo" width="120" />
  
  # Project Euler
  ### Context-Aware Data Correlation System
  
  > **AI-Powered CSV Data Correlation with Context Collection for Maximum Accuracy**
</div>

A full-stack application that intelligently correlates columns between two CSV files using AI-driven context collection, semantic matching, and statistical analysis. Reduces false positives by 30-50% through business context awareness.

---

## 🎯 What is Project Euler?

Project Euler helps you automatically discover relationships between columns in two different CSV files—even when they have different names, formats, or structures. By collecting business context about your datasets, it dramatically improves correlation accuracy and provides confidence scores for each match.

**Perfect for:**
- Data migration and ETL pipelines
- Database schema mapping
- Data integration projects
- Business intelligence workflows
- Legacy system modernization

---

## ✨ Key Features

### 🧠 **Context-Aware Correlation**
- **AI-Driven Question Generation**: Automatically creates relevant questions based on your data
- **Multi-Step Wizard**: Collects business context about datasets (purpose, domain, entities)
- **Smart Matching**: Uses context to filter false positives and boost confidence scores
- **Custom Mappings**: Define specific column pairs with 95% confidence guarantee
- **Column Exclusions**: Filter out debug/temporary columns from analysis

### 📊 **Advanced Correlation Engine**
- **Statistical Analysis**: Correlation coefficients for numeric data
- **Semantic Matching**: AI-powered name similarity and meaning analysis
- **Distribution Comparison**: Matches columns with similar data patterns
- **Confidence Scoring**: 0-100% confidence for each column pair
- **Interactive Visualization**: Flow diagram showing relationships with color-coded confidence

### 🔒 **Production-Grade Security**
- **API Key Encryption**: AES-GCM encryption for localStorage (Web Crypto API)
- **Security Headers**: CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy
- **Rate Limiting**: Sliding window algorithm with HTTP 429 responses
- **HTTPS Enforcement**: Production SSL/TLS support with Nginx reverse proxy
- **CORS Protection**: Configurable allowed origins for production

### 🎨 **Modern UI/UX**
- **React Portal Modal**: Full-screen context wizard with smooth animations
- **Two-Panel Layout**: Vertical stepper + questionnaire for intuitive navigation
- **Progress Indicators**: Real-time feedback on context collection progress
- **Export Functionality**: Download correlation mappings as JSON
- **Responsive Design**: Works seamlessly on desktop and tablet

### 🦙 **Flexible AI Backend**
- **Local LLM**: Ollama support (Llama3, Mistral, Qwen, etc.)
- **Cloud LLM**: Optional OpenAI/Anthropic/Gemini integration
- **Configurable UI**: Change model and endpoint through the app
- **Fallback Support**: Graceful degradation when LLM unavailable

---

## 🏗️ Architecture

```
┌─────────────────────────┐
│   Next.js Frontend      │
│  (React + TypeScript)   │
│                         │
│  • Context Wizard       │
│  • Dashboard            │
│  • API Key Manager      │
│  • Visualization        │
└──────────┬──────────────┘
           │
           │ REST API (Port 8001)
           ▼
┌──────────────────────────────────────────────┐
│        Backend (Choose One)                  │
├──────────────────┬───────────────────────────┤
│  Python (FastAPI)│      Go (Chi Router)      │
│                  │                           │
│ • Context Service│  • Context Service        │
│ • Question Gen   │  • Question Generator     │
│ • ML Matcher     │  • AI Semantic Matcher    │
│ • Rate Limiting  │  • Adaptive Learning      │
│ • Pandas Analysis│  • Pattern Learning       │
│                  │  • Confidence Calibration │
└────────┬─────────┴──────────┬────────────────┘
         │                    │
         └────────┬───────────┘
                  │
                  ├──► Ollama (Local LLM)
                  └──► OpenAI/Anthropic (Optional)
```

### Backend Options

| Feature | Python (FastAPI) | Go (Chi) |
|---------|------------------|----------|
| **CSV Parsing** | Pandas | Native Go |
| **ML Matching** | Sentence Transformers | Heuristic + LLM |
| **Learning** | Basic | Adaptive Weights, Pattern Learning |
| **Performance** | Good | Excellent |
| **Memory** | Higher | Lower |


---

## 🚀 Quick Start

### Prerequisites

- **Python 3.10+**
- **Node.js 18+**
- **Ollama** (for local LLM) - [Download](https://ollama.ai/download)
- Optional: **OpenAI/Anthropic API Key** (for cloud LLM)

### 1. Install Ollama

```bash
# Download from https://ollama.ai/download
# Then pull a model
ollama pull qwen3-vl:2b
# or
ollama pull llama3
ollama pull mistral
```

### 2a. Backend Setup (Python)

```bash
cd backend

# Create virtual environment
python -m venv venv

# Activate (Windows)
venv\Scripts\activate
# Activate (macOS/Linux)
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# (Optional) Create .env file
cp .env.template .env
# Edit .env with your API keys if using cloud LLM

# Start backend
python main.py
```

Backend runs on **`http://localhost:8001`**

### 2b. Backend Setup (Go - Alternative)

> **Go Backend Features**: Adaptive weight learning, pattern learning, confidence calibration, AI semantic matching via Ollama.

```bash
cd backend-go

# Build
go build ./cmd/server/main.go

# Run
go run ./cmd/server/main.go
# or
./main.exe  # Windows
./main      # Linux/macOS
```

Backend runs on **`http://localhost:8001`**

**Go Backend Endpoints:**
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/upload` | POST | Upload CSV files |
| `/column-similarity` | GET | Get column matches (add `?use_ai=true` for LLM) |
| `/correlation` | GET | Get numeric correlations |
| `/feedback/match` | POST | Submit match feedback (👍/👎) |
| `/feedback/stats` | GET | Get learning statistics |
| `/config/ollama` | GET/POST | Configure Ollama |

### 3. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# (Optional) Create .env.local for custom API URL
echo "NEXT_PUBLIC_API_URL=http://localhost:8001" > .env.local

# Start frontend
npm run dev
```

Frontend runs on **`http://localhost:3000`**


### 4. Open Browser

Navigate to `http://localhost:3000` and start correlating!

---

## 📖 How to Use

### Basic Workflow

1. **Upload Two CSV Files**  
   Click "Upload" for File 1 and File 2 (or drag & drop)

2. **Add Context (Recommended)**  
   Click "Add Context & Generate" to open the wizard:
   - **Step 1**: Answer questions about File 1 (purpose, domain, entities)
   - **Step 2**: Answer questions about File 2
   - **Step 3**: Describe relationship between files
   - **Step 4**: Review and confirm

3. **View Correlation Results**  
   Interactive flow diagram showing column relationships with confidence percentages

4. **Export Mapping**  
   Download the correlation results as JSON for use in ETL pipelines

### Advanced Features

#### Custom Column Mappings
Define specific column pairs that should map together:
- Example: `user_id` (File 1) → `customer_id` (File 2)
- Automatically assigned **95% confidence**

#### Column Exclusions
Exclude columns from correlation:
- Temp columns, debug fields, metadata, etc.
- Reduces noise and improves accuracy

#### Domain-Specific Boosting
When both files belong to the same business domain (e.g., "Sales"), similar column names receive a **10% confidence boost**.

#### Entity Overlap Scoring
Files with overlapping key entities (e.g., "Customer", "Order") get **up to 20% confidence boost** for related columns.

---

## ⚙️ Configuration

### Environment Variables

#### Backend (`.env`)

```env
# Environment
ENVIRONMENT=development  # or production

# CORS
ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
ALLOWED_ORIGINS_PROD=https://yourdomain.com  # Production only

# Ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=qwen3-vl:2b

# Cloud LLM (Optional)
OPENAI_API_KEY=sk-your-key-here

# Rate Limiting
RATE_LIMIT_ENABLED=True
MAX_REQUESTS_PER_MINUTE=60
MAX_LLM_CALLS_PER_HOUR=100

# File Upload
MAX_FILE_SIZE=104857600  # 100MB
MAX_ROWS_FOR_ANALYSIS=1000000
```

#### Frontend (`.env.local`)

```env
# API URL
NEXT_PUBLIC_API_URL=http://localhost:8001

# Environment
NEXT_PUBLIC_ENVIRONMENT=development
```

### Ollama Configuration UI

You can configure Ollama directly in the app:
1. Click the "API Keys" button in the dashboard
2. Scroll to "Ollama Local" section
3. Set Base URL and Model Name
4. Click "Save Ollama Config"

Changes take effect immediately without restarting the backend.

---

## 📦 Project Structure

```
project_euler/
├── backend/                        # Python Backend (FastAPI)
│   ├── app/
│   │   ├── routers/api.py          # API endpoints
│   │   ├── services/
│   │   │   ├── context_service.py  # Context management
│   │   │   ├── question_generator.py
│   │   │   ├── similarity.py
│   │   │   └── llm.py
│   │   ├── utils/
│   │   └── config.py
│   ├── main.py
│   └── requirements.txt
│
├── backend-go/                     # Go Backend (Chi Router)
│   ├── cmd/server/main.go          # Entry point
│   ├── internal/
│   │   ├── api/handlers.go         # HTTP handlers
│   │   ├── service/
│   │   │   ├── context.go          # Context management
│   │   │   ├── enhanced_similarity.go  # Column matching
│   │   │   ├── ai_matcher.go       # LLM-powered matching
│   │   │   ├── adaptive_learning.go    # Weight learning
│   │   │   ├── confidence_calibration.go
│   │   │   ├── pattern_learning.go
│   │   │   └── feedback_learning.go
│   │   ├── llm/service.go          # Ollama integration
│   │   └── state/state.go          # Global state
│   └── go.mod
│
├── frontend/                       # Next.js Frontend
│   ├── app/
│   ├── components/
│   │   ├── dashboard.tsx
│   │   ├── context-wizard.tsx
│   │   └── ui/
│   ├── lib/
│   │   ├── api-config.ts
│   │   └── crypto.ts
│   └── package.json
│
└── README.md
```



## 🤝 Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 💡 Tips for Best Results

1. **Provide detailed context**: More context = better accuracy
2. **Use consistent domains**: Files from the same business area correlate better
3. **Define custom mappings**: For known column pairs, set them explicitly
4. **Exclude irrelevant columns**: Temp/debug columns add noise
5. **Review confidence scores**: Values <50% may need manual verification
6. **Export mappings**: Save results for reuse in ETL pipelines

---

## 🙏 Acknowledgments

- **Ollama** - Local LLM runtime
- **Next.js** - React framework
- **FastAPI** - High-performance Python web framework
- **Shadcn UI** - Beautiful component library
- **pandas** - Data manipulation library