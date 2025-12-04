"""
Correlation analysis service - calculates numeric correlations between matched columns
"""
import pandas as pd
import numpy as np
from typing import List, Dict


def calculate_correlations(df1: pd.DataFrame, df2: pd.DataFrame, similarities: List[Dict]) -> List[Dict]:
    """
    Calculate numeric correlations for matched columns
    
    Args:
        df1: First dataframe
        df2: Second dataframe
        similarities: List of similarity matches from similarity service
        
    Returns:
        List of correlation results with Pearson and Spearman coefficients
    """
    correlations = []
    
    for sim in similarities:
        col1 = sim['file1_column']
        col2 = sim['file2_column']
        
        # Only calculate for numeric columns
        if col1 in df1.columns and col2 in df2.columns:
            series1 = df1[col1]
            series2 = df2[col2]
            
            # Check if both are numeric
            if pd.api.types.is_numeric_dtype(series1) and pd.api.types.is_numeric_dtype(series2):
                # Align lengths if different
                min_len = min(len(series1), len(series2))
                s1 = series1.iloc[:min_len]
                s2 = series2.iloc[:min_len]
                
                # Drop NaN values pairwise
                mask = ~(s1.isna() | s2.isna())
                s1_clean = s1[mask]
                s2_clean = s2[mask]
                
                if len(s1_clean) > 2:  # Need at least 3 points for correlation
                    try:
                        pearson = s1_clean.corr(s2_clean, method='pearson')
                        spearman = s1_clean.corr(s2_clean, method='spearman')
                        
                        correlations.append({
                            "file1_column": col1,
                            "file2_column": col2,
                            "pearson_correlation": float(pearson) if not pd.isna(pearson) else 0.0,
                            "spearman_correlation": float(spearman) if not pd.isna(spearman) else 0.0,
                            "sample_size": len(s1_clean),
                            "strength": classify_correlation_strength(abs(pearson) if not pd.isna(pearson) else 0),
                            "similarity_confidence": sim['confidence']
                        })
                    except Exception as e:
                        print(f"Error calculating correlation for {col1} - {col2}: {str(e)}")
                        continue
    
    # Sort by absolute Pearson correlation
    correlations.sort(key=lambda x: abs(x['pearson_correlation']), reverse=True)
    
    return correlations


def classify_correlation_strength(abs_corr: float) -> str:
    """Classify correlation strength based on absolute value"""
    if abs_corr >= 0.7:
        return "Strong"
    elif abs_corr >= 0.4:
        return "Moderate"
    elif abs_corr >= 0.2:
        return "Weak"
    else:
        return "Very Weak"
