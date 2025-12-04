from fastapi import APIRouter, UploadFile, File, HTTPException, Query, BackgroundTasks
from fastapi.responses import JSONResponse
import pandas as pd
import os
import shutil
import io
import json
from typing import Optional, List, Dict, Any

from app.config import UPLOAD_DIR, MAX_FILE_SIZE, MAX_ROWS_FOR_ANALYSIS
from app.state import current_df, current_df2, csv_file_path, csv_file_path2, uploaded_files
import app.state as state  # To modify global state
from app.models.schemas import QueryRequest, UploadResponse
from app.utils.security import sanitize_input
from app.utils.rate_limit import check_rate_limit
from app.utils.files import cleanup_old_files
from app.services.analysis import analyze_data_with_llm, analyze_query_without_llm, generate_comprehensive_analysis
from app.services.similarity import get_column_similarity_data
from app.services.visualization import smart_visualization_selection

router = APIRouter()

@router.post("/upload", response_model=UploadResponse)
async def upload_csv(
    background_tasks: BackgroundTasks,
    file: UploadFile = File(...), 
    file_index: int = Query(1, ge=1, le=2)
):
    """Upload a CSV file and load it into memory"""
    
    # Check file extension
    if not file.filename.lower().endswith('.csv'):
        raise HTTPException(status_code=400, detail="Only CSV files are allowed")
    
    # Check file size (approximate)
    file.file.seek(0, 2)
    size = file.file.tell()
    file.file.seek(0)
    
    if size > MAX_FILE_SIZE:
        raise HTTPException(status_code=400, detail=f"File too large. Maximum size is {MAX_FILE_SIZE/1024/1024}MB")
    
    # Create upload directory if not exists
    os.makedirs(UPLOAD_DIR, exist_ok=True)
    
    # Sanitize filename
    safe_filename = sanitize_input(file.filename)
    safe_filename = os.path.basename(safe_filename)
    file_path = os.path.join(UPLOAD_DIR, f"file{file_index}_{safe_filename}")
    
    try:
        # Save file to disk
        with open(file_path, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)
        
        # Track file for cleanup
        from datetime import datetime
        uploaded_files[file_path] = datetime.now()
        
        # Schedule cleanup
        background_tasks.add_task(cleanup_old_files)
        
        # Try to read CSV with different encodings
        df = None
        encodings = ['utf-8', 'latin1', 'iso-8859-1', 'cp1252']
        separators = [',', ';', '\t', '|']
        
        for encoding in encodings:
            for sep in separators:
                try:
                    df = pd.read_csv(file_path, encoding=encoding, sep=sep)
                    # Check if it looks like a valid CSV (more than 1 column or reasonable rows)
                    if len(df.columns) > 1 or len(df) > 0:
                        break
                except Exception:
                    continue
            if df is not None:
                break
        
        if df is None:
            # Last resort: try python engine
            try:
                df = pd.read_csv(file_path, engine='python')
            except Exception as e:
                os.remove(file_path)
                raise HTTPException(status_code=400, detail=f"Could not parse CSV file: {str(e)}")
        
        # Clean column names
        df.columns = [str(col).strip() for col in df.columns]
        
        # Convert date columns
        for col in df.columns:
            if df[col].dtype == 'object':
                try:
                    # Try to convert to datetime, but ignore if it fails
                    if len(df) < 10000:  # Only for smaller datasets to avoid performance hit
                        pd.to_datetime(df[col], errors='raise')
                        df[col] = pd.to_datetime(df[col])
                except (ValueError, TypeError):
                    pass
        
        # Update global state
        if file_index == 1:
            state.current_df = df
            state.csv_file_path = file_path
        else:
            state.current_df2 = df
            state.csv_file_path2 = file_path
            
        return {
            "message": f"File '{file.filename}' uploaded successfully",
            "rows": len(df),
            "columns": len(df.columns),
            "column_names": df.columns.tolist()
        }
        
    except Exception as e:
        if os.path.exists(file_path):
            os.remove(file_path)
        raise HTTPException(status_code=500, detail=f"Error processing file: {str(e)}")

