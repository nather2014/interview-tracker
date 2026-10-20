const $ = s => document.querySelector(s);
const admin = location.pathname === '/admin.html';
let token = '', page = 0, requestId = 0, editingId = null;
const loadedReports = new Map();
let editReport;
function announceReportChange() {
  try { localStorage.setItem('interview-reports-updated', String(Date.now())); } catch (_) {}
}
const escape = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function api(path, options = {}) {
  const response = await fetch('/api' + path, options);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.message || 'Something went wrong. Please try again.');
  }
  return response.status === 204 ? null : response.json();
}
async function filters(current) {
  const data = await api('/filters');
  if (current !== requestId) return;
  for (const [name, values, label] of [['field',data.fields,'All fields'],['role',data.roles,'All roles'],['company',data.companies,'All companies'],['topic',data.topics,'All topics'],['year',data.years,'All years']]) {
    const select = $(`#filters [name="${name}"]`);
    if (!select) continue;
    const old = [...select.selectedOptions].map(option => option.value);
    select.innerHTML = `<option value="">${label}</option>` + values.map(v => `<option value="${escape(v)}">${escape(v)}</option>`).join('');
    for (const option of select.options) option.selected = old.includes(option.value);
    if (![...select.selectedOptions].some(option => option.value)) select.value = '';
  }
}
async function browse() {
  const current = ++requestId;
  $('#result-count').textContent = 'Loading reports…';
  try {
    if (!admin) await filters(current);
    if (current !== requestId) return;
    const query = new URLSearchParams($('#filters') ? new FormData($('#filters')) : undefined);
    for (const [key,value] of [...query]) if (!value) query.delete(key);
    query.set('page', page);
    const data = await api('/interviews?' + query);
    if (current !== requestId) return;
    $('#result-count').textContent = `${data.total} interview report${data.total === 1 ? '' : 's'}`;
    loadedReports.clear();
    data.items.forEach(report => loadedReports.set(String(report.id), report));
    $('#reports').innerHTML = data.items.map(i => `<article class="report"><div class="report-header"><div><div class="company">${escape(i.company)}</div><h3>${escape(i.role)}</h3><div class="metadata">${[i.field, i.experience && i.experience + ' years experience', i.interviewType, i.interviewDate, i.location].filter(Boolean).map(escape).join(' · ')}</div></div><div>${[...new Set(i.questions.map(q => q.topic))].map(t => `<span class="tag">${escape(t)}</span>`).join('')}</div></div><ol class="questions">${i.questions.map(q => `<li>${escape(q.question)}<span class="question-topic">${escape(q.topic)}</span></li>`).join('')}</ol>${admin ? `<button data-edit="${i.id}">Edit report</button> <button class="danger" data-delete="${i.id}">Remove report</button>` : ''}</article>`).join('') || `<div class="empty"><h3>${query.size > 1 ? 'No matching interviews yet.' : 'No interviews yet.'}</h3><p class="muted">${query.size > 1 ? 'Try a different search or clear your filters.' : ''}</p></div>`;
    $('#previous').disabled = page === 0;
    $('#next').disabled = (page + 1) * data.pageSize >= data.total;
    $('#page-label').textContent = `Page ${page + 1} of ${Math.max(1, Math.ceil(data.total / data.pageSize))}`;
  } catch (error) {
    if (current !== requestId) return;
    $('#result-count').textContent = 'Could not load reports';
    $('#reports').innerHTML = `<div class="empty"><p class="error">${escape(error.message)}</p><button id="retry">Try again</button></div>`;
    $('#retry').onclick = browse;
  }
}
$('#previous').onclick = () => { if (page > 0) { page--; browse(); } };
$('#next').onclick = () => { page++; browse(); };
if (admin) {
  let visitorTimer, visitorRequest = 0;
  async function visitors() {
    const current = ++visitorRequest;
    try {
      const data = await api('/admin/visitors', {headers:{Authorization:'Bearer ' + token}});
      if (!token || current !== visitorRequest) return;
      $('#visitors-online').textContent = data.online;
      $('#visitors-today').textContent = data.today;
      $('#visitor-status').textContent = 'Updates every 15 seconds.';
    } catch (error) {
      if (token && current === visitorRequest) {
        $('#visitors-online').textContent = '—'; $('#visitors-today').textContent = '—';
        $('#visitor-status').textContent = 'Visitor counts unavailable. Retrying…';
      }
    }
  }
  $('#login').onsubmit = async event => {
    event.preventDefault();
    token = $('#admin-token').value;
    try {
      await api('/admin/session', {headers:{Authorization:'Bearer ' + token}});
      $('#login').hidden = true; $('#admin-content').hidden = false;
      $('#admin-token').value = ''; $('#login-error').textContent = ''; browse();
      clearInterval(visitorTimer); visitors();
      visitorTimer = setInterval(() => { if (!document.hidden) visitors(); }, 15000);
    } catch (error) { token = ''; $('#login-error').textContent = error.message; }
  };
  $('#logout').onclick = () => { clearInterval(visitorTimer); visitorRequest++; token = ''; $('#login').hidden = false; $('#admin-content').hidden = true; $('#reports').replaceChildren(); $('#submission').close(); editingId = null; };
  $('#reports').onclick = async event => {
    const edit = event.target.closest('[data-edit]');
    if (edit) { editReport(loadedReports.get(edit.dataset.edit)); return; }
    const button = event.target.closest('[data-delete]');
    if (!button || !confirm('Permanently remove this interview and all its questions?')) return;
    button.disabled = true;
    try {
      await api('/admin/interviews/' + button.dataset.delete, {method:'DELETE',headers:{Authorization:'Bearer ' + token}});
      page = 0; $('#notice').textContent = 'Report removed.'; announceReportChange(); browse();
    } catch (error) { $('#notice').textContent = error.message; button.disabled = false; }
  };
} else {
  const heartbeat = () => { if (!document.hidden) api('/visits', {method:'POST'}).catch(() => {}); };
  heartbeat();
  setInterval(heartbeat, 60000);
  document.addEventListener('visibilitychange', heartbeat);
  let timer;
  $('#filters').onsubmit = event => event.preventDefault();
  $('#filters').oninput = () => { clearTimeout(timer); timer = setTimeout(() => { page = 0; browse(); }, 250); };
  $('#filters').onreset = () => setTimeout(() => { page = 0; browse(); }, 0);
  const refresh = () => { if (!document.hidden) { page = 0; browse(); } };
  window.addEventListener('focus', refresh);
  document.addEventListener('visibilitychange', refresh);
  window.addEventListener('storage', event => { if (event.key === 'interview-reports-updated') refresh(); });
  browse();
}
if ($('#submission-form')) {
  const dialog = $('#submission'), form = $('#submission-form');
  const open = () => { $('#form-error').textContent = ''; dialog.showModal(); };
  if ($('#share')) $('#share').onclick = open;
  $('#close').onclick = () => { suggestionVersion++; clearTimeout(topicTimer); dialog.close(); };
  dialog.addEventListener('cancel', () => { suggestionVersion++; clearTimeout(topicTimer); });
  dialog.addEventListener('close', () => { suggestionVersion++; clearTimeout(topicTimer); });
  const today = new Date();
  form.elements.interviewDate.max = `${today.getFullYear()}-${String(today.getMonth()+1).padStart(2,'0')}-${String(today.getDate()).padStart(2,'0')}`;
  function addQuestion(question = '', topic = '', source = 'manual') {
    if ($('#question-fields').children.length >= 50) return;
    const row = document.createElement('div'); row.className = 'question-field';
    row.innerHTML = '<label>Question<textarea data-question maxlength="2000"></textarea></label><label>Topic<input data-topic maxlength="80"></label><span class="topic-hint muted" aria-live="polite"></span><button type="button">Remove question</button>';
    row.dataset.topicSource = source;
    row.querySelector('[data-question]').value = question;
    row.querySelector('[data-topic]').value = topic;
    row.querySelector('[data-question]').oninput = () => {
      if (row.dataset.topicSource === 'auto') { row.querySelector('[data-topic]').value = ''; row.querySelector('.topic-hint').textContent = ''; }
      scheduleTopics();
    };
    row.querySelector('[data-topic]').oninput = () => { row.dataset.topicSource = 'manual'; row.querySelector('.topic-hint').textContent = ''; };
    row.querySelector('button').onclick = () => { if ($('#question-fields').children.length > 1) row.remove(); };
    $('#question-fields').append(row);
  }
  let topicTimer, suggestionVersion = 0;
  async function suggestMissingTopics() {
    if (!$('#auto-topics').checked) return;
    const rows = [...$('#question-fields').children].filter(row => !row.querySelector('[data-topic]').value.trim() && row.querySelector('[data-question]').value.trim().length >= 3);
    if (!rows.length) { $('#topic-status').textContent = 'Topics are already set. Clear a topic to request a new suggestion.'; return; }
    const questions = rows.map(row => row.querySelector('[data-question]').value.trim());
    const current = ++suggestionVersion;
    $('#topic-status').textContent = 'Assigning topics…';
    try {
      const data = await api('/topics/suggest', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({questions})});
      if (current !== suggestionVersion || !$('#auto-topics').checked) return;
      let applied = 0;
      data.suggestions.forEach((suggestion, index) => {
        const row = rows[index];
        if (!row.isConnected || row.querySelector('[data-question]').value.trim() !== questions[index] || row.querySelector('[data-topic]').value.trim()) return;
        row.querySelector('[data-topic]').value = suggestion.topic;
        row.dataset.topicSource = 'auto';
        row.querySelector('.topic-hint').textContent = suggestion.needsReview ? 'Suggested topic — please review.' : 'Topic assigned automatically. You can change it.';
        applied++;
      });
      $('#topic-status').textContent = `${applied} topic${applied === 1 ? '' : 's'} assigned. Review and edit them before saving.`;
    } catch (error) { if (current === suggestionVersion) $('#topic-status').textContent = error.message; }
  }
  function scheduleTopics() { clearTimeout(topicTimer); topicTimer = setTimeout(suggestMissingTopics, 650); }
  $('#auto-topics').onchange = () => { suggestionVersion++; clearTimeout(topicTimer); if ($('#auto-topics').checked) scheduleTopics(); else $('#topic-status').textContent = ''; };
  $('#suggest-topics').onclick = () => { $('#auto-topics').checked = true; clearTimeout(topicTimer); suggestMissingTopics(); };
  $('#add-question').onclick = () => addQuestion(); addQuestion();
  function parseQuestions(raw) {
    const lines = raw.replace(/\r\n?/g, '\n').split('\n').map(line => line.replace(/^\s*\[[^\]\n]+\]\s*[^:\n]+:\s*/, '').trim()).filter(Boolean);
    const numbered = /^\d{1,3}\s*[.)、:\-]\s*/;
    const first = lines.findIndex(line => numbered.test(line));
    if (first < 0) return lines.map(line => line.replace(/^[-*•]\s+/, '')).filter(Boolean);
    const questions = [];
    for (const line of lines.slice(first)) {
      if (numbered.test(line)) questions.push(line.replace(numbered, ''));
      else questions[questions.length - 1] += '\n' + line;
    }
    return questions.filter(Boolean);
  }
  $('#import-questions').onclick = () => {
    const status = $('#bulk-status'), topic = $('#bulk-topic').value.trim();
    const questions = parseQuestions($('#bulk-questions').value);
    const rows = [...$('#question-fields').children];
    const blank = rows.filter(row => !row.querySelector('[data-question]').value.trim() && !row.querySelector('[data-topic]').value.trim());
    status.className = 'error';
    if (!questions.length) { status.textContent = 'Paste at least one question.'; return; }
    if (topic.length > 80) { status.textContent = 'Enter a topic (up to 80 characters) for these questions.'; return; }
    if (questions.length + rows.length - blank.length > 50) { status.textContent = 'An interview can contain up to 50 questions. Reduce the list or remove existing questions.'; return; }
    if (questions.some(question => question.length > 2000)) { status.textContent = 'Each question must be 2,000 characters or fewer. Shorten the long question and try again.'; return; }
    blank.forEach(row => row.remove());
    questions.forEach(question => addQuestion(question, topic));
    if (!topic) { clearTimeout(topicTimer); suggestMissingTopics(); }
    $('#bulk-questions').value = '';
    status.className = 'muted';
    status.textContent = `${questions.length} question${questions.length === 1 ? '' : 's'} added. Review the questions and topics below, then save the interview.`;
  };
  editReport = report => {
    if (!report) return;
    editingId = report.id;
    suggestionVersion++; clearTimeout(topicTimer); $('#topic-status').textContent = '';
    form.reset();
    for (const name of ['field','company','role','experience','interviewType','interviewDate','location']) {
      const input = form.elements[name], value = report[name] || '';
      if (input.tagName === 'SELECT' && ![...input.options].some(option => option.value === value)) input.add(new Option(value || 'Unspecified', value));
      input.value = value;
    }
    $('#question-fields').replaceChildren();
    report.questions.forEach(q => addQuestion(q.question, q.topic, q.topicSource || 'manual'));
    $('#bulk-status').textContent = ''; $('#bulk-entry').open = false;
    $('#publish').textContent = 'Save changes';
    open();
  };
  form.onsubmit = async event => {
    event.preventDefault(); $('#publish').disabled = true; $('#form-error').textContent = '';
    const body = Object.fromEntries(new FormData(form));
    body.questions = [...document.querySelectorAll('.question-field')].map(row => ({question:row.querySelector('[data-question]').value.trim(),topic:row.querySelector('[data-topic]').value.trim(),topicSource:row.dataset.topicSource || 'manual'})).filter(q => q.question || q.topic);
    try {
      await api(editingId === null ? '/interviews' : '/admin/interviews/' + editingId, {method:editingId === null ? 'POST' : 'PUT',headers:{'Content-Type':'application/json',...(editingId === null ? {} : {Authorization:'Bearer ' + token})},body:JSON.stringify(body)});
      dialog.close(); suggestionVersion++; clearTimeout(topicTimer); $('#topic-status').textContent = ''; form.reset(); $('#bulk-status').textContent = ''; $('#bulk-entry').open = false; $('#question-fields').replaceChildren(); addQuestion();
      if ($('#filters')) $('#filters').reset(); page = 0;
      $('#notice').className = 'success'; $('#notice').textContent = editingId === null ? 'Interview saved.' : 'Report updated.';
      editingId = null;
      announceReportChange();
      await browse();
    } catch (error) { $('#form-error').textContent = error.message; }
    finally { $('#publish').disabled = false; }
  };
}
