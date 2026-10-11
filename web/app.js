const $ = s => document.querySelector(s);
const admin = location.pathname === '/admin.html';
let token = '', page = 0, requestId = 0;
const escape = value => String(value).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
async function api(path, options = {}) {
  const response = await fetch('/api' + path, options);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.message || 'Something went wrong. Please try again.');
  }
  return response.status === 204 ? null : response.json();
}
async function filters() {
  const data = await api('/filters');
  for (const [name, values, label] of [['field',data.fields,'All fields'],['role',data.roles,'All roles'],['company',data.companies,'All companies'],['topic',data.topics,'All topics'],['year',data.years,'All years']]) {
    const select = $(`[name="${name}"]`);
    if (!select) continue;
    const old = select.value;
    select.innerHTML = `<option value="">${label}</option>` + values.map(v => `<option value="${escape(v)}">${escape(v)}</option>`).join('');
    if (values.map(String).includes(old)) select.value = old;
  }
  if ($('#stats')) $('#stats').textContent = `${data.interviews} interview reports  ·  ${data.questions} questions  ·  A little shared knowledge goes a long way`;
}
async function browse() {
  const current = ++requestId;
  const query = new URLSearchParams($('#filters') ? new FormData($('#filters')) : undefined);
  for (const [key,value] of [...query]) if (!value) query.delete(key);
  query.set('page', page);
  $('#result-count').textContent = 'Loading reports…';
  try {
    const data = await api('/interviews?' + query);
    if (current !== requestId) return;
    $('#result-count').textContent = `${data.total} interview report${data.total === 1 ? '' : 's'}`;
    $('#reports').innerHTML = data.items.map(i => `<article class="report"><div class="report-header"><div><div class="company">${escape(i.company)}</div><h3>${escape(i.role)}</h3><div class="metadata">${escape(i.field || 'Unspecified')} · ${escape(i.experience)} years experience · ${escape(i.interviewType)} · ${escape(i.interviewDate)}${i.location ? ' · ' + escape(i.location) : ''}</div></div><div>${[...new Set(i.questions.map(q => q.topic))].map(t => `<span class="tag">${escape(t)}</span>`).join('')}</div></div><ol class="questions">${i.questions.map(q => `<li>${escape(q.question)}<span class="question-topic">${escape(q.topic)}</span></li>`).join('')}</ol>${admin ? `<button class="danger" data-delete="${i.id}">Remove report</button>` : ''}</article>`).join('') || `<div class="empty"><h3>${query.size > 1 ? 'No matching interviews yet.' : 'The first note could be yours.'}</h3><p class="muted">${query.size > 1 ? 'Try a different search or clear your filters.' : 'Share a few questions from your last interview to get things started.'}</p></div>`;
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
  $('#login').onsubmit = async event => {
    event.preventDefault();
    token = $('#admin-token').value;
    try {
      await api('/admin/session', {headers:{Authorization:'Bearer ' + token}});
      $('#login').hidden = true; $('#admin-content').hidden = false;
      $('#admin-token').value = ''; $('#login-error').textContent = ''; browse();
    } catch (error) { token = ''; $('#login-error').textContent = error.message; }
  };
  $('#logout').onclick = () => { token = ''; $('#login').hidden = false; $('#admin-content').hidden = true; $('#reports').replaceChildren(); };
  $('#reports').onclick = async event => {
    const button = event.target.closest('[data-delete]');
    if (!button || !confirm('Permanently remove this interview and all its questions?')) return;
    button.disabled = true;
    try {
      await api('/admin/interviews/' + button.dataset.delete, {method:'DELETE',headers:{Authorization:'Bearer ' + token}});
      page = 0; $('#notice').textContent = 'Report removed.'; browse();
    } catch (error) { $('#notice').textContent = error.message; button.disabled = false; }
  };
} else {
  let timer;
  $('#filters').onsubmit = event => event.preventDefault();
  $('#filters').oninput = () => { clearTimeout(timer); timer = setTimeout(() => { page = 0; browse(); }, 250); };
  $('#filters').onreset = () => setTimeout(() => { page = 0; browse(); }, 0);
  const dialog = $('#submission'), form = $('#submission-form');
  const open = () => { $('#form-error').textContent = ''; dialog.showModal(); };
  $('#share').onclick = open; $('#share-bottom').onclick = open;
  $('#close').onclick = () => dialog.close();
  const today = new Date();
  form.elements.interviewDate.max = `${today.getFullYear()}-${String(today.getMonth()+1).padStart(2,'0')}-${String(today.getDate()).padStart(2,'0')}`;
  function addQuestion() {
    if ($('#question-fields').children.length >= 50) return;
    const row = document.createElement('div'); row.className = 'question-field';
    row.innerHTML = '<label>Question<textarea data-question required maxlength="2000" placeholder="What did the interviewer ask?"></textarea></label><label>Topic<input data-topic required maxlength="80" placeholder="e.g. Accounting, Communication, SQL"></label><button type="button">Remove question</button>';
    row.querySelector('button').onclick = () => { if ($('#question-fields').children.length > 1) row.remove(); };
    $('#question-fields').append(row);
  }
  $('#add-question').onclick = addQuestion; addQuestion();
  form.onsubmit = async event => {
    event.preventDefault(); $('#publish').disabled = true; $('#form-error').textContent = '';
    const body = Object.fromEntries(new FormData(form));
    body.questions = [...document.querySelectorAll('.question-field')].map(row => ({question:row.querySelector('[data-question]').value.trim(),topic:row.querySelector('[data-topic]').value.trim()}));
    try {
      await api('/interviews', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
      dialog.close(); form.reset(); $('#question-fields').replaceChildren(); addQuestion();
      $('#filters').reset(); page = 0;
      $('#notice').className = 'success'; $('#notice').textContent = 'Thank you — your interview report is now published.';
      await filters(); await browse();
    } catch (error) { $('#form-error').textContent = error.message; }
    finally { $('#publish').disabled = false; }
  };
  filters().catch(error => { $('#notice').textContent = error.message; }); browse();
}
