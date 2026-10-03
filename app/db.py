import sqlite3
import json
import time
from app.config import DB_PATH

def get_db():
    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    return conn

def init_db():
    with get_db() as conn:
        cursor = conn.cursor()
        
        # Nodes table
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS nodes (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            ip TEXT NOT NULL,
            role TEXT NOT NULL DEFAULT 'kharej', -- 'iran' or 'kharej'
            country_code TEXT DEFAULT 'GLOBAL',
            country_name TEXT DEFAULT 'خارج',
            flag TEXT DEFAULT '🌐',
            city TEXT DEFAULT '',
            token TEXT NOT NULL UNIQUE,
            status TEXT NOT NULL DEFAULT 'offline',
            last_seen REAL DEFAULT 0,
            cpu_percent REAL DEFAULT 0,
            ram_used_mb INTEGER DEFAULT 0,
            ram_total_mb INTEGER DEFAULT 0,
            uptime_seconds INTEGER DEFAULT 0,
            created_at REAL NOT NULL
        )
        """)
        
        # Tunnels table
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS tunnels (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            core_type TEXT NOT NULL DEFAULT 'hawal', -- 'hawal' or 'backhaul'
            server_node_id TEXT NOT NULL,
            client_node_id TEXT NOT NULL,
            core_port INTEGER NOT NULL,
            transport TEXT NOT NULL DEFAULT 'ws', -- 'tcp', 'ws', 'tcpmux', 'tls'
            ports_json TEXT NOT NULL DEFAULT '[]', -- JSON array of "443=127.0.0.1:443"
            token TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'stopped', -- 'running', 'stopped', 'error'
            nodelay INTEGER DEFAULT 1,
            snappy INTEGER DEFAULT 1,
            mux_con INTEGER DEFAULT 8,
            keepalive INTEGER DEFAULT 75,
            channel_size INTEGER DEFAULT 2048,
            restart_nonce INTEGER NOT NULL DEFAULT 0,
            created_at REAL NOT NULL,
            FOREIGN KEY (server_node_id) REFERENCES nodes (id) ON DELETE CASCADE,
            FOREIGN KEY (client_node_id) REFERENCES nodes (id) ON DELETE CASCADE
        )
        """)
        
        # Ping latency history
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS ping_history (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            source_node_id TEXT NOT NULL,
            target_node_id TEXT NOT NULL,
            latency_avg_ms REAL NOT NULL,
            latency_min_ms REAL NOT NULL,
            latency_max_ms REAL NOT NULL,
            packet_loss REAL NOT NULL,
            created_at REAL NOT NULL
        )
        """)
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS log_snapshots (
            node_id TEXT NOT NULL,
            source TEXT NOT NULL,
            content TEXT NOT NULL,
            updated_at REAL NOT NULL,
            PRIMARY KEY (node_id, source)
        )
        """)
        
        # Automatic migrations for existing databases
        try:
            cursor.execute("ALTER TABLE tunnels ADD COLUMN core_type TEXT NOT NULL DEFAULT 'hawal'")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE tunnels ADD COLUMN bytes_in INTEGER DEFAULT 0")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE tunnels ADD COLUMN bytes_out INTEGER DEFAULT 0")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE tunnels ADD COLUMN restart_nonce INTEGER NOT NULL DEFAULT 0")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE tunnels ADD COLUMN kcp_mode TEXT DEFAULT 'normal'")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE nodes ADD COLUMN agent_restart_nonce INTEGER NOT NULL DEFAULT 0")
        except:
            pass
        try:
            cursor.execute("ALTER TABLE nodes ADD COLUMN country_code TEXT DEFAULT 'GLOBAL'")
            cursor.execute("ALTER TABLE nodes ADD COLUMN country_name TEXT DEFAULT 'خارج'")
            cursor.execute("ALTER TABLE nodes ADD COLUMN flag TEXT DEFAULT '🌐'")
            cursor.execute("ALTER TABLE nodes ADD COLUMN city TEXT DEFAULT ''")
        except:
            pass

        # Traffic samples time-series table
        cursor.execute("""
        CREATE TABLE IF NOT EXISTS traffic_samples (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            target_type TEXT NOT NULL, -- 'node' or 'tunnel'
            target_id TEXT NOT NULL,
            timestamp INTEGER NOT NULL,
            bytes_in INTEGER NOT NULL DEFAULT 0,
            bytes_out INTEGER NOT NULL DEFAULT 0,
            rate_in_mbps REAL NOT NULL DEFAULT 0,
            rate_out_mbps REAL NOT NULL DEFAULT 0
        )
        """)
        cursor.execute("CREATE INDEX IF NOT EXISTS idx_traffic_target_ts ON traffic_samples(target_type, target_id, timestamp)")

        for col, col_type in [
            ("net_rx_bytes", "INTEGER DEFAULT 0"),
            ("net_tx_bytes", "INTEGER DEFAULT 0"),
            ("rate_in_mbps", "REAL DEFAULT 0"),
            ("rate_out_mbps", "REAL DEFAULT 0"),
            ("latency_ms", "REAL DEFAULT 0"),
        ]:
            try:
                cursor.execute(f"ALTER TABLE nodes ADD COLUMN {col} {col_type}")
            except Exception:
                pass
        
        conn.commit()

