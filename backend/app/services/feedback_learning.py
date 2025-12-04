"""
Feedback Learning System for Column Matching
Allows users to provide feedback on column matches and improves matching over time
"""

import json
import os
from datetime import datetime
from typing import Dict, List, Optional
from pathlib import Path

# Store feedback in a JSON file
FEEDBACK_FILE = Path("data/matching_feedback.json")
FEEDBACK_FILE.parent.mkdir(exist_ok=True)

class FeedbackLearningSystem:
    def __init__(self):
        self.feedback_data = self._load_feedback()
    
    def _load_feedback(self) -> Dict:
        """Load existing feedback from file"""
        if FEEDBACK_FILE.exists():
            try:
                with open(FEEDBACK_FILE, 'r') as f:
                    return json.load(f)
            except Exception as e:
                print(f"Error loading feedback: {e}")
                return {"matches": [], "corrections": {}}
        return {"matches": [], "corrections": {}}
    
    def _save_feedback(self):
        """Save feedback to file"""
        try:
            with open(FEEDBACK_FILE, 'w') as f:
                json.dump(self.feedback_data, f, indent=2)
        except Exception as e:
            print(f"Error saving feedback: {e}")
    
    def add_feedback(self, file1_col: str, file2_col: str, is_correct: bool, 
                    correct_match: Optional[str] = None, user_note: Optional[str] = None,
                    name_similarity: float = 0.0, data_similarity: float = 0.0, 
                    pattern_score: float = 0.0, confidence: float = 0.0):
        """
        Add user feedback about a column match and trigger ML learning
        
        Args:
            file1_col: Column name from file 1
            file2_col: Column name from file 2
            is_correct: Whether the match is correct
            correct_match: If incorrect, what the correct match should be
            user_note: Optional note from user
            name_similarity: Name similarity score (for ML training)
            data_similarity: Data similarity score (for ML training)
            pattern_score: Pattern match score (for ML training)
            confidence: Overall confidence score (for calibration)
        """
        feedback_entry = {
            "file1_column": file1_col,
            "file2_column": file2_col,
            "is_correct": is_correct,
            "correct_match": correct_match,
            "user_note": user_note,
            "timestamp": datetime.now().isoformat(),
            # ML training features
            "name_similarity": name_similarity,
            "data_similarity": data_similarity,
            "pattern_score": pattern_score,
            "confidence": confidence
        }
        
        self.feedback_data["matches"].append(feedback_entry)
        
        # Store corrections for learning
        if not is_correct and correct_match:
            key = f"{file1_col}|{file2_col}"
            self.feedback_data["corrections"][key] = {
                "suggested": file2_col,
                "correct": correct_match,
                "count": self.feedback_data["corrections"].get(key, {}).get("count", 0) + 1
            }
        
        self._save_feedback()
        
        # Trigger ML learning systems
        self._trigger_ml_learning(feedback_entry)
        
        return feedback_entry
    
    def _trigger_ml_learning(self, feedback: Dict):
        """Trigger all ML learning systems with new feedback"""
        try:
            # Import ML modules
            from app.services.adaptive_learning import adaptive_learner
            from app.services.confidence_calibration import confidence_calibrator
            from app.services.pattern_learning import pattern_learner
            
            # 1. Update confidence calibration
            confidence_calibrator.update(
                predicted_confidence=feedback.get('confidence', 0),
                actual_correct=feedback['is_correct']
            )
            
            # 2. Update pattern learning
            if feedback['is_correct']:
                pattern_learner.learn_from_positive(
                    feedback['file1_column'],
                    feedback['file2_column']
                )
            else:
                pattern_learner.learn_from_negative(
                    feedback['file1_column'],
                    feedback['file2_column'],
                    feedback.get('name_similarity', 0),
                    feedback.get('data_similarity', 0)
                )
            
            # 3. Update adaptive weights (batch update every 10 feedbacks)
            recent_feedback = self.feedback_data["matches"][-10:]
            if len(recent_feedback) >= 10:
                adaptive_learner.update_weights(recent_feedback)
            
            print(f"ML learning triggered for feedback on {feedback['file1_column']} ↔ {feedback['file2_column']}")
            
        except Exception as e:
            print(f"Error in ML learning: {e}")
            # Don't fail the feedback submission if ML fails
    
    def get_learned_boost(self, file1_col: str, file2_col: str) -> float:
        """
        Get a confidence boost/penalty based on historical feedback
        
        Returns:
            Float between -0.3 and +0.3 to adjust confidence
        """
        # Check if this exact match has feedback
        for match in self.feedback_data["matches"]:
            if match["file1_column"] == file1_col and match["file2_column"] == file2_col:
                if match["is_correct"]:
                    return 0.2  # Boost confidence by 20%
                else:
                    return -0.3  # Penalize by 30%
        
        # Check if this column pair has been corrected before
        key = f"{file1_col}|{file2_col}"
        if key in self.feedback_data["corrections"]:
            return -0.25  # Penalize known incorrect matches
        
        # Check for similar column name patterns that were corrected
        for correction_key, correction in self.feedback_data["corrections"].items():
            suggested_col = correction["suggested"]
            # If the current file2_col was previously suggested but wrong, penalize it
            if file2_col == suggested_col:
                return -0.15
        
        return 0.0  # No adjustment
    
    def get_suggested_match(self, file1_col: str) -> Optional[str]:
        """Get the learned correct match for a column based on feedback"""
        for match in self.feedback_data["matches"]:
            if match["file1_column"] == file1_col and match["is_correct"]:
                return match["file2_column"]
        
        # Check corrections
        for key, correction in self.feedback_data["corrections"].items():
            if key.startswith(f"{file1_col}|"):
                return correction["correct"]
        
        return None
    
    def get_feedback_stats(self) -> Dict:
        """Get statistics about feedback"""
        total_feedback = len(self.feedback_data["matches"])
        correct_matches = sum(1 for m in self.feedback_data["matches"] if m["is_correct"])
        incorrect_matches = total_feedback - correct_matches
        
        return {
            "total_feedback": total_feedback,
            "correct_matches": correct_matches,
            "incorrect_matches": incorrect_matches,
            "accuracy": (correct_matches / total_feedback * 100) if total_feedback > 0 else 0,
            "total_corrections": len(self.feedback_data["corrections"])
        }

# Global instance
feedback_system = FeedbackLearningSystem()
