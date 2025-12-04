import pandas as pd
import json
import re
from app.utils.security import sanitize_input
from app.services.llm import call_ollama

def execute_safe_pandas_code(df: pd.DataFrame, code: str) -> tuple[any, str]:  # type: ignore
    """Execute pandas code safely and return result with enhanced security"""
    # Sanitize code input
    code = sanitize_input(code, max_length=5000)
    
    # Additional security checks
    dangerous_keywords = [
        'import', 'from', '__', 'globals', 'locals', 'vars', 'dir',
        'eval', 'exec', 'compile', 'open', 'file', 'input', 'raw_input',
        'subprocess', 'os.', 'sys.', 'shutil', 'pickle', 'marshal',
        'ctypes', 'socket', 'urllib', 'requests', 'http', 'ftp',
        'sqlite3', 'mysql', 'psycopg2', 'pymongo', 'redis'
    ]
    
    code_lower = code.lower()
    for keyword in dangerous_keywords:
        if keyword in code_lower:
            return None, f"Security: Use of '{keyword}' is not allowed"
    
    # Create a safe execution environment with restricted access
    safe_dict = {
        'df': df.copy(),  # Use a copy to avoid modifying original
        'pd': pd,
        'len': len,
        'sum': sum,
        'max': max,
        'min': min,
        'avg': lambda x: sum(x) / len(x) if len(x) > 0 else 0,
        'mean': lambda x: sum(x) / len(x) if len(x) > 0 else 0,
        'round': round,
        'abs': abs,
        'str': str,
        'int': int,
        'float': float,
        'list': list,
        'dict': dict,
        'tuple': tuple,
        'set': set,
        'sorted': sorted,
        'range': range,
        'enumerate': enumerate,
        'zip': zip,
        '__builtins__': {  # Restricted builtins
            'len': len, 'sum': sum, 'max': max, 'min': min,
            'abs': abs, 'round': round, 'str': str, 'int': int,
            'float': float, 'bool': bool, 'type': type,
            'isinstance': isinstance, 'hasattr': hasattr,
            'getattr': getattr, 'setattr': setattr,
        }
    }
    
    try:
        # Remove markdown code blocks if present
        if "```python" in code:
            code = code.split("```python")[1].split("```")[0].strip()
        elif "```" in code:
            code = code.split("```")[1].split("```")[0].strip()
        
        # Clean up code - remove any leading/trailing whitespace
        code = code.strip()
        
        # Limit code length
        if len(code) > 5000:
            return None, "Code too long. Maximum 5000 characters allowed."
        
        # Handle multi-line code
        if '\n' in code:
            # Execute as a block with restricted globals
            exec(code, {"__builtins__": safe_dict['__builtins__']}, safe_dict)
            result = safe_dict.get('result', None)
        else:
            # Execute as expression
            exec(f"result = {code}", {"__builtins__": safe_dict['__builtins__']}, safe_dict)
            result = safe_dict.get('result', None)
        
        # Convert result to string
        if isinstance(result, pd.DataFrame):
            # Limit DataFrame display size
            if len(result) > 100:
                preview = result.head(100)
                return result, f"{preview.to_string()}\n\n... (showing first 100 of {len(result)} rows)"
            return result, result.to_string()
        elif isinstance(result, pd.Series):
            if len(result) > 100:
                preview = result.head(100)
                return result, f"{preview.to_string()}\n\n... (showing first 100 of {len(result)} values)"
            return result, result.to_string()
        elif isinstance(result, (int, float)):
            return result, str(result)
        elif isinstance(result, (list, dict)):
            return result, json.dumps(result, indent=2, default=str)
        else:
            return result, str(result)
    except Exception as e:
        return None, f"Error executing code: {str(e)}"

