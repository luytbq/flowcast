'use strict';
const form = document.getElementById('form');
const fileInput = document.getElementById('file');
const drop = document.getElementById('drop');
const dropText = document.getElementById('drop-text');
const result = document.getElementById('result');
const summary = document.getElementById('summary');
const issues = document.getElementById('issues');
const download = document.getElementById('download');
const buttons = [document.getElementById('build'), document.getElementById('check')];
let lastURL = null;

fileInput.addEventListener('change', () => {
  dropText.textContent = fileInput.files[0] ? fileInput.files[0].name : 'Choose or drop a file here';
});
drop.addEventListener('dragover', (e) => { e.preventDefault(); drop.classList.add('over'); });
drop.addEventListener('dragleave', () => drop.classList.remove('over'));
drop.addEventListener('drop', (e) => {
  e.preventDefault();
  drop.classList.remove('over');
  if (e.dataTransfer.files.length) {
    fileInput.files = e.dataTransfer.files;
    fileInput.dispatchEvent(new Event('change'));
  }
});

// The layout parameter form and the list of directions are declared by the server,
// so adding a parameter in core makes it appear here without editing the page.
async function loadFields() {
  const res = await fetch('/api/fields');
  const data = await res.json();
  const dir = document.getElementById('direction');
  for (const d of data.directions) {
    const o = document.createElement('option');
    o.value = d;
    o.textContent = d;
    dir.appendChild(o);
  }
  const box = document.getElementById('fields');
  for (const f of data.fields) {
    const label = document.createElement('label');
    label.textContent = f.name;
    label.title = f.help;
    const input = document.createElement('input');
    input.type = 'number';
    input.name = f.name;
    input.min = f.lo;
    input.max = f.hi;
    input.placeholder = f.default;
    label.appendChild(input);
    box.appendChild(label);
  }
}
loadFields();

function item(level, loc, text) {
  const li = document.createElement('li');
  li.className = 'lv-' + level;
  if (loc) {
    const s = document.createElement('span');
    s.className = 'loc';
    s.textContent = loc + ': ';
    li.appendChild(s);
  }
  li.appendChild(document.createTextNode(text));
  issues.appendChild(li);
}

function show(data, built) {
  result.hidden = false;
  issues.replaceChildren();
  download.hidden = true;
  if (data.error) {
    summary.className = 'bad';
    summary.textContent = data.error.message;
    return;
  }
  const errs = data.issues.filter((i) => i.level === 'error').length;
  const warns = data.issues.length - errs;
  for (const i of data.issues) item(i.level, (i.location || '') + (i.id ? ' [' + i.id + ']' : ''), i.message);
  for (const w of data.warnings || []) item('warning', 'layout', w.message);
  for (const f of data.findings || []) item(f.level, 'layout', f.message);
  if (!data.ok) {
    summary.className = 'bad';
    summary.textContent = `The table has ${errs} errors, ${warns} warnings. Fix the errors and try again.`;
    return;
  }
  summary.className = 'ok';
  if (!built) {
    summary.textContent = `The table is valid: 0 errors, ${warns} warnings.`;
    return;
  }
  const s = data.stats;
  summary.textContent = `Created ${data.filename}: ${s.lanes} lanes, ${s.items} elements, ${s.edges} edges.`;
  if (lastURL) URL.revokeObjectURL(lastURL);
  lastURL = URL.createObjectURL(new Blob([data.drawio], { type: 'application/vnd.jgraph.mxfile' }));
  download.href = lastURL;
  download.download = data.filename;
  download.hidden = false;
  download.click();
}

async function send(built) {
  if (!fileInput.files[0]) { fileInput.click(); return; }
  buttons.forEach((b) => { b.disabled = true; });
  try {
    const res = await fetch(built ? '/api/build' : '/api/check', { method: 'POST', body: new FormData(form) });
    show(await res.json(), built);
  } catch (e) {
    show({ error: { message: 'Could not send the file: ' + e.message } }, built);
  } finally {
    buttons.forEach((b) => { b.disabled = false; });
  }
}

form.addEventListener('submit', (e) => { e.preventDefault(); send(true); });
document.getElementById('check').addEventListener('click', () => send(false));
