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

document.querySelectorAll('.tab').forEach((btn) => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach((b) => b.classList.remove('active'));
    document.querySelectorAll('.panel').forEach((p) => p.classList.remove('active'));
    btn.classList.add('active');
    $('#' + btn.dataset.tab).classList.add('active');
    refresh();
  });
});

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
      <div>Segment: ${r.segment || '(all)'}</div>
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
  if (active === 'segments') loadSegments().catch((e) => setStatus(e.message));
  if (active === 'rules') loadRules().catch((e) => setStatus(e.message));
  if (active === 'runs') loadRuns().catch((e) => setStatus(e.message));
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
    setStatus('Campaign finished for ' + res.count + ' clients');
    loadRuns();
  } catch (err) {
    setStatus(err.message);
  }
});

refresh();
