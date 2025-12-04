import requests
import json
import re

ollama_base_url = "http://localhost:11434"
ollama_model = "qwen3-vl:2b"

def call_ollama(prompt: str) -> str:
    print(f"Calling Ollama with model: {ollama_model}")
    try:
        response = requests.post(
            f"{ollama_base_url}/api/generate",
            json={
                "model": ollama_model,
                "prompt": prompt,
                "stream": False
            },
            timeout=120
        )
        response.raise_for_status()
        result = response.json().get("response", "")
        return result
    except Exception as e:
        print(f"Error: {e}")
        return ""

def test_semantic_match():
    cols1 = ["id", "first_name", "salary", "department"]
    cols2 = ["emp_id", "given_name", "annual_income", "team"]
    
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
    
    print("Sending prompt...")
    response = call_ollama(prompt)
    print("\nResponse:")
    print(response)
    
    print("\nExtracting JSON...")
    json_match = re.search(r'\{.*\}', response, re.DOTALL)
    if json_match:
        json_str = json_match.group(0)
        try:
            data = json.loads(json_str)
            print("Parsed JSON:")
            print(json.dumps(data, indent=2))
        except Exception as e:
            print(f"JSON Parse Error: {e}")
    else:
        print("No JSON found in response")

if __name__ == "__main__":
    test_semantic_match()
