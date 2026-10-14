"""Client-side requests-per-minute limiter and retry backoff helper.

Why: free-tier endpoints (~40 RPM) answer 429 if we are greedy; staying under
the limit up front wastes less time than recovering from errors.
Deliberately does not: share state across processes (one process per eval).
"""
import random
import threading
import time
from collections import deque


class RateLimiter:
    def __init__(self, rpm: int):
        self.rpm, self.stamps, self.lock = rpm, deque(), threading.Lock()

    def wait(self):
        while True:
            with self.lock:
                now = time.monotonic()
                while self.stamps and now - self.stamps[0] >= 60:
                    self.stamps.popleft()
                if len(self.stamps) < self.rpm:
                    self.stamps.append(now)
                    return
                sleep = 60 - (now - self.stamps[0])
            time.sleep(max(sleep, 0.05))


def backoff_delay(attempt: int, retry_after: float | None = None) -> float:
    """Exponential backoff with jitter; a server-provided Retry-After wins."""
    if retry_after is not None:
        return retry_after
    return min(2 ** attempt, 60) + random.uniform(0, 1)
