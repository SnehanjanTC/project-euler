import pandas as pd
import numpy as np
import re
from sentence_transformers import SentenceTransformer, util
from sklearn.metrics.pairwise import cosine_similarity
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MLMatcher:
    _instance = None
    _model = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(MLMatcher, cls).__new__(cls)
            try:
                # Load a lightweight, fast model optimized for semantic similarity
                logger.info("Loading ML model for semantic matching...")
                cls._model = SentenceTransformer('all-MiniLM-L6-v2')
                logger.info("ML model loaded successfully.")
            except Exception as e:
                logger.error(f"Failed to load ML model: {e}")
                cls._model = None
        return cls._instance

    def compute_embeddings(self, texts: list[str]):
        """Generate vector embeddings for a list of texts."""
        if self._model is None:
            return None
        try:
            return self._model.encode(texts, convert_to_tensor=True)
        except Exception as e:
            logger.error(f"Error computing embeddings: {e}")
            return None

    def compute_semantic_similarity(self, emb1, emb2):
        """Compute cosine similarity between two sets of embeddings."""
        if emb1 is None or emb2 is None:
            return None
        try:
            # util.cos_sim returns a tensor, convert to numpy
            return util.cos_sim(emb1, emb2).cpu().numpy()
        except Exception as e:
            logger.error(f"Error computing semantic similarity: {e}")
            return None

    def compute_pattern_score(self, col1_name: str, col2_name: str) -> float:
        """Check for common regex patterns (IDs, Emails, Dates, etc.)"""
        patterns = {
            'email': r'email|e-mail|mail',
            'id': r'id|identifier|key|code|number|no\.',
            'date': r'date|time|year|month|day|created|updated|timestamp',
            'name': r'name|first|last|full|surname',
            'phone': r'phone|mobile|cell|contact',
            'address': r'address|city|state|zip|postal|country|location',
            'price': r'price|cost|amount|value|total|revenue|sales',
            'status': r'status|state|condition|phase'
        }
        
        c1 = col1_name.lower()
        c2 = col2_name.lower()
        
        score = 0.0
        for key, pattern in patterns.items():
            match1 = re.search(pattern, c1)
            match2 = re.search(pattern, c2)
            
            if match1 and match2:
                # Both match the same pattern category
                score = 1.0
                break
            elif (match1 and not match2) or (not match1 and match2):
                # Only one matches, potential mismatch if strong pattern
                pass
                
        return score

    def compute_name_similarity(self, name1: str, name2: str) -> float:
        """Compute simple string similarity (Levenshtein-like)"""
        n1 = name1.lower().replace('_', ' ').replace('-', ' ')
        n2 = name2.lower().replace('_', ' ').replace('-', ' ')
        
        if n1 == n2:
            return 1.0
        if n1 in n2 or n2 in n1:
            return 0.8
            
        # Jaccard on words
        words1 = set(n1.split())
        words2 = set(n2.split())
        if not words1 or not words2:
            return 0.0
            
        intersection = len(words1.intersection(words2))
        union = len(words1.union(words2))
        return intersection / union

    def compute_statistical_score(self, col1: pd.Series, col2: pd.Series) -> float:
        """Compute similarity based on data distribution statistics"""
        try:
            if not pd.api.types.is_numeric_dtype(col1) or not pd.api.types.is_numeric_dtype(col2):
                return 0.0
                
            stats1 = col1.describe()
            stats2 = col2.describe()
            
            # Compare Coefficient of Variation (CV)
            cv1 = stats1['std'] / stats1['mean'] if stats1['mean'] != 0 else 0
            cv2 = stats2['std'] / stats2['mean'] if stats2['mean'] != 0 else 0
            
            cv_diff = abs(cv1 - cv2) / (max(abs(cv1), abs(cv2)) + 1e-9)
            cv_sim = max(0, 1 - cv_diff)
            
            return cv_sim
        except Exception:
            return 0.0

    def get_matches(self, df1: pd.DataFrame, df2: pd.DataFrame, top_k: int = 10) -> list:
        """
        Main method to find best matches between two DataFrames using weighted ensemble scoring.
        """
        matches = []
        
        cols1 = df1.columns.tolist()
        cols2 = df2.columns.tolist()
        
        # 1. Compute Semantic Similarity Matrix (Batch Operation - Fast)
        semantic_matrix = None
        if self._model:
            emb1 = self.compute_embeddings(cols1)
            emb2 = self.compute_embeddings(cols2)
            semantic_matrix = self.compute_semantic_similarity(emb1, emb2)
        
        # 2. Iterate and Score
        for i, c1 in enumerate(cols1):
            for j, c2 in enumerate(cols2):
                
                # --- A. Semantic Score (40%) ---
                semantic_score = 0.0
                if semantic_matrix is not None:
                    semantic_score = float(semantic_matrix[i][j])
                
                # --- B. Name Score (20%) ---
                name_score = self.compute_name_similarity(c1, c2)
                
                # --- C. Pattern Score (10%) ---
                pattern_score = self.compute_pattern_score(c1, c2)
                
                # --- D. Statistical/Content Score (30%) ---
                content_score = 0.0
                # Check data types
                is_num1 = pd.api.types.is_numeric_dtype(df1[c1])
                is_num2 = pd.api.types.is_numeric_dtype(df2[c2])
                
                if is_num1 and is_num2:
                    # Numeric: Use statistical distribution
                    content_score = self.compute_statistical_score(df1[c1], df2[c2])
                elif not is_num1 and not is_num2:
                    # String: Use Jaccard on unique values (sample for speed)
                    try:
                        s1 = set(df1[c1].dropna().astype(str).unique()[:100])
                        s2 = set(df2[c2].dropna().astype(str).unique()[:100])
                        if s1 and s2:
                            content_score = len(s1.intersection(s2)) / len(s1.union(s2))
                    except Exception:
                        content_score = 0.0
                
                # --- Final Weighted Score ---
                # Weights: Semantic=0.4, Name=0.2, Pattern=0.1, Content=0.3
                final_score = (
                    (semantic_score * 0.4) + 
                    (name_score * 0.2) + 
                    (pattern_score * 0.1) + 
                    (content_score * 0.3)
                )
                
                # Boost exact name matches
                if name_score == 1.0:
                    final_score = max(final_score, 0.95)
                
                # Penalize type mismatch (strong penalty)
                if is_num1 != is_num2:
                    final_score *= 0.5
                
                # Threshold to keep
                if final_score > 0.1:
                    matches.append({
                        "file1_column": c1,
                        "file2_column": c2,
                        "similarity": float(final_score),
                        "confidence": float(final_score * 100),
                        "type": "ml_ensemble",
                        "details": {
                            "semantic": float(semantic_score),
                            "name": float(name_score),
                            "pattern": float(pattern_score),
                            "content": float(content_score)
                        }
                    })
        
        # Sort by score descending
        matches.sort(key=lambda x: x['similarity'], reverse=True)
        
        return matches[:top_k]
