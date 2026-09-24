"""
Unit tests for Hawal Authentication & First-Time Setup
"""

import os
import shutil
import tempfile
import unittest

class TestHawalAuth(unittest.TestCase):

    def setUp(self):
        # Create a temporary directory to isolate config.json
        self.test_dir = tempfile.mkdtemp()
        import app.config as config
        self.orig_config_path = config.CONFIG_PATH
        config.CONFIG_PATH = os.path.join(self.test_dir, "config.json")
        config.save_settings(dict(config.DEFAULT_SETTINGS))

    def tearDown(self):
        import app.config as config
        config.CONFIG_PATH = self.orig_config_path
        shutil.rmtree(self.test_dir, ignore_errors=True)

    def test_hash_and_verify_password(self):
        from app.auth import hash_password, verify_password
        pwd = "SuperSecretPassword123!"
        pw_hash, salt = hash_password(pwd)
        self.assertTrue(len(pw_hash) > 30)
        self.assertTrue(len(salt) > 10)
        self.assertTrue(verify_password(pwd, pw_hash, salt))
        self.assertFalse(verify_password("WrongPassword", pw_hash, salt))

    def test_first_time_setup_detection(self):
        from app.auth import is_first_time_setup
        self.assertTrue(is_first_time_setup())

    def test_setup_admin_validation(self):
        from app.auth import setup_admin, is_first_time_setup
        # Short username
        ok, tok, err = setup_admin("ab", "validpassword123")
        self.assertFalse(ok)
        self.assertIn("۳ کاراکتر", err)

        # Short password
        ok, tok, err = setup_admin("admin", "123")
        self.assertFalse(ok)
        self.assertIn("۶ کاراکتر", err)

        # Successful setup
        ok, tok, err = setup_admin("admin", "MySecurePass2026!")
        self.assertTrue(ok)
        self.assertTrue(len(tok) > 20)
        self.assertEqual(err, "")
        self.assertFalse(is_first_time_setup())

    def test_authenticate(self):
        from app.auth import setup_admin, authenticate, validate_session, invalidate_session
        setup_admin("admin", "MyPassword123")

        # Wrong user
        ok, tok, err = authenticate("wronguser", "MyPassword123")
        self.assertFalse(ok)

        # Wrong password
        ok, tok, err = authenticate("admin", "badpass")
        self.assertFalse(ok)

        # Correct credentials
        ok, session_tok, err = authenticate("admin", "MyPassword123")
        self.assertTrue(ok)
        self.assertTrue(validate_session(session_tok))

        # Logout / Invalidate
        invalidate_session(session_tok)
        self.assertFalse(validate_session(session_tok))

if __name__ == '__main__':
    unittest.main()