def generate_comprehensive_analysis(df: pd.DataFrame) -> dict:
    """Generate a comprehensive AI-based analysis report of the dataset - works with any CSV structure"""
    analysis = {
        "dataset_overview": {},
        "numeric_analysis": {},
        "categorical_analysis": {},
        "datetime_analysis": {},
        "correlations": {},
        "insights": [],
        "recommendations": []
    }
    
    # Dynamically detect column types - works with any CSV structure
    numeric_cols = df.select_dtypes(include=['number']).columns.tolist()
    categorical_cols = df.select_dtypes(include=['object', 'category']).columns.tolist()
    datetime_cols = df.select_dtypes(include=['datetime64']).columns.tolist()
    
    # Handle very large datasets by sampling for analysis
    sample_size = min(10000, len(df))  # Sample up to 10k rows for analysis
    analysis_df = df.sample(n=sample_size, random_state=42) if len(df) > sample_size else df
    
    # Dataset Overview
    analysis["dataset_overview"] = {
        "total_rows": len(df),
        "total_columns": len(df.columns),
        "column_names": df.columns.tolist(),
        "missing_values": df.isnull().sum().to_dict(),
        "duplicate_rows": df.duplicated().sum(),
        "memory_usage_mb": df.memory_usage(deep=True).sum() / 1024**2
    }
    
    # Numeric Analysis - handles any number of numeric columns
    if len(numeric_cols) > 0:
        for col in numeric_cols:
            try:
                col_data = analysis_df[col].dropna()  # Use sampled data and drop NaN
                if len(col_data) == 0:
                    continue
                    
                stats = col_data.describe()
                
                # Detect outliers using IQR method (only if we have enough data)
                outliers_count = 0
                outliers_percentage = 0
                if len(col_data) > 4:  # Need at least 4 values for quartiles
                    try:
                        Q1 = stats['25%']
                        Q3 = stats['75%']
                        IQR = Q3 - Q1
                        if IQR > 0:  # Avoid division by zero
                            lower_bound = Q1 - 1.5 * IQR
                            upper_bound = Q3 + 1.5 * IQR
                            outliers = col_data[(col_data < lower_bound) | (col_data > upper_bound)]
                            outliers_count = len(outliers)
                            outliers_percentage = len(outliers) / len(col_data) * 100
                    except Exception:
                        pass  # Skip outlier detection if it fails
                
                analysis["numeric_analysis"][col] = {
                    "mean": float(stats['mean']) if 'mean' in stats else None,
                    "median": float(stats['50%']) if '50%' in stats else None,
                    "std": float(stats['std']) if 'std' in stats else None,
                    "min": float(stats['min']) if 'min' in stats else None,
                    "max": float(stats['max']) if 'max' in stats else None,
                    "q25": float(stats['25%']) if '25%' in stats else None,
                    "q75": float(stats['75%']) if '75%' in stats else None,
                    "skewness": float(col_data.skew()) if len(col_data) > 2 else None,
                    "kurtosis": float(col_data.kurtosis()) if len(col_data) > 2 else None,
                    "outliers_count": outliers_count,
                    "outliers_percentage": outliers_percentage,
                    "null_count": int(df[col].isnull().sum()),
                    "null_percentage": float(df[col].isnull().sum() / len(df) * 100)
                }
                
                # Generate insights
                if len(col_data) > 2:
                    skew_val = col_data.skew()
                    if not pd.isna(skew_val) and abs(skew_val) > 1:
                        analysis["insights"].append(f"{col} shows {'positive' if skew_val > 0 else 'negative'} skewness (skew={skew_val:.2f}), indicating a non-normal distribution.")
                    
                    if outliers_count > len(col_data) * 0.05:  # More than 5% outliers
                        analysis["insights"].append(f"{col} has {outliers_count} potential outliers ({outliers_percentage:.1f}% of data).")
            except Exception as e:
                # Skip columns that cause errors
                analysis["insights"].append(f"Could not analyze {col}: {str(e)}")
                continue
    
    # Categorical Analysis - handles any categorical columns
    if len(categorical_cols) > 0:
        for col in categorical_cols:
            try:
                col_data = analysis_df[col].dropna()
                if len(col_data) == 0:
                    continue
                    
                value_counts = col_data.value_counts()
                unique_count = col_data.nunique()
                
                analysis["categorical_analysis"][col] = {
                    "unique_count": unique_count,
                    "most_common": value_counts.head(10).to_dict(),  # Show top 10
                    "least_common": value_counts.tail(5).to_dict() if len(value_counts) > 5 else {},
                    "null_count": int(df[col].isnull().sum()),
                    "null_percentage": float(df[col].isnull().sum() / len(df) * 100),
                    "max_length": int(col_data.astype(str).str.len().max()) if len(col_data) > 0 else 0,
                    "min_length": int(col_data.astype(str).str.len().min()) if len(col_data) > 0 else 0
                }
                
                # Generate insights
                if unique_count == len(df):
                    analysis["insights"].append(f"{col} appears to be a unique identifier (all values are unique).")
                elif unique_count < 10 and unique_count > 1:
                    analysis["insights"].append(f"{col} has {unique_count} categories, suggesting it could be used for grouping/segmentation.")
                elif unique_count == 1:
                    analysis["insights"].append(f"{col} has only one unique value - may not be useful for analysis.")
            except Exception as e:
                analysis["insights"].append(f"Could not analyze categorical column {col}: {str(e)}")
                continue
    
    # DateTime Analysis - handles date/time columns
    if len(datetime_cols) > 0:
        for col in datetime_cols:
            try:
                col_data = analysis_df[col].dropna()
                if len(col_data) == 0:
                    continue
                    
                analysis["datetime_analysis"][col] = {
                    "earliest": str(col_data.min()),
                    "latest": str(col_data.max()),
                    "span_days": (col_data.max() - col_data.min()).days,
                    "null_count": int(df[col].isnull().sum()),
                    "null_percentage": float(df[col].isnull().sum() / len(df) * 100)
                }
                
                analysis["insights"].append(f"{col} spans {(col_data.max() - col_data.min()).days} days from {col_data.min()} to {col_data.max()}.")
            except Exception as e:
                analysis["insights"].append(f"Could not analyze datetime column {col}: {str(e)}")
                continue
    
    # Correlations - only calculate if we have multiple numeric columns
    if len(numeric_cols) > 1:
        try:
            # Use sampled data for correlation to handle large datasets
            corr_matrix = analysis_df[numeric_cols].corr()
            analysis["correlations"] = corr_matrix.to_dict()
            
            # Find strong correlations (limit to avoid too many insights)
            strong_corrs = []
            for i, col1 in enumerate(numeric_cols):
                for col2 in numeric_cols[i+1:]:
                    try:
                        corr_value = corr_matrix.loc[col1, col2]
                        if not pd.isna(corr_value) and abs(corr_value) > 0.7:
                            strong_corrs.append((col1, col2, corr_value))
                    except Exception:
                        continue
            
            # Add top correlations as insights
            for col1, col2, corr_value in strong_corrs[:5]:  # Limit to top 5
                analysis["insights"].append(f"Strong {'positive' if corr_value > 0 else 'negative'} correlation ({corr_value:.2f}) between {col1} and {col2}.")
        except Exception as e:
            analysis["insights"].append(f"Could not calculate correlations: {str(e)}")
    
    # Generate recommendations based on dataset characteristics
    if len(numeric_cols) > 0 and len(categorical_cols) > 0:
        analysis["recommendations"].append("Consider analyzing relationships between numeric and categorical variables using groupby operations.")
    
    if df.isnull().sum().sum() > 0:
        missing_pct = (df.isnull().sum().sum() / (len(df) * len(df.columns))) * 100
        analysis["recommendations"].append(f"Dataset contains {df.isnull().sum().sum()} missing values ({missing_pct:.1f}%). Consider data cleaning or imputation strategies.")
    
    if len(df) > 10000:
        analysis["recommendations"].append(f"Large dataset detected ({len(df):,} rows). Analysis was performed on a sample of {sample_size:,} rows for performance.")
    
    if len(df.columns) > 50:
        analysis["recommendations"].append(f"Dataset has many columns ({len(df.columns)}). Consider focusing on specific columns for deeper analysis.")
    
    if len(datetime_cols) > 0:
        analysis["recommendations"].append("Date/time columns detected. Consider time-series analysis or temporal grouping.")
    
    if df.duplicated().sum() > 0:
        dup_pct = (df.duplicated().sum() / len(df)) * 100
        analysis["recommendations"].append(f"Dataset contains {df.duplicated().sum()} duplicate rows ({dup_pct:.1f}%). Consider removing duplicates if not needed.")
    
    return analysis

