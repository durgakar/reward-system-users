const $ = (sel) => document.querySelector(sel);

function headers() {
  const h = { 'Content-Type': 'application/json' };
  const key = $('#apiKey').value.trim();
  if (key) h['Authorization'] = 'Bearer ' + key;
  return h;
}

async function api(path, opts = {}) {
  const res = await fetch('/api/v1' + path, { ...opts, headers: { ...headers(), ...(opts.headers || {}) } });
  const text = await res.text();
  let body = text;
  try { body = JSON.parse(text); } catch (_) {}
  if (!res.ok) throw new Error(typeof body === 'object' && body.error ? body.error : text);
  return body;
}

function setStatus(msg) { $('#status').textContent = msg; }

function fmtMoney(n) { return '$' + Number(n).toFixed(2); }

function tagList(items) {
  if (!items || !items.length) return '<span class="tag muted">none</span>';
  return items.map((t) => `<span class="tag">${t}</span>`).join('');
}

function clientRow(c, compact) {
  const name = `${c.first_name} ${c.last_name}`;
  const segs = tagList(c.segments);
  const rules = tagList(c.expected_rules);
  if (compact) {
    return `<tr>
      <td>${name}</td>
      <td>${fmtMoney(c.lifetime_spend_usd)}</td>
      <td>${fmtMoney(c.last_order_total_usd)}</td>
      <td>${segs}</td>
      <td>${rules || '<span class="tag muted">—</span>'}</td>
    </tr>`;
  }
  return `<div class="card client-card">
    <div class="client-header"><strong>${name}</strong> <span class="muted">${c.email}</span></div>
    <div class="client-grid">
      <div><span class="label">Lifetime spend</span> ${fmtMoney(c.lifetime_spend_usd)}</div>
      <div><span class="label">Last order</span> ${fmtMoney(c.last_order_total_usd)}</div>
      <div><span class="label">Orders (90d)</span> ${c.orders_last_90_days}</div>
      <div><span class="label">Category</span> ${c.preferred_category}</div>
      <div><span class="label">Days since order</span> ${c.days_since_last_order}</div>
      <div><span class="label">Points balance</span> ${c.points_balance}</div>
    </div>
    <div class="client-tags"><span class="label">Segments</span> ${segs}</div>
    <div class="client-tags"><span class="label">Rules that would fire</span> ${rules || '<span class="tag muted">none on next run</span>'}</div>
  </div>`;
}

function renderRunResults(results) {
  if (!results || !results.length) {
    return '<p class="hint">No clients processed.</p>';
  }
  const rows = results.map((r) => `<tr>
    <td>${r.client_id}</td>
    <td>${tagList(r.segments)}</td>
    <td>${r.rule_matches}</td>
    <td>${r.points_awarded}</td>
    <td>${r.emails_sent}</td>
    <td>${r.errors && r.errors.length ? r.errors.join('; ') : '—'}</td>
  </tr>`).join('');
  return `<table class="data-table">
    <thead><tr><th>Client</th><th>Segments</th><th>Rules matched</th><th>Points</th><th>Emails</th><th>Errors</th></tr></thead>
    <tbody>${rows}</tbody>
  </table>`;
}

document.querySelectorAll('.tab').forEach((btn) => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach((b) => b.classList.remove('active'));
    document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
    btn.classList.add('active');
    $('#' + btn.dataset.tab).classList.add('active');
    refresh();
  });
});

async function loadClientDetails() {
  const items = await api('/clients/detail');
  $('#clientList').innerHTML = items.map((c) => clientRow(c, false)).join('');

  const overview = `<table class="data-table compact">
    <thead><tr><th>Client</th><th>Lifetime</th><th>Last order</th><th>Segments</th><th>Expected rules</th></tr></thead>
    <tbody>${items.map((c) => clientRow(c, true)).join('')}</tbody>
  </table>`;
  $('#overviewClients').innerHTML = overview;
  return items;
}

