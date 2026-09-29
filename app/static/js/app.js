let STATE = {
  nodes: [],
  tunnels: [],
  pings: [],
  settings: {},
  activeTab: 'dashboard',
  currentInstallerToken: '',
  currentInstallerRole: '',
  currentInstallerName: '',
  installerMode: 'native',
  activePortTags: [443, 2083],
  editPortTags: [],
  ws: null,
  tunnelTestResults: {},
  bandwidthRange: '24h',
  bandwidthTarget: 'all:all',
  bandwidthData: null,
  bandwidthAutoTimer: null
};

// UI UX Pro Max: Toast Notification System
function showToast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  
  let iconSvg = '';
  if (type === 'success') {
    iconSvg = '<svg width="20" height="20" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>';
  } else if (type === 'error') {
    iconSvg = '<svg width="20" height="20" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>';
  } else {
    iconSvg = '<svg width="20" height="20" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>';
  }

  toast.innerHTML = `${iconSvg}<span>${message}</span>`;
  container.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(15px)';
    toast.style.transition = 'all 0.25s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}

// -------------------------------------------------------------
// Engine Selector Segmented Cards
// -------------------------------------------------------------
function selectEngine(type) {
  document.getElementById('tunnel-core-type').value = type;
  const cardHawal = document.getElementById('card-engine-hawal');
  const cardPaqet = document.getElementById('card-engine-paqet');
  const cardBackhaul = document.getElementById('card-engine-backhaul');
  const cardGost = document.getElementById('card-engine-gost');
  const transportSelect = document.getElementById('tunnel-transport');

  if (cardHawal) cardHawal.classList.toggle('active', type === 'hawal');
  if (cardPaqet) cardPaqet.classList.toggle('active', type === 'paqet');
  if (cardBackhaul) cardBackhaul.classList.toggle('active', type === 'backhaul');
  if (cardGost) cardGost.classList.toggle('active', type === 'gost');

  if (transportSelect) {
    if (type === 'hawal') {
      transportSelect.innerHTML = '<option value="stealth" selected>⚡ Stealth Multi-Stream (ضد فیلتر و نویز متغیر)</option>';
    } else if (type === 'paqet') {
      transportSelect.innerHTML = '<option value="kcp" selected>🛡️ Raw Packet KCP (AES-128-GCM • ضد فیلترینگ شدید)</option>';
    } else if (type === 'gost') {
      transportSelect.innerHTML = `
        <option value="tls" selected>🔒 Relay over TLS (رمزنگاری‌شده و پایدار)</option>
        <option value="ws">WebSocket Relay</option>
        <option value="kcp">KCP Relay (UDP)</option>
        <option value="quic">QUIC Relay (UDP)</option>
      `;
    } else {
      transportSelect.innerHTML = `
        <option value="ws" selected>WebSocket (بکهول استاندارد)</option>
        <option value="tcp">TCP (خام و مستقیم)</option>
        <option value="tcpmux">TCP Mux (مالتی‌پلکس)</option>
        <option value="tls">TLS Encrypted</option>
      `;
    }
  }
}

function handleEditCoreTypeChange() {
  const coreType = document.getElementById('edit-tunnel-core-type').value;
  const transportSelect = document.getElementById('edit-tunnel-transport');
  if (!transportSelect) return;

  if (coreType === 'hawal') {
    transportSelect.innerHTML = '<option value="stealth" selected>⚡ Stealth Multi-Stream (ضد فیلتر و نویز متغیر)</option>';
  } else if (coreType === 'paqet') {
    transportSelect.innerHTML = '<option value="kcp" selected>🛡️ Raw Packet KCP (AES-128-GCM • ضد فیلترینگ شدید)</option>';
  } else if (coreType === 'gost') {
    transportSelect.innerHTML = `
      <option value="tls" selected>🔒 Relay over TLS</option>
      <option value="ws">WebSocket Relay</option>
      <option value="kcp">KCP Relay (UDP)</option>
      <option value="quic">QUIC Relay (UDP)</option>
    `;
  } else {
    transportSelect.innerHTML = `
      <option value="ws" selected>WebSocket (بکهول استاندارد)</option>
      <option value="tcp">TCP (خام و مستقیم)</option>
      <option value="tcpmux">TCP Mux (مالتی‌پلکس)</option>
      <option value="tls">TLS Encrypted</option>
    `;
  }
}

// -------------------------------------------------------------
// Port Chip Tag Manager (Add Modal)
// -------------------------------------------------------------
function renderPortChips() {
  const container = document.getElementById('port-chip-list');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.activePortTags.length === 0) {
    container.innerHTML = '<span style="color: var(--text-dim); font-size: 12px;">هنوز پورتی اضافه نشده است. از لیست زیر یا فیلد بالا پورت اضافه کنید.</span>';
    return;
  }

  STATE.activePortTags.forEach(port => {
    const chip = document.createElement('div');
    chip.className = 'port-chip';
    chip.innerHTML = `
      <span>${port} ──► ${port}</span>
      <button type="button" class="port-chip-remove" onclick="removePortChip(${port})">&times;</button>
    `;
    container.appendChild(chip);
  });
}

function addPortChip(portVal) {
  const port = parseInt(portVal);
  if (isNaN(port) || port < 1 || port > 65535) {
    showToast('شماره پورت نامعتبر است (باید بین ۱ تا ۶۵۵۳۵ باشد)', 'error');
    return;
  }
  if (STATE.activePortTags.includes(port)) {
    showToast(`پورت ${port} قبلاً اضافه شده است`, 'info');
    return;
  }
  STATE.activePortTags.push(port);
  renderPortChips();
}

function removePortChip(port) {
  STATE.activePortTags = STATE.activePortTags.filter(p => p !== port);
  renderPortChips();
}

// -------------------------------------------------------------
// Port Chip Tag Manager (Edit Modal)
// -------------------------------------------------------------
function renderEditPortChips() {
  const container = document.getElementById('edit-port-chip-list');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.editPortTags.length === 0) {
    container.innerHTML = '<span style="color: var(--text-dim); font-size: 12px;">هیچ پورتی برای این تانل ست نشده است.</span>';
    return;
  }

  STATE.editPortTags.forEach(port => {
    const chip = document.createElement('div');
    chip.className = 'port-chip';
    chip.innerHTML = `
      <span>${port} ──► ${port}</span>
      <button type="button" class="port-chip-remove" onclick="removeEditPortChip(${port})">&times;</button>
    `;
    container.appendChild(chip);
  });
}

function addEditPortChip(portVal) {
  const port = parseInt(portVal);
  if (isNaN(port) || port < 1 || port > 65535) {
    showToast('شماره پورت نامعتبر است', 'error');
    return;
  }
  if (STATE.editPortTags.includes(port)) {
    showToast(`پورت ${port} قبلاً در لیست وجود دارد`, 'info');
    return;
  }
  STATE.editPortTags.push(port);
  renderEditPortChips();
}

function removeEditPortChip(port) {
  STATE.editPortTags = STATE.editPortTags.filter(p => p !== port);
  renderEditPortChips();
}

// -------------------------------------------------------------
// Cloudflare Keyboard Shortcut & Quick Search
// -------------------------------------------------------------
document.addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault();
    const searchInput = document.getElementById('cf-home-search-input') || document.getElementById('sidebar-search-input');
    if (searchInput) searchInput.focus();
  }
});

function handleQuickFilter(query) {
  const q = (query || '').toLowerCase().trim();
  const tunnelRows = document.querySelectorAll('#cf-col-tunnels-list .cf-asset-row');
  tunnelRows.forEach(row => {
    const text = row.innerText.toLowerCase();
    row.style.display = text.includes(q) ? 'flex' : 'none';
  });

  const nodeRows = document.querySelectorAll('#cf-col-nodes-list .cf-asset-row');
  nodeRows.forEach(row => {
    const text = row.innerText.toLowerCase();
    row.style.display = text.includes(q) ? 'flex' : 'none';
  });

  const nodesTableRows = document.querySelectorAll('#nodes-table-body tr');
  nodesTableRows.forEach(row => {
    const text = row.innerText.toLowerCase();
    row.style.display = text.includes(q) ? '' : 'none';
  });

  const tunnelsTableRows = document.querySelectorAll('#tunnels-table-body tr');
  tunnelsTableRows.forEach(row => {
    const text = row.innerText.toLowerCase();
    row.style.display = text.includes(q) ? '' : 'none';
  });
}