# --- Node Operations ---
def list_nodes():
    with get_db() as conn:
        rows = conn.execute("SELECT * FROM nodes ORDER BY created_at ASC").fetchall()
        nodes = []
        now = time.time()
        for r in rows:
            d = dict(r)
            # consider offline if last_seen == 0 or > 15s ago
            if d.get("last_seen", 0) == 0:
                if d.get("status") != "enrolling":
                    d["status"] = "offline"
            elif now - d.get("last_seen", 0) > 15:
                d["status"] = "offline"
            nodes.append(d)
        return nodes

def get_node(node_id):
    with get_db() as conn:
        row = conn.execute("SELECT * FROM nodes WHERE id = ?", (node_id,)).fetchone()
        return dict(row) if row else None

def get_node_by_token(token):
    with get_db() as conn:
        row = conn.execute("SELECT * FROM nodes WHERE token = ?", (token,)).fetchone()
        return dict(row) if row else None

def save_node(node_id, name, ip, role, token, country_code="GLOBAL", country_name="خارج", flag="🌐", city=""):
    now = time.time()
    with get_db() as conn:
        conn.execute("""
        INSERT INTO nodes (id, name, ip, role, country_code, country_name, flag, city, token, status, last_seen, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'enrolling', 0, ?)
        ON CONFLICT(id) DO UPDATE SET
            name=excluded.name,
            ip=excluded.ip,
            role=excluded.role,
            country_code=excluded.country_code,
            country_name=excluded.country_name,
            flag=excluded.flag,
            city=excluded.city
        """, (node_id, name, ip, role, country_code, country_name, flag, city, token, now))
        conn.commit()

def update_node_heartbeat(node_id, ip, cpu, ram_used, ram_total, uptime, country_code=None, country_name=None, flag=None, city=None, net_rx_bytes=None, net_tx_bytes=None, rate_in_mbps=None, rate_out_mbps=None, latency_ms=None):
    now = time.time()
    with get_db() as conn:
        updates = [
            ("ip", ip),
            ("cpu_percent", cpu),
            ("ram_used_mb", ram_used),
            ("ram_total_mb", ram_total),
            ("uptime_seconds", uptime),
            ("last_seen", now),
            ("status", 'online')
        ]
        if country_code and country_name:
            updates.extend([
                ("country_code", country_code),
                ("country_name", country_name),
                ("flag", flag or '🌐'),
                ("city", city or '')
            ])
        if net_rx_bytes is not None:
            updates.append(("net_rx_bytes", int(net_rx_bytes)))
        if net_tx_bytes is not None:
            updates.append(("net_tx_bytes", int(net_tx_bytes)))
        if rate_in_mbps is not None:
            updates.append(("rate_in_mbps", round(float(rate_in_mbps), 2)))
        if rate_out_mbps is not None:
            updates.append(("rate_out_mbps", round(float(rate_out_mbps), 2)))
        if latency_ms is not None:
            updates.append(("latency_ms", round(float(latency_ms), 1)))

        set_clause = ", ".join([f"{col} = ?" for col, _ in updates])
        params = [val for _, val in updates]
        params.append(node_id)
        conn.execute(f"UPDATE nodes SET {set_clause} WHERE id = ?", params)
        conn.commit()

