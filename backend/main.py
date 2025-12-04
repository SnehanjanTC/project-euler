from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
import os
import shutil
from app.config import ALLOWED_ORIGINS, UPLOAD_DIR
from app.routers import api
from app.utils.files import cleanup_old_files

app = FastAPI(title="CSV Data Reader with AI", description="Backend API for CSV analysis using Ollama")

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=ALLOWED_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include API router
app.include_router(api.router)

@app.on_event("startup")
async def startup_event():
    """Run on startup"""
    # Create upload directory
    if not os.path.exists(UPLOAD_DIR):
        os.makedirs(UPLOAD_DIR)
        print(f"Created upload directory: {UPLOAD_DIR}")

@app.on_event("shutdown")
async def shutdown_event():
    """Cleanup on shutdown"""
    try:
        if os.path.exists(UPLOAD_DIR):
            shutil.rmtree(UPLOAD_DIR)
            print("Cleaned up upload directory")
    except Exception as e:
        print(f"Error cleaning up: {e}")

if __name__ == "__main__":
    uvicorn.run("main:app", host="0.0.0.0", port=8001, reload=True)
