const form = document.querySelector('#insight-filters');
const el = id => document.getElementById(id);
let requestVersion = 0;
async function get(path) {
  const response = await fetch('/api/' + path);
  if (!response.ok) throw new Error('Could not load insights. Please try again.');
  return response.json();
}
function chart(id, rows, unit) {
  const target = el(id);
  target.replaceChildren();
  if (!rows.length) { target.textContent = 'No data for this selection.'; return; }
  const max = Math.max(1, ...rows.map(row => row.count));
  for (const row of rows) {
    const item = document.createElement('div'); item.className = 'insight-bar-row';
    const label = document.createElement('span'); label.textContent = row.label;
    const count = document.createElement('span'); count.className = 'insight-bar-count';
    count.textContent = `${row.count.toLocaleString()} ${unit}${row.count === 1 ? '' : 's'}${row.reports === undefined ? '' : ` · ${row.reports} report${row.reports === 1 ? '' : 's'}`}`;
    const track = document.createElement('div'); track.className = 'insight-track'; track.setAttribute('aria-hidden', 'true');
    const bar = document.createElement('div'); bar.style.width = `${row.count / max * 100}%`; track.append(bar);
    item.append(label, count, track); target.append(item);
  }
}
function fillMonths(rows) {
  if (!rows.length) return [];
  const counts = new Map(rows.map(row => [row.label, row.count]));
  const [year, month] = rows[0].label.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, 1));
  const result = [];
  while (true) {
    const key = `${date.getUTCFullYear()}-${String(date.getUTCMonth()+1).padStart(2,'0')}`;
    result.push({label:key,count:counts.get(key) || 0});
    if (key === rows[rows.length-1].label) return result;
    date.setUTCMonth(date.getUTCMonth()+1);
  }
}
async function load() {
  const version = ++requestVersion;
  el('insight-status').textContent = 'Loading insights…';
  el('insight-results').hidden = true; el('insight-retry').hidden = true;
  try {
    const filters = await get('filters');
    if (version !== requestVersion) return;
    for (const [name, key] of [['company','companies'],['field','fields'],['role','roles'],['year','years']]) {
      const select = form.elements[name], value = select.value;
      while (select.options.length > 1) select.remove(1);
      filters[key].forEach(value => select.add(new Option(value, value)));
      select.value = [...select.options].some(option => option.value === value) ? value : '';
    }
    const query = new URLSearchParams();
    for (const [key,value] of new FormData(form)) if (value) query.set(key,value);
    const data = await get('insights?' + query);
    if (version !== requestVersion) return;
    for (const key of ['reports','questions','companies']) el('total-' + key).textContent = data[key].toLocaleString();
    chart('topic-chart', data.topics, 'question');
    chart('company-chart', data.companyBreakdown, 'report');
    chart('month-chart', fillMonths(data.months), 'report');
    el('untagged-note').textContent = `${data.untagged} questions have no topic and are excluded from topic counts.`;
    el('undated-note').textContent = `${data.undated} reports have no interview date and are excluded from the timeline.`;
    el('insight-empty').hidden = data.reports !== 0;
    el('insight-results').hidden = false;
    el('insight-status').textContent = `Based on ${data.reports} submitted report${data.reports === 1 ? '' : 's'}.`;
  } catch (error) {
    if (version !== requestVersion) return;
    el('insight-status').textContent = error.message; el('insight-retry').hidden = false;
  }
}
form.onsubmit = event => event.preventDefault();
form.onchange = load;
form.onreset = () => setTimeout(load, 0);
el('insight-retry').onclick = load;
window.addEventListener('focus', load);
window.addEventListener('storage', event => { if (event.key === 'interview-reports-updated') load(); });
const heartbeat = () => { if (!document.hidden) fetch('/api/visits', {method:'POST'}).catch(() => {}); };
document.addEventListener('visibilitychange', () => { if (!document.hidden) { heartbeat(); load(); } });
heartbeat(); setInterval(heartbeat,60000); load();
