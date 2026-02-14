package gdsserve

const uiHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>gdu search</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: system-ui, sans-serif; margin: 0; background: #1a1a1a; color: #e0e0e0; min-height: 100vh; }
    .container { max-width: 900px; margin: 0 auto; padding: 1.5rem; }
    h1 { font-size: 1.2rem; font-weight: 500; margin-bottom: 1rem; color: #888; }
    input { width: 100%; padding: 0.75rem 1rem; font-size: 1.1rem; border: 1px solid #333; border-radius: 6px; background: #252525; color: #e0e0e0; outline: none; }
    input:focus { border-color: #4a9eff; }
    input::placeholder { color: #666; }
    .results { margin-top: 1rem; border: 1px solid #333; border-radius: 6px; background: #252525; max-height: 70vh; overflow-y: auto; }
    .row { padding: 0.4rem 1rem; font-size: 0.9rem; font-family: ui-monospace, monospace; border-bottom: 1px solid #2a2a2a; display: flex; align-items: center; gap: 1rem; cursor: pointer; }
    .row:hover { background: #2a2a2a; }
    .row .path { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .row .size { color: #888; font-size: 0.85rem; min-width: 5rem; }
    .empty { padding: 2rem; text-align: center; color: #666; }
    .count { font-size: 0.85rem; color: #666; margin-top: 0.5rem; }
  </style>
</head>
<body>
  <div class="container">
    <h1>gdu search</h1>
    <input type="text" id="q" placeholder="Type to search (e.g. *.go or config)" autofocus autocomplete="off">
    <div class="count" id="count"></div>
    <div class="results" id="results"></div>
  </div>
  <script>
    const q = document.getElementById('q');
    const results = document.getElementById('results');
    const countEl = document.getElementById('count');
    let debounce = null;

    function search() {
      const query = q.value.trim();
      if (!query) {
        results.innerHTML = '<div class="empty">Type to search</div>';
        countEl.textContent = '';
        return;
      }
      fetch('/search?q=' + encodeURIComponent(query))
        .then(r => r.json())
        .then(data => {
          countEl.textContent = data.count + ' matches';
          if (data.results.length === 0) {
            results.innerHTML = '<div class="empty">No matches</div>';
            return;
          }
          results.innerHTML = data.results.map(e => {
            const size = e.size >= 1073741824 ? (e.size/1073741824).toFixed(1)+' GiB' :
              e.size >= 1048576 ? (e.size/1048576).toFixed(1)+' MiB' :
              e.size >= 1024 ? (e.size/1024).toFixed(1)+' KiB' : e.size + ' B';
            return '<div class="row" data-path="' + escapeHtml(e.path) + '"><span class="path" title="' + escapeHtml(e.path) + '">' + escapeHtml(e.path) + '</span><span class="size">' + size + '</span></div>';
          }).join('');
          results.querySelectorAll('.row').forEach(row => {
            row.addEventListener('click', () => {
              if (navigator.clipboard) navigator.clipboard.writeText(row.dataset.path);
            });
          });
        })
        .catch(() => { results.innerHTML = '<div class="empty">Error</div>'; countEl.textContent = ''; });
    }

    function escapeHtml(s) {
      const d = document.createElement('div');
      d.textContent = s;
      return d.innerHTML;
    }

    q.addEventListener('input', () => {
      clearTimeout(debounce);
      debounce = setTimeout(search, 80);
    });
    q.addEventListener('keydown', (e) => { if (e.key === 'Enter') search(); });
  </script>
</body>
</html>
`
