"""
Confidence Calibration System
Adjusts confidence scores based on historical accuracy
"""

import json
import numpy as np
from pathlib import Path
from typing import Dict
from datetime import datetime

CALIBRATION_FILE = Path("data/confidence_calibration.json")
CALIBRATION_FILE.parent.mkdir(exist_ok=True)

class ConfidenceCalibrator:
    """
    Calibrates confidence scores to match actual accuracy
    Uses binning approach: tracks accuracy for each confidence range
    """
    
    def __init__(self, bin_size: int = 10):
        self.bin_size = bin_size
        self.bins = self._load_bins()
    
    def _load_bins(self) -> Dict:
        """Load calibration data from file"""
        if CALIBRATION_FILE.exists():
            try:
                with open(CALIBRATION_FILE, 'r') as f:
                    return json.load(f)
            except Exception as e:
                print(f"Error loading calibration data: {e}")
                return self._initialize_bins()
        return self._initialize_bins()
    
    def _initialize_bins(self) -> Dict:
        """Initialize empty bins"""
        bins = {}
        for i in range(0, 100, self.bin_size):
            bins[str(i)] = {
                'correct': 0,
                'total': 0,
                'accuracy': 0.0
            }
        return bins
    
    def _save_bins(self):
        """Persist calibration data"""
        try:
            data = {
                **self.bins,
                'last_updated': datetime.now().isoformat()
            }
            with open(CALIBRATION_FILE, 'w') as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            print(f"Error saving calibration data: {e}")
    
    def _get_bin_key(self, confidence: float) -> str:
        """Get bin key for a confidence score"""
        bin_idx = int(confidence // self.bin_size) * self.bin_size
        bin_idx = max(0, min(90, bin_idx))  # Clamp to [0, 90]
        return str(bin_idx)
    
    def update(self, predicted_confidence: float, actual_correct: bool):
        """
        Update calibration data with new feedback
        
        Args:
            predicted_confidence: Original confidence score (0-100)
            actual_correct: Whether the match was actually correct
        """
        bin_key = self._get_bin_key(predicted_confidence)
        
        self.bins[bin_key]['total'] += 1
        if actual_correct:
            self.bins[bin_key]['correct'] += 1
        
        # Update accuracy
        if self.bins[bin_key]['total'] > 0:
            self.bins[bin_key]['accuracy'] = (
                self.bins[bin_key]['correct'] / self.bins[bin_key]['total']
            )
        
        self._save_bins()
    
    def calibrate(self, raw_confidence: float, min_samples: int = 5) -> float:
        """
        Calibrate a confidence score based on historical accuracy
        
        Args:
            raw_confidence: Original confidence score (0-100)
            min_samples: Minimum samples required for calibration
        
        Returns:
            Calibrated confidence score (0-100)
        """
        bin_key = self._get_bin_key(raw_confidence)
        bin_data = self.bins[bin_key]
        
        # If not enough samples, return raw confidence
        if bin_data['total'] < min_samples:
            return raw_confidence
        
        # Blend raw confidence with actual accuracy
        # More weight on actual accuracy as we get more samples
        blend_factor = min(0.7, bin_data['total'] / 50.0)  # Max 70% weight on accuracy
        
        actual_accuracy = bin_data['accuracy'] * 100  # Convert to 0-100 scale
        calibrated = (
            (1 - blend_factor) * raw_confidence +
            blend_factor * actual_accuracy
        )
        
        return calibrated
    
    def get_calibration_curve(self) -> Dict:
        """Get calibration curve data for visualization"""
        curve = []
        for bin_key in sorted(self.bins.keys(), key=int):
            if bin_key == 'last_updated':
                continue
            bin_data = self.bins[bin_key]
            if bin_data['total'] > 0:
                curve.append({
                    'predicted_range': f"{bin_key}-{int(bin_key)+self.bin_size}%",
                    'actual_accuracy': bin_data['accuracy'] * 100,
                    'sample_count': bin_data['total']
                })
        return {'calibration_curve': curve}
    
    def get_stats(self) -> Dict:
        """Get calibration statistics"""
        total_samples = sum(b['total'] for k, b in self.bins.items() if k != 'last_updated')
        
        # Calculate mean absolute calibration error
        errors = []
        for bin_key, bin_data in self.bins.items():
            if bin_key == 'last_updated' or bin_data['total'] == 0:
                continue
            predicted = int(bin_key) + self.bin_size / 2  # Mid-point of bin
            actual = bin_data['accuracy'] * 100
            errors.append(abs(predicted - actual))
        
        return {
            'total_samples': total_samples,
            'mean_calibration_error': np.mean(errors) if errors else 0.0,
            'calibrated_bins': len([b for k, b in self.bins.items() if k != 'last_updated' and b['total'] > 0])
        }

# Global instance
confidence_calibrator = ConfidenceCalibrator()
