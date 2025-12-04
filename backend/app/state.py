from typing import Optional, Dict
import pandas as pd
from datetime import datetime
import threading

# Global variables to store the current dataframes
current_df: Optional[pd.DataFrame] = None
current_df2: Optional[pd.DataFrame] = None
csv_file_path: Optional[str] = None
csv_file_path2: Optional[str] = None

# Context storage for datasets
file1_context: Optional[Dict] = None
file2_context: Optional[Dict] = None
correlation_metadata: Optional[Dict] = None

# Rate limiting storage
rate_limit_store: Dict[str, list] = {}
rate_limit_lock = threading.Lock()

# File cleanup tracking
uploaded_files: Dict[str, datetime] = {}