def delete_node(node_id):
    with get_db() as conn:
        conn.execute("DELETE FROM nodes WHERE id = ?", (node_id,))
        conn.commit()

# --- Tunnel Operations ---
def list_tunnels():
    with get_db() as conn:
        rows = conn.execute("""
        SELECT t.*, 
               sn.name as server_node_name, sn.ip as server_node_ip, sn.role as server_node_role, sn.latency_ms as server_node_latency,
               cn.name as client_node_name, cn.ip as client_node_ip, cn.role as client_node_role, cn.latency_ms as client_node_latency
        FROM tunnels t
        LEFT JOIN nodes sn ON t.server_node_id = sn.id
        LEFT JOIN nodes cn ON t.client_node_id = cn.id
        ORDER BY t.created_at ASC
        """).fetchall()
        tunnels = []
        for r in rows:
            d = dict(r)
            try:
                d["ports"] = json.loads(d["ports_json"])
            except:
                d["ports"] = []
            tunnels.append(d)
        return tunnels

def get_tunnel(tunnel_id):
    with get_db() as conn:
        row = conn.execute("SELECT * FROM tunnels WHERE id = ?", (tunnel_id,)).fetchone()
        if not row:
            return None
        d = dict(row)
        try:
            d["ports"] = json.loads(d["ports_json"])
        except:
            d["ports"] = []
        return d

def save_tunnel(tunnel_id, name, server_node_id, client_node_id, core_port, transport, ports, token, status='running', nodelay=1, snappy=1, mux_con=8, keepalive=75, channel_size=2048, core_type='hawal', kcp_mode='normal'):
    now = time.time()
    ports_json = json.dumps(ports)
    with get_db() as conn:
        conn.execute("""
        INSERT INTO tunnels (id, name, core_type, server_node_id, client_node_id, core_port, transport, ports_json, token, status, nodelay, snappy, mux_con, keepalive, channel_size, kcp_mode, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            name=excluded.name,
            core_type=excluded.core_type,
            server_node_id=excluded.server_node_id,
            client_node_id=excluded.client_node_id,
            core_port=excluded.core_port,
            transport=excluded.transport,
            ports_json=excluded.ports_json,
            token=excluded.token,
            status=excluded.status,
            nodelay=excluded.nodelay,
            snappy=excluded.snappy,
            mux_con=excluded.mux_con,
            keepalive=excluded.keepalive,
            channel_size=excluded.channel_size,
            kcp_mode=excluded.kcp_mode
        """, (tunnel_id, name, core_type, server_node_id, client_node_id, core_port, transport, ports_json, token, status, nodelay, snappy, mux_con, keepalive, channel_size, kcp_mode, now))
        conn.commit()

def update_tunnel(tunnel_id, name, core_port, transport, ports, core_type='hawal', kcp_mode=None):
    ports_json = json.dumps(ports)
    with get_db() as conn:
        if kcp_mode:
            conn.execute("""
            UPDATE tunnels SET
                name = ?,
                core_port = ?,
                transport = ?,
                ports_json = ?,
                core_type = ?,
                kcp_mode = ?
            WHERE id = ?
            """, (name, int(core_port), transport, ports_json, core_type, kcp_mode, tunnel_id))
        else:
            conn.execute("""
            UPDATE tunnels SET
                name = ?,
                core_port = ?,
                transport = ?,
                ports_json = ?,
                core_type = ?
            WHERE id = ?
            """, (name, int(core_port), transport, ports_json, core_type, tunnel_id))
        conn.commit()

def request_tunnel_restart(tunnel_id):
    with get_db() as conn:
        cursor = conn.execute(
            "UPDATE tunnels SET restart_nonce = restart_nonce + 1 WHERE id = ?", (tunnel_id,)
        )
        conn.commit()
        return cursor.rowcount > 0

