// Status boxes for the multi-connection resource pages (drawn by task-box.js, loaded before this script).
// Creates are listed bottom-left (green), deletes bottom-right (red), and the per-connection progress of
// a refresh bottom-center (blue); each box shows a spinner until
// its request answers, then the result (success boxes fade out, failures stay until dismissed).
// In AdminWeb the boxes are drawn in the main page over this page's frame, like the Topology.
//
//   const task = multiTaskBox('create', 'Creating ...');
//   task.finish(true, 'Created ...');                                    // fades out
//   task.finish(true, 'Created ...', { label: 'Download', onClick });    // stays until dismissed
//
// multiRefresh(withProgress): reloads the rows per connection (not the page), so running boxes are kept;
// window.onMultiRefreshed(), when defined by the page, is called after the new content is in place.
//
// The list APIs wait for the running create/delete requests of the same resources (per-resource locks),
// so a list refresh would only show the changes after all requests finish. Instead each finished request
// updates its row at once (multiAddRow / multiRemoveRow), and the content is refreshed from the server
// once when no request is running any more.
(function () {
    let running = 0;    // requests whose status box is still busy
    // results of the status boxes per type (create/delete): read by the Auto Play of the main page
    const taskResults = {};
    // {running, results: {create: {ok, fail, failures: [text]}, ...}}
    window.multiTaskState = () => ({ running, results: JSON.parse(JSON.stringify(taskResults)) });

    function emptyRow(table) {
        const tr = document.createElement('tr');
        tr.className = 'empty-row';
        const td = document.createElement('td');
        td.colSpan = table.rows[0] ? table.rows[0].cells.length : 1;
        td.className = 'center-align';
        td.textContent = table.dataset.emptyText || 'No resources found.';
        tr.appendChild(td);
        return tr;
    }

    // renumber the '#' column of the table's own rows (not of inner tables)
    function renumber(table) {
        let n = 0;
        Array.from(table.rows).forEach(row => {
            const cell = row.querySelector(':scope > td.column-num');
            if (cell) cell.textContent = ++n;
        });
        const empty = table.querySelector('tr.empty-row');
        if (n === 0 && !empty) (table.tBodies[0] || table).appendChild(emptyRow(table));
        if (n > 0 && empty) empty.remove();
    }

    // Insert a new resource row after the last row of the same connection (or at the end).
    window.multiAddRow = function (table, tr, connName) {
        let after = null;
        Array.from(table.rows).forEach(row => {
            const cell = row.querySelector(':scope > td.column-connection');
            if (cell && cell.textContent.trim() === connName) after = row;
        });
        if (after) after.after(tr);
        else (table.tBodies[0] || table).appendChild(tr);
        renumber(table);
        window.applyMultiFocus();
        syncSearchSelect();
    };

    // Remove the resource row that holds the given element (e.g. its delete checkbox).
    window.multiRemoveRow = function (el) {
        const tr = el && el.closest('tr');
        const table = tr && tr.parentElement && tr.closest('table');
        if (!tr) return;
        tr.remove();
        if (table) renumber(table);
        syncSearchSelect();
    };

    // type: 'create' | 'delete' | 'refresh'. Returns { setText, finish(ok, text, action) }
    // Send several Spider API requests in one call: the server sends them at the same time and streams
    // each response back as soon as it arrives (POST /spider/adminweb/fanout). The browser opens at most
    // 6 connections to one server, so sending one request per connection queued the rest behind slow
    // ones (ex: an IBM NLB takes about 7 minutes to create).
    // requests: [{method, path, headers, body}] -> one Promise per request, resolved with a Response, in order.
    window.multiFanout = function (requests) {
        // sent by the main page (task-box.js), so the requests go on when this page is left
        if (typeof window.taskFanout === 'function') return window.taskFanout(requests);
        const pending = requests.map(() => {
            let resolve, reject;
            const promise = new Promise((res, rej) => { resolve = res; reject = rej; });
            return { promise, resolve, reject, done: false };
        });
        const settle = (index, settleFn, value) => {
            const p = pending[index];
            if (!p || p.done) return;
            p.done = true;
            settleFn === 'resolve' ? p.resolve(value) : p.reject(value);
        };

        fetch('/spider/adminweb/fanout', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                requests: requests.map((r, i) => ({ id: String(i), method: r.method, path: r.path, headers: r.headers, body: r.body }))
            })
        })
        .then(async response => {
            if (!response.ok || !response.body) throw new Error(`HTTP ${response.status}`);
            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffered = '';
            for (;;) {
                const { value, done } = await reader.read();
                buffered += decoder.decode(value || new Uint8Array(), { stream: !done });
                let newline;
                while ((newline = buffered.indexOf('\n')) >= 0) {
                    const line = buffered.slice(0, newline).trim();
                    buffered = buffered.slice(newline + 1);
                    if (!line) continue;
                    const r = JSON.parse(line);
                    const index = parseInt(r.id, 10);
                    if (r.status >= 200 && r.status <= 599) {
                        // 204/205/304 responses cannot carry a body
                        const noBody = [204, 205, 304].includes(r.status);
                        settle(index, 'resolve', new Response(noBody ? null : r.body, {
                            status: r.status,
                            headers: { 'Content-Type': 'application/json' }
                        }));
                    } else {
                        settle(index, 'reject', new Error(r.error || 'the request could not be sent'));
                    }
                }
                if (done) break;
            }
            throw new Error('no response from the server');
        })
        .catch(error => pending.forEach((p, i) => settle(i, 'reject', error)));

        return pending.map(p => p.promise);
    };

    window.multiTaskBox = function (type, text) {
        // refresh boxes are not requests that change resources: they never trigger the sync refresh
        const counted = type !== 'refresh';
        if (counted) running++;
        const box = window.taskBox(type, text, {
            // the last running request: sync the content with the server once
            onDone: () => { if (counted && --running === 0) window.multiRefresh(); },
        });
        return {
            setText: box.setText,
            info: box.info,
            finish(ok, s, action) {
                if (counted) {
                    const r = taskResults[type] || (taskResults[type] = { ok: 0, fail: 0, failures: [] });
                    if (ok) r.ok++;
                    else { r.fail++; r.failures.push(s); }
                }
                box.finish(ok, s, action);
            },
        };
    };

    // ---- Connection bar: focus on one connection, or back to all ----
    // Clicking a connection chip shows the single-connection page of that connection below the bar (see
    // syncSingleFrame) and the main page shows it as the selected connection, in multi-connection mode;
    // "Active Connections (N)" goes back to the list of all of them.
    // The focus is kept in sessionStorage 'multiFocus' (shared with the main page and the other multi pages).
    const BAR_CSS = `
.connection-toggle { font: inherit; font-weight: bold; color: #0645AD; background: #fff; border: 1px solid #0645AD;
    border-radius: 10px; padding: 2px 10px; margin-right: 6px; cursor: pointer; }
.connection-toggle.active { color: #fff; background: #0645AD; }
button.connection-chip { font: inherit; font-weight: bold; cursor: pointer; }
button.connection-chip:hover, .connection-toggle:hover { filter: brightness(0.93); }
button.connection-chip.active { color: #fff; background: #0645AD; }
.multi-focus-hidden { display: none !important; }
.connection-error.connection-notice { color: #57606a; }
.single-frame { position: fixed; left: 0; width: 100%; border: none; background: #fff; z-index: 900; }
body.multi-single .search-select { display: none !important; }
.search-select { display: none; align-items: center; gap: 3px; margin-left: 8px; font-size: 12px; font-weight: bold;
    color: #555; white-space: nowrap; cursor: pointer; user-select: none; }
.search-select.shown { display: inline-flex; }
.search-select input { margin: 0; cursor: pointer; }
.fixed-action-buttons .add-button { margin-bottom: 0; }`;
    const barStyle = document.createElement('style');
    barStyle.textContent = BAR_CSS;
    document.head.appendChild(barStyle);

    function pageConnections() {
        return Array.from(document.querySelectorAll('.connection-bar .connection-chip')).map(chip => chip.dataset.connection);
    }

    // the connection this page is narrowed to, or '' for all
    window.multiFocusName = function () {
        let name = '';
        try { name = sessionStorage.getItem('multiFocus') || ''; } catch (e) {}
        return pageConnections().includes(name) ? name : '';
    };
    window.multiInFocus = function (connName) {
        const focus = window.multiFocusName();
        return !focus || connName === focus;
    };

    // Show only the focused connection: chips, resource rows, per-connection rows of the dialogs.
    window.applyMultiFocus = function () {
        const focus = window.multiFocusName();
        const toggle = document.querySelector('.connection-bar .connection-toggle');
        if (toggle) toggle.classList.toggle('active', !focus);
        document.querySelectorAll('.connection-bar .connection-chip').forEach(chip =>
            chip.classList.toggle('active', chip.dataset.connection === focus));

        document.querySelectorAll('table[data-empty-text]').forEach(table => {
            Array.from(table.rows).forEach(row => {
                const cell = row.querySelector(':scope > td.column-connection');
                if (!cell) return;
                const hide = !!focus && cell.textContent.trim() !== focus;
                row.classList.toggle('multi-focus-hidden', hide);
                if (hide) row.querySelectorAll('input[name="deleteCheckbox"]').forEach(cb => { cb.checked = false; });
            });
        });
        document.querySelectorAll('.overlay tr[data-connection]').forEach(row =>
            row.classList.toggle('multi-focus-hidden', !!focus && row.dataset.connection !== focus));
        const count = focus ? 1 : pageConnections().length;
        document.querySelectorAll('.multi-conn-count').forEach(span => { span.textContent = count; });
        syncSearchSelect();
        syncSingleFrame(focus);
    };

    // ---- Focused on one connection: its single-connection page ----
    // In place of the multi list, the page of this resource for that connection (e.g. /adminweb/vpc/<conn>)
    // is shown in a frame below the connection bar, without its own header: the search box and the refresh
    // button of this page work on it. Back to all connections, the multi list is refreshed (the single page
    // may have changed the resources), or loaded if it was not yet.
    var RESOURCE = (location.pathname.match(/\/adminweb\/([^/]+)\/multi$/) || [])[1] || '';
    var singleFrame = null;
    var listLoaded = new URLSearchParams(location.search).get('load') !== 'async';

    function singleFrameShown() {
        return !!(singleFrame && singleFrame.style.display !== 'none');
    }

    function placeSingleFrame() {
        if (!singleFrameShown()) return;
        // an iframe is not stretched by top/bottom (it keeps its default 150px height): set its height
        const bar = document.querySelector('.connection-bar');
        const top = bar ? bar.getBoundingClientRect().bottom : 0;
        singleFrame.style.top = top + 'px';
        singleFrame.style.height = Math.max(0, window.innerHeight - top) + 'px';
    }
    window.addEventListener('resize', placeSingleFrame);

    // The connection bar wraps to more lines with many connections (and grows with connection errors):
    // keep the list (+ button and table) right below it, as the single-connection page has it below its header.
    function placeContent() {
        const bar = document.querySelector('.connection-bar');
        const content = document.querySelector('.content');
        if (!bar || !content) return;
        content.style.marginTop = (bar.getBoundingClientRect().bottom + 8) + 'px';   // the same 8px as the single page
        placeSingleFrame();
    }
    window.addEventListener('resize', placeContent);
    (function watchConnectionBar() {
        const bar = document.querySelector('.connection-bar');
        if (bar && typeof ResizeObserver === 'function') new ResizeObserver(placeContent).observe(bar);
        placeContent();
    })();

    function syncSingleFrame(focus) {
        const content = document.querySelector('.content');
        if (!focus || !RESOURCE) {
            if (singleFrameShown()) {
                singleFrame.style.display = 'none';
                document.body.classList.remove('multi-single');
                if (content) content.style.display = '';
                window.multiRefresh(!listLoaded);
                listLoaded = true;
            }
            return;
        }
        if (!singleFrame) {
            singleFrame = document.createElement('iframe');
            singleFrame.className = 'single-frame';
            singleFrame.title = 'Single-connection page';
            singleFrame.addEventListener('load', onSingleFrameLoad);
            document.body.appendChild(singleFrame);
        }
        if (content) content.style.display = 'none';
        document.body.classList.add('multi-single');
        singleFrame.style.display = '';
        placeSingleFrame();
        const src = `/spider/adminweb/${RESOURCE}/${encodeURIComponent(focus)}`;
        if (singleFrame.dataset.src !== src) {
            singleFrame.dataset.src = src;
            window.parent.postMessage({ type: 'showProgressBar' }, '*');
            singleFrame.src = src;
        }
    }

    // on every load of the single page (also its own reloads): hide its header, keep the search applied
    function onSingleFrameLoad() {
        window.parent.postMessage({ type: 'hideProgressBar' }, '*');
        try {
            const doc = singleFrame.contentDocument;
            const style = doc.createElement('style');
            style.textContent = '.fixed-header { display: none !important; } .content { margin-top: 8px !important; }';
            doc.head.appendChild(style);
        } catch (e) {}
        forwardSearch();
    }

    // the search box of this page searches the single page
    function forwardSearch() {
        if (!singleFrameShown()) return;
        try {
            const win = singleFrame.contentWindow;
            const input = document.getElementById('searchInput');
            const target = win.document.getElementById('searchInput');
            if (!input || !target) return;
            target.value = input.value;
            if (input.value.trim()) {
                if (typeof win.searchKeyword === 'function') win.searchKeyword();
            } else if (typeof win.clearSearchInput === 'function') {
                win.clearSearchInput();
            }
        } catch (e) {}
    }
    (function setupSearchForward() {
        const input = document.getElementById('searchInput');
        if (!input) return;
        input.addEventListener('keyup', forwardSearch);
        const clear = document.getElementById('clearSearch');
        if (clear) clear.addEventListener('click', () => setTimeout(forwardSearch, 0));
    })();

    // the single page reports its progress to its parent (this page): pass it on to the main page
    window.addEventListener('message', event => {
        if (singleFrame && event.source === singleFrame.contentWindow) {
            try { window.parent.postMessage(event.data, '*'); } catch (e) {}
        }
    });

    function setMultiFocus(name) {
        try {
            if (name) sessionStorage.setItem('multiFocus', name);
            else sessionStorage.removeItem('multiFocus');
        } catch (e) {}
        try { window.parent.postMessage({ type: 'focusMultiConnection', name: name || '' }, '*'); } catch (e) {}
        window.applyMultiFocus();
    }

    document.addEventListener('click', event => {
        const toggle = event.target.closest('.connection-bar .connection-toggle');
        if (toggle) { setMultiFocus(''); return; }
        const chip = event.target.closest('.connection-bar .connection-chip');
        if (chip) setMultiFocus(chip.dataset.connection === window.multiFocusName() ? '' : chip.dataset.connection);
    });

    // checkboxes of the rows shown now (for "select all")
    window.multiVisibleCheckboxes = function () {
        return Array.from(document.querySelectorAll('input[name="deleteCheckbox"]:not(:disabled)'))
            .filter(cb => !cb.closest('tr.multi-focus-hidden'));
    };

    // ---- Search: select the matching rows ----
    // While a keyword is entered, a checkbox next to the search box checks (or unchecks) the delete
    // checkboxes of the rows highlighted by the search, so they can be deleted at once.
    function matchingCheckboxes() {
        const table = document.querySelector('table[data-empty-text]');
        if (!table) return [];
        return Array.from(table.rows)
            .filter(row => !row.classList.contains('multi-focus-hidden') && row.querySelector('td.highlight'))
            .map(row => row.querySelector(':scope > td input[name="deleteCheckbox"]'))
            .filter(cb => cb && !cb.disabled);
    }

    var searchSelect = null;    // var: safe to read before this line runs (syncSearchSelect returns early)
    function syncSearchSelect() {
        const input = document.getElementById('searchInput');
        if (!input || !searchSelect) return;
        const shown = !!input.value.trim();
        searchSelect.classList.toggle('shown', shown);
        const box = searchSelect.querySelector('input');
        const matches = shown ? matchingCheckboxes() : [];
        searchSelect.querySelector('span').textContent = `Select matches (${matches.length})`;
        box.checked = matches.length > 0 && matches.every(cb => cb.checked);
        if (!shown) box.checked = false;
    }

    (function setupSearchSelect() {
        const input = document.getElementById('searchInput');
        const container = input && input.closest('.searchContainer');
        if (!container) return;
        searchSelect = document.createElement('label');
        searchSelect.className = 'search-select';
        searchSelect.title = 'Check the rows matching the keyword (to delete them at once)';
        const box = document.createElement('input');
        box.type = 'checkbox';
        searchSelect.append(box, document.createElement('span'));
        container.after(searchSelect);

        box.addEventListener('change', () => {
            matchingCheckboxes().forEach(cb => { cb.checked = box.checked; });
            syncSearchSelect();
        });
        // after the page's own search handler (inline onkeyup) has highlighted the rows
        input.addEventListener('keyup', syncSearchSelect);
        input.addEventListener('input', syncSearchSelect);
        const clear = document.getElementById('clearSearch');
        if (clear) clear.addEventListener('click', () => setTimeout(syncSearchSelect, 0));
        // rows checked or unchecked by hand (or by "select all") update the state of the checkbox
        document.addEventListener('change', event => {
            if (event.target !== box) syncSearchSelect();
        });
        syncSearchSelect();
    })();

    // Reload the resource rows from the server, one request per connection (in parallel): the rows of a
    // connection are replaced as soon as its request answers. withProgress shows a status box per connection.
    // A connection whose request fails keeps its rows; one the server could not list shows its error.
    // Requests that finish while a refresh runs are covered by one more refresh afterwards.
    let refreshing = false, pending = false, pendingProgress = false;
    window.multiRefresh = function (withProgress) {
        // the refresh button while the single page is shown: reload that page
        if (withProgress && singleFrameShown()) {
            window.parent.postMessage({ type: 'showProgressBar' }, '*');
            singleFrame.contentWindow.location.reload();
            return;
        }
        if (refreshing) {
            pending = true;
            pendingProgress = pendingProgress || !!withProgress;
            return;
        }
        refreshing = true;
        const conns = pageConnections();

        const jobs = conns.map(conn => {
            const box = withProgress ? window.multiTaskBox('refresh', `Refreshing ${conn}…`) : null;
            const params = new URLSearchParams(location.search);
            params.set('ids', conn);
            params.delete('load');      // the rows themselves, not the page without them
            return fetch(`${location.pathname}?${params}`)
                .then(r => {
                    if (!r.ok) throw new Error(`HTTP ${r.status}`);
                    return r.text();
                })
                .then(html => {
                    const fresh = new DOMParser().parseFromString(html, 'text/html');
                    const freshTable = fresh.querySelector('table[data-empty-text]');
                    if (!freshTable) throw new Error('unexpected response');
                    const rows = Array.from(freshTable.rows).filter(row => row.querySelector(':scope > td.column-connection'));
                    // an error, or a notice (data-notice: e.g. the resource is not supported by the CSP)
                    const err = fresh.querySelector('.connection-bar .connection-error');
                    const notice = !!(err && err.dataset.notice === 'true');
                    const error = err ? err.textContent.trim().replace(/^[\u26a0\u2139]\s*/, '').replace(`${conn}: `, '') : '';
                    replaceConnectionRows(conn, rows, error, notice);
                    if (!box) return;
                    if (notice) box.info(`${conn}: ${error}`);
                    else if (error) box.finish(false, `Failed to list ${conn}: ${error}`);
                    else box.finish(true, `Refreshed ${conn} (${rows.length})`);
                })
                .catch(error => {
                    if (box) box.finish(false, `Failed to refresh ${conn}: ${error.message}`);
                    else console.error(`Error refreshing ${conn}:`, error);
                });
        });

        Promise.all(jobs).finally(() => {
            // all connections answered: an empty list is no longer "loading"
            document.querySelectorAll('table[data-empty-text] tr.empty-row td').forEach(td => {
                td.textContent = td.closest('table').dataset.emptyText;
            });
            refreshing = false;
            if (pending) {
                const progress = pendingProgress;
                pending = false;
                pendingProgress = false;
                window.multiRefresh(progress);
            }
        });
    };

    // Replace the rows of one connection with the fresh ones (kept in the connection order of the bar),
    // and its error (or notice) line in the connection bar.
    function replaceConnectionRows(conn, freshRows, error, notice) {
        const table = document.querySelector('table[data-empty-text]');
        if (!table) return;
        const order = pageConnections();
        const rank = name => order.indexOf(name);
        const connOf = row => {
            const cell = row.querySelector(':scope > td.column-connection');
            return cell ? cell.textContent.trim() : null;
        };

        Array.from(table.rows).filter(row => connOf(row) === conn).forEach(row => row.remove());
        const before = Array.from(table.rows).find(row => {
            const name = connOf(row);
            return name !== null && rank(name) > rank(conn);
        });
        freshRows.forEach(row => {
            const tr = document.importNode(row, true);
            if (before) before.before(tr);
            else (table.tBodies[0] || table).appendChild(tr);
        });
        renumber(table);

        const bar = document.querySelector('.connection-bar');
        if (bar) {
            bar.querySelectorAll('.connection-error').forEach(div => {
                if (div.dataset.connection === conn) div.remove();
            });
            if (error) {
                const div = document.createElement('div');
                div.className = notice ? 'connection-error connection-notice' : 'connection-error';
                div.dataset.connection = conn;
                if (notice) div.dataset.notice = 'true';
                div.textContent = `${notice ? '\u2139' : '\u26a0'} ${conn}: ${error}`;
                bar.appendChild(div);
            }
        }

        window.applyMultiFocus();
        // a page can set up the new content (e.g. render status cells) in onMultiRefreshed()
        if (typeof window.onMultiRefreshed === 'function') window.onMultiRefreshed();
        const search = document.getElementById('searchInput');
        if (search && search.value.trim() && typeof searchKeyword === 'function') searchKeyword();
        syncSearchSelect();
    }

    // this script is loaded at the end of the page: narrow the page to the stored focus at once
    window.applyMultiFocus();

    // opened without the lists (load=async, from the left menu): list every connection in parallel,
    // with a progress box per connection
    // (when the page opens focused on one connection, the list is loaded when going back to all of them)
    if (new URLSearchParams(location.search).get('load') === 'async') {
        document.querySelectorAll('table[data-empty-text] tr.empty-row td').forEach(td => { td.textContent = 'Loading…'; });
        if (!window.multiFocusName()) {
            window.multiRefresh(true);
            listLoaded = true;
        }
    }
})();
