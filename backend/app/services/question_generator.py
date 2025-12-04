"""
Question Generator - AI-driven intelligent question generation for context collection
"""
from typing import List, Dict, Any, Optional
import pandas as pd
import numpy as np
from datetime import datetime


class QuestionType:
    """Question type constants"""
    DATASET_PURPOSE = "dataset_purpose"
    BUSINESS_DOMAIN = "business_domain"
    KEY_ENTITIES = "key_entities"
    TEMPORAL_CONTEXT = "temporal_context"
    COLUMN_SEMANTIC = "column_semantic"
    RELATIONSHIPS = "relationships"
    CUSTOM_MAPPINGS = "custom_mappings"
    EXCLUSIONS = "exclusions"


class Question:
    """Question model"""
    def __init__(self, question_id: str, question_type: str, text: str, 
                 options: Optional[List[str]] = None, required: bool = True,
                 metadata: Optional[Dict] = None):
        self.id = question_id
        self.type = question_type
        self.text = text
        self.options = options or []
        self.required = required
        self.metadata = metadata or {}
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "type": self.type,
            "text": self.text,
            "options": self.options,
            "required": self.required,
            "metadata": self.metadata
        }


class QuestionGenerator:
    """Generate intelligent context questions based on data analysis"""
    
    DOMAIN_OPTIONS = [
        "Sales & Marketing",
        "Finance & Accounting",
        "Human Resources",
        "Operations & Supply Chain",
        "Customer Service",
        "Healthcare",
        "E-commerce",
        "Manufacturing",
        "Technology & IT",
        "Education",
        "Other"
    ]
    
    @staticmethod
    def analyze_dataframe(df: pd.DataFrame) -> Dict[str, Any]:
        """Analyze dataframe to inform question generation"""
        analysis = {
            "num_rows": len(df),
            "num_columns": len(df.columns),
            "column_names": df.columns.tolist(),
            "column_types": {},
            "has_dates": False,
            "has_numeric": False,
            "has_text": False,
            "potential_ids": [],
            "potential_dates": [],
            "potential_amounts": []
        }
        
        for col in df.columns:
            dtype = str(df[col].dtype)
            analysis["column_types"][col] = dtype
            
            # Detect column patterns
            col_lower = col.lower()
            
            if 'int' in dtype or 'float' in dtype:
                analysis["has_numeric"] = True
                if any(keyword in col_lower for keyword in ['id', 'number', 'code', 'key']):
                    analysis["potential_ids"].append(col)
                if any(keyword in col_lower for keyword in ['amount', 'price', 'cost', 'revenue', 'salary']):
                    analysis["potential_amounts"].append(col)
            
            if 'object' in dtype or 'string' in dtype:
                analysis["has_text"] = True
            
            if 'datetime' in dtype or any(keyword in col_lower for keyword in ['date', 'time', 'timestamp']):
                analysis["has_dates"] = True
                analysis["potential_dates"].append(col)
        
        return analysis
    
    @staticmethod
    def generate_questions(df: pd.DataFrame, file_index: int) -> List[Question]:
        """Generate context questions for a dataset"""
        analysis = QuestionGenerator.analyze_dataframe(df)
        questions = []
        
        # Q1: Dataset Purpose (Always ask)
        questions.append(Question(
            question_id=f"f{file_index}_purpose",
            question_type=QuestionType.DATASET_PURPOSE,
            text=f"What is the primary purpose of this dataset (File {file_index})?",
            options=[],
            required=True,
            metadata={"placeholder": "e.g., Customer transaction records, Employee performance data, etc."}
        ))
        
        # Q2: Business Domain (Always ask)
        questions.append(Question(
            question_id=f"f{file_index}_domain",
            question_type=QuestionType.BUSINESS_DOMAIN,
            text=f"Which business domain does this dataset belong to?",
            options=QuestionGenerator.DOMAIN_OPTIONS,
            required=True
        ))
        
        # Try to generate AI questions
        ai_questions = QuestionGenerator._generate_ai_questions(analysis, file_index)
        if ai_questions:
            questions.extend(ai_questions)
        else:
            # Fallback to heuristics if AI fails
            
            # Q3: Key Entities (Always ask)
            entity_hint = "e.g., Customer, Product, Order, Employee"
            questions.append(Question(
                question_id=f"f{file_index}_entities",
                question_type=QuestionType.KEY_ENTITIES,
                text=f"What are the main entities or subjects in this dataset?",
                options=[],
                required=True,
                metadata={
                    "placeholder": entity_hint,
                    "input_type": "tags",
                    "hint": "Enter multiple entities separated by commas"
                }
            ))
            
            # Q4: Temporal Context (if dates detected)
            if analysis["has_dates"]:
                date_cols = ", ".join(analysis["potential_dates"][:3])
                questions.append(Question(
                    question_id=f"f{file_index}_temporal",
                    question_type=QuestionType.TEMPORAL_CONTEXT,
                    text=f"What time period does this data cover? (Found date columns: {date_cols})",
                    options=[],
                    required=False,
                    metadata={"placeholder": "e.g., Q1 2024, Last 12 months, Historical data 2020-2023"}
                ))
            
            # Q5: Column Semantics (for ambiguous columns)
            ambiguous_columns = QuestionGenerator._find_ambiguous_columns(analysis["column_names"])
            if ambiguous_columns:
                col_list = ", ".join(ambiguous_columns[:5])
                questions.append(Question(
                    question_id=f"f{file_index}_column_semantics",
                    question_type=QuestionType.COLUMN_SEMANTIC,
                    text=f"Can you briefly describe what these columns represent: {col_list}?",
                    options=[],
                    required=False,
                    metadata={
                        "columns": ambiguous_columns[:5],
                        "input_type": "column_descriptions"
                    }
                ))
        
        # Q6: Exclusions (optional) - Always add this at the end
        questions.append(Question(
            question_id=f"f{file_index}_exclusions",
            question_type=QuestionType.EXCLUSIONS,
            text=f"Are there any columns that should be excluded from correlation analysis?",
            options=analysis["column_names"],
            required=False,
            metadata={
                "input_type": "multi_select",
                "hint": "Select columns like temporary fields, debug data, or irrelevant information"
            }
        ))
        
        return questions

    @staticmethod
    def _generate_ai_questions(analysis: Dict[str, Any], file_index: int) -> List[Question]:
        """Generate specific questions using LLM based on dataset analysis"""
        try:
            from app.services.llm import call_ollama
            import json
            import re
            
            prompt = f"""
            Analyze this dataset summary and generate 3 specific questions to understand its business context.
            
            Dataset Summary:
            - Columns: {', '.join(analysis['column_names'][:20])}
            - Row Count: {analysis['num_rows']}
            - Date Columns: {', '.join(analysis['potential_dates'])}
            - ID Columns: {', '.join(analysis['potential_ids'])}
            - Amount Columns: {', '.join(analysis['potential_amounts'])}
            
            Generate 3 questions that would help clarify:
            1. The specific business process this data represents
            2. The meaning of any ambiguous columns
            3. The time granularity or scope
            
            Return a JSON object with a 'questions' array. Each question should have:
            - 'text': The question text
            - 'type': One of ['text', 'select', 'multi_select']
            - 'options': Array of strings (only for select/multi_select)
            - 'id_suffix': A unique suffix for the ID (e.g., 'process_type')
            
            Example JSON:
            {{
                "questions": [
                    {{
                        "text": "What type of transactions does this represent?",
                        "type": "select",
                        "options": ["Online Sales", "In-store POS", "B2B Orders"],
                        "id_suffix": "trans_type"
                    }}
                ]
            }}
            
            Return ONLY the JSON.
            """
            
            response = call_ollama(prompt)
            if not response:
                return []
                
            # Extract JSON
            json_match = re.search(r'\{.*\}', response, re.DOTALL)
            if not json_match:
                return []
                
            data = json.loads(json_match.group(0))
            ai_questions = []
            
            for i, q_data in enumerate(data.get('questions', [])):
                q_id = f"f{file_index}_ai_{q_data.get('id_suffix', str(i))}"
                
                # Map LLM types to our QuestionType
                q_type = QuestionType.COLUMN_SEMANTIC # Default
                if 'entity' in q_data.get('text', '').lower():
                    q_type = QuestionType.KEY_ENTITIES
                elif 'time' in q_data.get('text', '').lower() or 'date' in q_data.get('text', '').lower():
                    q_type = QuestionType.TEMPORAL_CONTEXT
                
                ai_questions.append(Question(
                    question_id=q_id,
                    question_type=q_type, # Use generic type or infer
                    text=q_data['text'],
                    options=q_data.get('options', []),
                    required=False,
                    metadata={"generated_by": "ai"}
                ))
                
            return ai_questions
            
        except Exception as e:
            print(f"Error generating AI questions: {e}")
            return []
    
    @staticmethod
    def generate_relationship_questions(df1: pd.DataFrame, df2: pd.DataFrame) -> List[Question]:
        """Generate questions about relationships between two datasets"""
        analysis1 = QuestionGenerator.analyze_dataframe(df1)
        analysis2 = QuestionGenerator.analyze_dataframe(df2)
        
        questions = []
        
        # Q1: Relationship type
        questions.append(Question(
            question_id="relationship_type",
            question_type=QuestionType.RELATIONSHIPS,
            text="How are these two datasets related?",
            options=[
                "Same entity, different time periods",
                "Same entity, different sources",
                "Related entities (e.g., Orders and Customers)",
                "Completely different entities",
                "Not sure"
            ],
            required=True
        ))
        
        # Q2: Custom mappings (if user knows specific mappings)
        questions.append(Question(
            question_id="custom_mappings",
            question_type=QuestionType.CUSTOM_MAPPINGS,
            text="Do you know of any specific column mappings between the files?",
            options=[],
            required=False,
            metadata={
                "input_type": "mapping_pairs",
                "hint": "e.g., 'user_id' in File 1 maps to 'customer_id' in File 2",
                "file1_columns": analysis1["column_names"],
                "file2_columns": analysis2["column_names"]
            }
        ))
        
        return questions
    
    @staticmethod
    def _find_ambiguous_columns(column_names: List[str]) -> List[str]:
        """Find columns with ambiguous or unclear names"""
        ambiguous = []
        
        for col in column_names:
            col_lower = col.lower()
            
            # Skip clearly named columns
            if any(clear in col_lower for clear in [
                'id', 'name', 'email', 'phone', 'address', 'date', 'time',
                'amount', 'price', 'quantity', 'status', 'type', 'category'
            ]):
                continue
            
            # Check for abbreviations or short names
            if len(col) <= 3 or '_' not in col and len(col.split()) == 1:
                ambiguous.append(col)
            
            # Check for generic names
            if any(generic in col_lower for generic in [
                'col', 'field', 'data', 'value', 'item', 'attr'
            ]):
                ambiguous.append(col)
        
        return ambiguous[:10]  # Limit to 10 most ambiguous
    
    @staticmethod
    def generate_all_questions(df1: pd.DataFrame, df2: pd.DataFrame) -> Dict[str, List[Dict]]:
        """Generate all questions for both files and their relationship"""
        return {
            "file1_questions": [q.to_dict() for q in QuestionGenerator.generate_questions(df1, 1)],
            "file2_questions": [q.to_dict() for q in QuestionGenerator.generate_questions(df2, 2)],
            "relationship_questions": [q.to_dict() for q in QuestionGenerator.generate_relationship_questions(df1, df2)]
        }


def generate_context_questions(df1: pd.DataFrame, df2: Optional[pd.DataFrame] = None) -> Dict[str, Any]:
    """
    Generate context questions for one or two datasets
    
    Args:
        df1: First dataframe
        df2: Optional second dataframe
    
    Returns:
        Dictionary containing generated questions
    """
    if df2 is not None:
        return QuestionGenerator.generate_all_questions(df1, df2)
    else:
        return {
            "file1_questions": [q.to_dict() for q in QuestionGenerator.generate_questions(df1, 1)]
        }
