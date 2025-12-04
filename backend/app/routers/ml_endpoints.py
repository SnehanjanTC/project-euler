# Add these new ML stats endpoints after the existing feedback endpoints

@router.get("/ml/stats")
async def get_ml_stats():
    """Get comprehensive ML system statistics"""
    try:
        from app.services.adaptive_learning import adaptive_learner
        from app.services.confidence_calibration import confidence_calibrator
        from app.services.pattern_learning import pattern_learner
        from app.services.feedback_learning import feedback_system
        
        return {
            "adaptive_weights": adaptive_learner.get_training_stats(),
            "confidence_calibration": confidence_calibrator.get_stats(),
            "pattern_learning": pattern_learner.get_stats(),
            "feedback_stats": feedback_system.get_feedback_stats()
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error getting ML stats: {str(e)}")

@router.get("/ml/calibration-curve")
async def get_calibration_curve():
    """Get confidence calibration curve for visualization"""
    try:
        from app.services.confidence_calibration import confidence_calibrator
        return confidence_calibrator.get_calibration_curve()
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error getting calibration curve: {str(e)}")

@router.get("/ml/weights")
async def get_current_weights():
    """Get current learned weights"""
    try:
        from app.services.adaptive_learning import adaptive_learner
        return {
            "weights": adaptive_learner.get_weights(),
            "training_stats": adaptive_learner.get_training_stats()
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error getting weights: {str(e)}")
