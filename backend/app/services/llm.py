import requests
import json
import re
from app.config import OLLAMA_BASE_URL, OLLAMA_MODEL

def call_ollama(prompt: str) -> str:
    """Call Ollama API to get response. Returns empty string if Ollama is not available."""
    try:
        response = requests.post(
            f"{OLLAMA_BASE_URL}/api/generate",
            json={
                "model": OLLAMA_MODEL,
                "prompt": prompt,
                "stream": False
            },
            timeout=30  # Reduced timeout to prevent hanging
        )
        response.raise_for_status()
        result = response.json().get("response", "")
        if not result:
            return ""
        return result
    except requests.exceptions.ConnectionError:
        # Ollama is not running or not accessible
        return ""
    except requests.exceptions.Timeout:
        # Request timed out
        return ""
    except requests.exceptions.RequestException as e:
        # Other request errors
        print(f"Warning: Ollama request failed: {str(e)}")
        return ""

def get_semantic_matches_from_llm(cols1: list, cols2: list) -> dict:
    """Get semantic matches between two lists of columns using Ollama"""
    try:
        prompt = f"""
        You are an expert data integration specialist. Match columns from List A to List B based on semantic meaning.
        
        List A: {', '.join(cols1)}
        List B: {', '.join(cols2)}
        
        Return a JSON object where keys are columns from List A and values are the best matching column from List B.
        Only include matches where you are confident (score > 0.5).
        
        Format:
        {{
            "matches": [
                {{"col_a": "column_from_list_a", "col_b": "column_from_list_b", "confidence": 0.9, "reason": "Both refer to..."}}
            ]
        }}
        
        Return ONLY the JSON.
        """
        
        response = call_ollama(prompt)
        print(f"DEBUG: LLM Response for semantic match: {response[:500]}...")
        
        # Extract JSON from response
        json_match = re.search(r'\{.*\}', response, re.DOTALL)
        if json_match:
            json_str = json_match.group(0)
            return json.loads(json_str)
        return {"matches": []}
    except Exception as e:
        print(f"Error getting semantic matches: {str(e)}")
        return {"matches": []}
