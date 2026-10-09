"""
Unit tests for Hawal Tunnel MTU / channel_size persistence and Paqet config generation
(Issue #7: bug(panel/paqet): MTU/channel_size configuration persistence drop)
"""

import os
import shutil
import tempfile
import unittest

class TestTunnelMTUPersistence(unittest.TestCase):

    def setUp(self):
        self.test_dir = tempfile.mkdtemp()
        import app.db as db
        self.orig_db_path = db.DB_PATH
        db.DB_PATH = os.path.join(self.test_dir, "test_hawal.db")
        db.init_db()

    def tearDown(self):
        import app.db as db
        db.DB_PATH = self.orig_db_path
        shutil.rmtree(self.test_dir, ignore_errors=True)

    def test_save_tunnel_explicit_mtu(self):
        from app.db import save_tunnel, get_tunnel
        tun_id = "tun_test_mtu_explicit"
        save_tunnel(
            tunnel_id=tun_id,
            name="Paqet Wire Test",
            server_node_id="node_srv",
            client_node_id="node_cli",
            core_port=55011,
            transport="rawtcp",
            ports=["2087=127.0.0.1:2087"],
            token="sec_token_123",
            core_type="paqet",
            channel_size=1150
        )
        t = get_tunnel(tun_id)
        self.assertIsNotNone(t)
        self.assertEqual(t["channel_size"], 1150)
        self.assertEqual(t["core_port"], 55011)

    def test_save_tunnel_default_mtu_by_core_type(self):
        from app.db import save_tunnel, get_tunnel

        # Paqet default
        save_tunnel(
            tunnel_id="tun_paqet_def",
            name="Paqet Default",
            server_node_id="node_srv",
            client_node_id="node_cli",
            core_port=55012,
            transport="rawtcp",
            ports=["2087=127.0.0.1:2087"],
            token="token1",
            core_type="paqet",
            channel_size=None
        )
        t_paqet = get_tunnel("tun_paqet_def")
        self.assertEqual(t_paqet["channel_size"], 1150)

        # Hawal default
        save_tunnel(
            tunnel_id="tun_hawal_def",
            name="Hawal Default",
            server_node_id="node_srv",
            client_node_id="node_cli",
            core_port=55013,
            transport="rawtcp",
            ports=["2088=127.0.0.1:2088"],
            token="token2",
            core_type="hawal",
            channel_size=None
        )
        t_hawal = get_tunnel("tun_hawal_def")
        self.assertEqual(t_hawal["channel_size"], 1350)

    def test_update_tunnel_channel_size(self):
        from app.db import save_tunnel, update_tunnel, get_tunnel
        tun_id = "tun_update_mtu"
        save_tunnel(
            tunnel_id=tun_id,
            name="Before Update",
            server_node_id="node_srv",
            client_node_id="node_cli",
            core_port=55014,
            transport="rawtcp",
            ports=["2087=127.0.0.1:2087"],
            token="tok",
            core_type="paqet",
            channel_size=1350
        )
        t1 = get_tunnel(tun_id)
        self.assertEqual(t1["channel_size"], 1350)

        # Update MTU to 1150
        update_tunnel(
            tunnel_id=tun_id,
            name="After Update",
            core_port=55015,
            transport="rawtcp",
            ports=["2087=127.0.0.1:2087"],
            core_type="paqet",
            channel_size=1150
        )
        t2 = get_tunnel(tun_id)
        self.assertEqual(t2["name"], "After Update")
        self.assertEqual(t2["core_port"], 55015)
        self.assertEqual(t2["channel_size"], 1150)

    def test_paqet_engine_yaml_generation_mtu(self):
        from app.paqet_engine import generate_paqet_server_config, generate_paqet_client_config

        tun_data = {
            "core_port": 55011,
            "token": "test_secret_key",
            "mux_con": 8,
            "channel_size": 1150,
            "kcp_mode": "normal",
            "ports": ["2087=127.0.0.1:2087"]
        }

        server_yaml = generate_paqet_server_config(tun_data)
        self.assertIn("mtu: 1150", server_yaml)
        self.assertIn("addr: \":55011\"", server_yaml)
        self.assertIn("tcpbuf: 8192", server_yaml)
        self.assertIn("udpbuf: 4096", server_yaml)

        client_yaml = generate_paqet_client_config(tun_data, "192.209.62.115")
        self.assertIn("mtu: 1150", client_yaml)
        self.assertIn("addr: \"192.209.62.115:55011\"", client_yaml)
        self.assertIn("listen: \"0.0.0.0:2087\"", client_yaml)
        self.assertIn("target: \"127.0.0.1:2087\"", client_yaml)
        self.assertIn("tcpbuf: 8192", client_yaml)
        self.assertIn("udpbuf: 4096", client_yaml)

if __name__ == "__main__":
    unittest.main()
