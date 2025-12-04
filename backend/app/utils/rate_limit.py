"""
Rate Limiting - Production-Grade Implementation

Implements sliding window rate limiting with configurable limits,
proper HTTP 429 responses, and Retry-After headers.
"""

import time
from typing import Tuple, Optional
from collections import defaultdict
from fastapi import HTTPException, Request
from functools import wraps

from app.state import rate_limit_store, rate_limit_lock
from app.config import RATE_LIMIT_ENABLED, MAX_REQUESTS_PER_MINUTE


class RateLimiter:
    """Sliding window rate limiter with in-memory storage"""
    
    def __init__(self):
        self.store = rate_limit_store
        self.lock = rate_limit_lock
    
    def check_limit(
        self, 
        key: str, 
        max_requests: int, 
        window_seconds: int
    ) -> Tuple[bool, int, int]:
        """
        Check if request is within rate limit.
        
        Returns:
            (allowed: bool, remaining: int, reset_time: int)
        """
        current_time = time.time()
        
        with self.lock:
            # Initialize if not exists
            if key not in self.store:
                self.store[key] = []
            
            # Clean old entries (sliding window)
            self.store[key] = [
                req_time for req_time in self.store[key]
                if current_time - req_time < window_seconds
            ]
            
            # Calculate remaining requests
            current_count = len(self.store[key])
            remaining = max(0, max_requests - current_count)
            
            # Calculate reset time
            if self.store[key]:
                oldest_request = min(self.store[key])
                reset_time = int(oldest_request + window_seconds)
            else:
                reset_time = int(current_time + window_seconds)
            
            # Check limit
            if current_count >= max_requests:
                return False, 0, reset_time
            
            # Add current request
            self.store[key].append(current_time)
            return True, remaining - 1, reset_time
    
    def get_client_identifier(self, request: Request) -> str:
        """Get unique identifier for client (IP-based)"""
        # Try to get real IP from X-Forwarded-For header
        forwarded = request.headers.get("X-Forwarded-For")
        if forwarded:
            return forwarded.split(",")[0].strip()
        
        # Fall back to direct client IP
        if request.client:
            return request.client.host
        
        return "unknown"


# Global rate limiter instance
limiter = RateLimiter()


def check_rate_limit(identifier: str, max_requests: int = 60, window: int = 60) -> bool:
    """Legacy function for backward compatibility"""
    allowed, _, _ = limiter.check_limit(identifier, max_requests, window)
    return allowed


def rate_limit(
    max_requests: int = MAX_REQUESTS_PER_MINUTE,
    window_seconds: int = 60,
    key_prefix: str = "api"
):
    """
    Decorator for rate limiting FastAPI endpoints.
    
    Usage:
        @router.post("/endpoint")
        @rate_limit(max_requests=10, window_seconds=3600)
        async def my_endpoint():
            ...
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Skip rate limiting if disabled
            if not RATE_LIMIT_ENABLED:
                return await func(*args, **kwargs)
            
            # Extract request from args/kwargs
            request: Optional[Request] = None
            for arg in args:
                if isinstance(arg, Request):
                    request = arg
                    break
            if not request and 'request' in kwargs:
                request = kwargs['request']
            
            if not request:
                # Can't rate limit without request, allow through
                return await func(*args, **kwargs)
            
            # Generate rate limit key
            client_id = limiter.get_client_identifier(request)
            endpoint = request.url.path
            key = f"{key_prefix}:{endpoint}:{client_id}"
            
            # Check rate limit
            allowed, remaining, reset_time = limiter.check_limit(
                key, max_requests, window_seconds
            )
            
            if not allowed:
                retry_after = reset_time - int(time.time())
                raise HTTPException(
                    status_code=429,
                    detail={
                        "error": "Rate limit exceeded",
                        "message": f"Too many requests. Try again in {retry_after} seconds.",
                        "retry_after": retry_after,
                        "limit": max_requests,
                        "window": window_seconds
                    },
                    headers={"Retry-After": str(retry_after)}
                )
            
            # Add rate limit headers to response
            response = await func(*args, **kwargs)
            
            # If response is a Response object, add headers
            if hasattr(response, 'headers'):
                response.headers["X-RateLimit-Limit"] = str(max_requests)
                response.headers["X-RateLimit-Remaining"] = str(remaining)
                response.headers["X-RateLimit-Reset"] = str(reset_time)
            
            return response
        
        return wrapper
    return decorator