def find_matching_column(columns: list, col_name: str) -> str | None:
    """Find matching column name with fuzzy matching"""
    col_name_lower = col_name.lower().strip()
    # Exact match
    for col in columns:
        if col.lower() == col_name_lower:
            return col
    # Partial match
    for col in columns:
        if col_name_lower in col.lower() or col.lower() in col_name_lower:
            return col
        if col.lower().replace('_', ' ') == col_name_lower or col.lower().replace('-', ' ') == col_name_lower:
            return col
    return None

def smart_column_matching(question: str, columns: list) -> dict:
    """Intelligently match columns from natural language question"""
    question_lower = question.lower()
    matches = {}
    
    # Direct column name matching
    for col in columns:
        col_lower = col.lower()
        # Exact match
        if col_lower in question_lower or question_lower in col_lower:
            matches[col] = "exact"
        # Partial match (word boundaries)
        elif any(word == col_lower for word in question_lower.split()):
            matches[col] = "word"
        # Fuzzy match (contains)
        elif col_lower.replace('_', ' ') in question_lower or col_lower.replace('-', ' ') in question_lower:
            matches[col] = "fuzzy"
    
    # Semantic matching (common synonyms)
    semantic_map = {
        'salary': ['wage', 'pay', 'income', 'compensation', 'earnings'],
        'age': ['years', 'old', 'birth'],
        'name': ['person', 'employee', 'user', 'customer'],
        'city': ['location', 'place', 'address'],
        'department': ['division', 'team', 'group', 'unit'],
        'date': ['time', 'when', 'created', 'updated'],
        'count': ['number', 'total', 'quantity', 'amount'],
        'average': ['mean', 'avg', 'typical'],
        'maximum': ['max', 'highest', 'top', 'peak'],
        'minimum': ['min', 'lowest', 'bottom']
    }
    
    for col in columns:
        col_lower = col.lower()
        for key, synonyms in semantic_map.items():
            if key in col_lower:
                for synonym in synonyms:
                    if synonym in question_lower and col not in matches:
                        matches[col] = "semantic"
                        break
    
    return matches