async function loadSegments() {
  const items = await api('/segments');
  $('#segmentList').innerHTML = items.map((s) => `
    <div class="card">
      <strong>${s.id}</strong> — ${s.description || ''}
      <pre>${JSON.stringify(s.match, null, 2)}</pre>
      <button class="danger" data-del-seg="${s.id}">Delete</button>
    </div>`).join('');
  document.querySelectorAll('[data-del-seg]').forEach((b) => b.onclick = async () => {
    await api('/segments/' + b.dataset.delSeg, { method: 'DELETE' });
    loadSegments();
  });
}

async function loadRules() {
  const items = await api('/rules');
  $('#ruleList').innerHTML = items.map((r) => `
    <div class="card">
      <strong>${r.id}</strong> — ${r.name} ${r.enabled ? '' : '(disabled)'}
      <div>Segment: ${r.segment || '(all clients)'}</div>
      <div class="muted">${r.description || ''}</div>
      <pre>${JSON.stringify({ condition: r.condition, actions: r.actions }, null, 2)}</pre>
      <button class="danger" data-del-rule="${r.id}">Delete</button>
    </div>`).join('');
  document.querySelectorAll('[data-del-rule]').forEach((b) => b.onclick = async () => {
    await api('/rules/' + b.dataset.delRule, { method: 'DELETE' });
    loadRules();
  });
}

async function loadRuns() {
  const items = await api('/campaigns/runs');
  $('#runList').innerHTML = items.length ? items.map((r) => `
    <div class="card">
      #${r.id} ${r.campaign_id} — ${r.status}
      <div>clients=${r.clients_processed} points=${r.points_awarded} emails=${r.emails_sent} errors=${r.errors_count}</div>
      <div>${r.started_at}${r.finished_at ? ' → ' + r.finished_at : ''}</div>
    </div>`).join('') : '<p>No runs yet.</p>';
}

function refresh() {
  const active = document.querySelector('.tab.active')?.dataset.tab;
  setStatus('');
  const load = (fn) => fn().catch((e) => setStatus(e.message));
  if (active === 'overview' || active === 'clients') load(loadClientDetails);
  if (active === 'segments') load(loadSegments);
  if (active === 'rules') load(loadRules);
  if (active === 'runs') load(loadRuns);
}

$('#segmentForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const payload = {
    id: fd.get('id'),
    description: fd.get('description'),
    match: JSON.parse(fd.get('match_json')),
  };
  await api('/segments/' + payload.id, { method: 'PUT', body: JSON.stringify(payload) });
  setStatus('Segment saved');
  e.target.reset();
  loadSegments();
});

$('#ruleForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const payload = {
    id: fd.get('id'),
    name: fd.get('name'),
    segment: fd.get('segment') || '',
    enabled: fd.get('enabled') === 'on',
    condition: JSON.parse(fd.get('condition_json')),
    actions: JSON.parse(fd.get('actions_json')),
  };
  await api('/rules/' + payload.id, { method: 'PUT', body: JSON.stringify(payload) });
  setStatus('Rule saved');
  e.target.reset();
  loadRules();
});

$('#runCampaign').addEventListener('click', async () => {
  setStatus('Running campaign...');
  try {
    const res = await api('/campaigns/run', { method: 'POST' });
    const totalPts = res.results.reduce((s, r) => s + r.points_awarded, 0);
    const totalEmails = res.results.reduce((s, r) => s + r.emails_sent, 0);
    setStatus(`Done — ${res.count} clients, ${totalPts} points, ${totalEmails} emails`);
    $('#lastRunResults').innerHTML = renderRunResults(res.results);
    document.querySelector('[data-tab="overview"]').click();
    loadRuns();
    loadClientDetails();
  } catch (err) {
    setStatus(err.message);
  }
});

const saved = localStorage.getItem('admin_api_key');
if (saved) $('#apiKey').value = saved;
else $('#apiKey').value = 'dev-key';
$('#apiKey').addEventListener('change', () => localStorage.setItem('admin_api_key', $('#apiKey').value));

refresh();
