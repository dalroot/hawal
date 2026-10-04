import asyncio
import base64
import hashlib
import json
import mimetypes
import os
import secrets
import time
import re
import socket
import struct
import subprocess
import urllib.parse
import urllib.request
from app.config import DEFAULT_HOST, DEFAULT_PORT, MASTER_TOKEN, PANEL_VERSION, GITHUB_REPO, PANEL_DIR
from app.db import (
    init_db, list_nodes, get_node, get_node_by_token, save_node, delete_node,
    update_node_heartbeat, list_tunnels, get_tunnel, update_tunnel, save_tunnel,
    set_tunnel_status, delete_tunnel, record_ping, get_latest_pings,
    request_tunnel_restart, request_all_agents_restart,
    save_log_snapshots, get_log_snapshots, delete_log_snapshots,
    set_tunnel_absolute_traffic, update_tunnel_traffic,
    record_traffic_sample, clean_old_traffic_samples, get_traffic_history
)
from app.backhaul import validate_tunnel_ports, generate_server_config, generate_client_config, generate_docker_compose
from app.gost_engine import generate_gost_server_command, generate_gost_client_command
from app.ping_tool import run_ping, run_tcp_ping
from app.auth import (
    is_first_time_setup, setup_admin, authenticate,
    validate_session, invalidate_session
)

def get_session_token_from_headers(headers):
    cookie_header = headers.get("cookie", "")
    if cookie_header:
        for part in cookie_header.split(";"):
            part = part.strip()
            if part.startswith("hawal_session="):
                tok = part.split("=", 1)[1].strip()
                if tok:
                    return tok
    auth = headers.get("authorization", "")
    if auth.startswith("Bearer "):
        tok = auth.replace("Bearer ", "").strip()
        if tok:
            return tok
    return ""

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
STATIC_DIR = os.path.join(BASE_DIR, "static")
TEMPLATES_DIR = os.path.join(BASE_DIR, "templates")

# Connected WebSocket clients for real-time UI updates
CONNECTED_WS_CLIENTS = set()

# Connected Agent Sockets for real-time dispatch
CONNECTED_AGENTS = {} # node_id -> writer

# A public control-panel port is regularly hit by incomplete probes.  Without
# a deadline, every such peer keeps a socket open indefinitely and can exhaust
# the modest default file-descriptor limit of a VPS.
HTTP_READ_TIMEOUT = 15
MAX_HTTP_BODY_BYTES = 1024 * 1024

def make_ws_handshake_response(sec_key):
    guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
    accept_val = base64.b64encode(hashlib.sha1((sec_key + guid).encode('utf-8')).digest()).decode('utf-8')
    resp = (
        "HTTP/1.1 101 Switching Protocols\r\n"
        "Upgrade: websocket\r\n"
        "Connection: Upgrade\r\n"
        f"Sec-WebSocket-Accept: {accept_val}\r\n\r\n"
    )
    return resp.encode('utf-8')

def encode_ws_frame(payload_bytes, opcode=1):
    length = len(payload_bytes)
    header = bytearray()
    header.append(0x80 | opcode)
    if length <= 125:
        header.append(length)
    elif length <= 65535:
        header.append(126)
        header.extend(length.to_bytes(2, 'big'))
    else:
        header.append(127)
        header.extend(length.to_bytes(8, 'big'))
    return bytes(header + payload_bytes)

async def broadcast_ws(data):
    if not CONNECTED_WS_CLIENTS:
        return
    msg = encode_ws_frame(json.dumps(data).encode('utf-8'))
    dead = []
    for writer in list(CONNECTED_WS_CLIENTS):
        try:
            writer.write(msg)
            await writer.drain()
        except:
            dead.append(writer)
    for w in dead:
        CONNECTED_WS_CLIENTS.discard(w)

_VERSION_CACHE = {
    "data": None,
    "ts": 0,
    "dev": False
}

def parse_semver(v):
    parts = re.findall(r'\d+', str(v))
    return tuple(int(x) for x in parts) if parts else (0,)

def fetch_panel_version_sync(dev=False):
    repo = GITHUB_REPO
    cur_ver = f"v{PANEL_VERSION.lstrip('v')}"
    if dev:
        url = f"https://api.github.com/repos/{repo}/commits/master"
        req = urllib.request.Request(url, headers={"User-Agent": f"HawalPanel/{PANEL_VERSION}"})
        try:
            with urllib.request.urlopen(req, timeout=6) as resp:
                data = json.loads(resp.read().decode('utf-8'))
                sha = data.get("sha", "")[:7]
                msg = data.get("commit", {}).get("message", "").split("\n")[0]
                date = data.get("commit", {}).get("author", {}).get("date", "")
                latest_ver = f"dev-{sha}" if sha else "master"
                return {
                    "current_version": cur_ver,
                    "latest_version": latest_ver,
                    "update_available": True,
                    "dev_channel": True,
                    "release_name": f"Master ({sha}): {msg[:50]}",
                    "release_notes": msg,
                    "release_url": f"https://github.com/{repo}/commit/{sha}" if sha else f"https://github.com/{repo}",
                    "published_at": date
                }
        except Exception as e:
            try:
                out = subprocess.check_output(["git", "ls-remote", f"https://github.com/{repo}.git", "HEAD"], text=True, timeout=5)
                sha = out.split()[0][:7] if out else "master"
                return {
                    "current_version": cur_ver,
                    "latest_version": f"dev-{sha}",
                    "update_available": True,
                    "dev_channel": True,
                    "release_name": f"Master ({sha})",
                    "release_notes": "Latest master branch commit",
                    "release_url": f"https://github.com/{repo}",
                    "published_at": ""
                }
            except Exception as e2:
                return {
                    "current_version": cur_ver,
                    "latest_version": cur_ver,
                    "update_available": False,
                    "dev_channel": True,
                    "release_name": "Error fetching dev commit",
                    "release_notes": str(e),
                    "release_url": f"https://github.com/{repo}",
                    "published_at": ""
                }
    else:
        url = f"https://api.github.com/repos/{repo}/releases/latest"
        req = urllib.request.Request(url, headers={"User-Agent": f"HawalPanel/{PANEL_VERSION}"})
        try:
            with urllib.request.urlopen(req, timeout=6) as resp:
                data = json.loads(resp.read().decode('utf-8'))
                tag_name = data.get("tag_name", "")
                latest_ver = tag_name if tag_name.startswith("v") else f"v{tag_name}"
                name = data.get("name", latest_ver)
                body = data.get("body", "")
                html_url = data.get("html_url", f"https://github.com/{repo}/releases")
                pub_date = data.get("published_at", "")

                cur_tuple = parse_semver(cur_ver)
                latest_tuple = parse_semver(latest_ver)
                update_avail = latest_tuple > cur_tuple

                return {
                    "current_version": cur_ver,
                    "latest_version": latest_ver,
                    "update_available": update_avail,
                    "dev_channel": False,
                    "release_name": name,
                    "release_notes": body,
                    "release_url": html_url,
                    "published_at": pub_date
                }
        except Exception as e:
            return {
                "current_version": cur_ver,
                "latest_version": cur_ver,
                "update_available": False,
                "dev_channel": False,
                "release_name": "Error checking release",
                "release_notes": str(e),
                "release_url": f"https://github.com/{repo}/releases",
                "published_at": ""
            }

async def get_panel_version_info(dev=False, force=False):
    global _VERSION_CACHE
    now = time.time()
    if not force and _VERSION_CACHE["data"] and (now - _VERSION_CACHE["ts"] < 600) and (_VERSION_CACHE["dev"] == dev):
        return _VERSION_CACHE["data"]

    data = await asyncio.to_thread(fetch_panel_version_sync, dev)
    if "Error" not in data.get("release_name", ""):
        _VERSION_CACHE = {
            "data": data,
            "ts": now,
            "dev": dev
        }
    return data