def request_all_agents_restart():
    with get_db() as conn:
        cursor = conn.execute("UPDATE nodes SET agent_restart_nonce = agent_restart_nonce + 1")
        conn.commit()
        return cursor.rowcount

def save_log_snapshots(node_id, snapshots):
    now = time.time()
    with get_db() as conn:
        for source, content in snapshots.items():
            conn.execute("""
                INSERT INTO log_snapshots (node_id, source, content, updated_at)
                VALUES (?, ?, ?, ?)
                ON CONFLICT(node_id, source) DO UPDATE SET content=excluded.content, updated_at=excluded.updated_at
            """, (node_id, source, str(content)[-24000:], now))
        conn.commit()

def get_log_snapshots(node_id=None, source=None):
    sql = "SELECT * FROM log_snapshots"
    values = []
    where = []
    if node_id:
        where.append("node_id = ?")
        values.append(node_id)
    if source:
        where.append("source = ?")
        values.append(source)
    if where:
        sql += " WHERE " + " AND ".join(where)
    sql += " ORDER BY updated_at DESC"
    with get_db() as conn:
        return [dict(row) for row in conn.execute(sql, values).fetchall()]

def delete_log_snapshots(node_id=None, source=None):
    sql = "DELETE FROM log_snapshots"
    values = []
    where = []
    if node_id:
        where.append("node_id = ?")
        values.append(node_id)
    if source:
        where.append("source = ?")
        values.append(source)
    if where:
        sql += " WHERE " + " AND ".join(where)
    with get_db() as conn:
        cursor = conn.execute(sql, values)
        conn.commit()
        return cursor.rowcount

def update_tunnel_traffic(tunnel_id, bytes_in, bytes_out):
    with get_db() as conn:
        conn.execute("""
        UPDATE tunnels SET
            bytes_in = bytes_in + ?,
            bytes_out = bytes_out + ?
        WHERE id = ?
        """, (int(bytes_in), int(bytes_out), tunnel_id))
        conn.commit()

def set_tunnel_absolute_traffic(tunnel_id, bytes_in, bytes_out):
    with get_db() as conn:
        conn.execute("""
        UPDATE tunnels SET
            bytes_in = ?,
            bytes_out = ?
        WHERE id = ?
        """, (int(bytes_in), int(bytes_out), tunnel_id))
        conn.commit()

def set_tunnel_status(tunnel_id, status):
    with get_db() as conn:
        conn.execute("UPDATE tunnels SET status = ? WHERE id = ?", (status, tunnel_id))
        conn.commit()

def delete_tunnel(tunnel_id):
    with get_db() as conn:
        conn.execute("DELETE FROM tunnels WHERE id = ?", (tunnel_id,))
        conn.commit()

# --- Ping History ---
def record_ping(source_id, target_id, avg_ms, min_ms, max_ms, loss_pct):
    now = time.time()
    with get_db() as conn:
        conn.execute("""
        INSERT INTO ping_history (source_node_id, target_node_id, latency_avg_ms, latency_min_ms, latency_max_ms, packet_loss, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (source_id, target_id, avg_ms, min_ms, max_ms, loss_pct, now))
        conn.commit()

def get_latest_pings():
    with get_db() as conn:
        rows = conn.execute("""
        SELECT ph.*, 
               sn.name as source_name, sn.role as source_role,
               tn.name as target_name, tn.role as target_role
        FROM ping_history ph
        JOIN nodes sn ON ph.source_node_id = sn.id
        JOIN nodes tn ON ph.target_node_id = tn.id
        ORDER BY ph.created_at DESC LIMIT 20
        """).fetchall()
        return [dict(r) for r in rows]

# --- Time-Series Traffic Operations ---
def record_traffic_sample(target_type, target_id, bytes_in, bytes_out, rate_in_mbps=0.0, rate_out_mbps=0.0, timestamp=None):
    if timestamp is None:
        timestamp = int(time.time())
    with get_db() as conn:
        conn.execute("""
        INSERT INTO traffic_samples (target_type, target_id, timestamp, bytes_in, bytes_out, rate_in_mbps, rate_out_mbps)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (target_type, target_id, int(timestamp), int(bytes_in), int(bytes_out), round(float(rate_in_mbps), 2), round(float(rate_out_mbps), 2)))
        conn.commit()

