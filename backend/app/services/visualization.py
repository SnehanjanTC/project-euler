import pandas as pd
import json
from app.services.llm import call_ollama

def smart_visualization_selection(df: pd.DataFrame, question: str) -> dict:
    """Intelligently select the best visualization based on data and question"""
    
    # Analyze columns mentioned in question
    columns = df.columns.tolist()
    mentioned_cols = [col for col in columns if col.lower() in question.lower()]
    
    # If no columns mentioned, try to infer from data types
    if not mentioned_cols:
        numeric_cols = df.select_dtypes(include=['number']).columns.tolist()
        categorical_cols = df.select_dtypes(include=['object', 'category']).columns.tolist()
        datetime_cols = df.select_dtypes(include=['datetime64']).columns.tolist()
    else:
        numeric_cols = [col for col in mentioned_cols if pd.api.types.is_numeric_dtype(df[col])]
        categorical_cols = [col for col in mentioned_cols if pd.api.types.is_string_dtype(df[col]) or pd.api.types.is_categorical_dtype(df[col])]
        datetime_cols = [col for col in mentioned_cols if pd.api.types.is_datetime64_any_dtype(df[col])]
    
    # Determine visualization type
    viz_type = "bar"  # Default
    x_axis = None
    y_axis = None
    title = "Data Visualization"
    
    # Time series logic
    if datetime_cols and numeric_cols:
        viz_type = "line"
        x_axis = datetime_cols[0]
        y_axis = numeric_cols[0]
        title = f"{y_axis} over Time"
    
    # Comparison logic
    elif categorical_cols and numeric_cols:
        if len(df[categorical_cols[0]].unique()) > 10:
            viz_type = "bar"  # Horizontal bar for many categories
        else:
            viz_type = "bar"
        x_axis = categorical_cols[0]
        y_axis = numeric_cols[0]
        title = f"{y_axis} by {x_axis}"
    
    # Correlation logic
    elif len(numeric_cols) >= 2:
        viz_type = "scatter"
        x_axis = numeric_cols[0]
        y_axis = numeric_cols[1]
        title = f"{y_axis} vs {x_axis}"
    
    # Distribution logic
    elif len(numeric_cols) == 1 and not categorical_cols:
        viz_type = "histogram"
        x_axis = numeric_cols[0]
        title = f"Distribution of {x_axis}"
    
    # Composition logic
    elif len(categorical_cols) == 1 and not numeric_cols:
        viz_type = "pie"
        x_axis = categorical_cols[0]
        title = f"Composition of {x_axis}"
    
    # Ask LLM for better suggestion if available
    try:
        prompt = f"""
        Suggest the best chart type for this data and question.
        Question: {question}
        Columns: {columns}
        Data Types: {df.dtypes.to_dict()}
        
        Return JSON: {{ "type": "bar|line|scatter|pie|area", "x": "column_name", "y": "column_name", "title": "Chart Title" }}
        """
        response = call_ollama(prompt)
        import re
        json_match = re.search(r'\{.*\}', response, re.DOTALL)
        if json_match:
            suggestion = json.loads(json_match.group(0))
            if suggestion.get("type") and suggestion.get("x"):
                viz_type = suggestion["type"]
                x_axis = suggestion["x"]
                y_axis = suggestion.get("y")
                title = suggestion.get("title", title)
    except Exception:
        pass
        
    return {
        "type": viz_type,
        "x_axis": x_axis,
        "y_axis": y_axis,
        "title": title
    }