def analyze_query_without_llm(df: pd.DataFrame, question: str) -> dict:
    """Enhanced intelligent analysis method with smart pattern matching and context awareness"""
    # ... (This function is very long, I'll copy the core logic but for brevity in this tool call I'll truncate it slightly, assuming the user has the original file to copy from if needed. 
    # Actually, I should copy it fully to ensure it works. I will paste the full content.)
    
    question_lower = question.lower()
    columns = df.columns.tolist()
    numeric_cols = df.select_dtypes(include=['number']).columns.tolist()
    categorical_cols = df.select_dtypes(include=['object', 'category']).columns.tolist()
    
    # Smart column matching
    column_matches = smart_column_matching(question, columns)
    matched_columns = list(column_matches.keys())
    
    # Pattern 1: Comprehensive dataset overview/analysis
    if any(word in question_lower for word in ['overview', 'summary', 'analyze', 'analysis', 'insights', 'describe', 'comprehensive', 'full analysis']):
        comprehensive_analysis = generate_comprehensive_analysis(df)
        analysis_parts = []
        
        # Dataset Overview
        overview = comprehensive_analysis["dataset_overview"]
        analysis_parts.append(f"📊 DATASET OVERVIEW:")
        analysis_parts.append(f"• Total Rows: {overview['total_rows']}")
        analysis_parts.append(f"• Total Columns: {overview['total_columns']}")
        analysis_parts.append(f"• Column Names: {', '.join(overview['column_names'])}")
        analysis_parts.append(f"• Duplicate Rows: {overview['duplicate_rows']}")
        analysis_parts.append(f"• Memory Usage: {overview['memory_usage_mb']:.2f} MB")
        
        # Missing Values
        missing = overview['missing_values']
        if sum(missing.values()) > 0:
            analysis_parts.append(f"\n⚠️ MISSING VALUES:")
            for col, count in missing.items():
                if count > 0:
                    analysis_parts.append(f"  • {col}: {count} ({count/overview['total_rows']*100:.1f}%)")
        else:
            analysis_parts.append(f"\n✅ No missing values found")
        
        # Numeric Analysis
        if comprehensive_analysis["numeric_analysis"]:
            analysis_parts.append(f"\n📈 NUMERIC COLUMNS ANALYSIS:")
            for col, stats in comprehensive_analysis["numeric_analysis"].items():
                analysis_parts.append(f"\n  {col}:")
                analysis_parts.append(f"    Mean: {stats['mean']:.2f} | Median: {stats['median']:.2f} | Std: {stats['std']:.2f}")
                analysis_parts.append(f"    Range: [{stats['min']:.2f}, {stats['max']:.2f}]")
                analysis_parts.append(f"    IQR: [{stats['q25']:.2f}, {stats['q75']:.2f}]")
                analysis_parts.append(f"    Skewness: {stats['skewness']:.2f} | Kurtosis: {stats['kurtosis']:.2f}")
                if stats['outliers_count'] > 0:
                    analysis_parts.append(f"    ⚠️ Outliers: {stats['outliers_count']} ({stats['outliers_percentage']:.1f}%)")
        
        # Categorical Analysis
        if comprehensive_analysis["categorical_analysis"]:
            analysis_parts.append(f"\n📝 CATEGORICAL COLUMNS ANALYSIS:")
            for col, stats in comprehensive_analysis["categorical_analysis"].items():
                analysis_parts.append(f"\n  {col}:")
                analysis_parts.append(f"    Unique Values: {stats['unique_count']}")
                analysis_parts.append(f"    Most Common: {', '.join([f'{k}({v})' for k, v in list(stats['most_common'].items())[:3]])}")
                if stats['null_count'] > 0:
                    analysis_parts.append(f"    Missing: {stats['null_count']}")
        
        # Correlations
        if comprehensive_analysis["correlations"]:
            analysis_parts.append(f"\n🔗 CORRELATION ANALYSIS:")
            corr_data = comprehensive_analysis["correlations"]
            if len(numeric_cols) > 1:
                # Show top correlations
                strong_corrs = []
                for col1 in numeric_cols:
                    for col2 in numeric_cols:
                        if col1 != col2 and col1 in corr_data and col2 in corr_data[col1]:
                            corr_val = corr_data[col1][col2]
                            if abs(corr_val) > 0.5:
                                strong_corrs.append((col1, col2, corr_val))
                
                if strong_corrs:
                    analysis_parts.append("  Strong Correlations:")
                    for col1, col2, val in strong_corrs[:5]:  # Show top 5
                        analysis_parts.append(f"    {col1} ↔ {col2}: {val:.2f}")
        
        # Insights
        if comprehensive_analysis["insights"]:
            analysis_parts.append(f"\n💡 KEY INSIGHTS:")
            for insight in comprehensive_analysis["insights"][:10]:  # Limit to 10 insights
                analysis_parts.append(f"  • {insight}")
        
        # Recommendations
        if comprehensive_analysis["recommendations"]:
            analysis_parts.append(f"\n💼 RECOMMENDATIONS:")
            for rec in comprehensive_analysis["recommendations"]:
                analysis_parts.append(f"  • {rec}")
        
        return {
            "explanation": "\n".join(analysis_parts),
            "result": df.describe(include='all').to_string() if len(df) > 0 else "No data",
            "result_data": comprehensive_analysis,
            "result_type": "dataframe"
        }
    
    # Enhanced Pattern 2: Average/Mean/Median/Mode with smart matching
    avg_match = re.search(r'(?:average|mean|avg|median|mode|typical)\s+(?:of\s+)?(.+)', question_lower)
    if avg_match or any(word in question_lower for word in ['average', 'mean', 'avg', 'median', 'typical']):
        # Try to find column from smart matches first
        matching_col = None
        if matched_columns:
            # Prefer numeric columns
            for col in matched_columns:
                if col in numeric_cols:
                    matching_col = col
                    break
            # If no numeric match, try any match
            if not matching_col and matched_columns:
                matching_col = matched_columns[0]
        
        # Fallback to regex extraction
        if not matching_col and avg_match:
            col_name = avg_match.group(1).strip().strip('?').strip()
            matching_col = find_matching_column(columns, col_name)
        
        # If still no match, try to infer from context
        if not matching_col and numeric_cols:
            # Look for numeric column mentions in question
            for col in numeric_cols:
                if col.lower() in question_lower:
                    matching_col = col
                    break
            # Default to first numeric column if question is generic
            if not matching_col and ('average' in question_lower or 'mean' in question_lower):
                matching_col = numeric_cols[0]
        
        if matching_col and matching_col in numeric_cols:
            if 'median' in question_lower:
                result = df[matching_col].median()
                stat_name = "median"
            elif 'mode' in question_lower:
                mode_result = df[matching_col].mode()
                result = mode_result.iloc[0] if len(mode_result) > 0 else None
                stat_name = "mode"
            else:
                result = df[matching_col].mean()
                stat_name = "average"
            
            if result is not None:
                return {
                    "explanation": f"The {stat_name} {matching_col} is {result:.2f}" if isinstance(result, (int, float)) else f"The {stat_name} {matching_col} is {result}",
                    "result": str(result),
                    "result_data": float(result) if isinstance(result, (int, float)) else result,
                    "result_type": "scalar"
                }
    
    # Enhanced Pattern 3: Min/Max with smart matching
    minmax_match = re.search(r'(?:minimum|maximum|min|max|lowest|highest|top|bottom)\s+(?:of\s+)?(.+)', question_lower)
    if minmax_match or any(word in question_lower for word in ['min', 'max', 'lowest', 'highest', 'top', 'bottom']):
        matching_col = None
        # Use smart matches first
        if matched_columns:
            for col in matched_columns:
                if col in numeric_cols:
                    matching_col = col
                    break
        
        # Fallback to regex
        if not matching_col and minmax_match:
            col_name = minmax_match.group(1).strip().strip('?').strip()
            matching_col = find_matching_column(columns, col_name)
        
        # Context inference
        if not matching_col and numeric_cols:
            for col in numeric_cols:
                if col.lower() in question_lower:
                    matching_col = col
                    break
            if not matching_col:
                matching_col = numeric_cols[0]
        
        if matching_col and matching_col in numeric_cols:
            if any(word in question_lower for word in ['max', 'maximum', 'highest']):
                result_val = df[matching_col].max()
                result_row = df.loc[df[matching_col].idxmax()]
                stat_name = "maximum"
            else:
                result_val = df[matching_col].min()
                result_row = df.loc[df[matching_col].idxmin()]
                stat_name = "minimum"
            
            return {
                "explanation": f"The {stat_name} {matching_col} is {result_val:.2f}. Full record:\n{result_row.to_dict()}",
                "result": f"{stat_name.capitalize()}: {result_val}\n\nFull Record:\n{result_row.to_string()}",
                "result_data": {"value": float(result_val), "record": result_row.to_dict()},
                "result_type": "other"
            }
    
    # Pattern 4: Count/How many
    count_match = re.search(r'(?:count|how many|number of|total)', question_lower)
    if count_match and 'column' not in question_lower:
        result = len(df)
        return {
            "explanation": f"There are {result} rows in the dataset.",
            "result": str(result),
            "result_data": result,
            "result_type": "scalar"
        }
    
    # Pattern 5: Top N / Bottom N / Highest / Lowest
    top_match = re.search(r'(?:top|highest|largest|first|show)\s+(\d+)', question_lower)
    bottom_match = re.search(r'(?:bottom|lowest|smallest|last)\s+(\d+)', question_lower)
    
    if top_match or bottom_match:
        n = int(top_match.group(1)) if top_match else (int(bottom_match.group(1)) if bottom_match else 5)
        
        # Try to find column to sort by
        for col in numeric_cols:
            if col in question_lower:
                if top_match or 'highest' in question_lower or 'largest' in question_lower:
                    result_df = df.nlargest(n, col)
                else:
                    result_df = df.nsmallest(n, col)
                return {
                    "explanation": f"Here are the top {n} rows sorted by {col}:",
                    "result": result_df.to_string(),
                    "result_data": result_df.to_dict('records'),
                    "result_type": "dataframe"
                }
        
        # No specific column, just return first/last N
        if top_match or 'first' in question_lower:
            result_df = df.head(n)
        else:
            result_df = df.tail(n)
        return {
            "explanation": f"Here are the {'first' if top_match else 'last'} {n} rows:",
            "result": result_df.to_string(),
            "result_data": result_df.to_dict('records'),
            "result_type": "dataframe"
        }
    
    # Enhanced Pattern 6: Statistics/Summary with smart matching
    stats_match = re.search(r'(?:statistics|stats|summary|describe|info|details)\s+(?:of|for|about)\s+(.+)', question_lower)
    if stats_match or any(word in question_lower for word in ['statistics', 'stats', 'summary', 'describe']):
        matching_col = None
        # Use smart matches first
        if matched_columns:
            matching_col = matched_columns[0]
        
        # Fallback to regex
        if not matching_col and stats_match:
            col_name = stats_match.group(1).strip().strip('?').strip()
            matching_col = find_matching_column(columns, col_name)
        
        # Context inference
        if not matching_col:
            for col in columns:
                if col.lower() in question_lower:
                    matching_col = col
                    break
        
        if matching_col:
            if matching_col in numeric_cols:
                stats = df[matching_col].describe()
                return {
                    "explanation": f"Detailed statistics for {matching_col}:\n{stats.to_string()}",
                    "result": stats.to_string(),
                    "result_data": stats.to_dict(),
                    "result_type": "series"
                }
            else:
                # Categorical statistics
                value_counts = df[matching_col].value_counts()
                return {
                    "explanation": f"Value distribution for {matching_col}:\n{value_counts.to_string()}",
                    "result": value_counts.to_string(),
                    "result_data": value_counts.to_dict(),
                    "result_type": "series"
                }
    
    # Enhanced Pattern 7: Group by / Count by / Distribution with smart matching
    group_match = re.search(r'(?:how many|count|distribution|frequency)\s+(?:are|is|in|by|of)\s+(.+)', question_lower)
    if group_match or any(word in question_lower for word in ['how many', 'count', 'distribution', 'frequency']):
        matching_col = None
        # Use smart matches - prefer categorical columns
        if matched_columns:
            for col in matched_columns:
                if col in categorical_cols:
                    matching_col = col
                    break
            if not matching_col:
                matching_col = matched_columns[0]
        
        # Fallback to regex
        if not matching_col and group_match:
            col_name = group_match.group(1).strip().strip('?').strip()
            matching_col = find_matching_column(columns, col_name)
        
        # Context inference
        if not matching_col and categorical_cols:
            for col in categorical_cols:
                if col.lower() in question_lower:
                    matching_col = col
                    break
        
        if matching_col:
            counts = df[matching_col].value_counts()
            return {
                "explanation": f"Distribution of {matching_col}:\n{counts.to_string()}\n\nTotal unique values: {df[matching_col].nunique()}",
                "result": counts.to_string(),
                "result_data": counts.to_dict(),
                "result_type": "series"
            }
    
    # Enhanced Pattern 8: Filter/Where/Find with smart matching
    filter_match = re.search(r'(?:find|show|filter|where|which|list|get)\s+(?:rows|records|people|items|all|data)', question_lower)
    if filter_match or any(word in question_lower for word in ['find', 'show', 'filter', 'where', 'which']):
        # Use smart column matching for filters
        filter_cols = matched_columns if matched_columns else columns
        
        # Try to extract conditions with smart matching
        for col in filter_cols:
            if col.lower() in question_lower or any(word in col.lower() for word in question_lower.split() if len(word) > 3):
                # Look for comparison operators
                if '>' in question_lower or 'greater' in question_lower or 'above' in question_lower:
                    # Extract number
                    num_match = re.search(r'(\d+)', question_lower)
                    if num_match:
                        threshold = float(num_match.group(1))
                        if col in numeric_cols:
                            result_df = df[df[col] > threshold]
                            return {
                                "explanation": f"Found {len(result_df)} rows where {col} > {threshold}:",
                                "result": result_df.to_string() if len(result_df) <= 50 else f"{result_df.head(50).to_string()}\n... ({len(result_df) - 50} more rows)",
                                "result_data": result_df.to_dict('records') if len(result_df) <= 100 else result_df.head(100).to_dict('records'),
                                "result_type": "dataframe"
                            }
                elif '<' in question_lower or 'less' in question_lower or 'below' in question_lower:
                    num_match = re.search(r'(\d+)', question_lower)
                    if num_match:
                        threshold = float(num_match.group(1))
                        if col in numeric_cols:
                            result_df = df[df[col] < threshold]
                            return {
                                "explanation": f"Found {len(result_df)} rows where {col} < {threshold}:",
                                "result": result_df.to_string() if len(result_df) <= 50 else f"{result_df.head(50).to_string()}\n... ({len(result_df) - 50} more rows)",
                                "result_data": result_df.to_dict('records') if len(result_df) <= 100 else result_df.head(100).to_dict('records'),
                                "result_type": "dataframe"
                            }
                elif '=' in question_lower or 'equal' in question_lower or 'is' in question_lower:
                    # Try to find value
                    if col in categorical_cols:
                        # Look for quoted strings or common values
                        for val in df[col].unique()[:10]:  # Check first 10 unique values
                            if str(val).lower() in question_lower:
                                result_df = df[df[col] == val]
                                return {
                                    "explanation": f"Found {len(result_df)} rows where {col} = '{val}':",
                                    "result": result_df.to_string() if len(result_df) <= 50 else f"{result_df.head(50).to_string()}\n... ({len(result_df) - 50} more rows)",
                                    "result_data": result_df.to_dict('records') if len(result_df) <= 100 else result_df.head(100).to_dict('records'),
                                    "result_type": "dataframe"
                                }
    
    # Pattern 9: Correlation between columns
    corr_match = re.search(r'(?:correlation|relationship|correlate|related)\s+(?:between|of)', question_lower)
    if corr_match and len(numeric_cols) >= 2:
        corr_matrix = df[numeric_cols].corr()
        return {
            "explanation": f"Correlation matrix between numeric columns:\n{corr_matrix.to_string()}\n\nValues range from -1 (negative correlation) to +1 (positive correlation).",
            "result": corr_matrix.to_string(),
            "result_data": corr_matrix.to_dict(),
            "result_type": "dataframe"
        }
    
    # Pattern 10: Aggregate by group
    groupby_match = re.search(r'(?:group|aggregate|sum|total)\s+(?:by|of)', question_lower)
    if groupby_match:
        for cat_col in categorical_cols:
            if cat_col in question_lower:
                for num_col in numeric_cols:
                    if num_col in question_lower or 'sum' in question_lower or 'total' in question_lower:
                        grouped = df.groupby(cat_col)[num_col].agg(['sum', 'mean', 'count'])
                        return {
                            "explanation": f"Aggregated {num_col} by {cat_col}:\n{grouped.to_string()}",
                            "result": grouped.to_string(),
                            "result_data": grouped.to_dict(),
                            "result_type": "dataframe"
                        }
    
    # Pattern 11: Unique values
    unique_match = re.search(r'(?:unique|distinct|different)\s+(?:values|items)', question_lower)
    if unique_match:
        for col in columns:
            if col in question_lower:
                unique_vals = df[col].unique()
                return {
                    "explanation": f"Unique values in {col} ({len(unique_vals)} total):\n{', '.join([str(v) for v in unique_vals[:50]])}{'...' if len(unique_vals) > 50 else ''}",
                    "result": f"Total unique: {len(unique_vals)}\nValues: {', '.join([str(v) for v in unique_vals[:50]])}",
                    "result_data": {"count": len(unique_vals), "values": unique_vals.tolist()[:100]},
                    "result_type": "other"
                }
    
    # Default: Comprehensive dataset summary
    summary = []
    summary.append(f"📊 Dataset Summary:")
    summary.append(f"• Shape: {df.shape[0]} rows × {df.shape[1]} columns")
    summary.append(f"• Columns: {', '.join(columns)}")
    
    if len(numeric_cols) > 0:
        summary.append(f"\n📈 Numeric Columns Summary:")
        for col in numeric_cols:
            summary.append(f"  {col}: mean={df[col].mean():.2f}, min={df[col].min():.2f}, max={df[col].max():.2f}")
    
    if len(categorical_cols) > 0:
        summary.append(f"\n📝 Categorical Columns Summary:")
        for col in categorical_cols:
            summary.append(f"  {col}: {df[col].nunique()} unique values")
    
    return {
        "explanation": "\n".join(summary) + "\n\n💡 Try asking:\n• 'What is the average of [column]?'\n• 'Show me the top 10 rows'\n• 'Statistics for [column]'\n• 'How many are in [category]?'\n• 'Find rows where [column] > [value]'\n• 'Correlation between columns'\n• 'Group by [column] and sum [numeric column]'",
        "result": df.head(10).to_string(),
        "result_data": df.head(10).to_dict('records'),
        "result_type": "dataframe"
    }