def clean_old_traffic_samples(retention_days=35):
    cutoff = int(time.time()) - (retention_days * 86400)
    with get_db() as conn:
        conn.execute("DELETE FROM traffic_samples WHERE timestamp < ?", (cutoff,))
        conn.commit()

def get_traffic_history(target_type="all", target_id=None, time_range="24h"):
    now = int(time.time())
    range_config = {
        "1h":  {"duration": 3600,        "bucket": 60},      # 60 points (1m)
        "12h": {"duration": 12 * 3600,   "bucket": 300},     # 144 points (5m)
        "24h": {"duration": 24 * 3600,   "bucket": 300},     # 288 points (5m)
        "7d":  {"duration": 7 * 86400,   "bucket": 1800},    # 336 points (30m)
        "30d": {"duration": 30 * 86400,  "bucket": 7200}     # 360 points (2h)
    }
    cfg = range_config.get(time_range, range_config["24h"])
    duration = cfg["duration"]
    bucket_size = cfg["bucket"]
    start_time = now - duration

    where_clauses = ["timestamp >= ?"]
    params = [start_time]

    if target_type in ("node", "tunnel"):
        where_clauses.append("target_type = ?")
        params.append(target_type)
        if target_id and target_id != "all":
            where_clauses.append("target_id = ?")
            params.append(target_id)
    elif target_type == "all":
        # Aggregate based on Iran gateway node to represent overall network transit without double counting
        with get_db() as conn:
            gateway = conn.execute("SELECT id FROM nodes WHERE role = 'iran' LIMIT 1").fetchone()
        if gateway:
            where_clauses.append("target_type = 'node' AND target_id = ?")
            params.append(gateway["id"])
        else:
            where_clauses.append("target_type = 'node'")

    where_sql = " AND ".join(where_clauses)

    query = f"""
    SELECT
        (timestamp / {bucket_size}) * {bucket_size} AS bucket_ts,
        SUM(bytes_in) AS total_in,
        SUM(bytes_out) AS total_out,
        ROUND(AVG(rate_in_mbps), 2) AS avg_rate_in,
        ROUND(AVG(rate_out_mbps), 2) AS avg_rate_out,
        ROUND(MAX(rate_in_mbps), 2) AS peak_rate_in,
        ROUND(MAX(rate_out_mbps), 2) AS peak_rate_out
    FROM traffic_samples
    WHERE {where_sql}
    GROUP BY bucket_ts
    ORDER BY bucket_ts ASC
    """

    with get_db() as conn:
        rows = conn.execute(query, params).fetchall()

    points = []
    total_bytes_in = 0
    total_bytes_out = 0
    peak_in = 0.0
    peak_out = 0.0
    latest_in = 0.0
    latest_out = 0.0

    for r in rows:
        d = dict(r)
        b_ts = d["bucket_ts"]
        b_in = d["total_in"] or 0
        b_out = d["total_out"] or 0
        r_in = d["avg_rate_in"] or 0.0
        r_out = d["avg_rate_out"] or 0.0
        p_in = d["peak_rate_in"] or 0.0
        p_out = d["peak_rate_out"] or 0.0

        total_bytes_in += b_in
        total_bytes_out += b_out
        if p_in > peak_in: peak_in = p_in
        if p_out > peak_out: peak_out = p_out
        latest_in = r_in
        latest_out = r_out

        points.append({
            "timestamp": b_ts,
            "bytes_in": b_in,
            "bytes_out": b_out,
            "rate_in_mbps": r_in,
            "rate_out_mbps": r_out
        })

    return {
        "range": time_range,
        "target_type": target_type,
        "target_id": target_id or "all",
        "summary": {
            "total_bytes_in": total_bytes_in,
            "total_bytes_out": total_bytes_out,
            "peak_rate_in_mbps": peak_in,
            "peak_rate_out_mbps": peak_out,
            "current_rate_in_mbps": latest_in,
            "current_rate_out_mbps": latest_out
        },
        "points": points
    }

