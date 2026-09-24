"""
Integration test for HTTP Server Authentication & Route Protection
"""

import asyncio
import json
import os
import shutil
import tempfile
import urllib.request
import urllib.error
import unittest
from app.server import HTTPServer

class TestServerAuthIntegration(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.test_dir = tempfile.mkdtemp()
        import app.config as config
        cls.orig_config_path = config.CONFIG_PATH
        cls.orig_db_path = config.DB_PATH
        config.CONFIG_PATH = os.path.join(cls.test_dir, "config.json")
        config.DB_PATH = os.path.join(cls.test_dir, "test.db")
        config.save_settings(dict(config.DEFAULT_SETTINGS))

        cls.port = 19192
        cls.host = "127.0.0.1"
        cls.base_url = f"http://{cls.host}:{cls.port}"

        # Start server in a background thread or async loop
        cls.loop = asyncio.new_event_loop()
        cls.server = HTTPServer(host=cls.host, port=cls.port)

        import threading
        def run_loop():
            asyncio.set_event_loop(cls.loop)
            cls.loop.run_until_complete(cls.server.start())
            cls.loop.run_forever()

        cls.thread = threading.Thread(target=run_loop, daemon=True)
        cls.thread.start()

        # Wait for server to bind
        import time
        time.sleep(0.5)

    @classmethod
    def tearDownClass(cls):
        import app.config as config
        config.CONFIG_PATH = cls.orig_config_path
        config.DB_PATH = cls.orig_db_path
        shutil.rmtree(cls.test_dir, ignore_errors=True)

    def test_full_auth_flow(self):
        # Bypass any environment proxy for 127.0.0.1
        proxy_handler = urllib.request.ProxyHandler({})
        self.opener = urllib.request.build_opener(proxy_handler)

        class NoRedirectHandler(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                return None

        no_redirect_opener = urllib.request.build_opener(proxy_handler, NoRedirectHandler)
        try:
            resp = no_redirect_opener.open(f"{self.base_url}/")
            status = resp.getcode()
        except urllib.error.HTTPError as e:
            status = e.code
            self.assertEqual(e.headers.get("Location"), "/login")
        self.assertEqual(status, 302)

        # 2. Check Auth Status (First time)
        req = urllib.request.Request(f"{self.base_url}/api/auth/status")
        with self.opener.open(req) as resp:
            data = json.loads(resp.read().decode('utf-8'))
            self.assertTrue(data.get("is_first_time"))
            self.assertFalse(data.get("authenticated"))

        # 3. Access protected API without auth should return 401
        try:
            self.opener.open(f"{self.base_url}/api/nodes")
            unauth_status = 200
        except urllib.error.HTTPError as e:
            unauth_status = e.code
        self.assertEqual(unauth_status, 401)

        # 4. Setup admin account via POST /api/auth/setup
        setup_payload = json.dumps({"username": "myadmin", "password": "StrongSecret2026!"}).encode('utf-8')
        req = urllib.request.Request(
            f"{self.base_url}/api/auth/setup",
            data=setup_payload,
            headers={"Content-Type": "application/json"}
        )
        with self.opener.open(req) as resp:
            cookie = resp.headers.get("Set-Cookie")
            self.assertIn("hawal_session=", cookie)
            data = json.loads(resp.read().decode('utf-8'))
            self.assertTrue(data.get("success"))

        session_cookie = cookie.split(";")[0]

        # 5. Access protected API WITH session cookie -> 200 OK
        req = urllib.request.Request(
            f"{self.base_url}/api/nodes",
            headers={"Cookie": session_cookie}
        )
        with self.opener.open(req) as resp:
            self.assertEqual(resp.getcode(), 200)
            data = json.loads(resp.read().decode('utf-8'))
            self.assertIn("nodes", data)

        # 6. Access / with session cookie -> 200 OK (Dashboard HTML)
        req = urllib.request.Request(
            f"{self.base_url}/",
            headers={"Cookie": session_cookie}
        )
        with self.opener.open(req) as resp:
            self.assertEqual(resp.getcode(), 200)
            body = resp.read().decode('utf-8')
            self.assertIn("Hawal", body)

        # 7. Logout via POST /api/auth/logout
        req = urllib.request.Request(
            f"{self.base_url}/api/auth/logout",
            data=b"{}",
            headers={"Content-Type": "application/json", "Cookie": session_cookie}
        )
        with self.opener.open(req) as resp:
            self.assertEqual(resp.getcode(), 200)

        # 8. Access protected API again -> 401
        try:
            req = urllib.request.Request(
                f"{self.base_url}/api/nodes",
                headers={"Cookie": session_cookie}
            )
            self.opener.open(req)
            after_logout_status = 200
        except urllib.error.HTTPError as e:
            after_logout_status = e.code
        self.assertEqual(after_logout_status, 401)

if __name__ == '__main__':
    unittest.main()