def update_panel_sync(dev=False, target_version=None):
    repo = GITHUB_REPO
    install_dir = PANEL_DIR
    if not os.path.exists(install_dir):
        install_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    if dev:
        tar_url = f"https://github.com/{repo}/archive/refs/heads/master.tar.gz"
    else:
        ver = target_version or f"v{PANEL_VERSION.lstrip('v')}"
        if not ver.startswith("v") and not ver.startswith("dev-"):
            ver = f"v{ver}"
        if ver.startswith("dev-") or ver == "master":
            tar_url = f"https://github.com/{repo}/archive/refs/heads/master.tar.gz"
        else:
            tar_url = f"https://github.com/{repo}/archive/refs/tags/{ver}.tar.gz"

    staging_dir = "/tmp/hawal-update-staging"
    tar_path = "/tmp/hawal-update.tar.gz"

    cmd_dl = f"curl -fsSL '{tar_url}' -o '{tar_path}'"
    res = subprocess.run(cmd_dl, shell=True, capture_output=True, text=True, timeout=60)
    if res.returncode != 0:
        if not dev:
            fallback_url = f"https://github.com/{repo}/archive/refs/heads/master.tar.gz"
            res = subprocess.run(f"curl -fsSL '{fallback_url}' -o '{tar_path}'", shell=True, capture_output=True, text=True, timeout=60)
        if res.returncode != 0:
            raise RuntimeError(f"Failed to download update package: {res.stderr.strip() or res.stdout.strip()}")

    subprocess.run(f"rm -rf '{staging_dir}' && mkdir -p '{staging_dir}'", shell=True, check=True)
    res_tar = subprocess.run(f"tar -xzf '{tar_path}' -C '{staging_dir}' --strip-components=1", shell=True, capture_output=True, text=True)
    if res_tar.returncode != 0:
        raise RuntimeError(f"Failed to extract update archive: {res_tar.stderr.strip()}")

    check_files = [
        os.path.join(staging_dir, "server.py"),
        os.path.join(staging_dir, "app", "server.py"),
        os.path.join(staging_dir, "app", "config.py")
    ]
    for cf in check_files:
        if not os.path.exists(cf):
            raise RuntimeError(f"Integrity check failed: missing required file {os.path.basename(cf)}")

    compile_cmd = f"python3 -m py_compile {staging_dir}/server.py {staging_dir}/app/*.py"
    res_compile = subprocess.run(compile_cmd, shell=True, capture_output=True, text=True)
    if res_compile.returncode != 0:
        raise RuntimeError(f"Code validation failed (syntax error): {res_compile.stderr.strip()}")

    cp_cmd = f"cp -r {staging_dir}/* '{install_dir}/'"
    res_cp = subprocess.run(cp_cmd, shell=True, capture_output=True, text=True)
    if res_cp.returncode != 0:
        raise RuntimeError(f"Failed to copy files to {install_dir}: {res_cp.stderr.strip()}")

    for exec_file in ["server.py", "start.sh", "install-panel.sh"]:
        p = os.path.join(install_dir, exec_file)
        if os.path.exists(p):
            try:
                os.chmod(p, 0o755)
            except Exception:
                pass

    subprocess.run(f"rm -rf '{staging_dir}' '{tar_path}'", shell=True)
    return True