// -------------------------------------------------------------
// SPA Routing & Tab Navigation
// -------------------------------------------------------------
const ROUTE_TAB_MAP = {
  '': 'dashboard',
  '/': 'dashboard',
  '/dashboard': 'dashboard',
  '/node': 'nodes',
  '/nodes': 'nodes',
  '/tunnel': 'tunnels',
  '/tunnels': 'tunnels',
  '/ping': 'ping',
  '/diagnostics': 'ping',
  '/log': 'logs',
  '/logs': 'logs',
  '/settings': 'settings'
};

const TAB_PATH_MAP = {
  'dashboard': '/',
  'nodes': '/nodes',
  'tunnels': '/tunnels',
  'ping': '/ping',
  'logs': '/logs',
  'settings': '/settings'
};

function getTabFromUrl() {
  const path = window.location.pathname.replace(/\/$/, '') || '/';
  if (ROUTE_TAB_MAP[path]) {
    return ROUTE_TAB_MAP[path];
  }
  const hash = window.location.hash.replace(/^#\/?/, '');
  if (hash && (ROUTE_TAB_MAP['/' + hash] || ROUTE_TAB_MAP[hash])) {
    return ROUTE_TAB_MAP['/' + hash] || ROUTE_TAB_MAP[hash];
  }
  return 'dashboard';
}

function switchTab(tabId, pushState = true) {
  if (!tabId) tabId = 'dashboard';
  STATE.activeTab = tabId;

  document.querySelectorAll('.tab-content').forEach(el => el.style.display = 'none');
  const activeSection = document.getElementById(`tab-${tabId}`);
  if (activeSection) activeSection.style.display = 'block';

  document.querySelectorAll('.cf-nav-item').forEach(el => el.classList.remove('active'));
  const activeNav = document.getElementById(`nav-${tabId}`);
  if (activeNav) activeNav.classList.add('active');

  const titles = {
    'dashboard': 'نمای کلی',
    'nodes': 'نودها',
    'tunnels': 'تانل‌ها',
    'ping': 'آزمایش شبکه',
    'logs': 'لاگ‌ها',
    'settings': 'تنظیمات'
  };
  const titleFa = titles[tabId] || 'نمای کلی';
  document.title = `Hawal — ${titleFa}`;

  const breadcrumbs = document.getElementById('cf-breadcrumbs');
  if (breadcrumbs) {
    breadcrumbs.innerHTML = `
      <span style="cursor: pointer;" onclick="switchTab('dashboard')">Hawal</span>
      <span>/</span>
      <span style="color: var(--cf-text-primary); font-weight: 700;">${titleFa}</span>
    `;
  }

  if (pushState) {
    const targetPath = TAB_PATH_MAP[tabId] || '/';
    if (window.location.pathname !== targetPath) {
      window.history.pushState({ tab: tabId }, '', targetPath);
    }
  }

  if (tabId === 'dashboard') {
    setTimeout(loadBandwidthMetrics, 50);
  } else if (tabId === 'logs') {
    loadLogs();
  }
}

window.addEventListener('popstate', (e) => {
  const tab = (e.state && e.state.tab) ? e.state.tab : getTabFromUrl();
  switchTab(tab, false);
});

// -------------------------------------------------------------
// Live WebSocket Connection
// -------------------------------------------------------------
function connectWebSocket() {
  const loc = window.location;
  const wsProtocol = loc.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${wsProtocol}//${loc.host}/ws`;

  STATE.ws = new WebSocket(wsUrl);

  STATE.ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      if (msg.event === 'node_updated' || msg.event === 'tunnel_updated' || msg.event === 'ping_updated') {
        fetchData();
      }
    } catch (e) {}
  };

  STATE.ws.onclose = () => {
    setTimeout(connectWebSocket, 3000);
  };
}

