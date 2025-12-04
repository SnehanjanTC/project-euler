"""
OpenAI integration for fast, high-quality semantic matching
"""
import json
import os
from typing import Optional
from openai import OpenAI

# Initialize OpenAI client
client: Optional[OpenAI] = None
try:
    api_key = os.getenv("OPENAI_API_KEY")
    if api_key:
        client = OpenAI(api_key=api_key)
        print("OpenAI client initialized successfully")
    else:
        print("OPENAI_API_KEY not found, OpenAI features will be disabled")
except Exception as e:
    print(f"Failed to initialize OpenAI client: {e}")
    client = None


def get_semantic_matches_openai(cols1: list, cols2: list, timeout: int = 10) -> dict:
    """
    Fast, high-quality semantic matching using OpenAI GPT-4o-mini
    
    Args:
        cols1: List of column names from first file
        cols2: List of column names from second file
        timeout: Request timeout in seconds
        
    Returns:
        Dictionary with matches: [{col_a, col_b, confidence, reason}]
    """
    if not client:
        raise ValueError("OpenAI client not initialized")
    
    # Build prompt with few-shot examples
    prompt = f"""You are an expert data integration specialist. Match columns from List A to List B based on semantic meaning.

EXAMPLE MATCHES (for reference):
- "employee_id" ↔ "emp_id" (confidence: 0.95, reason: "Both uniquely identify employees")
- "first_name" ↔ "fname" (confidence: 0.90, reason: "Common abbreviation for first name")
- "salary" ↔ "annual_income" (confidence: 0.85, reason: "Both represent employee compensation")
- "dept" ↔ "department" (confidence: 0.80, reason: "Standard abbreviation")
- "hire_date" ↔ "join_date" (confidence: 0.85, reason: "Both indicate when employee started")

NOW MATCH THESE COLUMNS:
List A (File 1): {', '.join(cols1)}
List B (File 2): {', '.join(cols2)}

RULES:
1. Only include matches with confidence > 0.6
2. Confidence should reflect semantic similarity (0.0 to 1.0)
3. Provide a brief reason for each match
4. Return ONLY the JSON, no other text

OUTPUT FORMAT (JSON only):
{{
    "matches": [
        {{"col_a": "column_from_A", "col_b": "column_from_B", "confidence": 0.85, "reason": "description"}}
    ]
}}
"""
    
    try:
        response = client.chat.completions.create(
            model="gpt-4o-mini",
            messages=[
                {"role": "system", "content": "You are a data integration expert. Always return valid JSON."},
                {"role": "user", "content": prompt}
            ],
            response_format={"type": "json_object"},
            temperature=0.3,
            max_tokens=1000,
            timeout=timeout
        )
        
        result = json.loads(response.choices[0].message.content)
        print(f"OpenAI returned {len(result.get('matches', []))} matches")
        return result
        
    except Exception as e:
        print(f"OpenAI API error: {str(e)}")
        raise


def get_semantic_matches_hybrid(cols1: list, cols2: list) -> dict:
    """
    Hybrid approach: Try OpenAI first, fallback to Ollama if it fails
    
    Args:
        cols1: List of column names from first file
        cols2: List of column names from second file
        
    Returns:
        Dictionary with matches from either OpenAI or Ollama
    """
    try:
        # Try OpenAI first (fast and high quality)
        print("Attempting OpenAI semantic matching...")
        return get_semantic_matches_openai(cols1, cols2)
    except Exception as e:
        # Fallback to local Ollama
        print(f"OpenAI failed ({str(e)}), falling back to Ollama...")
        from app.services.llm import get_semantic_matches_from_llm
        return get_semantic_matches_from_llm(cols1, cols2)
