import requests
import json
import os

BASE_URL = "http://localhost:8001"
ARTIFACTS_DIR = r"C:\Users\ArkapravaGhosh\.gemini\antigravity\brain\8568f57c-9393-466f-b838-802afdfb4d72"

def upload_file(file_path, index):
    print(f"Uploading {file_path} as file {index}...")
    with open(file_path, 'rb') as f:
        files = {'file': (os.path.basename(file_path), f, 'text/csv')}
        response = requests.post(f"{BASE_URL}/upload?file_index={index}", files=files)
        if response.status_code == 200:
            print(f"Success: {response.json()['message']}")
        else:
            print(f"Error: {response.text}")

def check_similarity():
    print("\nChecking column similarity...")
    response = requests.get(f"{BASE_URL}/column-similarity")
    if response.status_code == 200:
        data = response.json()
        print(f"Found {len(data['similarities'])} similarities.")
        print("\nTop Similarities:")
        for sim in data['similarities'][:10]:
            print(f"{sim['file1_column']} <-> {sim['file2_column']}")
            print(f"  Confidence: {sim['confidence']:.2f}%")
            print(f"  Type: {sim['type']}")
            print(f"  LLM Score: {sim.get('llm_semantic_score', 0):.2f}")
            print(f"  Dist Sim: {sim.get('distribution_similarity', 0):.2f}")
            print("-" * 30)
    else:
        print(f"Error: {response.text}")

if __name__ == "__main__":
    file1 = os.path.join(ARTIFACTS_DIR, "test_data_1.csv")
    file2 = os.path.join(ARTIFACTS_DIR, "test_data_2.csv")
    
    upload_file(file1, 1)
    upload_file(file2, 2)
    check_similarity()