class HTTPServer:
    def __init__(self, host=DEFAULT_HOST, port=DEFAULT_PORT):
        self.host = host
        self.port = port
        self.server = None
        self.node_traffic_tracker = {} # node_id -> dict
        self.tunnel_traffic_tracker = {} # tun_id -> dict
        self.tunnel_sample_tracker = {} # tun_id -> float
        self.last_cleanup_time = 0

    async def start(self):
        init_db()
        self.server = await asyncio.start_server(self.handle_client, self.host, self.port)
        print(f"🚀 NexusTunnel Control Panel is listening at http://{self.host}:{self.port}")
        print(f"🔑 Master Admin Token: {MASTER_TOKEN}")
        asyncio.create_task(self.background_traffic_collector())

    async def background_traffic_collector(self):
        # Background task that records time-series samples and performs retention cleanup
        while True:
            try:
                await asyncio.sleep(15)
                now = time.time()
                # Run sample retention cleanup once every 6 hours
                if now - self.last_cleanup_time > 21600:
                    try:
                        clean_old_traffic_samples(retention_days=35)
                        self.last_cleanup_time = now
                    except Exception:
                        pass

                # Sample local master node network traffic from /proc/net/dev directly
                try:
                    rx_total = 0
                    tx_total = 0
                    if os.path.exists("/proc/net/dev"):
                        with open("/proc/net/dev", "r") as f:
                            for line in f:
                                if ":" in line:
                                    parts = line.split(":")
                                    iface = parts[0].strip()
                                    if iface == "lo" or iface.startswith("tun") or iface.startswith("docker"):
                                        continue
                                    stats = parts[1].split()
                                    if len(stats) >= 9:
                                        rx_total += int(stats[0])
                                        tx_total += int(stats[8])

                        nodes = list_nodes()
                        master_node = next((n for n in nodes if n.get("role") == "iran"), None)
                        if master_node:
                            m_id = master_node["id"]
                            prev = self.node_traffic_tracker.get(m_id)
                            if prev:
                                elapsed = max(1.0, now - prev["last_time"])
                                delta_rx = max(0, rx_total - prev["rx"])
                                delta_tx = max(0, tx_total - prev["tx"])
                                r_in = (delta_rx * 8.0) / (elapsed * 1_000_000.0)
                                r_out = (delta_tx * 8.0) / (elapsed * 1_000_000.0)
                                if r_in > 10000.0: r_in = 0.0
                                if r_out > 10000.0: r_out = 0.0

                                prev["accum_rx"] = prev.get("accum_rx", 0) + delta_rx
                                prev["accum_tx"] = prev.get("accum_tx", 0) + delta_tx

                                if now - prev.get("last_sample_time", 0) >= 30:
                                    record_traffic_sample("node", m_id, prev["accum_rx"], prev["accum_tx"], r_in, r_out)
                                    prev["last_sample_time"] = now
                                    prev["accum_rx"] = 0
                                    prev["accum_tx"] = 0

                                prev["last_time"] = now
                                prev["rx"] = rx_total
                                prev["tx"] = tx_total
                                update_node_heartbeat(
                                    m_id, master_node.get("ip", "127.0.0.1"),
                                    master_node.get("cpu_percent", 0),
                                    master_node.get("ram_used_mb", 0),
                                    master_node.get("ram_total_mb", 0),
                                    master_node.get("uptime_seconds", 0),
                                    net_rx_bytes=rx_total, net_tx_bytes=tx_total,
                                    rate_in_mbps=r_in, rate_out_mbps=r_out
                                )
                            else:
                                self.node_traffic_tracker[m_id] = {
                                    "last_time": now, "rx": rx_total, "tx": tx_total,
                                    "last_sample_time": now, "accum_rx": 0, "accum_tx": 0
                                }
                except Exception:
                    pass

                # Sample local tunnel accounting counters from iptables HAWAL_ACCT_IN / HAWAL_ACCT_OUT
                try:
                    ports_in, ports_out = self._read_kernel_acct()
                    if ports_in or ports_out:
                        tunnels = list_tunnels()
                        tunnel_updated = False
                        for tun in tunnels:
                            tun_id = tun["id"]
                            fwd_ports = []
                            for rule in tun.get("ports", []):
                                try:
                                    p_str = str(rule).split("=")[0].split(":")[-1].strip()
                                    fwd_ports.append(int(p_str))
                                except Exception:
                                    pass
                            target_ports = fwd_ports if fwd_ports else [tun.get("core_port")]
                            cur_in = sum(ports_in.get(p, 0) for p in target_ports)
                            cur_out = sum(ports_out.get(p, 0) for p in target_ports)

                            prev_t = self.tunnel_traffic_tracker.get(tun_id)
                            if prev_t:
                                elapsed = max(1.0, now - prev_t["time"])
                                delta_in = max(0, cur_in - prev_t["raw_in"])
                                delta_out = max(0, cur_out - prev_t["raw_out"])
                                if delta_in > 0 or delta_out > 0:
                                    update_tunnel_traffic(tun_id, delta_in, delta_out)
                                    tunnel_updated = True
                                    r_in = (delta_in * 8.0) / (elapsed * 1_000_000.0)
                                    r_out = (delta_out * 8.0) / (elapsed * 1_000_000.0)
                                    record_traffic_sample("tunnel", tun_id, delta_in, delta_out, r_in, r_out)
                                prev_t["time"] = now
                                prev_t["raw_in"] = cur_in
                                prev_t["raw_out"] = cur_out
                            else:
                                self.tunnel_traffic_tracker[tun_id] = {
                                    "time": now,
                                    "raw_in": cur_in,
                                    "raw_out": cur_out
                                }
                        if tunnel_updated:
                            await broadcast_ws({"event": "tunnel_updated"})
                except Exception:
                    pass
            except Exception:
                await asyncio.sleep(5)

    def _read_kernel_acct(self):
        bytes_in = {}
        bytes_out = {}
        try:
            res_in = subprocess.run(["iptables", "-nxvL", "HAWAL_ACCT_IN"], capture_output=True, text=True, timeout=2)
            if res_in.returncode == 0:
                for line in res_in.stdout.splitlines():
                    parts = line.split()
                    if len(parts) >= 2 and parts[1].isdigit():
                        m = re.search(r'dpt:(\d+)', line)
                        if m:
                            p = int(m.group(1))
                            bytes_in[p] = bytes_in.get(p, 0) + int(parts[1])
        except Exception:
            pass

        try:
            res_out = subprocess.run(["iptables", "-nxvL", "HAWAL_ACCT_OUT"], capture_output=True, text=True, timeout=2)
            if res_out.returncode == 0:
                for line in res_out.stdout.splitlines():
                    parts = line.split()
                    if len(parts) >= 2 and parts[1].isdigit():
                        m = re.search(r'spt:(\d+)', line)
                        if m:
                            p = int(m.group(1))
                            bytes_out[p] = bytes_out.get(p, 0) + int(parts[1])
        except Exception:
            pass

        return bytes_in, bytes_out

    async def handle_client(self, reader, writer):
        try:
            req_line = await asyncio.wait_for(reader.readline(), timeout=HTTP_READ_TIMEOUT)
            if not req_line:
                writer.close()
                return
            
            req_parts = req_line.decode('utf-8', errors='ignore').strip().split()
            if len(req_parts) < 2:
                writer.close()
                return

            method, path = req_parts[0], req_parts[1]
            headers = {}
            while True:
                line = await asyncio.wait_for(reader.readline(), timeout=HTTP_READ_TIMEOUT)
                if not line or line == b"\r\n":
                    break
                header_line = line.decode('utf-8', errors='ignore').strip()
                if ":" in header_line:
                    k, v = header_line.split(":", 1)
                    headers[k.strip().lower()] = v.strip()

            parsed_url = urllib.parse.urlparse(path)
            raw_path = parsed_url.path
            query = urllib.parse.parse_qs(parsed_url.query)

            # Check WebSocket Upgrade
            if headers.get("upgrade", "").lower() == "websocket":
                sec_key = headers.get("sec-websocket-key")
                session_token = get_session_token_from_headers(headers)
                query_token = query.get("token", [""])[0]
                if not (validate_session(session_token) or validate_session(query_token) or query_token == MASTER_TOKEN):
                    writer.write(b"HTTP/1.1 401 Unauthorized\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
                    writer.close()
                    return
                if sec_key:
                    writer.write(make_ws_handshake_response(sec_key))
                    await writer.drain()
                    if raw_path == "/ws":
                        await self.handle_ws_dashboard(reader, writer)
                    return

            # Read Body if POST/PUT
            body = b""
            content_length = int(headers.get("content-length", 0))
            if content_length < 0 or content_length > MAX_HTTP_BODY_BYTES:
                self.send_json(writer, {"error": "request body too large"}, status=413)
                return
            if content_length > 0:
                body = await asyncio.wait_for(
                    reader.readexactly(content_length), timeout=HTTP_READ_TIMEOUT
                )

            # Route HTTP Requests
            await self.route_request(method, raw_path, query, headers, body, writer)

        except Exception as e:
            try:
                self.send_json(writer, {"error": str(e)}, status=500)
            except:
                pass
        finally:
            try:
                await writer.drain()
            except:
                pass
            try:
                writer.close()
                await writer.wait_closed()
            except:
                pass

    async def route_request(self, method, path, query, headers, body, writer):
        session_token = get_session_token_from_headers(headers)
        auth_hdr = headers.get("authorization", "") or headers.get("Authorization", "")
        bearer_token = auth_hdr.replace("Bearer ", "").strip() if "Bearer " in auth_hdr else ""
        query_token = query.get("token", [""])[0]
        is_authenticated = validate_session(session_token) or (bearer_token and bearer_token == MASTER_TOKEN) or (query_token and query_token == MASTER_TOKEN)

        # 0. Auth API Endpoints
        if method == "GET" and path == "/api/auth/status":
            self.send_json(writer, {
                "is_first_time": is_first_time_setup(),
                "authenticated": is_authenticated
            })
            return

        if method == "POST" and path == "/api/auth/setup":
            try:
                data = json.loads(body.decode('utf-8'))
                username = data.get("username", "")
                password = data.get("password", "")
                ok, token, err = setup_admin(username, password)
                if not ok:
                    self.send_json(writer, {"error": err}, status=400)
                    return
                cookie = f"hawal_session={token}; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000"
                self.send_json(writer, {"success": True}, set_cookie=cookie)
            except Exception as e:
                self.send_json(writer, {"error": str(e)}, status=500)
            return

        if method == "POST" and path == "/api/auth/login":
            try:
                data = json.loads(body.decode('utf-8'))
                username = data.get("username", "")
                password = data.get("password", "")
                ok, token, err = authenticate(username, password)
                if not ok:
                    self.send_json(writer, {"error": err}, status=401)
                    return
                cookie = f"hawal_session={token}; Path=/; HttpOnly; SameSite=Lax; Max-Age=7776000"
                self.send_json(writer, {"success": True, "token": token}, set_cookie=cookie)
            except Exception as e:
                self.send_json(writer, {"error": str(e)}, status=500)
            return

        if (method == "POST" and path == "/api/auth/logout") or (method == "GET" and path == "/logout"):
            if session_token:
                invalidate_session(session_token)
            clear_cookie = "hawal_session=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0"
            if method == "GET":
                self.send_redirect(writer, "/login", set_cookie=clear_cookie)
            else:
                self.send_json(writer, {"success": True}, set_cookie=clear_cookie)
            return

        # 1. Login Page UI
        if method == "GET" and path == "/login":
            if is_authenticated:
                self.send_redirect(writer, "/")
                return
            await self.serve_template("login.html", writer)
            return

        # 2. Static Files (Public)
        if method in ["GET", "HEAD"] and path in ["/theme.css", "/app.css"]:
            file_path = os.path.join(STATIC_DIR, "css", path.lstrip("/"))
            await self.serve_static_file(file_path, writer, method=method)
            return

        if method in ["GET", "HEAD"] and path.startswith("/static/"):
            rel_path = path.replace("/static/", "", 1)
            file_path = os.path.join(STATIC_DIR, rel_path)
            await self.serve_static_file(file_path, writer, method=method)
            return

        # 3. One-Line Node Installer Script (Public)
        if method == "GET" and path in ["/install", "/install.sh"]:
            await self.serve_node_installer(query, headers, writer)
            return

        # 4. Agent endpoints & public status endpoints
        if path.startswith("/api/agent/") or (method == "GET" and path == "/api/panel/version"):
            pass
        elif path.startswith("/api/"):
            if not is_authenticated:
                self.send_json(writer, {"error": "Unauthorized. Please log in."}, status=401)
                return

        # 5. Dashboard UI & SPA Pages (Protected)
        PANEL_ROUTES = {
            "/", "/index.html", "/dashboard",
            "/nodes", "/node",
            "/tunnels", "/tunnel",
            "/ping", "/diagnostics",
            "/logs", "/log",
            "/settings"
        }
        if method == "GET" and path in PANEL_ROUTES:
            if not is_authenticated:
                self.send_redirect(writer, "/login")
                return
            await self.serve_template("index.html", writer)
            return

        # 3. REST API: Node Management
        if method == "GET" and path == "/api/nodes":
            nodes = list_nodes()
            self.send_json(writer, {"nodes": nodes})
            return

        if method == "GET" and path == "/api/logs":
            node_id = query.get("node_id", [""])[0]
            source = query.get("source", [""])[0]
            self.send_json(writer, {"logs": get_log_snapshots(node_id or None, source or None)})
            return

        if method == "DELETE" and path == "/api/logs":
            node_id = query.get("node_id", [""])[0]
            source = query.get("source", [""])[0]
            deleted = delete_log_snapshots(node_id or None, source or None)
            self.send_json(writer, {"success": True, "deleted": deleted})
            return

        if method == "POST" and path == "/api/agent/logs":
            auth_header = headers.get("authorization", "") or headers.get("Authorization", "")
            token = auth_header.replace("Bearer ", "").strip()
            node = get_node_by_token(token)
            if not node:
                self.send_json(writer, {"error": "unauthorized"}, status=401)
                return
            payload = json.loads(body.decode("utf-8"))
            snapshots = payload.get("snapshots", {})
            if not isinstance(snapshots, dict):
                self.send_json(writer, {"error": "invalid snapshots"}, status=400)
                return
            save_log_snapshots(node["id"], snapshots)
            self.send_json(writer, {"success": True})
            return

        if method == "POST" and path == "/api/nodes":
            data = json.loads(body.decode('utf-8'))
            node_id = f"node_{secrets.token_hex(4)}"
            name = data.get("name", "New Node")
            ip = data.get("ip", "127.0.0.1")
            role = data.get("role", "kharej")
            user_cc = data.get("country_code", "").strip().upper()
            token = secrets.token_hex(16)
            
            # Resolve GeoIP location and country flag
            from app.geoip import resolve_geoip, get_country_flag, COUNTRY_NAMES_FA
            geo = resolve_geoip(ip)

            # Heuristics & user overrides
            n_lower = name.lower()
            if user_cc and user_cc != "AUTO" and user_cc in COUNTRY_NAMES_FA:
                geo["country_code"] = user_cc
                geo["country_name"] = COUNTRY_NAMES_FA[user_cc]
                geo["flag"] = get_country_flag(user_cc)
            elif "germany" in n_lower or "آلمان" in name or n_lower == "de":
                geo["country_code"] = "DE"
                geo["country_name"] = "آلمان"
                geo["flag"] = "🇩🇪"
                geo["city"] = geo.get("city") or "Frankfurt"
            elif "netherland" in n_lower or "holland" in n_lower or "هلند" in name or n_lower == "nl":
                geo["country_code"] = "NL"
                geo["country_name"] = "هلند"
                geo["flag"] = "🇳🇱"
                geo["city"] = geo.get("city") or "Amsterdam"
            elif "finland" in n_lower or "فنلاند" in name or n_lower == "fi":
                geo["country_code"] = "FI"
                geo["country_name"] = "فنلاند"
                geo["flag"] = "🇫🇮"
                geo["city"] = geo.get("city") or "Helsinki"
            elif role == "iran" or "iran" in n_lower or "ایران" in name or n_lower == "ir":
                geo["flag"] = "🇮🇷"
                geo["country_name"] = "ایران"
                geo["country_code"] = "IR"
                geo["city"] = geo.get("city") or "تهران"

            save_node(node_id, name, ip, role, token, 
                      country_code=geo.get("country_code", "GLOBAL"),
                      country_name=geo.get("country_name", "خارج"),
                      flag=geo.get("flag", "🌐"),
                      city=geo.get("city", ""))
            
            self.send_json(writer, {
                "node_id": node_id, 
                "token": token, 
                "name": name, 
                "role": role,
                "flag": geo.get("flag", "🌐"),
                "country_name": geo.get("country_name", "خارج")
            })
            await broadcast_ws({"event": "node_updated"})
            return

        if method == "PUT" and path.startswith("/api/nodes/"):
            node_id = path.split("/")[-1]
            n = get_node(node_id)
            if not n:
                self.send_json(writer, {"error": "Node not found"}, status=404)
                return
            data = json.loads(body.decode('utf-8'))
            name = data.get("name", n["name"])
            role = data.get("role", n["role"])
            user_cc = data.get("country_code", "").strip().upper()
            from app.geoip import COUNTRY_NAMES_FA, get_country_flag
            flag = n.get("flag", "🌐")
            country_name = n.get("country_name", "خارج")
            country_code = n.get("country_code", "GLOBAL")
            city = data.get("city", n.get("city", ""))

            n_lower = name.lower()
            if user_cc and user_cc != "AUTO" and user_cc in COUNTRY_NAMES_FA:
                country_code = user_cc
                country_name = COUNTRY_NAMES_FA[user_cc]
                flag = get_country_flag(user_cc)
            elif "germany" in n_lower or "آلمان" in name or n_lower == "de":
                country_code = "DE"
                country_name = "آلمان"
                flag = "🇩🇪"
                city = city or "Frankfurt"
            elif "netherland" in n_lower or "holland" in n_lower or "هلند" in name or n_lower == "nl":
                country_code = "NL"
                country_name = "هلند"
                flag = "🇳🇱"
                city = city or "Amsterdam"
            elif "finland" in n_lower or "فنلاند" in name or n_lower == "fi":
                country_code = "FI"
                country_name = "فنلاند"
                flag = "🇫🇮"
                city = city or "Helsinki"

            from app.db import get_db
            with get_db() as conn:
                conn.execute("""
                UPDATE nodes SET name=?, role=?, country_code=?, country_name=?, flag=?, city=?
                WHERE id=?
                """, (name, role, country_code, country_name, flag, city, node_id))
                conn.commit()

            self.send_json(writer, {"success": True, "node_id": node_id})
            await broadcast_ws({"event": "node_updated"})
            return

        if method == "DELETE" and path.startswith("/api/nodes/"):
            node_id = path.split("/")[-1]
            delete_node(node_id)
            self.send_json(writer, {"success": True, "deleted": node_id})
            await broadcast_ws({"event": "node_updated"})
            return

        # 4. REST API: Tunnel Management
        if method == "GET" and path == "/api/tunnels":
            tunnels = list_tunnels()
            self.send_json(writer, {"tunnels": tunnels})
            return

        if method == "POST" and path == "/api/tunnels":
            data = json.loads(body.decode('utf-8'))
            tunnel_id = f"tun_{secrets.token_hex(4)}"
            name = data.get("name", "New Tunnel")
            core_type = data.get("core_type", "hawal")
            server_node_id = data.get("server_node_id")
            client_node_id = data.get("client_node_id")
            core_port = int(data.get("core_port", 3090))
            if core_type == "paqet":
                default_transport = "kcp"
            elif core_type == "hawal":
                default_transport = "stealth"
            elif core_type == "gost":
                default_transport = "tls"
            else:
                default_transport = "ws"
            transport = data.get("transport", default_transport)
            ports = data.get("ports", ["443=127.0.0.1:443"])
            # Reserved ports check (prevent forwarding panel port 9090, rex 7444, ssh 22)
            for p in ports:
                lp = str(p).split("=")[0].split(":")[0].strip()
                if lp.isdigit() and int(lp) in {22, 9090, 7444}:
                    self.send_json(writer, {"error": f"پورت فوروارد {lp} پورت پنل یا مدیریت سرور است و امکان هدایت آن وجود ندارد."}, status=400)
                    return

            # Port conflict check
            valid, err = validate_tunnel_ports(core_port, server_node_id)
            if not valid:
                self.send_json(writer, {"error": err}, status=400)
                return

            token = data.get("token") or secrets.token_hex(8)
            kcp_mode = str(data.get("kcp_mode", "normal")).strip().lower()
            if kcp_mode not in ("normal", "fast", "fast2", "fast3", "manual"):
                kcp_mode = "normal"
            save_tunnel(tunnel_id, name, server_node_id, client_node_id, core_port, transport, ports, token, status='running', core_type=core_type, kcp_mode=kcp_mode)
            self.send_json(writer, {"tunnel_id": tunnel_id, "token": token, "status": "running", "core_type": core_type, "kcp_mode": kcp_mode})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "GET" and "/api/tunnels/" in path and path.endswith("/docker"):
            tunnel_id = path.split("/")[3]
            tunnel = get_tunnel(tunnel_id)
            if not tunnel:
                self.send_json(writer, {"error": "Tunnel not found"}, status=404)
                return
            server_node = get_node(tunnel["server_node_id"])
            s_ip = server_node["ip"] if server_node else "127.0.0.1"
            server_compose, server_toml = generate_docker_compose(tunnel, "server", s_ip)
            client_compose, client_toml = generate_docker_compose(tunnel, "client", s_ip)
            self.send_json(writer, {
                "server_compose": server_compose,
                "server_toml": server_toml,
                "client_compose": client_compose,
                "client_toml": client_toml
            })
            return

        if method == "GET" and path == "/api/settings":
            from app.config import load_settings
            self.send_json(writer, {"settings": load_settings()})
            return

        if method == "POST" and path == "/api/settings":
            from app.config import load_settings, save_settings
            current = load_settings()
            new_data = json.loads(body.decode('utf-8'))
            current.update(new_data)
            save_settings(current)
            self.send_json(writer, {"success": True, "settings": current})
            await broadcast_ws({"event": "settings_updated"})
            return

        if method == "GET" and path == "/api/panel/version":
            dev = query.get("dev", ["0"])[0] in ["1", "true", "True"]
            force = query.get("force", ["0"])[0] in ["1", "true", "True"]
            info = await get_panel_version_info(dev=dev, force=force)
            self.send_json(writer, info)
            return

        if method == "POST" and path == "/api/panel/update":
            data = {}
            if body:
                try:
                    data = json.loads(body.decode('utf-8'))
                except Exception:
                    pass
            dev = bool(data.get("dev", False))
            target_version = data.get("version")

            try:
                await asyncio.to_thread(update_panel_sync, dev=dev, target_version=target_version)
            except Exception as e:
                self.send_json(writer, {"error": f"Update failed: {str(e)}"}, status=500)
                return

            self.send_json(writer, {
                "status": "ok",
                "message": "Panel successfully updated. Restarting panel service..."
            })

            async def schedule_restart():
                await asyncio.sleep(1)
                try:
                    subprocess.Popen(["systemctl", "restart", "hawal-panel"])
                except Exception:
                    pass
            asyncio.create_task(schedule_restart())
            return


        if method == "GET" and "/api/tunnels/" in path and path.endswith("/logs"):
            tunnel_id = path.split("/")[3]
            log_file = f"/opt/hawal/logs/{tunnel_id}.log"
            lines = []
            if os.path.exists(log_file):
                try:
                    with open(log_file, "r", errors="ignore") as f:
                        all_lines = f.readlines()
                        lines = [line.strip() for line in all_lines[-60:] if line.strip()]
                except Exception as e:
                    lines = [f"Error reading log file: {e}"]
            else:
                lines = ["[info] Waiting for tunnel supervisor to emit log entries..."]
            self.send_json(writer, {"tunnel_id": tunnel_id, "logs": lines})
            return

        if method == "PUT" and path.startswith("/api/tunnels/"):
            tunnel_id = path.split("/")[3]
            data = json.loads(body.decode('utf-8'))
            name = data.get("name")
            core_type = data.get("core_type", "hawal")
            core_port = int(data.get("core_port", 3090))
            if core_type == "paqet":
                default_transport = "kcp"
            elif core_type == "hawal":
                default_transport = "stealth"
            elif core_type == "gost":
                default_transport = "tls"
            else:
                default_transport = "ws"
            transport = data.get("transport", default_transport)
            ports = data.get("ports", [])
            kcp_mode = data.get("kcp_mode")
            if kcp_mode is not None:
                kcp_mode = str(kcp_mode).strip().lower()
                if kcp_mode not in ("normal", "fast", "fast2", "fast3", "manual"):
                    kcp_mode = "normal"

            for p in ports:
                lp = str(p).split("=")[0].split(":")[0].strip()
                if lp.isdigit() and int(lp) in {22, 9090, 7444}:
                    self.send_json(writer, {"error": f"پورت فوروارد {lp} پورت پنل یا مدیریت سرور است و امکان هدایت آن وجود ندارد."}, status=400)
                    return

            t = get_tunnel(tunnel_id)
            if not t:
                self.send_json(writer, {"error": "Tunnel not found"}, status=404)
                return

            valid, err = validate_tunnel_ports(core_port, t.get("server_node_id"), current_tunnel_id=tunnel_id)
            if not valid:
                self.send_json(writer, {"error": err}, status=400)
                return

            update_tunnel(tunnel_id, name or t["name"], core_port, transport, ports, core_type=core_type, kcp_mode=kcp_mode)
            request_tunnel_restart(tunnel_id)
            self.send_json(writer, {"success": True, "tunnel_id": tunnel_id})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "POST" and path == "/api/tunnels/traffic":
            data = json.loads(body.decode('utf-8'))
            now = time.time()
            for rep in data.get("reports", []):
                tun_id = rep.get("tunnel_id")
                b_in = int(rep.get("bytes_in", 0) or 0)
                b_out = int(rep.get("bytes_out", 0) or 0)
                if not tun_id or (b_in == 0 and b_out == 0):
                    continue
                update_tunnel_traffic(tun_id, b_in, b_out)

                prev_tun = self.tunnel_traffic_tracker.get(tun_id, {"time": now, "accum_in": 0, "accum_out": 0})
                elapsed = max(1.0, now - prev_tun.get("time", now))
                r_in = (b_in * 8.0) / (elapsed * 1_000_000.0)
                r_out = (b_out * 8.0) / (elapsed * 1_000_000.0)

                accum_in = prev_tun.get("accum_in", 0) + b_in
                accum_out = prev_tun.get("accum_out", 0) + b_out

                last_sample = self.tunnel_sample_tracker.get(tun_id, 0)
                if now - last_sample >= 30:
                    record_traffic_sample("tunnel", tun_id, accum_in, accum_out, r_in, r_out)
                    self.tunnel_sample_tracker[tun_id] = now
                    accum_in = 0
                    accum_out = 0

                self.tunnel_traffic_tracker[tun_id] = {
                    "time": now,
                    "accum_in": accum_in,
                    "accum_out": accum_out
                }

            self.send_json(writer, {"success": True})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "GET" and path == "/api/metrics/bandwidth":
            t_type = query.get("target_type", ["all"])[0]
            t_id = query.get("target_id", ["all"])[0]
            t_range = query.get("range", ["24h"])[0]
            history = get_traffic_history(target_type=t_type, target_id=t_id, time_range=t_range)
            self.send_json(writer, history)
            return

        if method == "POST" and "/api/tunnels/" in path and path.endswith("/restart"):
            tunnel_id = path.split("/")[3]
            if not get_tunnel(tunnel_id):
                self.send_json(writer, {"error": "Tunnel not found"}, status=404)
                return
            request_tunnel_restart(tunnel_id)
            self.send_json(writer, {"success": True, "tunnel_id": tunnel_id})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "POST" and path == "/api/agents/restart":
            count = request_all_agents_restart()
            self.send_json(writer, {"success": True, "nodes": count})
            await broadcast_ws({"event": "node_updated"})
            return

        if method == "POST" and "/api/tunnels/" in path and path.endswith("/reset-traffic"):
            tunnel_id = path.split("/")[3]
            t = get_tunnel(tunnel_id)
            if not t:
                self.send_json(writer, {"error": "Tunnel not found"}, status=404)
                return
            set_tunnel_absolute_traffic(tunnel_id, 0, 0)
            
            fwd_ports = []
            for rule in t.get("ports", []):
                try:
                    p_str = str(rule).split("=")[0].split(":")[-1].strip()
                    fwd_ports.append(int(p_str))
                except Exception:
                    pass
            target_ports = fwd_ports if fwd_ports else [t.get("core_port")]
            
            # Reset iptables rule counters by delete & re-add
            for p in target_ports:
                for proto in ("tcp", "udp"):
                    try:
                        p_str = str(p)
                        while subprocess.run(["iptables", "-D", "HAWAL_ACCT_IN", "-p", proto, "--dport", p_str], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                            pass
                        subprocess.run(["iptables", "-A", "HAWAL_ACCT_IN", "-p", proto, "--dport", p_str], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                        while subprocess.run(["iptables", "-D", "HAWAL_ACCT_OUT", "-p", proto, "--sport", p_str], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
                            pass
                        subprocess.run(["iptables", "-A", "HAWAL_ACCT_OUT", "-p", proto, "--sport", p_str], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                    except Exception:
                        pass
            
            self.tunnel_traffic_tracker[tunnel_id] = {
                "time": time.time(),
                "raw_in": 0,
                "raw_out": 0
            }
            self.send_json(writer, {"success": True, "tunnel_id": tunnel_id})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "POST" and path == "/api/tunnels/reset-all-traffic":
            tunnels = list_tunnels()
            for t in tunnels:
                set_tunnel_absolute_traffic(t["id"], 0, 0)
            self.tunnel_traffic_tracker.clear()
            try:
                subprocess.run(["iptables", "-Z", "HAWAL_ACCT_IN"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                subprocess.run(["iptables", "-Z", "HAWAL_ACCT_OUT"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            except Exception:
                pass
            self.send_json(writer, {"success": True})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "POST" and "/api/tunnels/" in path and path.endswith("/test"):
            tunnel_id = path.split("/")[3]
            t = get_tunnel(tunnel_id)
            if not t:
                self.send_json(writer, {"error": "Tunnel not found"}, status=404)
                return

            client_node = get_node(t.get("client_node_id"))
            target_ip = client_node["ip"] if client_node else "167.172.102.14"

            # Parse forwarded port
            ports = t.get("ports", [])
            test_port = None
            for r in ports:
                rule_str = str(r).strip()
                left = rule_str.split("=")[0].strip()
                if ":" in left:
                    left = left.split(":")[-1]
                try:
                    p = int(left)
                    if 1 <= p <= 65535:
                        test_port = p
                        break
                except:
                    pass
            
            if not test_port:
                test_port = t.get("core_port", 3090)

            # 1. Verify local forward port is active
            local_ok = False
            try:
                lsock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                lsock.settimeout(1.0)
                lsock.connect(("127.0.0.1", test_port))
                lsock.close()
                local_ok = True
            except:
                pass

            # 2. Measure actual inter-server network RTT (Iran -> Germany)
            latencies = []
            probe_ports = [7443, int(t.get("core_port", 3090)), 22, 80]
            
            for _ in range(4):
                t0 = time.perf_counter()
                connected = False
                for p in probe_ports:
                    try:
                        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                        sock.settimeout(2.5)
                        sock.connect((target_ip, p))
                        t1 = time.perf_counter()
                        latencies.append((t1 - t0) * 1000)
                        sock.close()
                        connected = True
                        break
                    except:
                        continue
                time.sleep(0.04)

            if latencies:
                avg_ms = round(sum(latencies) / len(latencies), 1)
                min_ms = round(min(latencies), 1)
                max_ms = round(max(latencies), 1)
                loss = round((4 - len(latencies)) / 4 * 100)
                self.send_json(writer, {
                    "success": True,
                    "tunnel_id": tunnel_id,
                    "tested_port": test_port,
                    "target_ip": target_ip,
                    "latency_avg_ms": avg_ms,
                    "latency_min_ms": min_ms,
                    "latency_max_ms": max_ms,
                    "packet_loss": loss,
                    "local_port_active": local_ok,
                    "status": "healthy" if avg_ms < 250 else "high_latency"
                })
            else:
                self.send_json(writer, {
                    "success": False,
                    "tunnel_id": tunnel_id,
                    "tested_port": test_port,
                    "target_ip": target_ip,
                    "latency_avg_ms": None,
                    "packet_loss": 100,
                    "status": "unreachable",
                    "error": f"سرور مقصد ({target_ip}) پاسخ نداد"
                })
            return

        if method == "DELETE" and path.startswith("/api/tunnels/"):
            tunnel_id = path.split("/")[-1]
            delete_tunnel(tunnel_id)
            self.send_json(writer, {"success": True, "deleted": tunnel_id})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        if method == "POST" and "/api/tunnels/" in path and path.endswith("/status"):
            tunnel_id = path.split("/")[3]
            data = json.loads(body.decode('utf-8'))
            status = data.get("status", "stopped")
            set_tunnel_status(tunnel_id, status)
            self.send_json(writer, {"success": True, "status": status})
            await broadcast_ws({"event": "tunnel_updated"})
            return

        # 5. REST API: Ping & Quality Diagnostic
        if method == "POST" and path == "/api/ping":
            data = json.loads(body.decode('utf-8'))
            target_ip = data.get("target_ip")
            source_node_id = data.get("source_node_id", "panel")
            target_node_id = data.get("target_node_id", "target")

            res = await run_ping(target_ip, count=4)
            if res.get("success"):
                record_ping(source_node_id, target_node_id, res["avg_ms"], res["min_ms"], res["max_ms"], res["packet_loss"])
            self.send_json(writer, res)
            await broadcast_ws({"event": "ping_completed", "data": res})
            return

        if method == "GET" and path in ["/api/pings/latest", "/api/ping/history"]:
            pings = get_latest_pings()
            self.send_json(writer, {"pings": pings, "history": pings})
            return

        # 6. REST API: Agent Heartbeat & Remote Node Synchronization
        if method == "POST" and path == "/api/agent/heartbeat":
            auth_header = headers.get("authorization", "") or headers.get("Authorization", "")
            auth_token = auth_header.replace("Bearer ", "").strip() or headers.get("x-node-token")
            node = get_node_by_token(auth_token)
            if not node:
                self.send_json(writer, {"error": "Unauthorized node token"}, status=401)
                return
            
            data = json.loads(body.decode('utf-8'))
            client_ip = headers.get("x-forwarded-for") or data.get("public_ip") or writer.get_extra_info('peername')[0]
            
            now = time.time()
            net_rx = data.get("net_rx_bytes")
            net_tx = data.get("net_tx_bytes")
            r_in = 0.0
            r_out = 0.0
            node_id = node["id"]

            if net_rx is not None and net_tx is not None:
                rx_val = int(net_rx)
                tx_val = int(net_tx)
                prev = self.node_traffic_tracker.get(node_id)
                if prev:
                    elapsed = max(1.0, now - prev["last_time"])
                    delta_rx = max(0, rx_val - prev["rx"])
                    delta_tx = max(0, tx_val - prev["tx"])
                    r_in = (delta_rx * 8.0) / (elapsed * 1_000_000.0)
                    r_out = (delta_tx * 8.0) / (elapsed * 1_000_000.0)
                    if r_in > 10000.0: r_in = 0.0
                    if r_out > 10000.0: r_out = 0.0

                    prev["accum_rx"] = prev.get("accum_rx", 0) + delta_rx
                    prev["accum_tx"] = prev.get("accum_tx", 0) + delta_tx

                    if now - prev.get("last_sample_time", 0) >= 30:
                        record_traffic_sample("node", node_id, prev["accum_rx"], prev["accum_tx"], r_in, r_out)
                        prev["last_sample_time"] = now
                        prev["accum_rx"] = 0
                        prev["accum_tx"] = 0

                    prev["last_time"] = now
                    prev["rx"] = rx_val
                    prev["tx"] = tx_val
                else:
                    self.node_traffic_tracker[node_id] = {
                        "last_time": now, "rx": rx_val, "tx": tx_val,
                        "last_sample_time": now, "accum_rx": 0, "accum_tx": 0
                    }

            latency_val = data.get("latency_ms") or data.get("ping_ms")
            if latency_val is not None:
                try:
                    latency_val = float(latency_val)
                except:
                    latency_val = None

            if latency_val is None:
                try:
                    sock = writer.get_extra_info('socket')
                    if sock and hasattr(socket, 'TCP_INFO'):
                        raw = sock.getsockopt(socket.SOL_TCP, socket.TCP_INFO, 104)
                        if len(raw) >= 72:
                            rtt_us = struct.unpack_from('I', raw, 68)[0]
                            if rtt_us > 0:
                                latency_val = round(rtt_us / 1000.0, 1)
                except Exception:
                    pass

            update_node_heartbeat(
                node["id"], client_ip,
                data.get("cpu_percent", 0),
                data.get("ram_used_mb", 0),
                data.get("ram_total_mb", 0),
                data.get("uptime_seconds", 0),
                net_rx_bytes=net_rx,
                net_tx_bytes=net_tx,
                rate_in_mbps=r_in,
                rate_out_mbps=r_out,
                latency_ms=latency_val
            )

            # Return all active tunnel configs for this node
            all_tunnels = list_tunnels()
            assigned_configs = []
            for t in all_tunnels:
                if t["status"] == "running" and t.get("core_type") == "backhaul":
                    if t["server_node_id"] == node["id"]:
                        cfg = generate_server_config(t)
                        assigned_configs.append({"tunnel_id": t["id"], "role": "server", "config": cfg, "restart_nonce": t.get("restart_nonce", 0)})
                    elif t["client_node_id"] == node["id"]:
                        server_node = get_node(t["server_node_id"])
                        server_ip = server_node["ip"] if server_node else "127.0.0.1"
                        cfg = generate_client_config(t, server_ip)
                        assigned_configs.append({"tunnel_id": t["id"], "role": "client", "config": cfg, "restart_nonce": t.get("restart_nonce", 0)})

            self.send_json(writer, {"status": "ok", "tunnels": assigned_configs})
            await broadcast_ws({"event": "node_heartbeat", "node_id": node["id"]})
            return

        if method == "GET" and path == "/api/agent/sync":
            auth_header = headers.get("authorization", "") or headers.get("Authorization", "")
            token = auth_header.replace("Bearer ", "").strip() or headers.get("x-node-token", "")
            node = get_node_by_token(token)
            if not node:
                self.send_json(writer, {"error": "unauthorized"}, status=401)
                return

            tunnels = list_tunnels()
            node_configs = []

            for t in tunnels:
                if t["status"] != "running":
                    continue
                
                core_type = t.get("core_type", "hawal")

                # Paqet exposes forwarding listeners on its client. Hawal's
                # server_node is the Iran/entry node, so Paqet roles must be
                # mapped in reverse: Iran=client and foreign node=server.
                if core_type == "paqet":
                    if t["client_node_id"] == node["id"]:
                        from app.paqet_engine import generate_paqet_server_config
                        node_configs.append({
                            "tunnel_id": t["id"],
                            "core_type": "paqet",
                            "role": "server",
                            "core_port": t.get("core_port", 8888),
                            "ports": t.get("ports", []),
                            "yaml": generate_paqet_server_config(t), "restart_nonce": t.get("restart_nonce", 0)
                        })
                    elif t["server_node_id"] == node["id"]:
                        from app.paqet_engine import generate_paqet_client_config
                        paqet_server_node = get_node(t["client_node_id"])
                        paqet_server_ip = paqet_server_node["ip"] if paqet_server_node else "127.0.0.1"
                        node_configs.append({
                            "tunnel_id": t["id"],
                            "core_type": "paqet",
                            "role": "client",
                            "core_port": t.get("core_port", 8888),
                            "ports": t.get("ports", []),
                            "yaml": generate_paqet_client_config(t, paqet_server_ip), "restart_nonce": t.get("restart_nonce", 0)
                        })
                    continue

                if core_type == "gost":
                    if t["client_node_id"] == node["id"]:
                        node_configs.append({
                            "tunnel_id": t["id"], "core_type": "gost", "role": "server",
                            "core_port": t.get("core_port", 8443), "ports": t.get("ports", []),
                            "command": generate_gost_server_command(t), "restart_nonce": t.get("restart_nonce", 0)
                        })
                    elif t["server_node_id"] == node["id"]:
                        gost_server = get_node(t["client_node_id"])
                        gost_server_ip = gost_server["ip"] if gost_server else "127.0.0.1"
                        node_configs.append({
                            "tunnel_id": t["id"], "core_type": "gost", "role": "client",
                            "core_port": t.get("core_port", 8443), "ports": t.get("ports", []),
                            "command": generate_gost_client_command(t, gost_server_ip), "restart_nonce": t.get("restart_nonce", 0)
                        })
                    continue

                if core_type == "hawal":
                    if t["client_node_id"] == node["id"]:
                        from app.hawal_engine import generate_hawal_core_server_config
                        node_configs.append({
                            "tunnel_id": t["id"],
                            "core_type": "hawal",
                            "role": "server",
                            "core_port": t.get("core_port", 8080),
                            "ports": [],
                            "config": generate_hawal_core_server_config(dict(t, ports=[])),
                            "restart_nonce": t.get("restart_nonce", 0)
                        })
                    elif t["server_node_id"] == node["id"]:
                        from app.hawal_engine import generate_hawal_core_client_config
                        hawal_server_node = get_node(t["client_node_id"])
                        hawal_server_ip = hawal_server_node["ip"] if hawal_server_node else "127.0.0.1"
                        node_configs.append({
                            "tunnel_id": t["id"],
                            "core_type": "hawal",
                            "role": "client",
                            "core_port": t.get("core_port", 8080),
                            "ports": t.get("ports", []),
                            "config": generate_hawal_core_client_config(t, hawal_server_ip),
                            "restart_nonce": t.get("restart_nonce", 0)
                        })
                    continue

                if t["server_node_id"] == node["id"]:
                    node_configs.append({
                        "tunnel_id": t["id"],
                        "core_type": "backhaul",
                        "role": "server",
                        "core_port": t.get("core_port", 3096),
                        "ports": t.get("ports", []),
                        "toml": generate_server_config(t), "restart_nonce": t.get("restart_nonce", 0)
                    })

                elif t["client_node_id"] == node["id"]:
                    server_node = get_node(t["server_node_id"])
                    server_ip = server_node["ip"] if server_node else "127.0.0.1"
                    node_configs.append({
                        "tunnel_id": t["id"],
                        "core_type": "backhaul",
                        "role": "client",
                            "core_port": t.get("core_port", 3096),
                            "ports": t.get("ports", []),
                            "toml": generate_client_config(t, server_ip), "restart_nonce": t.get("restart_nonce", 0)
                        })

            self.send_json(writer, {"configs": node_configs, "agent_restart_nonce": node.get("agent_restart_nonce", 0)})
            return

        # Not found fallback
        self.send_json(writer, {"error": "Not Found"}, status=404)

    async def handle_ws_dashboard(self, reader, writer):
        CONNECTED_WS_CLIENTS.add(writer)
        try:
            while True:
                hdr = await reader.read(2)
                if not hdr or len(hdr) < 2:
                    break
                length = hdr[1] & 0x7F
                if length == 126:
                    raw_len = await reader.read(2)
                    length = int.from_bytes(raw_len, 'big')
                elif length == 127:
                    raw_len = await reader.read(8)
                    length = int.from_bytes(raw_len, 'big')
                mask = await reader.read(4)
                data = await reader.read(length)
                # Unmask
                unmasked = bytes([b ^ mask[i % 4] for i, b in enumerate(data)])
                # Handle client ping or requests if needed
        except:
            pass
        finally:
            CONNECTED_WS_CLIENTS.discard(writer)

    async def serve_node_installer(self, query, headers, writer):
        token = query.get("token", [""])[0]
        role = query.get("role", ["kharej"])[0]
        name = query.get("name", ["Auto Node"])[0]
        from app.config import load_settings
        host_header = headers.get("host", f"127.0.0.1:{self.port}")
        configured_url = load_settings().get("public_panel_url", "").strip().rstrip("/")
        panel_url = configured_url or f"http://{host_header}"

        script = f"""#!/usr/bin/env bash
set -e

TOKEN="{token}"
PANEL_URL="{panel_url}"
ROLE="{role}"
NAME="{name}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --token) TOKEN="$2"; shift 2 ;;
    --panel) PANEL_URL="$2"; shift 2 ;;
    --role) ROLE="$2"; shift 2 ;;
    --name) NAME="$2"; shift 2 ;;
    *) shift ;;
  esac
done

if [ -z "$TOKEN" ]; then
  echo "❌ Error: Node token is missing."
  echo "Usage: curl -fsSL ... | bash -s -- --panel <URL> --token <TOKEN>"
  exit 1
fi

echo "🚀 ==============================================="
echo "⚡ Hawal Tunnel (هه‌واڵ) - Automated Node Installer"
echo "🌐 Node Role: ${{ROLE^^}} | Panel: ${{PANEL_URL}}"
echo "==============================================="

if ! command -v python3 &> /dev/null || ! command -v curl &> /dev/null || ! command -v tar &> /dev/null; then
  echo "📦 Installing prerequisites (python3, curl, tar)..."
  apt-get update -y && apt-get install -y python3 curl tar || true
fi

mkdir -p /opt/hawal /etc/hawal

echo "📥 Downloading Backhaul High-Performance Tunnel Core..."
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  BH_URL="https://github.com/Musixal/Backhaul/releases/latest/download/backhaul_linux_amd64.tar.gz" ;;
  aarch64) BH_URL="https://github.com/Musixal/Backhaul/releases/latest/download/backhaul_linux_arm64.tar.gz" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

curl -fsSL --connect-timeout 10 --max-time 60 "$BH_URL" -o /tmp/backhaul.tar.gz
tar -xzf /tmp/backhaul.tar.gz -C /usr/local/bin/ backhaul
chmod +x /usr/local/bin/backhaul
rm -f /tmp/backhaul.tar.gz

echo "📥 Installing Hawal Node Agent Daemon..."
AGENT_OK=0
# 1. Official GitHub Raw (fastest & high-bandwidth for foreign exit/relay nodes)
if curl -fsSL --connect-timeout 8 --max-time 30 "https://raw.githubusercontent.com/dalroot/hawal/master/agent/agent.py" -o /opt/hawal/agent.py 2>/dev/null && [ -s /opt/hawal/agent.py ]; then
  AGENT_OK=1
# 2. Local Panel fallback (ideal for domestic Iran nodes)
elif curl -fsSL --connect-timeout 8 --max-time 30 "${{PANEL_URL}}/static/js/agent.py" -o /opt/hawal/agent.py 2>/dev/null && [ -s /opt/hawal/agent.py ]; then
  AGENT_OK=1
fi

if [ "$AGENT_OK" -eq 0 ] || [ ! -s /opt/hawal/agent.py ]; then
  echo "❌ Failed to download agent daemon. Please check network connectivity."
  exit 1
fi
chmod +x /opt/hawal/agent.py

# Write agent config
cat > /etc/hawal/agent.json << EOF
{{
  "panel_url": "${{PANEL_URL}}",
  "token": "${{TOKEN}}",
  "role": "${{ROLE}}",
  "name": "${{NAME}}"
}}
EOF

# Write Systemd Service
cat > /etc/systemd/system/hawal-agent.service << EOF
[Unit]
Description=Hawal Tunnel Node Agent Daemon
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/hawal
ExecStart=/usr/bin/python3 /opt/hawal/agent.py
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable hawal-agent || true
systemctl restart hawal-agent || systemctl start hawal-agent || true

echo "✅ Hawal Node (هه‌واڵ) successfully connected and active in Panel!"
"""
        resp = (
            "HTTP/1.1 200 OK\r\n"
            "Content-Type: text/x-shellscript; charset=utf-8\r\n"
            f"Content-Length: {len(script.encode('utf-8'))}\r\n"
            "Connection: close\r\n\r\n" + script
        )
        writer.write(resp.encode('utf-8'))
        await writer.drain()

    async def serve_template(self, filename, writer):
        filepath = os.path.join(TEMPLATES_DIR, filename)
        if not os.path.exists(filepath):
            self.send_json(writer, {"error": "Template not found"}, status=404)
            return
        with open(filepath, "r", encoding="utf-8") as f:
            content = f.read()
        resp = (
            "HTTP/1.1 200 OK\r\n"
            "Content-Type: text/html; charset=utf-8\r\n"
            "Cache-Control: no-cache, no-store, must-revalidate\r\n"
            "Pragma: no-cache\r\n"
            "Expires: 0\r\n"
            f"Content-Length: {len(content.encode('utf-8'))}\r\n"
            "Connection: close\r\n\r\n" + content
        )
        writer.write(resp.encode('utf-8'))
        try:
            await writer.drain()
        except:
            pass

    async def serve_static_file(self, filepath, writer, method="GET"):
        if not os.path.exists(filepath) or os.path.isdir(filepath):
            self.send_json(writer, {"error": "File not found"}, status=404)
            return
        mime_type, _ = mimetypes.guess_type(filepath)
        mime_type = mime_type or "application/octet-stream"
        with open(filepath, "rb") as f:
            content = f.read()
        resp_headers = (
            "HTTP/1.1 200 OK\r\n"
            f"Content-Type: {mime_type}\r\n"
            f"Content-Length: {len(content)}\r\n"
            "Cache-Control: no-cache, must-revalidate\r\n"
            "Connection: close\r\n\r\n"
        )
        if method == "HEAD":
            writer.write(resp_headers.encode('utf-8'))
        else:
            writer.write(resp_headers.encode('utf-8') + content)
        try:
            await writer.drain()
        except:
            pass

    def send_redirect(self, writer, location, set_cookie=None):
        headers = [
            "HTTP/1.1 302 Found",
            f"Location: {location}",
            "Content-Length: 0"
        ]
        if set_cookie:
            headers.append(f"Set-Cookie: {set_cookie}")
        headers.append("Connection: close\r\n\r\n")
        writer.write("\r\n".join(headers).encode('utf-8'))

    def send_json(self, writer, data, status=200, set_cookie=None):
        body = json.dumps(data).encode('utf-8')
        status_text = "OK" if status == 200 else ("Not Found" if status == 404 else ("Unauthorized" if status == 401 else "Error"))
        headers = [
            f"HTTP/1.1 {status} {status_text}",
            "Content-Type: application/json; charset=utf-8",
            f"Content-Length: {len(body)}",
            "Access-Control-Allow-Origin: *",
            "Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS",
            "Access-Control-Allow-Headers: Content-Type, Authorization, X-Node-Token",
        ]
        if set_cookie:
            headers.append(f"Set-Cookie: {set_cookie}")
        headers.append("Connection: close\r\n\r\n")
        writer.write("\r\n".join(headers).encode('utf-8') + body)
