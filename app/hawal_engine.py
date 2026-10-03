import json

def generate_hawal_core_server_config(tunnel_dict):
    """
    Generates JSON configuration dictionary for Hawal Core (Server Mode)
    """
    carrier = tunnel_dict.get("transport", "tls").lower()
    if carrier in ("kcp", "rawpaq", "rawtcp", "raw"):
        carrier = "rawpaq"
    elif carrier == "tcp":
        carrier = "tcp"
    else:
        carrier = "tls"

    return {
        "mode": "server",
        "carrier": carrier,
        "bind_addr": f"0.0.0.0:{tunnel_dict['core_port']}",
        "ports": tunnel_dict.get("ports", []),
        "token": tunnel_dict.get("token", ""),
        "enable_padding": bool(tunnel_dict.get("snappy", 1)),
        "nodelay": bool(tunnel_dict.get("nodelay", 1)),
        "insecure_tls": True,
        "server_name": tunnel_dict.get("sni", "") or "www.cloudflare.com"
    }

def generate_hawal_core_client_config(tunnel_dict, server_ip):
    """
    Generates JSON configuration dictionary for Hawal Core (Client Mode)
    """
    carrier = tunnel_dict.get("transport", "tls").lower()
    if carrier in ("kcp", "rawpaq", "rawtcp", "raw"):
        carrier = "rawpaq"
    elif carrier == "tcp":
        carrier = "tcp"
    else:
        carrier = "tls"

    return {
        "mode": "client",
        "carrier": carrier,
        "connect_addr": f"{server_ip}:{tunnel_dict['core_port']}",
        "ports": tunnel_dict.get("ports", []),
        "token": tunnel_dict.get("token", ""),
        "enable_padding": bool(tunnel_dict.get("snappy", 1)),
        "nodelay": bool(tunnel_dict.get("nodelay", 1)),
        "insecure_tls": True,
        "server_name": tunnel_dict.get("sni", "") or "www.cloudflare.com"
    }
