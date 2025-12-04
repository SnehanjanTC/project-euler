# Add this new endpoint to api.py

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