// -------------------------------------------------------------
// Fetch and Render
// -------------------------------------------------------------
async function fetchData() {
  try {
    const [nodesRes, tunnelsRes, pingsRes, settingsRes] = await Promise.all([
      fetch('/api/nodes').then(r => r.json()),
      fetch('/api/tunnels').then(r => r.json()),
      fetch('/api/ping/history').then(r => r.json()),
      fetch('/api/settings').then(r => r.json())
    ]);

    STATE.nodes = nodesRes.nodes || [];
    STATE.tunnels = tunnelsRes.tunnels || [];
    STATE.pings = pingsRes.history || pingsRes.pings || [];
    STATE.settings = settingsRes.settings || {};
    const publicPanelUrl = document.getElementById('setting-public-panel-url');
    if (publicPanelUrl) publicPanelUrl.value = STATE.settings.public_panel_url || '';

    renderDashboard();
    renderNodes();
    renderTunnels();
    renderPingSection();
  } catch (e) {
    console.error('Error fetching state:', e);
  }
}

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1);
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatUptime(seconds) {
  if (!seconds || seconds <= 0) return 'به تازگی فعال شده';
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days} روز و ${hours} ساعت`;
  if (hours > 0) return `${hours} ساعت و ${mins} دقیقه`;
  return `${mins} دقیقه`;
}

function renderServerResourceNodes() {
  const container = document.getElementById('server-nodes-resource-container');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.nodes.length === 0) {
    container.innerHTML = `
      <div style="grid-column: 1 / -1; padding: 28px; text-align: center; background: var(--bg-surface); border: 1px dashed var(--border-default); border-radius: var(--radius-xl); color: var(--text-muted);">
        <p style="font-size: 14px; font-weight: 600; color: var(--text-secondary);">هنوز هیچ سروری به شبکه متصل نشده است</p>
        <button class="btn btn-primary btn-sm" onclick="openAddNodeModal()" style="margin-top: 12px;">+ تعریف نود سرور</button>
      </div>
    `;
    return;
  }

  STATE.nodes.forEach(node => {
    const isOnline = node.status === 'online';
    const isIran = node.role === 'iran';
    const roleText = isIran ? 'سرور ایران (Gateway)' : 'سرور خارج (Exit Node)';
    const roleClass = isIran ? 'role-iran' : 'role-kharej';

    const cpu = Math.min(100, Math.max(0, node.cpu_percent || 0));
    const ramU = node.ram_used_mb || 0;
    const ramT = Math.max(1, node.ram_total_mb || 1024);
    const ramPct = Math.min(100, Math.round((ramU / ramT) * 100));

    let cpuColorClass = 'fill-emerald';
    if (cpu > 80) cpuColorClass = 'fill-rose';
    else if (cpu > 50) cpuColorClass = 'fill-amber';

    let ramColorClass = 'fill-sky';
    if (ramPct > 85) ramColorClass = 'fill-rose';
    else if (ramPct > 65) ramColorClass = 'fill-amber';

    const uptimeStr = formatUptime(node.uptime_seconds);

    const card = document.createElement('div');
    card.className = 'server-node-card';
    card.innerHTML = `
      <div class="node-card-top">
        <div class="node-identity">
          <div class="node-flag-box">${node.flag || (isIran ? '🇮🇷' : '🌐')}</div>
          <div class="node-title-wrap">
            <div class="node-name">
              <span>${node.name}</span>
              <span class="node-role-pill ${roleClass}">${roleText}</span>
            </div>
            <div class="node-ip-tag" onclick="navigator.clipboard.writeText('${node.ip}'); showToast('آدرس IP کپی شد', 'success')" title="کلیک برای کپی آدرس IP">
              <span>${node.ip}</span>
              <svg width="12" height="12" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
            </div>
          </div>
        </div>
        <div class="status-pill ${isOnline ? 'online' : 'offline'}">
          <span class="status-dot ${isOnline ? 'online' : 'offline'}"></span>
          <span>${isOnline ? 'آنلاین' : 'آفلاین'}</span>
        </div>
      </div>

      <div class="node-meters-grid">
        <div class="meter-item">
          <div class="meter-header">
            <span>بار پردازنده (CPU)</span>
            <span class="meter-val tabular-nums">${cpu.toFixed(1)}%</span>
          </div>
          <div class="meter-bar-track">
            <div class="meter-bar-fill ${cpuColorClass}" style="width: ${cpu}%;"></div>
          </div>
        </div>

        <div class="meter-item">
          <div class="meter-header">
            <span>حافظه رم (RAM)</span>
            <span class="meter-val tabular-nums">${ramU} / ${ramT} MB (${ramPct}%)</span>
          </div>
          <div class="meter-bar-track">
            <div class="meter-bar-fill ${ramColorClass}" style="width: ${ramPct}%;"></div>
          </div>
        </div>
      </div>

      <div class="node-card-footer">
        <div class="uptime-badge">
          <svg width="13" height="13" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
          <span>آپتایم: <span class="tabular-nums">${uptimeStr}</span></span>
        </div>
        <button class="btn btn-secondary btn-sm" onclick="showInstallModal('${node.token}', '${node.role}', '${node.name}')">
          اسکریپت اتصال
        </button>
      </div>
    `;
    container.appendChild(card);
  });
}

function renderDashboard() {
  const colCountTunnels = document.getElementById('col-count-tunnels');
  const colCountNodes = document.getElementById('col-count-nodes');
  const colTotalTraffic = document.getElementById('col-total-traffic');
  const kpiCycleRx = document.getElementById('kpi-cycle-rx');
  const kpiCycleTx = document.getElementById('kpi-cycle-tx');
  const kpiActiveEngines = document.getElementById('kpi-active-engines-summary');

  if (colCountTunnels) colCountTunnels.innerText = `${STATE.tunnels.length} تانل فعال`;
  if (colCountNodes) colCountNodes.innerText = STATE.nodes.length;

  const totalIn = STATE.tunnels.reduce((acc, t) => acc + (t.bytes_in || 0), 0);
  const totalOut = STATE.tunnels.reduce((acc, t) => acc + (t.bytes_out || 0), 0);
  const totalNetworkTraffic = totalIn + totalOut;

  if (colTotalTraffic) colTotalTraffic.innerText = formatBytes(totalNetworkTraffic);
  if (kpiCycleRx) kpiCycleRx.innerText = formatBytes(totalIn);
  if (kpiCycleTx) kpiCycleTx.innerText = formatBytes(totalOut);

  if (kpiActiveEngines) {
    const engines = [...new Set(STATE.tunnels.map(t => {
      if (t.core_type === 'paqet') return 'Paqet Raw';
      if (t.core_type === 'hawal') return 'Stealth Core';
      if (t.core_type === 'gost') return 'GOST';
      return 'Backhaul';
    }))];
    kpiActiveEngines.innerText = engines.length > 0 ? engines.join(', ') : 'بدون تانل';
  }

  renderServerResourceNodes();
  renderCloudflareTunnelsColumn();
  renderCloudflareNodesColumn();
  renderCloudflareAnalyticsColumn(totalIn, totalOut);
  renderTopology();
  populateTrafficTargetOptions();
  loadBandwidthMetrics();
  initBandwidthAutoRefresh();
}

function renderCloudflareTunnelsColumn() {
  const container = document.getElementById('cf-col-tunnels-list');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.tunnels.length === 0) {
    container.innerHTML = '<div style="grid-column: 1 / -1; padding: 24px; text-align: center; color: var(--text-muted); font-size: 13px;">هیچ تانلی ایجاد نشده است.</div>';
    return;
  }

  STATE.tunnels.forEach(tun => {
    const card = document.createElement('div');
    card.className = 'tunnel-item-card';
    card.onclick = () => switchTab('tunnels');

    let engineLabel = 'Backhaul';
    let engineClass = 'engine-backhaul';
    if (tun.core_type === 'paqet') {
      engineLabel = 'Paqet Raw KCP';
      engineClass = 'engine-paqet';
    } else if (tun.core_type === 'hawal') {
      engineLabel = 'Stealth Core v2';
      engineClass = 'engine-hawal';
    } else if (tun.core_type === 'gost') {
      engineLabel = 'GOST Relay';
      engineClass = 'engine-gost';
    }

    const totalBytes = (tun.bytes_in || 0) + (tun.bytes_out || 0);

    let portsStr = 'پورت هسته فقط';
    try {
      const p = typeof tun.ports_json === 'string' ? JSON.parse(tun.ports_json) : (tun.ports_json || []);
      if (Array.isArray(p) && p.length > 0) {
        portsStr = p.map(item => item.split('=')[0]).join(', ');
      }
    } catch(e) {}

    card.innerHTML = `
      <div class="tun-card-header">
        <div class="tun-name-wrap">
          <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24" style="color: var(--accent-primary);"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
          <span class="tun-name">${tun.name}</span>
        </div>
        <span class="engine-pill ${engineClass}">${engineLabel}</span>
      </div>

      <div class="tun-ports-box">
        <span style="color: var(--text-muted);">پورت‌های فوروارد:</span>
        <span class="tabular-nums" style="color: var(--accent-sky); font-weight: 600;">${portsStr}</span>
      </div>

      <div class="tun-card-footer">
        <span>پورت هسته: <span class="tabular-nums">${tun.core_port}</span></span>
        <span class="tabular-nums" style="font-weight: 600; color: var(--text-primary);">${formatBytes(totalBytes)}</span>
      </div>
    `;
    container.appendChild(card);
  });
}

function renderCloudflareNodesColumn() {
  const container = document.getElementById('cf-col-nodes-list');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.nodes.length === 0) {
    container.innerHTML = '<div style="padding: 16px; color: var(--cf-text-muted); font-size: 13px;">هیچ نودی یافت نشد.</div>';
    return;
  }

  STATE.nodes.forEach(node => {
    const row = document.createElement('div');
    row.className = 'cf-asset-row';
    row.onclick = () => switchTab('nodes');

    const isOnline = node.status === 'online';

    row.innerHTML = `
      <div class="cf-asset-left">
        <span style="font-size: 16px;">${node.flag || '🌐'}</span>
        <div>
          <div class="cf-asset-title">${node.name}</div>
          <div class="cf-asset-subtitle">
            ${node.ip} • <span style="color: ${isOnline ? 'var(--cf-green)' : 'var(--cf-red)'}; font-weight: 700;">${isOnline ? '🟢 آنلاین' : '🔴 آفلاین'}</span>
          </div>
        </div>
      </div>
      <svg class="cf-asset-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
    `;
    container.appendChild(row);
  });
}

function renderCloudflareAnalyticsColumn(totalIn, totalOut) {
  const container = document.getElementById('cf-col-analytics-list');
  if (!container) return;

  container.innerHTML = `
    <div class="cf-asset-row" onclick="switchTab('tunnels')">
      <div class="cf-asset-left">
        <svg class="cf-asset-icon" style="color: var(--cf-green);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
        <div>
          <div class="cf-asset-title">ترافیک ورودی</div>
          <div class="cf-asset-subtitle">${formatBytes(totalIn)}</div>
        </div>
      </div>
      <svg class="cf-asset-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
    </div>

    <div class="cf-asset-row" onclick="switchTab('tunnels')">
      <div class="cf-asset-left">
        <svg class="cf-asset-icon" style="color: var(--cf-blue);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>
        <div>
          <div class="cf-asset-title">ترافیک خروجی</div>
          <div class="cf-asset-subtitle">${formatBytes(totalOut)}</div>
        </div>
      </div>
      <svg class="cf-asset-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
    </div>

    <div class="cf-asset-row" onclick="switchTab('ping')">
      <div class="cf-asset-left">
        <svg class="cf-asset-icon" style="color: var(--cf-orange);" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
        <div>
          <div class="cf-asset-title">تاخیر میانگین شبکه</div>
          <div class="cf-asset-subtitle">برای مشاهده، آزمایش شبکه را اجرا کنید</div>
        </div>
      </div>
      <svg class="cf-asset-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path></svg>
    </div>
  `;
}

// -------------------------------------------------------------
// Bandwidth & Traffic Analytics Suite
// -------------------------------------------------------------
function populateTrafficTargetOptions() {
  const sel = document.getElementById('traffic-target-select');
  if (!sel) return;

  const currentVal = sel.value || STATE.bandwidthTarget || 'all:all';
  const options = [{ value: 'all:all', label: '🌐 کل ترافیک شبکه (All Traffic)' }];

  STATE.nodes.forEach(n => {
    const flag = n.flag || '🌐';
    const roleLabel = n.role === 'iran' ? 'ایران' : 'خارج';
    options.push({
      value: `node:${n.id}`,
      label: `🖥️ ${flag} نود ${n.name} (${roleLabel})`
    });
  });

  STATE.tunnels.forEach(t => {
    options.push({
      value: `tunnel:${t.id}`,
      label: `⚡ تانل ${t.name} (پورت ${t.core_port})`
    });
  });

  sel.innerHTML = options.map(opt => `<option value="${opt.value}" ${opt.value === currentVal ? 'selected' : ''}>${opt.label}</option>`).join('');
}

function changeBandwidthRange(range) {
  STATE.bandwidthRange = range;
  const container = document.getElementById('traffic-range-buttons');
  if (container) {
    container.querySelectorAll('.btn-range, .btn-range-pill').forEach(btn => {
      if (btn.dataset.range === range) btn.classList.add('active');
      else btn.classList.remove('active');
    });
  }

  const rangeLabels = {
    '1h': 'بازهٔ زمانی: ۱ ساعت گذشته',
    '12h': 'بازهٔ زمانی: ۱۲ ساعت گذشته',
    '24h': 'بازهٔ زمانی: امروز (۲۴ ساعت گذشته)',
    '7d': 'بازهٔ زمانی: ۷ روز گذشته (۱ هفته)',
    '30d': 'بازهٔ زمانی: ۳۰ روز گذشته (۱ ماه)'
  };
  const labelEl = document.getElementById('chart-time-range-display');
  if (labelEl) labelEl.innerText = rangeLabels[range] || `بازهٔ زمانی: ${range}`;

  loadBandwidthMetrics();
}

async function loadBandwidthMetrics() {
  const sel = document.getElementById('traffic-target-select');
  const targetVal = sel ? sel.value : (STATE.bandwidthTarget || 'all:all');
  STATE.bandwidthTarget = targetVal;

  const parts = targetVal.split(':');
  const targetType = parts[0] || 'all';
  const targetId = parts[1] || 'all';
  const range = STATE.bandwidthRange || '24h';

  try {
    const res = await fetch(`/api/metrics/bandwidth?target_type=${encodeURIComponent(targetType)}&target_id=${encodeURIComponent(targetId)}&range=${encodeURIComponent(range)}`);
    if (!res.ok) return;
    const data = await res.json();
    STATE.bandwidthData = data;

    const summ = data.summary || {};
    const currIn = document.getElementById('stat-curr-rate-in');
    const currOut = document.getElementById('stat-curr-rate-out');
    const totIn = document.getElementById('stat-total-in');
    const totOut = document.getElementById('stat-total-out');
    const peakEl = document.getElementById('stat-peak-rate');

    const rxMbps = summ.current_rate_in_mbps || 0;
    const txMbps = summ.current_rate_out_mbps || 0;
    const totalMbps = rxMbps + txMbps;

    if (currIn) currIn.innerText = `${rxMbps.toFixed(2)} Mbps`;
    if (currOut) currOut.innerText = `${txMbps.toFixed(2)} Mbps`;
    if (totIn) totIn.innerText = formatBytes(summ.total_bytes_in || 0);
    if (totOut) totOut.innerText = formatBytes(summ.total_bytes_out || 0);
    if (peakEl) {
      const peakVal = Math.max(summ.peak_rate_in_mbps || 0, summ.peak_rate_out_mbps || 0);
      peakEl.innerText = `${peakVal.toFixed(2)} Mbps`;
    }

    // Update Overview Top KPI cards
    const kpiLiveRate = document.getElementById('kpi-live-rate');
    const kpiLiveRx = document.getElementById('kpi-live-rx');
    const kpiLiveTx = document.getElementById('kpi-live-tx');
    const kpiToday = document.getElementById('kpi-today-traffic');

    if (kpiLiveRate) kpiLiveRate.innerText = `${totalMbps.toFixed(2)} Mbps`;
    if (kpiLiveRx) kpiLiveRx.innerText = `${rxMbps.toFixed(2)} Mbps`;
    if (kpiLiveTx) kpiLiveTx.innerText = `${txMbps.toFixed(2)} Mbps`;

    if (kpiToday) {
      const dayBytes = (summ.total_bytes_in || 0) + (summ.total_bytes_out || 0);
      kpiToday.innerText = formatBytes(dayBytes);
    }

    renderBandwidthChart(data);
  } catch (e) {
    console.error("Failed to load bandwidth metrics:", e);
  }
}

function renderBandwidthChart(data) {
  const canvas = document.getElementById('bandwidthCanvas');
  if (!canvas) return;

  const ctx = canvas.getContext('2d');
  const rect = canvas.getBoundingClientRect();
  if (rect.width === 0 || rect.height === 0) return;

  const dpr = window.devicePixelRatio || 1;
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  ctx.scale(dpr, dpr);

  const w = rect.width;
  const h = rect.height;
  ctx.clearRect(0, 0, w, h);

  const points = data.points || [];
  if (points.length === 0) {
    ctx.fillStyle = '#64748b';
    ctx.font = '13px Vazirmatn, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('در حال جمع‌آوری اولین نمونه‌های ترافیکی... داده‌ها به‌زودی نمایش داده می‌شوند.', w / 2, h / 2);
    return;
  }

  const padLeft = 55;
  const padRight = 20;
  const padTop = 20;
  const padBottom = 35;

  const chartW = w - padLeft - padRight;
  const chartH = h - padTop - padBottom;

  let maxRate = 0.5;
  points.forEach(p => {
    if (p.rate_in_mbps > maxRate) maxRate = p.rate_in_mbps;
    if (p.rate_out_mbps > maxRate) maxRate = p.rate_out_mbps;
  });
  maxRate = maxRate * 1.18;

  // Grid Lines & Y Axis Labels
  const gridLines = 4;
  ctx.strokeStyle = 'rgba(15, 23, 42, 0.06)';
  ctx.lineWidth = 1;
  ctx.fillStyle = '#64748b';
  ctx.font = '10px JetBrains Mono, monospace';
  ctx.textAlign = 'right';

  for (let i = 0; i <= gridLines; i++) {
    const yVal = (maxRate / gridLines) * i;
    const y = padTop + chartH - (i / gridLines) * chartH;
    ctx.beginPath();
    ctx.moveTo(padLeft, y);
    ctx.lineTo(w - padRight, y);
    ctx.stroke();

    ctx.fillText(`${yVal.toFixed(1)}M`, padLeft - 8, y + 3);
  }

  const coordsIn = [];
  const coordsOut = [];
  const count = points.length;

  points.forEach((p, idx) => {
    const x = count === 1 ? padLeft + chartW / 2 : padLeft + (idx / (count - 1)) * chartW;
    const yIn = padTop + chartH - (Math.min(p.rate_in_mbps, maxRate) / maxRate) * chartH;
    const yOut = padTop + chartH - (Math.min(p.rate_out_mbps, maxRate) / maxRate) * chartH;
    coordsIn.push({ x, y: yIn, pt: p });
    coordsOut.push({ x, y: yOut, pt: p });
  });

  function drawSeries(coords, lineColor, fillColor) {
    if (coords.length === 0) return;

    const grad = ctx.createLinearGradient(0, padTop, 0, padTop + chartH);
    grad.addColorStop(0, fillColor);
    grad.addColorStop(1, 'rgba(0, 0, 0, 0)');

    ctx.beginPath();
    ctx.moveTo(coords[0].x, padTop + chartH);
    coords.forEach(c => ctx.lineTo(c.x, c.y));
    ctx.lineTo(coords[coords.length - 1].x, padTop + chartH);
    ctx.closePath();
    ctx.fillStyle = grad;
    ctx.fill();

    ctx.beginPath();
    coords.forEach((c, i) => {
      if (i === 0) ctx.moveTo(c.x, c.y);
      else ctx.lineTo(c.x, c.y);
    });
    ctx.strokeStyle = lineColor;
    ctx.lineWidth = 2;
    ctx.lineJoin = 'round';
    ctx.stroke();
  }

  drawSeries(coordsOut, '#f59e0b', 'rgba(245, 158, 11, 0.22)');
  drawSeries(coordsIn, '#10b981', 'rgba(16, 185, 129, 0.25)');

  // X Axis Timestamps
  ctx.fillStyle = '#64748b';
  ctx.font = '10px JetBrains Mono, monospace';
  ctx.textAlign = 'center';
  const labelStep = Math.max(1, Math.floor(count / 5));

  for (let i = 0; i < count; i += labelStep) {
    const pt = points[i];
    const d = new Date(pt.timestamp * 1000);
    let timeStr = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
    if (data.range === '7d' || data.range === '30d') {
      timeStr = `${d.getMonth() + 1}/${d.getDate()} ${timeStr}`;
    }
    const x = count === 1 ? padLeft + chartW / 2 : padLeft + (i / (count - 1)) * chartW;
    ctx.fillText(timeStr, x, h - 10);
  }

  setupChartHover(canvas, coordsIn, coordsOut, chartW, chartH, padLeft, padTop);
}

function setupChartHover(canvas, coordsIn, coordsOut, chartW, chartH, padLeft, padTop) {
  const tooltip = document.getElementById('chartTooltip');
  if (!tooltip) return;

  canvas.onmousemove = (e) => {
    const rect = canvas.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;

    if (mouseX < padLeft || mouseX > rect.width - 20) {
      tooltip.style.display = 'none';
      return;
    }

    let nearestIdx = 0;
    let minDiff = Infinity;
    coordsIn.forEach((c, idx) => {
      const diff = Math.abs(c.x - mouseX);
      if (diff < minDiff) {
        minDiff = diff;
        nearestIdx = idx;
      }
    });

    const ptIn = coordsIn[nearestIdx];
    const ptOut = coordsOut[nearestIdx];
    if (!ptIn) return;

    const d = new Date(ptIn.pt.timestamp * 1000);
    const dateStr = d.toLocaleDateString('fa-IR');
    const timeStr = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;

    tooltip.innerHTML = `
      <div style="font-weight: 700; color: #94a3b8; margin-bottom: 4px; border-bottom: 1px solid #334155; padding-bottom: 3px;">
        🕒 ${timeStr} • ${dateStr}
      </div>
      <div style="color: #10b981; margin-bottom: 2px;">
        ▼ دانلود: <b>${ptIn.pt.rate_in_mbps.toFixed(2)} Mbps</b> <span style="color: #94a3b8; font-size: 10px;">(${formatBytes(ptIn.pt.bytes_in)})</span>
      </div>
      <div style="color: #f59e0b;">
        ▲ آپلود: <b>${ptOut.pt.rate_out_mbps.toFixed(2)} Mbps</b> <span style="color: #94a3b8; font-size: 10px;">(${formatBytes(ptOut.pt.bytes_out)})</span>
      </div>
    `;

    tooltip.style.display = 'block';
    let tipLeft = mouseX + 15;
    if (tipLeft + 200 > rect.width) tipLeft = mouseX - 205;
    tooltip.style.left = `${tipLeft}px`;
    tooltip.style.top = `${Math.max(10, mouseY - 40)}px`;
  };

  canvas.onmouseleave = () => {
    tooltip.style.display = 'none';
  };
}

function initBandwidthAutoRefresh() {
  if (!STATE.bandwidthAutoTimer) {
    STATE.bandwidthAutoTimer = setInterval(() => {
      if (STATE.activeTab === 'dashboard') {
        loadBandwidthMetrics();
      }
    }, 10000);

    window.addEventListener('resize', () => {
      if (STATE.bandwidthData && STATE.activeTab === 'dashboard') {
        renderBandwidthChart(STATE.bandwidthData);
      }
    });
  }
}

function renderTopology() {
  const container = document.getElementById('topology-container');
  if (!container) return;

  const iranNode = STATE.nodes.find(n => n.role === 'iran');
  const kharejNode = STATE.nodes.find(n => n.role === 'kharej');
  const indicator = document.getElementById('cf-rtt-indicator');

  if (!iranNode || !kharejNode) {
    if (indicator) indicator.textContent = 'برای نمایش مسیر، دو نود اضافه کنید';
    container.innerHTML = `
      <div class="topology-empty">
        <div class="topology-empty-icon">↔</div>
        <div><strong>مسیر ارتباطی هنوز آماده نیست</strong><span>یک نود ایران و یک نود خارج اضافه کنید تا نقشهٔ ارتباط نمایش داده شود.</span></div>
        <button class="btn btn-secondary btn-sm" onclick="openAddNodeModal()">افزودن نود</button>
      </div>`;
    return;
  }

  const activeTunnels = STATE.tunnels.filter(t => t.status === 'running').length;
  const iranOnline = iranNode.status === 'online';
  const foreignOnline = kharejNode.status === 'online';
  const routeHealthy = iranOnline && foreignOnline;
  if (indicator) indicator.textContent = routeHealthy ? `${activeTunnels} تانل فعال` : 'نیازمند بررسی نودها';

  container.innerHTML = `
    <div class="topology-network">
      <div class="route-node ${iranOnline ? 'online' : 'offline'}">
        <div class="route-flag">${iranNode.flag || '🇮🇷'}</div>
        <div class="route-copy"><span class="route-label">نود ایران</span><strong>${iranNode.name}</strong><code>${iranNode.ip}</code></div>
        <span class="route-status"><i></i>${iranOnline ? 'متصل' : 'قطع'}</span>
      </div>
      <div class="route-connector ${routeHealthy ? 'healthy' : ''}">
        <span>${activeTunnels} تانل فعال</span><div><i></i><i></i><i></i></div><small>مسیر رمزگذاری‌شده</small>
      </div>
      <div class="route-node ${foreignOnline ? 'online' : 'offline'}">
        <div class="route-flag">${kharejNode.flag || '🌐'}</div>
        <div class="route-copy"><span class="route-label">نود خارج</span><strong>${kharejNode.name}</strong><code>${kharejNode.ip}</code></div>
        <span class="route-status"><i></i>${foreignOnline ? 'متصل' : 'قطع'}</span>
      </div>
    </div>
  `;
}

function renderDashboardNodeCards() {
  const container = document.getElementById('dashboard-nodes-grid');
  if (!container) return;
  container.innerHTML = '';

  if (STATE.nodes.length === 0) {
    container.innerHTML = '<div style="color: var(--text-muted); font-size: 13px;">هیچ نودی یافت نشد.</div>';
    return;
  }

  STATE.nodes.forEach(node => {
    const card = document.createElement('div');
    card.className = 'stat-card';
    card.style.flexDirection = 'column';
    card.style.alignItems = 'stretch';
    card.style.gap = '14px';

    const isOnline = node.status === 'online';
    const cpu = node.cpu_percent || 0.1;
    const ramU = node.ram_used_mb || 12;
    const ramT = node.ram_total_mb || 2048;

    card.innerHTML = `
      <div style="display: flex; justify-content: space-between; align-items: center;">
        <div style="display: flex; align-items: center; gap: 10px;">
          <span style="font-size: 22px;">${node.flag || '🌐'}</span>
          <div>
            <div style="font-weight: 800; font-size: 14px;">${node.name}</div>
            <div style="font-size: 12px; color: var(--text-muted); font-family: 'JetBrains Mono';">${node.ip}</div>
          </div>
        </div>
        <div class="badge ${isOnline ? 'badge-online' : 'badge-offline'}">
          <span class="status-dot ${isOnline ? 'online' : 'offline'}"></span>
          ${isOnline ? 'آنلاین' : 'آفلاین'}
        </div>
      </div>

      <div style="display: flex; flex-direction: column; gap: 8px; font-size: 12px;">
        <div>
          <div style="display: flex; justify-content: space-between; color: var(--text-muted);">
            <span>مصرف پردازنده (CPU):</span>
            <span style="font-family: 'JetBrains Mono'; color: #f8fafc;">${cpu}%</span>
          </div>
          <div style="background: rgba(255,255,255,0.06); height: 4px; border-radius: 2px; margin-top: 4px; overflow: hidden;">
            <div style="width: ${Math.min(100, cpu * 3)}%; background: var(--accent-amber); height: 100%;"></div>
          </div>
        </div>

        <div>
          <div style="display: flex; justify-content: space-between; color: var(--text-muted);">
            <span>مصرف رم (RAM):</span>
            <span style="font-family: 'JetBrains Mono'; color: #f8fafc;">${ramU} MB / ${ramT} MB</span>
          </div>
          <div style="background: rgba(255,255,255,0.06); height: 4px; border-radius: 2px; margin-top: 4px; overflow: hidden;">
            <div style="width: ${Math.min(100, (ramU / ramT) * 100)}%; background: var(--accent-emerald); height: 100%;"></div>
          </div>
        </div>
      </div>
    `;
    container.appendChild(card);
  });
}

function renderNodes() {
  const tbody = document.getElementById('nodes-table-body');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (STATE.nodes.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="7" style="text-align: center; padding: 36px 16px; color: var(--cf-text-muted);">
          <div style="font-size: 32px; margin-bottom: 8px;">🌐</div>
          <div style="font-size: 14px; font-weight: 600; color: var(--cf-text-secondary);">هنوز هیچ نودی ثبت نشده است</div>
          <p style="font-size: 12px; margin-top: 4px;">برای شروع، حداقل یک نود ایران و یک نود خارج اضافه کنید.</p>
          <button class="btn btn-primary btn-sm" onclick="openAddNodeModal()" style="margin-top: 14px;">+ تعریف نود جدید</button>
        </td>
      </tr>
    `;
    return;
  }

  STATE.nodes.forEach(node => {
    const tr = document.createElement('tr');
    const isOnline = node.status === 'online';
    const roleBadge = node.role === 'iran'
      ? '<span class="badge" style="background: rgba(16, 185, 129, 0.15); color: #34d399; font-size: 11px; margin-right: 6px;">سرور ایران</span>'
      : '<span class="badge" style="background: rgba(59, 130, 246, 0.15); color: #60a5fa; font-size: 11px; margin-right: 6px;">سرور خارج</span>';

    tr.innerHTML = `
      <td style="font-weight: 800;">
        <span style="margin-left: 6px;">${node.flag || '🌐'}</span>
        ${node.name}
        ${roleBadge}
      </td>
      <td>${node.country_name || 'نامشخص'} (${node.country_code || 'XX'})</td>
      <td>
        <span style="font-family: 'JetBrains Mono', monospace; color: var(--cf-text-primary); cursor: pointer;" onclick="navigator.clipboard.writeText('${node.ip}'); showToast('آدرس IP کپی شد', 'success')" title="کلیک برای کپی IP">
          ${node.ip} 📋
        </span>
      </td>
      <td>
        <div class="badge ${isOnline ? 'badge-online' : 'badge-offline'}">
          <span class="status-dot ${isOnline ? 'online' : 'offline'}"></span>
          ${isOnline ? 'آنلاین' : 'آفلاین'}
        </div>
      </td>
      <td>
        <div style="font-family: 'JetBrains Mono', monospace; font-size: 11px; display: flex; flex-direction: column; gap: 4px;">
          <div style="display: flex; justify-content: space-between; gap: 8px;">
            <span style="color: var(--cf-text-muted);">CPU:</span>
            <span style="color: ${(node.cpu_percent || 0) > 80 ? '#ef4444' : '#10b981'}; font-weight: 600;">${node.cpu_percent || 0}%</span>
          </div>
          <div style="display: flex; justify-content: space-between; gap: 8px;">
            <span style="color: var(--cf-text-muted);">RAM:</span>
            <span>${node.ram_used_mb || 0} MB</span>
          </div>
        </div>
      </td>
      <td>
        <button class="btn btn-secondary btn-sm" onclick="showInstallModal('${node.token}', '${node.role}', '${node.name}')" title="دستور نصب روی سرور">
          دستور نصب 📋
        </button>
      </td>
      <td>
        <button class="btn btn-danger btn-sm" onclick="deleteNode('${node.id}')" title="حذف نود">حذف</button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

function populateLogSelectors() {
  const nodeSelect = document.getElementById('logs-node');
  const sourceSelect = document.getElementById('logs-source');
  if (!nodeSelect || !sourceSelect) return;
  const selectedNode = nodeSelect.value;
  const selectedSource = sourceSelect.value;
  nodeSelect.innerHTML = '<option value="">همهٔ نودها</option>' + STATE.nodes.map(n => `<option value="${n.id}">${n.flag || '🌐'} ${n.name}</option>`).join('');
  sourceSelect.innerHTML = '<option value="">همهٔ لاگ‌ها</option><option value="agent">لاگ Agent</option>' + STATE.tunnels.map(t => `<option value="tunnel:${t.id}">تانل: ${t.name}</option>`).join('');
  nodeSelect.value = selectedNode;
  sourceSelect.value = selectedSource;
}

async function loadLogs() {
  populateLogSelectors();
  const node = document.getElementById('logs-node')?.value || '';
  const source = document.getElementById('logs-source')?.value || '';
  const output = document.getElementById('logs-output');
  const meta = document.getElementById('logs-meta');
  if (!output) return;
  output.textContent = 'در حال دریافت لاگ…';
  try {
    const params = new URLSearchParams(); if (node) params.set('node_id', node); if (source) params.set('source', source);
    const data = await fetch(`/api/logs?${params}`).then(r => r.json());
    const nodes = Object.fromEntries(STATE.nodes.map(n => [n.id, n]));
    output.textContent = (data.logs || []).map(l => `===== ${(nodes[l.node_id]?.name || l.node_id)} • ${l.source} =====\n${l.content}`).join('\n\n') || 'هنوز لاگی از agent دریافت نشده؛ حداکثر ۱۲ ثانیه صبر و دوباره بروزرسانی کن.';
    if (meta) meta.textContent = `${(data.logs || []).length} منبع لاگ`;
  } catch (e) { output.textContent = `خطا در دریافت لاگ: ${e.message}`; }
}

async function copyVisibleLogs() {
  const content = document.getElementById('logs-output')?.textContent || '';
  if (!content || content.startsWith('در حال دریافت')) return showToast('لاگی برای کپی وجود ندارد.', 'info');
  try {
    await navigator.clipboard.writeText(content);
    showToast('لاگ کپی شد.', 'success');
  } catch (e) { showToast('مرورگر اجازهٔ کپی نداد.', 'error'); }
}

async function clearVisibleLogs() {
  const node = document.getElementById('logs-node')?.value || '';
  const source = document.getElementById('logs-source')?.value || '';
  const scope = node || source ? 'لاگ انتخاب‌شده' : 'همهٔ snapshotهای لاگ';
  if (!confirm(`${scope} از پنل پاک می‌شود. فایل لاگ سرورها حذف نمی‌شود. ادامه می‌دهید؟`)) return;
  try {
    const params = new URLSearchParams(); if (node) params.set('node_id', node); if (source) params.set('source', source);
    const data = await fetch(`/api/logs?${params}`, { method: 'DELETE' }).then(r => r.json());
    if (!data.success) throw new Error('درخواست ناموفق بود');
    showToast(`${data.deleted} snapshot پاک شد؛ گزارش جدید agent دوباره ظاهر می‌شود.`, 'success');
    loadLogs();
  } catch (e) { showToast(`پاک‌سازی ناموفق بود: ${e.message}`, 'error'); }
}

function renderTunnels() {
  const tbody = document.getElementById('tunnels-table-body');
  if (!tbody) return;
  tbody.innerHTML = '';

  if (STATE.tunnels.length === 0) {
    tbody.innerHTML = `
      <tr>
        <td colspan="9" style="text-align: center; padding: 36px 16px; color: var(--cf-text-muted);">
          <div style="font-size: 32px; margin-bottom: 8px;">⚡</div>
          <div style="font-size: 14px; font-weight: 600; color: var(--cf-text-secondary);">هنوز هیچ تانلی ساخته نشده است</div>
          <p style="font-size: 12px; margin-top: 4px;">برای اتصال نود ایران به خارج، یک تانل جدید ایجاد کنید.</p>
          <button class="btn btn-primary btn-sm" onclick="openAddTunnelModal()" style="margin-top: 14px;">+ ساخت تانل جدید</button>
        </td>
      </tr>
    `;
    return;
  }

  STATE.tunnels.forEach(tun => {
    const tr = document.createElement('tr');
    let engineBadge = '<span style="color: #60a5fa; font-weight: 700;">🚀 Backhaul Core</span>';
    if (tun.core_type === 'hawal') {
      engineBadge = '<span style="color: var(--accent-amber); font-weight: 700;">⚡ هسته Go Stealth</span>';
    } else if (tun.core_type === 'paqet') {
      engineBadge = '<span style="color: #10b981; font-weight: 700;">🛡️ Paqet Raw KCP</span>';
    } else if (tun.core_type === 'gost') {
      engineBadge = '<span style="color: #7c3aed; font-weight: 700;">👻 GOST Relay</span>';
    }
    const isRunning = tun.status === 'running';
    const bIn = tun.bytes_in || 0;
    const bOut = tun.bytes_out || 0;

    let portsBadges = '';
    (tun.ports || []).forEach(p => {
      portsBadges += `<span class="badge badge-tag" style="margin-left: 4px;">${p}</span>`;
    });

    const testRes = STATE.tunnelTestResults[tun.id];
    let testBadgeHtml = '';
    if (testRes) {
      if (testRes.loading) {
        testBadgeHtml = '<span class="badge" style="background:rgba(245,158,11,0.2); color:#fde68a;">در حال تست... ⏳</span>';
      } else if (testRes.success) {
        testBadgeHtml = `<span class="badge badge-online">🟢 ${testRes.latency_avg_ms}ms • ${testRes.packet_loss}% Loss</span>`;
      } else {
        testBadgeHtml = '<span class="badge badge-offline">🔴 عدم اتصال</span>';
      }
    } else {
      testBadgeHtml = `<button class="btn btn-test btn-sm" onclick="testTunnel('${tun.id}')">⚡ تست پینگ و سلامت</button>`;
    }

    tr.innerHTML = `
      <td>
        <div style="font-weight: 800;">${tun.name}</div>
        <div style="font-size: 11px; margin-top: 2px;">
          ${engineBadge}
        </div>
      </td>
      <td style="font-family: 'JetBrains Mono'; font-size: 13px;">${tun.server_name || 'Iran Node'}</td>
      <td style="font-family: 'JetBrains Mono'; font-size: 13px;">${tun.client_name || 'Germany Node'}</td>
      <td style="font-family: 'JetBrains Mono'; color: var(--accent-amber); font-weight: 700;">${tun.core_port}</td>
      <td>${portsBadges}</td>
      <td>
        <div style="font-family: 'JetBrains Mono'; font-size: 12px; display: flex; flex-direction: column; gap: 2px;">
          <span style="color: #34d399;">📥 ${formatBytes(bIn)}</span>
          <span style="color: #38bdf8;">📤 ${formatBytes(bOut)}</span>
        </div>
      </td>
      <td>
        <div id="test-col-${tun.id}">${testBadgeHtml}</div>
      </td>
      <td>
        <div class="badge ${isRunning ? 'badge-online' : 'badge-offline'}">
          <span class="status-dot ${isRunning ? 'online' : 'offline'}"></span>
          ${isRunning ? 'فعال' : 'متوقف'}
        </div>
      </td>
      <td>
        <div style="display: flex; gap: 6px;">
          <button class="btn btn-secondary btn-sm" onclick="restartTunnel('${tun.id}')" title="ری‌استارت همین تانل در هر دو نود">
            ↻ ری‌استارت
          </button>
          <button class="btn btn-edit btn-sm" onclick="openEditTunnelModal('${tun.id}')" title="ویرایش پورت‌ها و تنظیمات">
            ✏️ ویرایش
          </button>
          <button class="btn btn-secondary btn-sm" onclick="showDockerModal('${tun.id}')" title="فایل داکر">
            🐳
          </button>
          <button class="btn btn-danger btn-sm" onclick="deleteTunnel('${tun.id}')" title="حذف تانل">
            حذف
          </button>
        </div>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

// -------------------------------------------------------------
// Live Tunnel Health & Ping Tester
// -------------------------------------------------------------
async function testTunnel(tunnelId) {
  STATE.tunnelTestResults[tunnelId] = { loading: true };
  renderTunnels();

  try {
    const res = await fetch(`/api/tunnels/${tunnelId}/test`, { method: 'POST' });
    const data = await res.json();
    STATE.tunnelTestResults[tunnelId] = data;
    renderTunnels();

    if (data.success) {
      showToast(`تست تانل موفق بود: تاخیر ${data.latency_avg_ms}ms روی پورت ${data.tested_port}`, 'success');
    } else {
      showToast(`خطا در ارتباط با تانل: ${data.error || 'پاسخ دریافت نشد'}`, 'error');
    }
  } catch (e) {
    STATE.tunnelTestResults[tunnelId] = { success: false, error: e.message };
    renderTunnels();
    showToast('خطا در اجرای تست پینگ تانل', 'error');
  }
}

async function restartTunnel(tunnelId) {
  if (!confirm('این تانل در هر دو نود برای چند ثانیه قطع و دوباره اجرا می‌شود. ادامه می‌دهید؟')) return;
  try {
    const res = await fetch(`/api/tunnels/${tunnelId}/restart`, { method: 'POST' });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'درخواست ناموفق بود');
    showToast('دستور ری‌استارت تانل ارسال شد؛ حداکثر تا چند ثانیه اعمال می‌شود.', 'success');
  } catch (e) {
    showToast(`ری‌استارت تانل ناموفق بود: ${e.message}`, 'error');
  }
}

async function restartAllAgents() {
  if (!confirm('همهٔ ایجنت‌ها و تانل‌ها برای چند ثانیه ری‌استارت می‌شوند. ادامه می‌دهید؟')) return;
  try {
    const res = await fetch('/api/agents/restart', { method: 'POST' });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'درخواست ناموفق بود');
    showToast(`دستور ری‌استارت برای ${data.nodes} ایجنت ارسال شد.`, 'success');
  } catch (e) {
    showToast(`ری‌استارت ایجنت‌ها ناموفق بود: ${e.message}`, 'error');
  }
}

