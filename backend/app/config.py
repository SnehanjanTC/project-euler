import os

# CORS settings
ALLOWED_ORIGINS = os.getenv("ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000").split(",")
ENVIRONMENT = os.getenv("ENVIRONMENT", "development")
IS_PRODUCTION = ENVIRONMENT == "production"

# Production CORS origins (override ALLOWED_ORIGINS in production)
if IS_PRODUCTION:
    prod_origins = os.getenv("ALLOWED_ORIGINS_PROD")
    if prod_origins:
        ALLOWED_ORIGINS = prod_origins.split(",")

# Ollama settings
OLLAMA_MODEL = "qwen3-vl:2b"
OLLAMA_BASE_URL = "http://localhost:11434"

# Rate limiting
RATE_LIMIT_ENABLED = os.getenv("RATE_LIMIT_ENABLED", "True").lower() == "true"
MAX_REQUESTS_PER_MINUTE = int(os.getenv("MAX_REQUESTS_PER_MINUTE", "60"))
MAX_LLM_CALLS_PER_HOUR = int(os.getenv("MAX_LLM_CALLS_PER_HOUR", "100"))

# File settings
MAX_FILE_SIZE = 50 * 1024 * 1024  # 50MB
UPLOAD_DIR = "uploads"
FILE_CLEANUP_INTERVAL = 3600  # 1 hour
MAX_FILE_AGE = 86400  # 24 hours

# Analysis settings
MAX_ROWS_FOR_ANALYSIS = 1000000

