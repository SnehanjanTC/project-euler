"""
Pattern Learning System
Learns domain-specific patterns from confirmed matches and anti-patterns from rejections
"""

import json
import re
from pathlib import Path
from typing import Dict, List, Set
from datetime import datetime
from collections import Counter

PATTERN_FILE = Path("data/learned_patterns.json")
PATTERN_FILE.parent.mkdir(exist_ok=True)

class PatternLearner:
    """
    Learns column matching patterns from user feedback
    - Positive patterns: Common substrings in confirmed matches
    - Negative patterns: Features that lead to false positives
    """
    
    def __init__(self):
        self.data = self._load_data()
        self.positive_patterns = self.data.get('positive_patterns', {})
        self.negative_patterns = self.data.get('negative_patterns', [])
    
    def _load_data(self) -> Dict:
        """Load learned patterns from file"""
        if PATTERN_FILE.exists():
            try:
                with open(PATTERN_FILE, 'r') as f:
                    return json.load(f)
            except Exception as e:
                print(f"Error loading patterns: {e}")
                return {'positive_patterns': {}, 'negative_patterns': []}
        return {'positive_patterns': {}, 'negative_patterns': []}
    
    def _save_data(self):
        """Persist learned patterns"""
        try:
            data = {
                'positive_patterns': self.positive_patterns,
                'negative_patterns': self.negative_patterns,
                'last_updated': datetime.now().isoformat()
            }
            with open(PATTERN_FILE, 'w') as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            print(f"Error saving patterns: {e}")
    
    def _extract_common_tokens(self, col1: str, col2: str) -> Set[str]:
        """Extract common tokens between two column names"""
        # Normalize and tokenize
        tokens1 = set(re.split(r'[_\-\s]+', col1.lower()))
        tokens2 = set(re.split(r'[_\-\s]+', col2.lower()))
        
        # Find common tokens (length > 2 to avoid noise)
        common = {t for t in tokens1 & tokens2 if len(t) > 2}
        return common
    
    def _extract_anti_pattern_features(self, col1: str, col2: str, 
                                      name_sim: float, data_sim: float) -> Dict:
        """Extract features that characterize a false positive"""
        return {
            'col1_tokens': set(re.split(r'[_\-\s]+', col1.lower())),
            'col2_tokens': set(re.split(r'[_\-\s]+', col2.lower())),
            'name_sim_range': f"{int(name_sim*10)*10}-{int(name_sim*10)*10+10}",
            'data_sim_range': f"{int(data_sim*10)*10}-{int(data_sim*10)*10+10}",
            'length_ratio': len(col1) / len(col2) if len(col2) > 0 else 1.0,
            'has_common_prefix': col1[:3].lower() == col2[:3].lower() if len(col1) >= 3 and len(col2) >= 3 else False
        }
    
    def learn_from_positive(self, col1: str, col2: str):
        """Learn patterns from a confirmed match"""
        common_tokens = self._extract_common_tokens(col1, col2)
        
        for token in common_tokens:
            # Increment frequency count
            self.positive_patterns[token] = self.positive_patterns.get(token, 0) + 1
        
        self._save_data()
    
    def learn_from_negative(self, col1: str, col2: str, name_sim: float, data_sim: float):
        """Learn anti-patterns from a rejected match"""
        features = self._extract_anti_pattern_features(col1, col2, name_sim, data_sim)
        
        # Store anti-pattern
        self.negative_patterns.append({
            'features': {k: list(v) if isinstance(v, set) else v for k, v in features.items()},
            'timestamp': datetime.now().isoformat()
        })
        
        # Keep only recent anti-patterns (last 100)
        self.negative_patterns = self.negative_patterns[-100:]
        
        self._save_data()
    
    def get_positive_pattern_score(self, col1: str, col2: str) -> float:
        """
        Calculate score based on learned positive patterns
        Returns 0.0 to 1.0
        """
        common_tokens = self._extract_common_tokens(col1, col2)
        
        if not common_tokens:
            return 0.0
        
        # Calculate weighted score based on pattern frequency
        total_score = 0.0
        max_frequency = max(self.positive_patterns.values()) if self.positive_patterns else 1
        
        for token in common_tokens:
            if token in self.positive_patterns:
                # Normalize by max frequency
                frequency = self.positive_patterns[token]
                total_score += (frequency / max_frequency)
        
        # Average and cap at 1.0
        avg_score = total_score / len(common_tokens) if common_tokens else 0.0
        return min(1.0, avg_score)
    
    def get_negative_pattern_penalty(self, col1: str, col2: str, 
                                    name_sim: float, data_sim: float) -> float:
        """
        Calculate penalty based on learned anti-patterns
        Returns 0.0 (no penalty) to 0.5 (strong penalty)
        """
        if not self.negative_patterns:
            return 0.0
        
        current_features = self._extract_anti_pattern_features(col1, col2, name_sim, data_sim)
        
        # Count how many anti-patterns match
        match_count = 0
        for anti in self.negative_patterns:
            anti_features = anti['features']
            
            # Check feature similarity
            matches = 0
            total_checks = 0
            
            # Check token overlap
            if 'col1_tokens' in anti_features and 'col2_tokens' in anti_features:
                anti_tokens1 = set(anti_features['col1_tokens'])
                anti_tokens2 = set(anti_features['col2_tokens'])
                curr_tokens1 = current_features['col1_tokens']
                curr_tokens2 = current_features['col2_tokens']
                
                overlap1 = len(anti_tokens1 & curr_tokens1) / len(anti_tokens1) if anti_tokens1 else 0
                overlap2 = len(anti_tokens2 & curr_tokens2) / len(anti_tokens2) if anti_tokens2 else 0
                
                if overlap1 > 0.5 and overlap2 > 0.5:
                    matches += 1
                total_checks += 1
            
            # Check similarity ranges
            if anti_features.get('name_sim_range') == current_features['name_sim_range']:
                matches += 1
            total_checks += 1
            
            # If significant match, increment count
            if total_checks > 0 and matches / total_checks > 0.5:
                match_count += 1
        
        # Calculate penalty (more matches = higher penalty)
        if match_count > 0:
            penalty = min(0.5, match_count * 0.1)
            return penalty
        
        return 0.0
    
    def get_stats(self) -> Dict:
        """Get pattern learning statistics"""
        return {
            'positive_patterns_count': len(self.positive_patterns),
            'negative_patterns_count': len(self.negative_patterns),
            'top_positive_patterns': sorted(
                self.positive_patterns.items(), 
                key=lambda x: x[1], 
                reverse=True
            )[:10]
        }

# Global instance
pattern_learner = PatternLearner()
