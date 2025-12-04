"""
Adaptive Weight Learning System
Learns optimal weights for name, data, and pattern similarity based on user feedback
"""

import json
import numpy as np
from pathlib import Path
from typing import Dict, List, Tuple
from datetime import datetime

WEIGHTS_FILE = Path("data/adaptive_weights.json")
WEIGHTS_FILE.parent.mkdir(exist_ok=True)

class AdaptiveWeightLearner:
    """
    Uses gradient descent to learn optimal weights from feedback
    Objective: Maximize confidence for correct matches, minimize for incorrect ones
    """
    
    def __init__(self, learning_rate: float = 0.01):
        self.learning_rate = learning_rate
        self.weights = self._load_weights()
        self.training_history = []
        
    def _load_weights(self) -> Dict[str, float]:
        """Load weights from file or use defaults"""
        if WEIGHTS_FILE.exists():
            try:
                with open(WEIGHTS_FILE, 'r') as f:
                    data = json.load(f)
                    return data.get('weights', self._default_weights())
            except Exception as e:
                print(f"Error loading weights: {e}")
                return self._default_weights()
        return self._default_weights()
    
    def _default_weights(self) -> Dict[str, float]:
        """Default weights (current heuristic)"""
        return {
            'name': 0.45,
            'data': 0.35,
            'pattern': 0.20
        }
    
    def _save_weights(self):
        """Persist weights to disk"""
        try:
            data = {
                'weights': self.weights,
                'last_updated': datetime.now().isoformat(),
                'training_history': self.training_history[-100:]  # Keep last 100
            }
            with open(WEIGHTS_FILE, 'w') as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            print(f"Error saving weights: {e}")
    
    def calculate_confidence(self, name_sim: float, data_sim: float, pattern_sim: float) -> float:
        """Calculate confidence using current weights"""
        return (
            name_sim * self.weights['name'] +
            data_sim * self.weights['data'] +
            pattern_sim * self.weights['pattern']
        ) * 100
    
    def update_weights(self, feedback_batch: List[Dict]):
        """
        Update weights using gradient descent on a batch of feedback
        
        Args:
            feedback_batch: List of dicts with keys:
                - name_similarity: float
                - data_similarity: float  
                - pattern_score: float (json_confidence)
                - is_correct: bool
        """
        if not feedback_batch:
            return
        
        # Accumulate gradients
        gradients = {'name': 0.0, 'data': 0.0, 'pattern': 0.0}
        total_loss = 0.0
        
        for feedback in feedback_batch:
            # Get features
            name_sim = feedback.get('name_similarity', 0.0)
            data_sim = feedback.get('data_similarity', 0.0)
            pattern_sim = feedback.get('pattern_score', 0.0)
            
            # Calculate predicted confidence (0-1 scale)
            predicted = self.calculate_confidence(name_sim, data_sim, pattern_sim) / 100.0
            
            # True label (1 for correct, 0 for incorrect)
            actual = 1.0 if feedback.get('is_correct', False) else 0.0
            
            # Binary cross-entropy loss
            # L = -[y*log(p) + (1-y)*log(1-p)]
            epsilon = 1e-7  # Prevent log(0)
            predicted = np.clip(predicted, epsilon, 1 - epsilon)
            loss = -(actual * np.log(predicted) + (1 - actual) * np.log(1 - predicted))
            total_loss += loss
            
            # Gradient of loss w.r.t. predicted
            # dL/dp = -y/p + (1-y)/(1-p)
            grad_pred = -actual / predicted + (1 - actual) / (1 - predicted)
            
            # Chain rule: dL/dw = dL/dp * dp/dw
            # dp/dw_name = name_sim (and similarly for others)
            gradients['name'] += grad_pred * name_sim
            gradients['data'] += grad_pred * data_sim
            gradients['pattern'] += grad_pred * pattern_sim
        
        # Average gradients
        n = len(feedback_batch)
        for key in gradients:
            gradients[key] /= n
        
        # Update weights using gradient descent
        for key in self.weights:
            self.weights[key] -= self.learning_rate * gradients[key]
        
        # Normalize weights to sum to 1.0
        total = sum(self.weights.values())
        if total > 0:
            self.weights = {k: v / total for k, v in self.weights.items()}
        
        # Record training history
        avg_loss = total_loss / n
        self.training_history.append({
            'timestamp': datetime.now().isoformat(),
            'loss': avg_loss,
            'weights': self.weights.copy(),
            'batch_size': n
        })
        
        # Save updated weights
        self._save_weights()
        
        print(f"Weights updated: {self.weights}")
        print(f"Average loss: {avg_loss:.4f}")
    
    def get_weights(self) -> Dict[str, float]:
        """Get current weights"""
        return self.weights.copy()
    
    def get_training_stats(self) -> Dict:
        """Get training statistics"""
        if not self.training_history:
            return {
                'total_updates': 0,
                'current_weights': self.weights,
                'avg_loss': None
            }
        
        recent_losses = [h['loss'] for h in self.training_history[-10:]]
        return {
            'total_updates': len(self.training_history),
            'current_weights': self.weights,
            'avg_loss': np.mean(recent_losses),
            'loss_trend': 'improving' if len(recent_losses) > 1 and recent_losses[-1] < recent_losses[0] else 'stable'
        }

# Global instance
adaptive_learner = AdaptiveWeightLearner()
