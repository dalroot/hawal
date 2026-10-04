"""
⚡ Hawal Tunnel - Authentication & Session Management (هه‌واڵ)
Provides PBKDF2-HMAC-SHA256 password hashing, first-time setup detection,
secure authentication, and sliding session token management.
"""

import hashlib
import os
import secrets
import time
import json
from app.config import load_settings, save_settings, DATA_DIR

SESSIONS_FILE = os.path.join(DATA_DIR, "sessions.json")
SESSION_LIFETIME_SECONDS = 90 * 24 * 60 * 60  # 90 days

def _load_sessions_from_disk():
    if not os.path.exists(SESSIONS_FILE):
        return {}
    try:
        with open(SESSIONS_FILE, "r", encoding="utf-8") as f:
            data = json.load(f)
            now = time.time()
            return {k: v for k, v in data.items() if isinstance(v, (int, float)) and v > now}
    except Exception:
        return {}

def _save_sessions_to_disk(sessions):
    try:
        now = time.time()
        valid = {k: v for k, v in sessions.items() if isinstance(v, (int, float)) and v > now}
        with open(SESSIONS_FILE, "w", encoding="utf-8") as f:
            json.dump(valid, f)
    except (OSError, TypeError):
        pass

# Active in-memory session tokens loaded from disk
ACTIVE_SESSIONS = _load_sessions_from_disk()


def hash_password(password: str, salt_bytes: bytes = None) -> tuple[str, str]:
    """
    Hashes password using PBKDF2-HMAC-SHA256 with 200,000 iterations.
    Returns (hash_hex, salt_hex).
    """
    if salt_bytes is None:
        salt_bytes = secrets.token_bytes(16)
    pw_hash = hashlib.pbkdf2_hmac(
        'sha256',
        password.encode('utf-8'),
        salt_bytes,
        200_000
    )
    return pw_hash.hex(), salt_bytes.hex()


def verify_password(password: str, stored_hash_hex: str, salt_hex: str) -> bool:
    """
    Verifies a password against the stored hash and salt using constant-time comparison.
    """
    try:
        salt_bytes = bytes.fromhex(salt_hex)
        candidate_hash = hashlib.pbkdf2_hmac(
            'sha256',
            password.encode('utf-8'),
            salt_bytes,
            200_000
        ).hex()
        return secrets.compare_digest(candidate_hash, stored_hash_hex)
    except Exception:
        return False


def is_first_time_setup() -> bool:
    """
    Returns True if no administrator account has been initialized yet.
    """
    settings = load_settings()
    has_hash = bool(settings.get("admin_password_hash"))
    has_user = bool(settings.get("admin_username"))
    return not (has_hash and has_user)


def get_admin_username() -> str:
    settings = load_settings()
    return settings.get("admin_username", "admin")


def create_session() -> str:
    """
    Generates a cryptographically secure 32-byte session token with 90-day persistent lifetime.
    """
    token = secrets.token_hex(32)
    ACTIVE_SESSIONS[token] = time.time() + SESSION_LIFETIME_SECONDS
    _save_sessions_to_disk(ACTIVE_SESSIONS)
    return token


def validate_session(token: str) -> bool:
    """
    Checks if session token is valid and not expired, refreshing its lifetime.
    """
    if not token or not isinstance(token, str):
        return False
    global ACTIVE_SESSIONS
    if token not in ACTIVE_SESSIONS:
        ACTIVE_SESSIONS = _load_sessions_from_disk()
    expiry = ACTIVE_SESSIONS.get(token)
    if not expiry:
        return False
    now = time.time()
    if now > expiry:
        ACTIVE_SESSIONS.pop(token, None)
        _save_sessions_to_disk(ACTIVE_SESSIONS)
        return False
    # Slide session expiration window
    ACTIVE_SESSIONS[token] = now + SESSION_LIFETIME_SECONDS
    return True


def invalidate_session(token: str) -> None:
    """
    Removes session token upon logout.
    """
    if token:
        ACTIVE_SESSIONS.pop(token, None)
        _save_sessions_to_disk(ACTIVE_SESSIONS)


def setup_admin(username: str, password: str) -> tuple[bool, str, str]:
    """
    Configures the administrator account for the first time.
    Returns (success, session_token_or_empty, error_message).
    """
    username = (username or "").strip()
    password = (password or "")

    if not is_first_time_setup():
        return False, "", "حساب مدیریت قبلاً راه‌اندازی شده است."

    if len(username) < 3:
        return False, "", "نام کاربری باید حداقل ۳ کاراکتر باشد."

    if len(password) < 6:
        return False, "", "رمز عبور باید حداقل ۶ کاراکتر باشد."

    pw_hash, salt_hex = hash_password(password)

    settings = load_settings()
    settings["admin_username"] = username
    settings["admin_password_hash"] = pw_hash
    settings["admin_password_salt"] = salt_hex
    save_settings(settings)

    session_token = create_session()
    return True, session_token, ""


def authenticate(username: str, password: str) -> tuple[bool, str, str]:
    """
    Authenticates administrator username & password.
    Returns (success, session_token_or_empty, error_message).
    """
    username = (username or "").strip()
    password = (password or "")

    if is_first_time_setup():
        return False, "", "پنل هنوز راه‌اندازی اولیه نشده است."

    settings = load_settings()
    stored_user = settings.get("admin_username", "admin")
    stored_hash = settings.get("admin_password_hash", "")
    stored_salt = settings.get("admin_password_salt", "")

    if username != stored_user:
        return False, "", "نام کاربری یا رمز عبور اشتباه است."

    if not verify_password(password, stored_hash, stored_salt):
        return False, "", "نام کاربری یا رمز عبور اشتباه است."

    session_token = create_session()
    return True, session_token, ""
