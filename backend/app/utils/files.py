import os
from datetime import datetime
from app.state import uploaded_files
from app.config import MAX_FILE_AGE

def cleanup_old_files():
    """Clean up old uploaded files"""
    current_time = datetime.now()
    files_to_remove = []
    
    for file_path, upload_time in uploaded_files.items():
        if (current_time - upload_time).total_seconds() > MAX_FILE_AGE:
            files_to_remove.append(file_path)
    
    for file_path in files_to_remove:
        try:
            if os.path.exists(file_path):
                os.remove(file_path)
            del uploaded_files[file_path]
        except Exception as e:
            print(f"Error cleaning up file {file_path}: {str(e)}")