// -------------------------------------------------------------
// Edit Tunnel Flow
// -------------------------------------------------------------
function openEditTunnelModal(tunnelId) {
  const tunnel = STATE.tunnels.find(t => t.id === tunnelId);
  if (!tunnel) return;

  document.getElementById('edit-tunnel-id').value = tunnel.id;
  document.getElementById('edit-tunnel-name').value = tunnel.name;
  document.getElementById('edit-tunnel-core-port').value = tunnel.core_port;
  
  const coreTypeSelect = document.getElementById('edit-tunnel-core-type');
  if (coreTypeSelect) coreTypeSelect.value = tunnel.core_type || 'hawal';
  handleEditCoreTypeChange();

  // Extract port numbers
  STATE.editPortTags = (tunnel.ports || []).map(rule => {
    const p = parseInt(rule.split('=')[0]);
    return isNaN(p) ? 80 : p;
  });

  renderEditPortChips();
  document.getElementById('modal-edit-tunnel').classList.add('active');
}

async function handleSaveEditTunnel(e) {
  e.preventDefault();
  const tunnelId = document.getElementById('edit-tunnel-id').value;
  const name = document.getElementById('edit-tunnel-name').value;
  const coreType = document.getElementById('edit-tunnel-core-type').value;
  const corePort = parseInt(document.getElementById('edit-tunnel-core-port').value);
  const transport = document.getElementById('edit-tunnel-transport').value;

  if (STATE.editPortTags.length === 0) {
    showToast('حداقل یک پورت باید برای تانل مشخص شود', 'error');
    return;
  }

  const ports = STATE.editPortTags.map(p => `${p}=127.0.0.1:${p}`);

  try {
    const res = await fetch(`/api/tunnels/${tunnelId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: jsonStringifySafe({
        name,
        core_type: coreType,
        core_port: corePort,
        transport,
        ports
      })
    });

    const data = await res.json();
    if (res.ok) {
      showToast('تنظیمات و پورت‌های تانل با موفقیت بروزرسانی شد', 'success');
      closeModal('modal-edit-tunnel');
      fetchData();
    } else {
      showToast(data.error || 'خطا در ویرایش تانل', 'error');
    }
  } catch (err) {
    showToast('خطا در برقراری ارتباط با سرور', 'error');
  }
}

function jsonStringifySafe(obj) {
  return JSON.stringify(obj);
}

// -------------------------------------------------------------
// Modals & Handlers
// -------------------------------------------------------------
function openAddTunnelModal() {
  document.getElementById('modal-add-tunnel').classList.add('active');
  const sSelect = document.getElementById('tunnel-server-node');
  const cSelect = document.getElementById('tunnel-client-node');
  if (sSelect && cSelect) {
    sSelect.innerHTML = '';
    cSelect.innerHTML = '';
    STATE.nodes.forEach(n => {
      sSelect.innerHTML += `<option value="${n.id}" ${n.role === 'iran' ? 'selected' : ''}>${n.flag || '🇮🇷'} ${n.name} (${n.ip})</option>`;
      cSelect.innerHTML += `<option value="${n.id}" ${n.role === 'kharej' ? 'selected' : ''}>${n.flag || '🌐'} ${n.name} (${n.ip})</option>`;
    });
  }
  renderPortChips();
}

function openAddNodeModal() {
  document.getElementById('modal-add-node').classList.add('active');
}

function closeModal(modalId) {
  const modal = document.getElementById(modalId);
  if (modal) modal.classList.remove('active');
}

async function handleAddNode(e) {
  e.preventDefault();
  const name = document.getElementById('node-name').value;
  const ip = document.getElementById('node-ip').value;
  const role = document.getElementById('node-role').value;

  try {
    const res = await fetch('/api/nodes', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, ip, role })
    });
    const data = await res.json();
    closeModal('modal-add-node');
    showInstallModal(data.token, role, name);
    showToast('نود جدید با موفقیت ایجاد شد', 'success');
    fetchData();
  } catch (err) {
    showToast('خطا در ساخت نود', 'error');
  }
}

async function handleAddTunnel(e) {
  e.preventDefault();
  const name = document.getElementById('tunnel-name').value;
  const coreType = document.getElementById('tunnel-core-type').value;
  const serverNodeId = document.getElementById('tunnel-server-node').value;
  const clientNodeId = document.getElementById('tunnel-client-node').value;
  const corePort = parseInt(document.getElementById('tunnel-core-port').value);
  const transport = document.getElementById('tunnel-transport').value;

  if (STATE.activePortTags.length === 0) {
    showToast('حداقل یک پورت باید اضافه شود', 'error');
    return;
  }

  const ports = STATE.activePortTags.map(p => `${p}=127.0.0.1:${p}`);

  try {
    const res = await fetch('/api/tunnels', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        core_type: coreType,
        server_node_id: serverNodeId,
        client_node_id: clientNodeId,
        core_port: corePort,
        transport,
        ports
      })
    });
    const data = await res.json();
    if (res.ok) {
      showToast('تانل با موفقیت ایجاد و فعال شد', 'success');
      closeModal('modal-add-tunnel');
      fetchData();
    } else {
      showToast(data.error || 'خطا در ساخت تانل', 'error');
    }
  } catch (err) {
    showToast('خطا در برقراری ارتباط با سرور', 'error');
  }
}

async function deleteTunnel(tunnelId) {
  if (!confirm('آیا از حذف این تانل مطمئن هستید؟')) return;
  try {
    await fetch(`/api/tunnels/${tunnelId}`, { method: 'DELETE' });
    showToast('تانل با موفقیت حذف شد', 'info');
    fetchData();
  } catch (e) {
    showToast('خطا در حذف تانل', 'error');
  }
}

async function deleteNode(nodeId) {
  if (!confirm('آیا از حذف این نود مطمئن هستید؟')) return;
  try {
    await fetch(`/api/nodes/${nodeId}`, { method: 'DELETE' });
    showToast('نود با موفقیت حذف شد', 'info');
    fetchData();
  } catch (e) {
    showToast('خطا در حذف نود', 'error');
  }
}

function showInstallModal(token, role, name) {
  STATE.currentInstallerToken = token;
  STATE.currentInstallerRole = role;
  STATE.currentInstallerName = name;
  updateInstallerCommand();
  document.getElementById('modal-install-code').classList.add('active');
}

function switchInstallerMode(mode) {
  STATE.installerMode = mode;
  document.getElementById('btn-mode-native').className = mode === 'native' ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm';
  document.getElementById('btn-mode-docker').className = mode === 'docker' ? 'btn btn-primary btn-sm' : 'btn btn-secondary btn-sm';
  updateInstallerCommand();
}

function updateInstallerCommand() {
  const origin = window.location.origin;
  const cmdBox = document.getElementById('install-command-text');
  if (!cmdBox) return;

  if (STATE.installerMode === 'native') {
    cmdBox.innerText = `curl -fsSL ${origin}/install.sh | bash -s -- --panel ${origin} --token ${STATE.currentInstallerToken}`;
  } else {
    cmdBox.innerText = `docker run -d --name hawal-agent --restart=always --network=host -e PANEL_URL="${origin}" -e AGENT_TOKEN="${STATE.currentInstallerToken}" ghcr.io/dalroot/hawal-agent:latest`;
  }
}

function copyInstallCmd() {
  const text = document.getElementById('install-command-text').innerText;
  navigator.clipboard.writeText(text);
  showToast('دستور با موفقیت در کلیپ‌بورد کپی شد', 'success');
}

async function showDockerModal(tunnelId) {
  try {
    const res = await fetch(`/api/tunnels/${tunnelId}/docker`);
    const data = await res.json();
    document.getElementById('docker-compose-text').innerText = data.server_compose || '# Docker Compose File';
    document.getElementById('modal-tunnel-docker').classList.add('active');
  } catch (e) {
    showToast('خطا در بارگذاری اطلاعات داکر', 'error');
  }
}

function copyDockerCompose() {
  const text = document.getElementById('docker-compose-text').innerText;
  navigator.clipboard.writeText(text);
  showToast('فایل docker-compose کپی شد', 'success');
}

function renderPingSection() {
  const select = document.getElementById('ping-target-select');
  if (select) {
    select.innerHTML = '';
    STATE.nodes.forEach(n => {
      select.innerHTML += `<option value="${n.id}">${n.flag || '🌐'} ${n.name} (${n.ip})</option>`;
    });
  }

  const tbody = document.getElementById('ping-history-body');
  if (tbody) {
    tbody.innerHTML = '';
    STATE.pings.forEach(p => {
      const tr = document.createElement('tr');
      const d = new Date(p.created_at * 1000).toLocaleTimeString('fa-IR');
      tr.innerHTML = `
        <td style="font-family: 'JetBrains Mono'; font-size: 12px;">${d}</td>
        <td>${p.source_name}</td>
        <td>${p.target_name}</td>
        <td style="font-family: 'JetBrains Mono'; color: var(--accent-amber); font-weight: 700;">${p.latency_avg_ms} ms</td>
        <td style="font-family: 'JetBrains Mono'; color: ${p.packet_loss === 0 ? 'var(--accent-emerald)' : 'var(--accent-rose)'};">${p.packet_loss}%</td>
      `;
      tbody.appendChild(tr);
    });
  }
}

async function triggerPingTest() {
  const targetId = document.getElementById('ping-target-select').value;
  if (!targetId) return;

  showToast('در حال ارسال پکت‌های تست پینگ و پکت‌لاس...', 'info');

  try {
    const res = await fetch('/api/ping/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target_node_id: targetId })
    });
    const data = await res.json();
    
    document.getElementById('ping-result-container').style.display = 'block';
    document.getElementById('ping-res-avg').innerText = `${data.latency_avg_ms} ms`;
    document.getElementById('ping-res-min').innerText = `${data.latency_min_ms} ms`;
    document.getElementById('ping-res-max').innerText = `${data.latency_max_ms} ms`;
    document.getElementById('ping-res-loss').innerText = `${data.packet_loss}%`;

    showToast('تست پینگ با موفقیت انجام شد', 'success');
    fetchData();
  } catch (e) {
    showToast('خطا در اجرای تست پینگ', 'error');
  }
}

async function handleSaveSettings(e) {
  e.preventDefault();
  const panelPort = parseInt(document.getElementById('setting-panel-port').value);
  const defaultTransport = document.getElementById('setting-default-transport').value;
  const defaultMux = parseInt(document.getElementById('setting-default-mux').value);
  const mtuClamp = parseInt(document.getElementById('setting-mtu-clamp').value);
  const publicPanelUrl = document.getElementById('setting-public-panel-url').value.trim().replace(/\/$/, '');

  try {
    const res = await fetch('/api/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        panel_port: panelPort,
        default_transport: defaultTransport,
        default_mux: defaultMux,
        mtu_clamp: mtuClamp,
        public_panel_url: publicPanelUrl
      })
    });
    if (res.ok) {
      showToast('تنظیمات با موفقیت ذخیره شد', 'success');
    }
  } catch (e) {
    showToast('خطا در ذخیره تنظیمات', 'error');
  }
}

// -------------------------------------------------------------
// App Initialization
// -------------------------------------------------------------
document.addEventListener('DOMContentLoaded', () => {
  const initialTab = getTabFromUrl();
  switchTab(initialTab, false);
  fetchData();
  connectWebSocket();
  renderPortChips();
});
