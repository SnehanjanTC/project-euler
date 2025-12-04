import pandas as pd
import numpy as np
from app.services.llm import get_semantic_matches_from_llm
from app.state import current_df, current_df2

def calculate_distribution_similarity(col1: pd.Series, col2: pd.Series) -> float:
    """Calculate similarity based on data distribution (normalized stats)"""
    try:
        # Only for numeric columns
        if not pd.api.types.is_numeric_dtype(col1) or not pd.api.types.is_numeric_dtype(col2):
            return 0.0
            
        stats1 = col1.describe()
        stats2 = col2.describe()
        
        # Normalize and compare key metrics
        # We compare the relative shape, not absolute values (unless they are close)
        
        # Coefficient of Variation (std / mean) - measures dispersion relative to mean
        cv1 = stats1['std'] / stats1['mean'] if stats1['mean'] != 0 else 0
        cv2 = stats2['std'] / stats2['mean'] if stats2['mean'] != 0 else 0
        
        # Skewness
        skew1 = col1.skew()
        skew2 = col2.skew()
        
        # Calculate differences
        cv_diff = abs(cv1 - cv2) / (max(abs(cv1), abs(cv2)) + 1e-9)
        skew_diff = abs(skew1 - skew2) / (max(abs(skew1), abs(skew2)) + 1e-9)
        
        # Similarity scores (inverse of difference)
        cv_sim = max(0, 1 - cv_diff)
        skew_sim = max(0, 1 - skew_diff)
        
        return (cv_sim * 0.6) + (skew_sim * 0.4)
    except Exception:
        return 0.0

def calculate_column_similarity(col1: pd.Series, col2: pd.Series) -> dict:
    """Calculate similarity between two columns with confidence score"""
    similarity_score = 0.0
    similarity_type = "none"
    
    # 1. Data Type Similarity
    if pd.api.types.is_numeric_dtype(col1) and pd.api.types.is_numeric_dtype(col2):
        # For numeric columns, check correlation if lengths match
        if len(col1) == len(col2):
            try:
                correlation = col1.corr(col2)
                if not pd.isna(correlation):
                    similarity_score = abs(correlation)
                    similarity_type = "correlation"
            except Exception:
                pass
    elif pd.api.types.is_string_dtype(col1) and pd.api.types.is_string_dtype(col2):
        # For string columns, check Jaccard similarity of unique values
        set1 = set(col1.dropna().unique())
        set2 = set(col2.dropna().unique())
        if len(set1) > 0 and len(set2) > 0:
            intersection = len(set1.intersection(set2))
            union = len(set1.union(set2))
            similarity_score = intersection / union
            similarity_type = "jaccard"
            
    # 2. Name Similarity
    name1 = str(col1.name).lower().replace('_', ' ').replace('-', ' ')
    name2 = str(col2.name).lower().replace('_', ' ').replace('-', ' ')
    
    name_similarity = 0.0
    if name1 == name2:
        name_similarity = 1.0
    elif name1 in name2 or name2 in name1:
        name_similarity = 0.8
    else:
        # Check for common words
        words1 = set(name1.split())
        words2 = set(name2.split())
        if words1 and words2:
            common = len(words1.intersection(words2))
            if common > 0:
                name_similarity = 0.5 * (common / max(len(words1), len(words2)))
    
    # Combine similarity scores
    if similarity_type != "none":
        # Weight: 40% data similarity, 40% semantic/name, 20% distribution (if numeric)
        
        dist_sim = 0.0
        if pd.api.types.is_numeric_dtype(col1) and pd.api.types.is_numeric_dtype(col2):
            dist_sim = calculate_distribution_similarity(col1, col2)
            final_similarity = (similarity_score * 0.4) + (name_similarity * 0.4) + (dist_sim * 0.2)
        else:
            final_similarity = (similarity_score * 0.6) + (name_similarity * 0.4)
            
        final_confidence = min(100.0, final_similarity * 100)
    else:
        # Only name similarity available
        final_similarity = name_similarity
        final_confidence = min(100.0, name_similarity * 100)
        similarity_type = "name_only"
    
    return {
        "similarity": float(final_similarity),
        "confidence": float(final_confidence),
        "type": similarity_type,
        "data_similarity": float(similarity_score) if similarity_type != "none" else 0.0,
        "name_similarity": float(name_similarity),
        "distribution_similarity": float(dist_sim) if 'dist_sim' in locals() else 0.0
    }