@router.post("/query")
async def query_data(request: QueryRequest):
    """Analyze the data based on user question"""
    if state.current_df is None:
        raise HTTPException(status_code=400, detail="No CSV file loaded. Please upload a file first.")
    
    # Rate limiting
    if not check_rate_limit("query"):
        raise HTTPException(status_code=429, detail="Rate limit exceeded. Please try again later.")
        
    try:
        # Check if we should use both files
        if state.current_df2 is not None and ("both" in request.question.lower() or "compare" in request.question.lower() or "file 2" in request.question.lower()):
            # For now, just analyze the first file, but we could extend this to multi-file analysis
            # Or merge them if possible
            pass
            
        # Use LLM for analysis
        result = analyze_data_with_llm(state.current_df, request.question)
        
        # Add column similarity info if available and relevant
        if state.current_df2 is not None:
            # Check if question asks about mapping or comparison
            if any(word in request.question.lower() for word in ['map', 'match', 'compare', 'similar', 'link']):
                similarity_data = get_column_similarity_data(state.current_df, state.current_df2)
                result["column_similarities"] = similarity_data
        
        return result
        
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error analyzing data: {str(e)}")

@router.get("/status")
async def get_status():
    """Get status of loaded data"""
    file1_loaded = state.current_df is not None
    file2_loaded = state.current_df2 is not None
    
    return {
        "file1_loaded": file1_loaded,
        "file2_loaded": file2_loaded,
        "file1": {
            "loaded": file1_loaded,
            "rows": len(state.current_df) if state.current_df is not None else 0,
            "columns": len(state.current_df.columns) if state.current_df is not None else 0,
            "filename": os.path.basename(state.csv_file_path) if state.csv_file_path else None
        },
        "file2": {
            "loaded": file2_loaded,
            "rows": len(state.current_df2) if state.current_df2 is not None else 0,
            "columns": len(state.current_df2.columns) if state.current_df2 is not None else 0,
            "filename": os.path.basename(state.csv_file_path2) if state.csv_file_path2 else None
        }
    }

@router.get("/preview")
async def get_preview(rows: int = 10, file_index: int = 1):
    """Get preview of data"""
    target_df = state.current_df if file_index == 1 else state.current_df2
    
    if target_df is None:
        raise HTTPException(status_code=400, detail=f"File {file_index} not loaded")
    
    return target_df.head(rows).to_dict('records')

@router.get("/analyze")
async def analyze_dataset(file_index: int = 1):
    """Get comprehensive analysis of the dataset"""
    target_df = state.current_df if file_index == 1 else state.current_df2
    
    if target_df is None:
        raise HTTPException(status_code=400, detail=f"File {file_index} not loaded")
        
    try:
        return generate_comprehensive_analysis(target_df)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error analyzing dataset: {str(e)}")

@router.get("/correlation")
async def get_correlation(col1: str, col2: str, file_index: int = 1):
    """Get correlation between two columns"""
    target_df = state.current_df if file_index == 1 else state.current_df2
    
    if target_df is None:
        raise HTTPException(status_code=400, detail=f"File {file_index} not loaded")
        
    if col1 not in target_df.columns or col2 not in target_df.columns:
        raise HTTPException(status_code=404, detail="Column not found")
        
    if not pd.api.types.is_numeric_dtype(target_df[col1]) or not pd.api.types.is_numeric_dtype(target_df[col2]):
        raise HTTPException(status_code=400, detail="Both columns must be numeric")
        
    correlation = target_df[col1].corr(target_df[col2])
    
    return {
        "column1": col1,
        "column2": col2,
        "correlation": float(correlation),
        "interpretation": "Strong positive" if correlation > 0.7 else "Strong negative" if correlation < -0.7 else "Weak/None"
    }

@router.get("/column-similarity")
async def get_column_similarity():
    """Calculate similarity between columns of two loaded files"""
    if state.current_df is None or state.current_df2 is None:
        raise HTTPException(status_code=400, detail="Both files must be loaded to calculate similarity")
    
    try:
        # Get stored context if available
        context1 = state.file1_context
        context2 = state.file2_context
        
        # Calculate similarity with context
        similarity_data = get_column_similarity_data(
            state.current_df, 
            state.current_df2,
            context1=context1,
            context2=context2
        )
        
        # Also calculate numeric correlations for matched columns
        from app.services.correlation import calculate_correlations
        correlations = calculate_correlations(
            state.current_df, 
            state.current_df2, 
            similarity_data['similarities']
        )
        
        # Add correlations to the response
        similarity_data['correlations'] = correlations
        
        # Add context info to response
        similarity_data['context_used'] = {
            'file1_has_context': context1 is not None,
            'file2_has_context': context2 is not None
        }
        
        print(f"Returning {len(similarity_data['similarities'])} similarities and {len(correlations)} correlations")
        return similarity_data
    except Exception as e:
        print(f"Error calculating similarities: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error calculating correlations: {str(e)}")

