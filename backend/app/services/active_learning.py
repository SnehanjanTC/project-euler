"""
Active Learning System
Identifies matches that would benefit most from user feedback
"""

import numpy as np
from typing import List, Dict

class ActiveLearner:
    """
    Calculates uncertainty scores to prioritize feedback requests
    High uncertainty = high value feedback
    """
    
    @staticmethod
    def calculate_uncertainty(match: Dict) -> float:
        """
        Calculate uncertainty score for a match
        Higher score = more valuable feedback
        
        Returns: 0.0 to 1.0
        """
        confidence = match.get('confidence', 0) / 100.0  # Normalize to 0-1
        name_sim = match.get('name_similarity', 0)
        data_sim = match.get('data_similarity', 0)
        pattern_sim = match.get('json_confidence', 0)
        
        # 1. Confidence near decision boundary (40-60%) = high uncertainty
        boundary_distance = abs(confidence - 0.5)
        boundary_uncertainty = 1.0 - (boundary_distance * 2)  # Max at 0.5, min at 0 or 1
        boundary_uncertainty = max(0, boundary_uncertainty)
        
        # 2. Signal conflict = high uncertainty
        # If name says yes but data says no (or vice versa)
        signals = [name_sim, data_sim, pattern_sim]
        signal_variance = np.var(signals) if len(signals) > 1 else 0
        signal_uncertainty = min(1.0, signal_variance * 3)  # Scale variance
        
        # 3. Novel pattern = high uncertainty
        # If pattern score is low (not matching known patterns)
        novelty_uncertainty = 1.0 - pattern_sim
        
        # Combine uncertainties (weighted average)
        total_uncertainty = (
            0.5 * boundary_uncertainty +
            0.3 * signal_uncertainty +
            0.2 * novelty_uncertainty
        )
        
        return total_uncertainty
    
    @staticmethod
    def prioritize_for_feedback(matches: List[Dict], top_k: int = 5) -> List[Dict]:
        """
        Prioritize matches for feedback based on uncertainty
        
        Args:
            matches: List of match dictionaries
            top_k: Number of top uncertain matches to return
        
        Returns:
            List of matches sorted by uncertainty (highest first)
        """
        # Calculate uncertainty for each match
        for match in matches:
            match['uncertainty_score'] = ActiveLearner.calculate_uncertainty(match)
        
        # Sort by uncertainty (descending)
        sorted_matches = sorted(
            matches,
            key=lambda x: x.get('uncertainty_score', 0),
            reverse=True
        )
        
        return sorted_matches[:top_k]
    
    @staticmethod
    def should_request_feedback(match: Dict, threshold: float = 0.6) -> bool:
        """
        Determine if feedback should be requested for this match
        
        Args:
            match: Match dictionary
            threshold: Uncertainty threshold (0-1)
        
        Returns:
            True if feedback would be valuable
        """
        uncertainty = ActiveLearner.calculate_uncertainty(match)
        return uncertainty >= threshold

# Global instance
active_learner = ActiveLearner()
