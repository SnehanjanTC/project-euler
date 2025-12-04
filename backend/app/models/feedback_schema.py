from pydantic import BaseModel
from typing import Optional

class FeedbackRequest(BaseModel):
    file1_column: str
    file2_column: str
    is_correct: bool
    correct_match: Optional[str] = None
    user_note: Optional[str] = None