def compare_json_structures(json1: list, json2: list, col1: str, col2: str) -> float:
    """Compare JSON structures for similarity confidence"""
    try:
        # Extract values for the specified columns
        values1 = [item.get(col1) for item in json1 if item.get(col1) is not None]
        values2 = [item.get(col2) for item in json2 if item.get(col2) is not None]
        
        if not values1 or not values2:
            return 0.0
            
        # Check type consistency
        type1 = type(values1[0])
        type2 = type(values2[0])
        
        if type1 != type2:
            return 0.0
            
        # Check value distribution similarity (simple version)
        if isinstance(values1[0], (int, float)):
            avg1 = sum(values1) / len(values1)
            avg2 = sum(values2) / len(values2)
            # If averages are within 20% of each other
            if abs(avg1 - avg2) < 0.2 * max(abs(avg1), abs(avg2)):
                return 80.0
        elif isinstance(values1[0], str):
            # Check length similarity
            avg_len1 = sum(len(s) for s in values1) / len(values1)
            avg_len2 = sum(len(s) for s in values2) / len(values2)
            if abs(avg_len1 - avg_len2) < 2:
                return 60.0
                
        return 40.0
    except Exception:
        return 0.0

def get_column_similarity_data(df1: pd.DataFrame, df2: pd.DataFrame, 
                               context1: dict = None, context2: dict = None):
    """Calculate similarity between all columns of two dataframes"""
    similarities = []
    nodes = []
    edges = []
    
    file1_cols = df1.columns.tolist()
    file2_cols = df2.columns.tolist()
    
    # Create nodes for graph
    for col in file1_cols:
        nodes.append({"id": f"file1_{col}", "label": col, "group": "file1"})
    
    for col in file2_cols:
        nodes.append({"id": f"file2_{col}", "label": col, "group": "file2"})
    
    # Lighter Heuristic Matching (No Heavy ML)
    # Enhanced with stricter thresholds to reduce false positives
    
    print(f"Calculating similarity for {len(file1_cols)}x{len(file2_cols)} matrix using enhanced heuristics...")
    
    import re
    
    def get_pattern_score(c1, c2):
        """Enhanced pattern matching with stricter rules"""
        patterns = {
            'email': r'\b(email|e[-_]?mail|mail)\b',
            'id': r'\b(id|identifier|key|code|number|no\.?)\b',
            'date': r'\b(date|time|year|month|day|created|updated|timestamp)\b',
            'name': r'\b(name|first|last|full|surname|given)\b',
            'phone': r'\b(phone|mobile|cell|contact|tel)\b',
            'address': r'\b(address|city|state|zip|postal|country|location)\b',
            'price': r'\b(price|cost|amount|value|total|revenue|fee)\b',
            'status': r'\b(status|state|condition|flag)\b',
            'description': r'\b(desc|description|details|notes|comment)\b',
        }
        c1_lower = c1.lower()
        c2_lower = c2.lower()
        
        # Check if both columns match the same pattern category
        for key, pattern in patterns.items():
            match1 = bool(re.search(pattern, c1_lower))
            match2 = bool(re.search(pattern, c2_lower))
            if match1 and match2:
                return 0.9  # Strong signal if both match same pattern category
        
        return 0.0
    
    def calculate_name_similarity_strict(name1, name2):
        """Stricter name similarity to reduce false positives"""
        n1 = name1.lower().strip()
        n2 = name2.lower().strip()
        
        # Exact match
        if n1 == n2:
            return 1.0
        
        # Very similar (just case/underscore/hyphen differences)
        n1_normalized = n1.replace('_', '').replace('-', '').replace(' ', '')
        n2_normalized = n2.replace('_', '').replace('-', '').replace(' ', '')
        if n1_normalized == n2_normalized:
            return 0.95
        
        # One contains the other (but not too short)
        if len(n1) > 3 and len(n2) > 3:
            if n1 in n2 or n2 in n1:
                return 0.7
        
        # Jaccard similarity (existing logic)
        set1 = set(n1.split('_') + n1.split('-') + n1.split(' '))
        set2 = set(n2.split('_') + n2.split('-') + n2.split(' '))
        set1 = {s for s in set1 if len(s) > 2}  # Filter out very short tokens
        set2 = {s for s in set2 if len(s) > 2}
        
        if not set1 or not set2:
            return 0.0
        
        intersection = len(set1.intersection(set2))
        union = len(set1.union(set2))
        
        return intersection / union if union > 0 else 0.0


    # Import context service for context-aware enhancements
    try:
        from app.services.context_service import ContextService
        context_available = True
    except ImportError:
        context_available = False
        print("Context service not available, proceeding without context")
    
    for col1 in file1_cols:
        # Skip excluded columns from context
        if context_available and context1 and ContextService.should_exclude_column(context1, col1):
            print(f"Skipping excluded column from File 1: {col1}")
            continue
            
        for col2 in file2_cols:
            # Skip excluded columns from context
            if context_available and context2 and ContextService.should_exclude_column(context2, col2):
                continue
                
            try:
                # Check for custom mappings first
                custom_mapping_score = None
                if context_available and context1 and context2:
                    custom_mapping_score = ContextService.get_custom_mapping(context1, context2, col1, col2)
                
                # If custom mapping exists, use it with high confidence
                if custom_mapping_score is not None:
                    similarities.append({
                        "file1_column": col1,
                        "file2_column": col2,
                        "similarity": custom_mapping_score,
                        "confidence": 95.0,  # High confidence for user-defined mappings
                        "type": "custom_mapping",
                        "data_similarity": 0.0,
                        "name_similarity": 1.0,
                        "distribution_similarity": 0.0,
                        "json_confidence": 0.0,
                        "llm_semantic_score": 1.0,
                        "reason": "User-defined custom mapping"
                    })
                    continue
                
                # 1. Calculate Basic Similarity (Name + Data)
                sim_data = calculate_column_similarity(df1[col1], df2[col2])
                
                # 2. Use stricter name similarity
                name_sim = calculate_name_similarity_strict(col1, col2)
                
                # 3. Add Pattern Bonus
                pattern_score = get_pattern_score(col1, col2)
                
                # 4. Calculate Final Weighted Confidence with stricter rules
                # Weights: 35% Data, 45% Name, 20% Pattern
                # Increased name weight to prioritize column name matching
                data_weight = 0.35
                name_weight = 0.45
                pattern_weight = 0.20
                
                final_confidence = (
                    (sim_data['similarity'] * data_weight) + 
                    (name_sim * name_weight) + 
                    (pattern_score * pattern_weight)
                ) * 100
                
                # Apply context-aware confidence enhancement
                if context_available and context1 and context2:
                    final_confidence = ContextService.enhance_confidence_with_context(
                        final_confidence, context1, context2, col1, col2
                    )
                
                # STRICTER THRESHOLD: Only keep matches with reasonable confidence
                # Also require either good name similarity OR pattern match
                min_confidence = 20.0
                min_name_or_pattern = 0.3
                
                if final_confidence > min_confidence and (name_sim > min_name_or_pattern or pattern_score > min_name_or_pattern):
                    # Get column descriptions from context if available
                    col1_desc = ""
                    col2_desc = ""
                    if context_available and context1:
                        col1_desc = ContextService.get_column_context(context1, col1) or ""
                    if context_available and context2:
                        col2_desc = ContextService.get_column_context(context2, col2) or ""
                    
                    reason = f"Name({name_sim:.2f}), Data({sim_data['similarity']:.2f}), Pattern({pattern_score:.2f})"
                    if col1_desc or col2_desc:
                        reason += f" | Context: {col1_desc[:30]} <-> {col2_desc[:30]}"
                    
                    similarities.append({
                        "file1_column": col1,
                        "file2_column": col2,
                        "similarity": sim_data['similarity'],
                        "confidence": final_confidence,
                        "type": sim_data['type'],
                        "data_similarity": sim_data['similarity'],
                        "name_similarity": name_sim,
                        "distribution_similarity": sim_data.get("distribution_similarity", 0.0),
                        "json_confidence": pattern_score,
                        "llm_semantic_score": 0.0,
                        "reason": reason,
                        "context_enhanced": context_available and (context1 is not None or context2 is not None)
                    })
            except Exception as e:
                # Skip columns that cause errors
                continue

    
    # Apply feedback learning adjustments
    try:
        from app.services.feedback_learning import feedback_system
        from app.services.adaptive_learning import adaptive_learner
        from app.services.pattern_learning import pattern_learner
        from app.services.confidence_calibration import confidence_calibrator
        from app.services.active_learning import active_learner
        
        # Get learned weights (use adaptive weights if available)
        learned_weights = adaptive_learner.get_weights()
        print(f"Using learned weights: {learned_weights}")
        
        for sim in similarities:
            # 1. Apply feedback boost/penalty (existing logic)
            boost = feedback_system.get_learned_boost(sim['file1_column'], sim['file2_column'])
            
            # 2. Add learned pattern scores
            positive_pattern_score = pattern_learner.get_positive_pattern_score(
                sim['file1_column'], 
                sim['file2_column']
            )
            negative_pattern_penalty = pattern_learner.get_negative_pattern_penalty(
                sim['file1_column'],
                sim['file2_column'],
                sim['name_similarity'],
                sim['data_similarity']
            )
            
            # 3. Recalculate confidence using learned weights
            raw_confidence = (
                sim['name_similarity'] * learned_weights['name'] +
                sim['data_similarity'] * learned_weights['data'] +
                sim['json_confidence'] * learned_weights['pattern']
            ) * 100
            
            # 4. Apply learned pattern adjustments
            raw_confidence += (positive_pattern_score * 10)  # Boost up to 10%
            raw_confidence -= (negative_pattern_penalty * 100)  # Penalty up to 50%
            
            # 5. Apply feedback boost
            raw_confidence = max(0, min(100, raw_confidence + (boost * 100)))
            
            # 6. Apply confidence calibration
            calibrated_confidence = confidence_calibrator.calibrate(raw_confidence)
            
            # Update similarity with ML-enhanced confidence
            sim['confidence'] = calibrated_confidence
            sim['ml_enhanced'] = True
            sim['learned_weights_used'] = learned_weights
            
            if boost != 0:
                sim['feedback_adjusted'] = True
                sim['feedback_boost'] = boost
            
            if positive_pattern_score > 0:
                sim['positive_pattern_boost'] = positive_pattern_score
            
            if negative_pattern_penalty > 0:
                sim['negative_pattern_penalty'] = negative_pattern_penalty
        
        # 7. Calculate uncertainty scores for active learning
        for sim in similarities:
            sim['uncertainty_score'] = active_learner.calculate_uncertainty(sim)
        
        print(f"ML enhancements applied to {len(similarities)} similarities")
        
    except Exception as e:
        print(f"Could not apply ML enhancements: {e}")
        import traceback
        traceback.print_exc()
                
    # Sort by confidence
    similarities.sort(key=lambda x: x["confidence"], reverse=True)
    
    # Always show top 10 matches in the flow diagram, regardless of confidence
    top_n = min(10, len(similarities))
    for i in range(top_n):
        sim = similarities[i]
        edges.append({
            "source": f"file1_{sim['file1_column']}",
            "target": f"file2_{sim['file2_column']}",
            "value": sim['confidence'],
            "label": f"{int(sim['confidence'])}%"
        })
    
    print(f"Created {len(edges)} edges from top {top_n} similarities")
    
    return {
        "nodes": nodes,
        "edges": edges,
        "similarities": similarities[:top_n], # Return top 10 for the table as well
        "total_relationships": len(similarities)
    }
