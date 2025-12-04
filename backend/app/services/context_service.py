"""
Context Service - Manages data context collection and storage
"""
from typing import Dict, List, Optional, Any
import pandas as pd
from datetime import datetime


class ContextService:
    """Service for managing dataset context"""
    
    @staticmethod
    def create_context_schema() -> Dict[str, Any]:
        """Create empty context schema"""
        return {
            "dataset_purpose": None,
            "business_domain": None,
            "key_entities": [],
            "temporal_context": None,
            "column_descriptions": {},
            "relationships": [],
            "custom_mappings": {},
            "exclusions": [],
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat()
        }
    
    @staticmethod
    def validate_context(context: Dict[str, Any]) -> bool:
        """Validate context data structure"""
        required_fields = ["dataset_purpose", "business_domain"]
        return all(field in context for field in required_fields)
    
    @staticmethod
    def merge_context(existing: Dict[str, Any], new: Dict[str, Any]) -> Dict[str, Any]:
        """Merge new context with existing context"""
        merged = existing.copy()
        merged.update(new)
        merged["updated_at"] = datetime.now().isoformat()
        return merged
    
    @staticmethod
    def build_context_prompt(context1: Optional[Dict], context2: Optional[Dict]) -> str:
        """Build enhanced LLM prompt with context information"""
        if not context1 and not context2:
            return ""
        
        prompt_parts = ["Consider the following context:\n"]
        
        if context1:
            prompt_parts.append(f"File 1 Context:")
            prompt_parts.append(f"  - Purpose: {context1.get('dataset_purpose', 'Unknown')}")
            prompt_parts.append(f"  - Domain: {context1.get('business_domain', 'Unknown')}")
            if context1.get('key_entities'):
                prompt_parts.append(f"  - Key Entities: {', '.join(context1['key_entities'])}")
            prompt_parts.append("")
        
        if context2:
            prompt_parts.append(f"File 2 Context:")
            prompt_parts.append(f"  - Purpose: {context2.get('dataset_purpose', 'Unknown')}")
            prompt_parts.append(f"  - Domain: {context2.get('business_domain', 'Unknown')}")
            if context2.get('key_entities'):
                prompt_parts.append(f"  - Key Entities: {', '.join(context2['key_entities'])}")
            prompt_parts.append("")
        
        return "\n".join(prompt_parts)
    
    @staticmethod
    def get_column_context(context: Dict[str, Any], column_name: str) -> Optional[str]:
        """Get specific column description from context"""
        if not context:
            return None
        return context.get('column_descriptions', {}).get(column_name)
    
    @staticmethod
    def should_exclude_column(context: Dict[str, Any], column_name: str) -> bool:
        """Check if column should be excluded from correlation"""
        if not context:
            return False
        return column_name in context.get('exclusions', [])
    
    @staticmethod
    def get_custom_mapping(context1: Dict[str, Any], context2: Dict[str, Any], 
                          col1: str, col2: str) -> Optional[float]:
        """Check if there's a custom mapping defined by user"""
        if context1 and col1 in context1.get('custom_mappings', {}):
            if context1['custom_mappings'][col1] == col2:
                return 1.0  # Perfect match for custom mapping
        
        if context2 and col2 in context2.get('custom_mappings', {}):
            if context2['custom_mappings'][col2] == col1:
                return 1.0  # Perfect match for custom mapping
        
        return None
    
    @staticmethod
    def enhance_confidence_with_context(
        base_confidence: float,
        context1: Optional[Dict],
        context2: Optional[Dict],
        col1: str,
        col2: str
    ) -> float:
        """Enhance confidence score based on context alignment"""
        if not context1 or not context2:
            return base_confidence
        
        # Check for custom mappings first
        custom_score = ContextService.get_custom_mapping(context1, context2, col1, col2)
        if custom_score is not None:
            return min(100.0, base_confidence * 1.5)  # Boost confidence for custom mappings
        
        # Check domain alignment
        domain1 = context1.get('business_domain', '').lower()
        domain2 = context2.get('business_domain', '').lower()
        
        if domain1 and domain2 and domain1 == domain2:
            # Same domain - boost confidence slightly
            base_confidence *= 1.1
        
        # Check entity alignment
        entities1 = set(e.lower() for e in context1.get('key_entities', []))
        entities2 = set(e.lower() for e in context2.get('key_entities', []))
        
        if entities1 and entities2:
            entity_overlap = len(entities1.intersection(entities2)) / max(len(entities1), len(entities2))
            if entity_overlap > 0.5:
                base_confidence *= (1.0 + entity_overlap * 0.2)  # Up to 20% boost
        
        return min(100.0, base_confidence)  # Cap at 100%


def store_context(file_index: int, context_data: Dict[str, Any]) -> Dict[str, Any]:
    """Store context for a specific file"""
    from app import state
    
    # Validate context
    if not ContextService.validate_context(context_data):
        raise ValueError("Invalid context data: missing required fields")
    
    # Store in appropriate state variable
    if file_index == 1:
        if state.file1_context:
            state.file1_context = ContextService.merge_context(state.file1_context, context_data)
        else:
            state.file1_context = context_data
        return state.file1_context
    elif file_index == 2:
        if state.file2_context:
            state.file2_context = ContextService.merge_context(state.file2_context, context_data)
        else:
            state.file2_context = context_data
        return state.file2_context
    else:
        raise ValueError("Invalid file_index: must be 1 or 2")


def get_context(file_index: int) -> Optional[Dict[str, Any]]:
    """Retrieve context for a specific file"""
    from app import state
    
    if file_index == 1:
        return state.file1_context
    elif file_index == 2:
        return state.file2_context
    else:
        raise ValueError("Invalid file_index: must be 1 or 2")


def clear_context(file_index: Optional[int] = None):
    """Clear context for specific file or all files"""
    from app import state
    
    if file_index == 1 or file_index is None:
        state.file1_context = None
    if file_index == 2 or file_index is None:
        state.file2_context = None
    if file_index is None:
        state.correlation_metadata = None