@router.get("/visualizations")
async def get_visualizations(question: str = ""):
    """Get suggested visualizations"""
    if state.current_df is None:
        raise HTTPException(status_code=400, detail="No CSV file loaded")
        
    try:
        return smart_visualization_selection(state.current_df, question)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error generating visualization: {str(e)}")

@router.get("/column-types")
async def get_column_types(file_index: int = 1):
    """Get data types of columns"""
    target_df = state.current_df if file_index == 1 else state.current_df2
    
    if target_df is None:
        raise HTTPException(status_code=400, detail=f"File {file_index} not loaded")
        
    types = {}
    for col in target_df.columns:
        dtype = str(target_df[col].dtype)
        if 'int' in dtype or 'float' in dtype:
            types[col] = 'numeric'
        elif 'datetime' in dtype:
            types[col] = 'datetime'
        else:
            types[col] = 'categorical'
            
    return types

@router.post("/filter")
async def filter_data(request: dict):
    """Filter data based on conditions"""
    if state.current_df is None:
        raise HTTPException(status_code=400, detail="No CSV file loaded")
        
    try:
        conditions = request.get('conditions', [])
        filtered_df = state.current_df.copy()
        
        for condition in conditions:
            col = condition.get('column')
            op = condition.get('operator')
            val = condition.get('value')
            
            if col not in filtered_df.columns:
                continue
                
            if op == 'equals':
                filtered_df = filtered_df[filtered_df[col] == val]
            elif op == 'contains':
                filtered_df = filtered_df[filtered_df[col].astype(str).str.contains(str(val), case=False, na=False)]
            elif op == 'greater_than':
                filtered_df = filtered_df[filtered_df[col] > float(val)]
            elif op == 'less_than':
                filtered_df = filtered_df[filtered_df[col] < float(val)]
                
        return {
            "rows": len(filtered_df),
            "data": filtered_df.head(100).to_dict('records')
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error filtering data: {str(e)}")

@router.get("/kpis")
async def get_kpis(file_index: int = 1):
    """Get Key Performance Indicators"""
    target_df = state.current_df if file_index == 1 else state.current_df2
    
    if target_df is None:
        raise HTTPException(status_code=400, detail=f"File {file_index} not loaded")
        
    kpis = []
    numeric_cols = target_df.select_dtypes(include=['number']).columns
    
    for col in numeric_cols:
        kpis.append({
            "name": col,
            "value": float(target_df[col].sum()),
            "avg": float(target_df[col].mean()),
            "type": "sum"
        })
    return kpis

@router.post("/chat-insights")
async def get_chat_insights(request: QueryRequest):
    """Get intelligent insights about column matching when both CSVs are loaded"""
    if state.current_df is None:
        raise HTTPException(status_code=400, detail="No data loaded. Please upload CSV files first.")
    
    # If both files are loaded, provide column matching insights
    if state.current_df2 is not None:
        try:
            # Get column similarity data
            similarity_data = get_column_similarity_data(state.current_df, state.current_df2)
            
            # Get top matches
            top_matches = similarity_data.get('similarities', [])[:10]
            
            # Build intelligent response
            response_parts = []
            response_parts.append("## 🔍 Column Matching Analysis\n\n")
            response_parts.append(f"I've analyzed **{len(state.current_df.columns)} columns from File 1** and **{len(state.current_df2.columns)} columns from File 2**.\n\n")
            
            if top_matches:
                response_parts.append("### ✅ Best Column Matches\n\n")
                response_parts.append("Here are the top column pairs that are most likely to match:\n\n")
                
                for i, match in enumerate(top_matches[:5], 1):
                    confidence = match.get('confidence', 0)
                    file1_col = match.get('file1_column', '')
                    file2_col = match.get('file2_column', '')
                    
                    # Determine match quality
                    if confidence > 70:
                        quality = "🟢 **Excellent Match**"
                    elif confidence > 50:
                        quality = "🟡 **Good Match**"
                    else:
                        quality = "🟠 **Moderate Match**"
                    
                    response_parts.append(f"**{i}. `{file1_col}` ↔️ `{file2_col}`**\n")
                    response_parts.append(f"   - {quality} ({confidence:.1f}% confidence)\n")
                    
                    # Add reasoning
                    reasons = []
                    name_sim = match.get('name_similarity', 0)
                    data_sim = match.get('data_similarity', 0)
                    pattern_sim = match.get('json_confidence', 0)
                    
                    if name_sim > 0.7:
                        reasons.append(f"Very similar names ({name_sim*100:.0f}%)")
                    elif name_sim > 0.4:
                        reasons.append(f"Similar names ({name_sim*100:.0f}%)")
                    
                    if data_sim > 0.7:
                        reasons.append(f"Similar data patterns ({data_sim*100:.0f}%)")
                    elif data_sim > 0.4:
                        reasons.append(f"Matching data ({data_sim*100:.0f}%)")
                    
                    if pattern_sim > 0.5:
                        reasons.append("Matching data types/patterns")
                    
                    if reasons:
                        response_parts.append(f"   - **Why:** {', '.join(reasons)}\n")
                    response_parts.append("\n")
                
                # Add recommendations
                response_parts.append("### 💡 Recommendations\n\n")
                
                high_confidence = [m for m in top_matches if m.get('confidence', 0) > 70]
                if high_confidence:
                    response_parts.append(f"- **{len(high_confidence)} column pairs** have high confidence (>70%) and are excellent candidates for joining/merging.\n")
                
                medium_confidence = [m for m in top_matches if 50 < m.get('confidence', 0) <= 70]
                if medium_confidence:
                    response_parts.append(f"- **{len(medium_confidence)} column pairs** have good confidence (50-70%) and should be reviewed manually.\n")
                
                # Suggest best join keys
                potential_keys = [m for m in top_matches if any(keyword in m.get('file1_column', '').lower() for keyword in ['id', 'key', 'code', 'number'])]
                if potential_keys:
                    best_key = potential_keys[0]
                    response_parts.append(f"\n**💎 Best Join Key:** `{best_key['file1_column']}` ↔️ `{best_key['file2_column']}` ({best_key.get('confidence', 0):.1f}% match)\n")
            else:
                response_parts.append("\n⚠️ No strong column matches found. The datasets may have very different structures.\n")
            
            return {
                "answer": "".join(response_parts),
                "result_type": "column_matching",
                "column_similarities": similarity_data,
                "top_matches": top_matches[:5]
            }
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Error analyzing column matches: {str(e)}")
    else:
        # Only one file loaded
        return {
            "answer": "Please upload both CSV files to get column matching insights. Currently, only one file is loaded.",
            "result_type": "info"
        }

@router.post("/feedback/match")
async def submit_match_feedback(feedback: dict):
    """Submit feedback on a column match with ML training features"""
    try:
        from app.services.feedback_learning import feedback_system
        
        file1_col = feedback.get('file1_column')
        file2_col = feedback.get('file2_column')
        is_correct = feedback.get('is_correct', False)
        correct_match = feedback.get('correct_match')
        user_note = feedback.get('user_note')
        
        # ML training features (optional, for better learning)
        name_similarity = feedback.get('name_similarity', 0.0)
        data_similarity = feedback.get('data_similarity', 0.0)
        pattern_score = feedback.get('pattern_score', 0.0)
        confidence = feedback.get('confidence', 0.0)
        
        if not file1_col or not file2_col:
            raise HTTPException(status_code=400, detail="file1_column and file2_column are required")
        
        result = feedback_system.add_feedback(
            file1_col=file1_col,
            file2_col=file2_col,
            is_correct=is_correct,
            correct_match=correct_match,
            user_note=user_note,
            name_similarity=name_similarity,
            data_similarity=data_similarity,
            pattern_score=pattern_score,
            confidence=confidence
        )
        
        return {
            "success": True,
            "message": "Feedback recorded and ML systems updated",
            "feedback": result
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error recording feedback: {str(e)}")

@router.get("/feedback/stats")
async def get_feedback_stats():
    """Get feedback statistics"""
    try:
        from app.services.feedback_learning import feedback_system
        stats = feedback_system.get_feedback_stats()
        return stats
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error getting feedback stats: {str(e)}")

# ============================================================================
# CONTEXT MANAGEMENT ENDPOINTS
# ============================================================================

@router.post("/context/questions")
async def generate_context_questions():
    """Generate AI-driven context questions for loaded datasets"""
    if state.current_df is None or state.current_df2 is None:
        raise HTTPException(status_code=400, detail="Both files must be loaded to generate context questions")
    
    try:
        from app.services.question_generator import generate_context_questions
        
        questions = generate_context_questions(state.current_df, state.current_df2)
        
        return {
            "success": True,
            "questions": questions,
            "total_questions": (
                len(questions.get('file1_questions', [])) + 
                len(questions.get('file2_questions', [])) + 
                len(questions.get('relationship_questions', []))
            )
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error generating questions: {str(e)}")


@router.post("/context/submit")
async def submit_context(request: dict):
    """Store user-provided context for a dataset"""
    try:
        from app.services.context_service import store_context, ContextService
        
        file_index = request.get('file_index')
        context_data = request.get('context_data')
        
        if not file_index or not context_data:
            raise HTTPException(status_code=400, detail="file_index and context_data are required")
        
        if file_index not in [1, 2]:
            raise HTTPException(status_code=400, detail="file_index must be 1 or 2")
        
        # Ensure context has required fields
        if 'dataset_purpose' not in context_data or 'business_domain' not in context_data:
            # Create schema and merge with provided data
            schema = ContextService.create_context_schema()
            schema.update(context_data)
            context_data = schema
        
        # Store context
        stored_context = store_context(file_index, context_data)
        
        return {
            "success": True,
            "message": f"Context stored for File {file_index}",
            "context": stored_context
        }
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error storing context: {str(e)}")


@router.get("/context/{file_index}")
async def get_context_endpoint(file_index: int):
    """Retrieve stored context for a dataset"""
    if file_index not in [1, 2]:
        raise HTTPException(status_code=400, detail="file_index must be 1 or 2")
    
    try:
        from app.services.context_service import get_context
        
        context = get_context(file_index)
        
        if context is None:
            return {
                "success": True,
                "has_context": False,
                "context": None
            }
        
        return {
            "success": True,
            "has_context": True,
            "context": context
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error retrieving context: {str(e)}")


@router.delete("/context/{file_index}")
async def clear_context_endpoint(file_index: Optional[int] = None):
    """Clear context for specific file or all files"""
    try:
        from app.services.context_service import clear_context
        
        if file_index is not None and file_index not in [1, 2]:
            raise HTTPException(status_code=400, detail="file_index must be 1 or 2")
        
        clear_context(file_index)
        
        if file_index:
            message = f"Context cleared for File {file_index}"
        else:
            message = "All context cleared"
        
        return {
            "success": True,
            "message": message
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error clearing context: {str(e)}")


@router.get("/context/status")
async def get_context_status():
    """Get context status for both files"""
    return {
        "file1": {
            "has_context": state.file1_context is not None,
            "context_summary": {
                "dataset_purpose": state.file1_context.get('dataset_purpose') if state.file1_context else None,
                "business_domain": state.file1_context.get('business_domain') if state.file1_context else None,
            } if state.file1_context else None
        },
        "file2": {
            "has_context": state.file2_context is not None,
            "context_summary": {
                "dataset_purpose": state.file2_context.get('dataset_purpose') if state.file2_context else None,
                "business_domain": state.file2_context.get('business_domain') if state.file2_context else None,
            } if state.file2_context else None
        }
    }


@router.post("/config/ollama")
async def save_ollama_config(config: dict):
    """Save Ollama configuration"""
    try:
        import app.config as config_module
        
        # Update runtime config
        if 'baseUrl' in config:
            config_module.OLLAMA_BASE_URL = config['baseUrl']
        if 'model' in config:
            config_module.OLLAMA_MODEL = config['model']
        
        # Optional: Write to config.py file for persistence
        config_file_path = os.path.join(os.path.dirname(__file__), '..', 'config.py')
        if os.path.exists(config_file_path):
            with open(config_file_path, 'r') as f:
                lines = f.readlines()
            
            with open(config_file_path, 'w') as f:
                for line in lines:
                    if line.startswith('OLLAMA_MODEL =') and 'model' in config:
                        f.write(f'OLLAMA_MODEL = "{config["model"]}"\n')
                    elif line.startswith('OLLAMA_BASE_URL =') and 'baseUrl' in config:
                        f.write(f'OLLAMA_BASE_URL = "{config["baseUrl"]}"\n')
                    else:
                        f.write(line)
        
        return {
            "success": True,
            "message": "Ollama configuration saved successfully",
            "config": {
                "baseUrl": config_module.OLLAMA_BASE_URL,
                "model": config_module.OLLAMA_MODEL
            }
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error saving Ollama config: {str(e)}")


@router.get("/config/ollama")
async def get_ollama_config():
    """Get current Ollama configuration"""
    from app.config import OLLAMA_BASE_URL, OLLAMA_MODEL
    return {
        "baseUrl": OLLAMA_BASE_URL,
        "model": OLLAMA_MODEL
    }

