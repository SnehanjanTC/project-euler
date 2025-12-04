from pydantic import BaseModel, Field, validator
from typing import Optional, List, Dict, Any
import re

def sanitize_input(text: str, max_length: int = 10000) -> str:
    """Sanitize user input to prevent injection attacks"""
    if not text or not isinstance(text, str):
        return ""
    
    # Limit length
    text = text[:max_length]
    
    # Remove potentially dangerous patterns
    dangerous_patterns = [
        r'__import__',
        r'eval\s*\(',
        r'exec\s*\(',
        r'compile\s*\(',
        r'open\s*\(',
        r'file\s*\(',
        r'input\s*\(',
        r'raw_input\s*\(',
        r'subprocess',
        r'os\.system',
        r'shell\s*=',
        r'import\s+os',
        r'import\s+sys',
        r'import\s+subprocess',
    ]
    
    for pattern in dangerous_patterns:
        text = re.sub(pattern, '', text, flags=re.IGNORECASE)
    
    # Remove null bytes
    text = text.replace('\x00', '')
    
    return text.strip()

class QueryRequest(BaseModel):
    question: str = Field(..., min_length=1, max_length=10000)
    
    @validator('question')
    def validate_question(cls, v):
        return sanitize_input(v)

class UploadResponse(BaseModel):
    message: str
    rows: int
    columns: int
    column_names: list