def analyze_data_with_llm(df: pd.DataFrame, question: str) -> dict:
    """Use Ollama to analyze the dataframe and answer the question, with fallback to rule-based analysis"""
    
    # Get dataframe info
    df_info = {
        "shape": df.shape,
        "columns": df.columns.tolist(),
        "dtypes": df.dtypes.astype(str).to_dict(),
        "sample_data": df.head(10).to_dict('records'),
    }
    
    # Get summary statistics for numeric columns
    numeric_cols = df.select_dtypes(include=['number']).columns
    if len(numeric_cols) > 0:
        df_info["summary_stats"] = df[numeric_cols].describe().to_dict()
    else:
        df_info["summary_stats"] = {}
    
    # Get null counts
    null_counts = df.isnull().sum().to_dict()
    df_info["null_counts"] = {k: int(v) for k, v in null_counts.items()}
    
    # Create a comprehensive prompt for the LLM
    prompt = f"""You are an expert data analyst and AI assistant specializing in comprehensive data analysis. You have access to a pandas DataFrame called 'df' with the following information:

📊 DATASET INFORMATION:
- Shape: {df_info['shape'][0]} rows, {df_info['shape'][1]} columns
- Columns: {', '.join(df_info['columns'])}
- Column Types: {json.dumps(df_info['dtypes'], indent=2)}

📋 SAMPLE DATA (first 10 rows):
{json.dumps(df_info['sample_data'], indent=2, default=str)}

📈 SUMMARY STATISTICS (for numeric columns):
{json.dumps(df_info['summary_stats'], indent=2, default=str)}

⚠️ MISSING VALUES:
{json.dumps(df_info['null_counts'], indent=2)}

❓ USER QUESTION: {question}

🎯 YOUR TASK:
Analyze the user's question and provide comprehensive insights. You can perform:
1. Statistical Analysis: mean, median, mode, std dev, min, max, percentiles
2. Data Exploration: filtering, sorting, grouping, aggregations
3. Pattern Detection: correlations, distributions, trends, outliers
4. Comparative Analysis: comparisons between groups, columns, or categories
5. Data Summaries: overviews, summaries, key insights
6. Advanced Queries: complex filters, multi-column analysis, conditional aggregations

📝 RESPONSE FORMAT:
You MUST format your response EXACTLY as:
CODE: <pandas code expression>
EXPLANATION: <detailed, conversational explanation with insights>

🔧 PANDAS OPERATIONS AVAILABLE:
- Basic: df['col'], df[['col1', 'col2']], len(df), df.shape
- Statistics: df['col'].mean(), .median(), .mode(), .std(), .var(), .describe()
- Aggregations: df.groupby('col').agg({{'col2': ['mean', 'sum', 'count']}})
- Filtering: df[df['col'] > value], df[df['col'] == 'value'], df.query('col > value')
- Sorting: df.sort_values('col', ascending=False), df.nlargest(n, 'col'), df.nsmallest(n, 'col')
- Grouping: df.groupby('col')['col2'].sum(), df.groupby('col').size()
- Correlations: df[['col1', 'col2']].corr()
- Value counts: df['col'].value_counts(), df['col'].nunique()
- String operations: df['col'].str.contains('text'), df['col'].str.upper()
- Date operations: pd.to_datetime(df['col']), df['col'].dt.year

💡 IMPORTANT GUIDELINES:
- Always use safe pandas operations that won't cause errors
- For DataFrames: return result.to_dict('records') for JSON serialization
- For Series: return result.to_dict() or result.to_string()
- For scalars: return the value directly
- Include context and insights in your explanation, not just the result
- If the question asks for analysis, provide multiple insights and observations
- Detect patterns, trends, and interesting findings in the data

📚 EXAMPLE RESPONSES:

Example 1 - Simple Query:
Question: "What is the average salary?"
CODE: df['salary'].mean()
EXPLANATION: The average salary across all {df_info['shape'][0]} records is calculated by taking the mean of the salary column. This gives us a central tendency measure that represents the typical salary in the dataset.

Example 2 - Complex Analysis:
Question: "Analyze the relationship between age and salary"
CODE: df[['age', 'salary']].corr().iloc[0, 1]
EXPLANATION: I've calculated the correlation coefficient between age and salary. A positive value indicates that as age increases, salary tends to increase. A value close to 1 or -1 indicates a strong relationship, while values near 0 suggest little to no linear relationship. Additionally, I can see that the average salary by age group shows [provide insights based on the data].

Example 3 - Grouped Analysis:
Question: "Show me salary statistics by city"
CODE: df.groupby('city')['salary'].agg(['mean', 'median', 'count', 'min', 'max']).to_dict('index')
EXPLANATION: I've analyzed salary distributions across different cities. This reveals which cities have the highest and lowest average salaries, the number of employees in each city, and the salary ranges. Key insights include [provide specific findings from the data].

Example 4 - Comprehensive Overview:
Question: "Give me a comprehensive analysis of this dataset"
CODE: {{'summary': df.describe(include='all').to_dict(), 'correlations': df.select_dtypes(include=['number']).corr().to_dict(), 'value_counts': {{col: df[col].value_counts().to_dict() for col in df.select_dtypes(include=['object']).columns}}}}
EXPLANATION: Here's a comprehensive analysis of your dataset: [Provide detailed insights about distributions, patterns, correlations, key findings, outliers, data quality, etc.]

Now analyze the user's question and provide a comprehensive response:

Answer:"""

    # Constants for parsing LLM response
    CODE_TAG = "CODE:"
    EXPLANATION_TAG = "EXPLANATION:"
    
    # Try to get response from Ollama
    llm_response = call_ollama(prompt)
    
    # If Ollama is not available, use fallback analysis
    if not llm_response or len(llm_response.strip()) == 0:
        print("Ollama not available, using fallback analysis")
        return analyze_query_without_llm(df, question)
    
    result_data = None
    result_type = None
    
    # Try to extract and execute pandas code
    try:
        # Extract code from response if CODE: tag is present
        if CODE_TAG in llm_response:
            parts = llm_response.split(CODE_TAG)
            if len(parts) > 1:
                code_explanation = parts[1].split(EXPLANATION_TAG)
                code_part = code_explanation[0].strip()
                explanation_part = code_explanation[1].strip() if len(code_explanation) > 1 else ""
                
                # Execute the code
                result, result_str = execute_safe_pandas_code(df, code_part)
                
                if result is not None:
                    # Determine result type and format accordingly
                    if isinstance(result, pd.DataFrame):
                        result_data = result.to_dict('records')
                        result_type = "dataframe"
                        result_str = result.to_string()
                    elif isinstance(result, pd.Series):
                        result_data = result.to_dict()
                        result_type = "series"
                        result_str = result.to_string()
                    elif isinstance(result, (int, float)):
                        result_data = result
                        result_type = "scalar"
                    else:
                        result_data = str(result)
                        result_type = "other"
                    
                    return {
                        "explanation": explanation_part,
                        "result": result_str,
                        "result_data": result_data,
                        "result_type": result_type,
                        "raw_response": llm_response
                    }
                else:
                    # Code execution failed, try fallback
                    print(f"Code execution failed: {result_str}, trying fallback")
                    return analyze_query_without_llm(df, question)
        
        # If no CODE: tag found, try to find pandas expressions
        import re
        pandas_pattern = r"df\[['\"]([^'\"]+)['\"]\](?:\.\w+\([^)]*\))?"
        matches = re.findall(pandas_pattern, llm_response)
        
        if matches and matches[0] in df.columns:
            col_name = matches[0]
            simple_code = f"df['{col_name}']"
            result, result_str = execute_safe_pandas_code(df, simple_code)
            if result is not None:
                return {
                    "explanation": llm_response,
                    "result": result_str[:500],
                    "result_data": result.to_dict() if isinstance(result, pd.Series) else str(result),
                    "result_type": "series" if isinstance(result, pd.Series) else "other",
                    "raw_response": llm_response
                }
        
        # If LLM response doesn't contain executable code, use fallback
        print("LLM response doesn't contain executable code, using fallback")
        return analyze_query_without_llm(df, question)
        
    except Exception as e:
        # If anything fails, use fallback analysis
        print(f"Error processing LLM response: {str(e)}, using fallback")
        return analyze_query_without_llm(df, question)
