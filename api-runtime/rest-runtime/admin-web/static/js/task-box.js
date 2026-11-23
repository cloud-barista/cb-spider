// Progress / status boxes of the AdminWeb pages (same look as the Topology create/delete boxes).
// Creates are listed bottom-left (green), deletes bottom-right (red), and other requests (loading,
// refresh, updates) bottom-center (blue); each box shows a spinner and the elapsed time until its
// request answers, then the result (success boxes fade out after 3 seconds, failures stay until dismissed).
//
// Host and pages
//   The main page (main.html) loads this script as the HOST (window.TASK_HOST = true before loading it).
//   Every resource page in its frames loads it as a CLIENT: the boxes are drawn and the requests that change
//   resources (POST/PUT/DELETE) are sent by the host, which is never reloaded. So a create or delete goes on
//   when the user moves to another page, and its box still shows how it ends:
//     - leaving a page, its running create/delete boxes and its failure boxes move to the "Task Status" panel
//       of the main page's top bar (with the page and connection they come from), apart from the boxes of
//       the page now shown; a reload of the same page keeps them in place.
//     - the result of a box whose page is gone comes from its request(s) (status and Result of the response).
//   A page opened alone (no host) is its own host.
//
// API of a page
//   fetchWithProgress(url, options[, label])        // fetch() with a box per request (type from the method)
//   showProgressBar([label]) / hideProgressBar()    // a box around any work (stacked: hide ends the last one)
//   taskBox(type, text, opts)                       // a box: { setText, info, finish(ok, text, action), dismiss }
//   taskFanout(requests)                            // multiFanout (multi-task.js) sent by the host
(function () {
    if (window.taskBox) return;

    const HIDDEN = new URLSearchParams(location.search).has('topoCreate') && window.parent !== window;
    const FADE_AFTER = 3000;   // ms a success stays

    const TASK_CSS = `
.mc-task-tray { position: fixed; z-index: 1600; display: flex; flex-direction: column; gap: 6px;
    overflow-y: auto; pointer-events: none; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif; }
.mc-task-tray.create { align-items: flex-start; }
.mc-task-tray.delete { align-items: flex-end; }
.mc-task-tray.refresh { align-items: center; transform: translateX(-50%); }
.mc-task { pointer-events: auto; display: flex; align-items: center; gap: 8px; padding: 7px 10px; max-width: 100%;
    box-sizing: border-box; border-radius: 8px; font-size: 12px; line-height: 1.35; border: 1px solid;
    box-shadow: 0 2px 8px rgba(15, 23, 42, 0.15); }
.mc-task.create { background: #f0fdf4; border-color: #86efac; color: #14532d; }
.mc-task.delete { background: #fff1f2; border-color: #fda4af; color: #881337; }
.mc-task.refresh { background: #eff6ff; border-color: #93c5fd; color: #1e3a8a; }
.mc-task.fail { background: #fef2f2; border-color: #f87171; color: #991b1b; }
.mc-task.autofade { animation: mc-task-out 0.45s ease ${FADE_AFTER / 1000}s forwards; }
.mc-task.dismissed { animation: mc-task-out 0.45s ease forwards; pointer-events: none; }
@keyframes mc-task-out { to { opacity: 0; visibility: hidden; } }
.mc-task-text { word-break: break-word; }
.mc-task-time { flex: none; opacity: 0.7; font-variant-numeric: tabular-nums; }
.mc-task-icon { flex: none; width: 14px; height: 14px; display: flex; align-items: center; justify-content: center; font-weight: 800; }
.mc-task.ok .mc-task-icon { color: #16a34a; }
.mc-task.fail .mc-task-icon { color: #dc2626; }
.mc-task-spin { flex: none; width: 14px; height: 14px; border-radius: 50%; border: 2.5px solid; box-sizing: border-box;
    animation: mc-task-spin 0.8s linear infinite; }
.mc-task.create .mc-task-spin { border-color: #bbf7d0; border-top-color: #16a34a; }
.mc-task.delete .mc-task-spin { border-color: #fecdd3; border-top-color: #e11d48; }
.mc-task.refresh .mc-task-spin { border-color: #bfdbfe; border-top-color: #2563eb; }
.mc-task.refresh.ok .mc-task-icon { color: #2563eb; }
.mc-task.info { background: #f6f8fa; border-color: #d0d7de; color: #57606a; }
.mc-task.info .mc-task-icon { color: #57606a; }
@keyframes mc-task-spin { to { transform: rotate(360deg); } }
.mc-task-action { flex: none; font-size: 11px; cursor: pointer; }
.mc-task-close { flex: none; border: none; background: none; cursor: pointer; color: inherit; opacity: 0.6; font-size: 12px; padding: 0 2px; }
.mc-task-close:hover { opacity: 1; }
.mc-task.busy .mc-task-close { display: none; }
/* "Task Status" panel: tasks of pages the user has left */
.bg-tasks { position: relative; display: inline-flex; align-items: center; margin-left: 14px; flex: none;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif; z-index: 2100; }
.bg-tasks.hidden { display: none; }
.bg-tasks-pill { display: inline-flex; align-items: center; gap: 6px; padding: 3px 10px; border-radius: 12px;
    border: 1px solid #94a3b8; background: #f8fafc; color: #334155; font-size: 12px; font-weight: 600; cursor: pointer; white-space: nowrap; }
.bg-tasks-pill:hover { background: #eef2f7; }
.bg-tasks-pill .mc-task-spin { width: 12px; height: 12px; border-width: 2px; border-color: #cbd5e1; border-top-color: #475569; }
.bg-tasks-pill .bg-fail { color: #dc2626; }
.bg-tasks-caret { font-size: 10px; opacity: 0.7; }
.bg-tasks-panel { position: absolute; top: calc(100% + 6px); left: 0; width: 440px; max-width: 70vw; max-height: 55vh;
    overflow-y: auto; background: #ffffff; border: 1px solid #cbd5e1; border-radius: 10px;
    box-shadow: 0 10px 28px rgba(15, 23, 42, 0.22); padding: 8px; box-sizing: border-box; }
.bg-tasks.collapsed .bg-tasks-panel { display: none; }
.bg-tasks-head { display: flex; align-items: center; justify-content: space-between; font-size: 12px; font-weight: 700;
    color: #475569; padding: 2px 4px 8px; border-bottom: 1px solid #e2e8f0; margin-bottom: 8px;
    position: sticky; top: -8px; background: #ffffff; z-index: 1; margin-top: -8px; padding-top: 10px; }
.bg-tasks-head button { border: none; background: none; cursor: pointer; color: #64748b; font-size: 12px; }
.bg-tasks-list { display: flex; flex-direction: column; gap: 8px; }
.bg-task-item { display: flex; flex-direction: column; gap: 4px; }
.bg-task-boxes { display: flex; flex-direction: column; gap: 6px; }
.bg-task-src { font-size: 11px; color: #64748b; padding-left: 2px; }
.bg-task-src b { color: #334155; }
.bg-task-item .mc-task { box-shadow: none; }`;

    const VERB_DONE = {
        Creating: 'Created', Deleting: 'Deleted', Updating: 'Updated', Adding: 'Added', Removing: 'Removed',
        Attaching: 'Attached', Detaching: 'Detached', Associating: 'Associated', Disassociating: 'Disassociated',
        Uploading: 'Uploaded', Emptying: 'Emptied', Enabling: 'Enabled', Disabling: 'Disabled', Resizing: 'Resized',
        Registering: 'Registered', Unregistering: 'Unregistered', Starting: 'Started', Stopping: 'Stopped',
    };
    const VERB_BASE = {
        Creating: 'Create', Deleting: 'Delete', Updating: 'Update', Adding: 'Add', Removing: 'Remove',
        Attaching: 'Attach', Detaching: 'Detach', Associating: 'Associate', Disassociating: 'Disassociate',
        Uploading: 'Upload', Emptying: 'Empty', Enabling: 'Enable', Disabling: 'Disable', Resizing: 'Resize',
        Registering: 'Register', Unregistering: 'Unregister', Starting: 'Start', Stopping: 'Stop',
    };
    const baseText = s => String(s || '').replace(/\s*(…|\.\.\.)\s*$/, '');
    // "Creating VPC x…" -> "Create VPC x" (for "... failed")
    const verbText = s => {
        const b = baseText(s);
        const m = b.match(/^(\w+)\b/);
        return m && VERB_BASE[m[1]] ? VERB_BASE[m[1]] + b.slice(m[1].length) : b;
    };
    const doneText = s => {
        const b = baseText(s);
        const m = b.match(/^(\w+)\b/);
        return m && VERB_DONE[m[1]] ? VERB_DONE[m[1]] + b.slice(m[1].length) : `${b} done`;
    };
    const methodOf = (url, options) => String((options && options.method) || (url && url.method) || 'GET').toUpperCase();
    const typeOfMethod = method => method === 'DELETE' ? 'delete' : (method === 'POST' ? 'create' : (method === 'GET' ? 'refresh' : 'update'));
    const isSpiderUrl = url => {
        if (typeof url !== 'string') return false;
        return url.startsWith('/spider/') || url.startsWith(location.origin + '/spider/');
    };

    // ======================================================================================================
    // HOST: draws the boxes and sends the requests that change resources
    // ======================================================================================================
    function createHost(hostWin) {
        const doc = hostWin.document;
        const hostFetch = hostWin.fetch.bind(hostWin);   // before the page wraps it
        const clients = [];                // { win, key, path, alive, trays, boxes:Set, unlinked:[], awaiting, source }
        const traysByKey = new Map();      // frame element (kept across reloads) -> trays
        let bg = null;                     // "Task Status" panel

        function ensureStyle() {
            if (doc.getElementById('mc-task-style-4')) return;
            const st = doc.createElement('style');
            st.id = 'mc-task-style-4';
            st.textContent = TASK_CSS;
            (doc.head || doc.documentElement).appendChild(st);
        }

        const clientOf = win => clients.find(c => c.win === win);

        // the viewport of a page in the host's coordinates (frames inside frames)
        function viewportOf(win) {
            let x = 0, y = 0;
            try {
                let w = win;
                while (w !== hostWin && w.frameElement) {
                    const fe = w.frameElement, r = fe.getBoundingClientRect();
                    x += r.left + fe.clientLeft; y += r.top + fe.clientTop;
                    w = w.parent;
                }
                return { x, y, w: win.innerWidth, h: win.innerHeight };
            } catch (e) {
                return { x: 0, y: 0, w: hostWin.innerWidth, h: hostWin.innerHeight };
            }
        }

        function traysFor(client) {
            if (client.trays && client.trays.create.isConnected) return client.trays;
            ensureStyle();
            let trays = traysByKey.get(client.key);
            if (!trays || !trays.create.isConnected) {
                const mk = type => {
                    const el = doc.createElement('div');
                    el.className = 'mc-task-tray ' + type;
                    doc.body.appendChild(el);
                    return el;
                };
                trays = { create: mk('create'), delete: mk('delete'), refresh: mk('refresh') };
                traysByKey.set(client.key, trays);
            }
            client.trays = trays;
            place(client.win);
            return trays;
        }

        // keep the trays at the bottom corners of the page's frame
        function place(win) {
            const client = clientOf(win);
            if (!client || !client.trays) return;
            const t = client.trays, v = viewportOf(win);
            const vw = doc.documentElement.clientWidth, vh = doc.documentElement.clientHeight;
            const bottom = vh - (v.y + v.h) + 12;
            const maxW = Math.max(200, v.w * 0.32) + 'px', maxH = Math.max(80, v.h * 0.45) + 'px';
            Object.assign(t.create.style, { left: (v.x + 12) + 'px', right: 'auto', bottom: bottom + 'px', maxWidth: maxW, maxHeight: maxH });
            Object.assign(t.delete.style, { right: (vw - (v.x + v.w) + 12) + 'px', left: 'auto', bottom: bottom + 'px', maxWidth: maxW, maxHeight: maxH,
                transform: '', alignItems: '' });
            Object.assign(t.refresh.style, { left: (v.x + v.w / 2) + 'px', right: 'auto', bottom: bottom + 'px', maxWidth: maxW, maxHeight: maxH });
            // a page that only deletes (the Dashboard) sets MULTI_TASK_DELETE_ALIGN = 'center': deletes bottom-center, wider
            let center = false;
            try { center = win.MULTI_TASK_DELETE_ALIGN === 'center'; } catch (e) {}
            if (center) {
                Object.assign(t.delete.style, { left: (v.x + v.w / 2) + 'px', right: 'auto', transform: 'translateX(-50%)',
                    alignItems: 'center', maxWidth: Math.max(300, v.w * 0.7) + 'px' });
            }
        }
        hostWin.addEventListener('resize', () => clients.forEach(c => c.alive && place(c.win)));

        // where a box comes from: "VPC/Subnet · aws-config01"
        function sourceOf(win) {
            let page = '', conn = '';
            try {
                page = (win.document.title || '').replace(/\s*\(Multi-Connection\)\s*$/i, ' (multi)').replace(/\s*Management\b/i, '').trim();
                const sel = win.document.getElementById('connConfig');
                conn = sel && sel.value ? sel.value : '';
            } catch (e) {}
            if (!conn) {
                try {
                    const multi = JSON.parse(hostWin.sessionStorage.getItem('multiConnections') || 'null');
                    if (Array.isArray(multi) && multi.length > 1 && /\(multi\)/.test(page)) conn = `${multi.length} connections`;
                    else conn = hostWin.sessionStorage.getItem('selectedConfigName') || '';
                } catch (e) {}
            }
            return { page: page || 'page', conn };
        }

        // ---- pages ----
        function attach(win) {
            let key = null, path = '';
            try { key = win.frameElement || win; path = win.location.pathname; } catch (e) { key = win; }
            const client = { win, key, path, alive: true, trays: null, boxes: new Set(), unlinked: [], awaiting: null, source: null };
            // the page before it in the same frame
            clients.filter(c => c.key === key && !c.alive).forEach(prev => {
                clearTimeout(prev.relocateTimer);
                if (prev.path === path) {
                    // a reload: its boxes stay where they are
                    prev.boxes.forEach(b => { b.client = client; client.boxes.add(b); });
                    prev.boxes.clear();
                } else {
                    relocate(prev);
                }
                drop(prev);
            });
            clients.push(client);
            return client;
        }

        function drop(client) {
            const i = clients.indexOf(client);
            if (i >= 0) clients.splice(i, 1);
        }

        // the page is left: its boxes are orphans now (results come from their requests)
        function detach(win) {
            const client = clientOf(win);
            if (!client || !client.alive) return;
            client.alive = false;
            client.awaiting = null;
            client.unlinked = [];
            client.boxes.forEach(b => {
                b.orphan = true;
                if (!b.busy) return;
                if (!b.mutating) { b.remove(); return; }            // a load: nothing changes, nothing to show
                // its request is not known, or was sent by the page (cancelled with it): the result cannot be shown
                if (!b.links.length || b.links.some(r => r.pageSide && !r.settled)) b.untracked();
                else b.check();                                     // its requests may have answered already
            });
            // no page loaded in the frame after it (e.g. a page without boxes, or the frame was removed)
            client.relocateTimer = hostWin.setTimeout(() => { relocate(client); drop(client); }, 2500);
        }

        // boxes of a left page: running creates/deletes and failures go to the "Task Status" panel
        function relocate(client) {
            client.boxes.forEach(b => {
                if (!b.el.isConnected) return;
                if ((b.busy && b.mutating) || b.keep) toPanel(b);
                else b.remove();
            });
            client.boxes.clear();
        }

        // ---- "Task Status" panel (top bar of the main page) ----
        function panel() {
            if (bg && bg.root.isConnected) return bg;
            ensureStyle();
            const root = doc.createElement('div');
            root.className = 'bg-tasks hidden collapsed';   // folded until the pill is clicked
            const pill = doc.createElement('button');
            pill.type = 'button';
            pill.className = 'bg-tasks-pill';
            pill.title = 'Status of the running and finished tasks';
            const box = doc.createElement('div');
            box.className = 'bg-tasks-panel';
            const head = doc.createElement('div');
            head.className = 'bg-tasks-head';
            head.innerHTML = '<span>Task Status</span>';
            const fold = doc.createElement('button');
            fold.type = 'button';
            fold.textContent = 'Hide ▴';
            fold.title = 'Hide the list (the tasks go on)';
            head.appendChild(fold);
            const list = doc.createElement('div');
            list.className = 'bg-tasks-list';
            box.append(head, list);
            root.append(pill, box);
            const bar = doc.querySelector('.top-menu');
            const right = bar && bar.querySelector('.top-menu-right');
            if (bar && right) bar.insertBefore(root, right);
            else { Object.assign(root.style, { position: 'fixed', top: '8px', right: '12px' }); doc.body.appendChild(root); }
            pill.addEventListener('click', () => root.classList.toggle('collapsed'));
            fold.addEventListener('click', () => root.classList.add('collapsed'));
            bg = { root, pill, list };
            // boxes remove themselves (fade out, ✕): drop their rows and update the pill
            new hostWin.MutationObserver(updatePanel).observe(list, { childList: true, subtree: true, attributes: true, attributeFilter: ['class'] });
            return bg;
        }

        // tasks of the same page (and connection) share one group under one title
        function toPanel(b) {
            const p = panel();
            const s = b.source || { page: 'page', conn: '' };
            const key = `${s.page}\u0000${s.conn}`;
            let item = Array.from(p.list.children).find(el => el.dataset.key === key);
            if (!item) {
                item = doc.createElement('div');
                item.className = 'bg-task-item';
                item.dataset.key = key;
                const src = doc.createElement('div');
                src.className = 'bg-task-src';
                const page = doc.createElement('b');
                page.textContent = s.page;
                src.append(page, doc.createTextNode(s.conn ? ` · ${s.conn}` : ''));
                const boxes = doc.createElement('div');
                boxes.className = 'bg-task-boxes';
                item.append(src, boxes);
                p.list.appendChild(item);
            }
            item.querySelector('.bg-task-boxes').appendChild(b.el);
            b.inPanel = true;
            p.root.classList.remove('hidden');   // the pill shows up; the list stays as the user left it
            updatePanel();
        }

        function updatePanel() {
            if (!bg) return;
            bg.list.querySelectorAll('.bg-task-item').forEach(item => { if (!item.querySelector('.mc-task')) item.remove(); });
            const boxes = Array.from(bg.list.querySelectorAll('.mc-task'));
            if (!boxes.length) { bg.root.classList.add('hidden'); return; }
            // creating(7) deleting(3) failed(2): running tasks by kind, then failures
            const busy = boxes.filter(el => el.classList.contains('busy'));
            const count = cls => busy.filter(el => el.classList.contains(cls)).length;
            const failed = boxes.filter(el => el.classList.contains('fail')).length;
            const parts = [];
            [['create', 'creating'], ['delete', 'deleting'], ['refresh', 'updating']].forEach(([cls, word]) => {
                const n = count(cls);
                if (n) parts.push(`${word}(${n})`);
            });
            if (failed) parts.push(`<span class="bg-fail">failed(${failed})</span>`);
            if (!parts.length) parts.push(`done(${boxes.length})`);
            bg.pill.innerHTML = (busy.length ? '<span class="mc-task-spin"></span>' : '') +
                `Tasks: ${parts.join(' ')} <span class="bg-tasks-caret">▾</span>`;
        }

        // ---- boxes ----
        // opts: timer, onDone (called while the page is shown), mutating (changes resources),
        //       link: 'one' | 'all' (takes the next request(s) of the page), autoText: [present, past, base]
        //       (the host sets the result from the request: fetchWithProgress)
        function box(win, type, text, opts) {
            opts = opts || {};
            const client = clientOf(win) || attach(win);
            const trays = traysFor(client);
            const el = doc.createElement('div');
            el.className = 'mc-task busy ' + (type === 'update' ? 'refresh' : type);
            const spin = doc.createElement('span');
            spin.className = 'mc-task-spin';
            const txt = doc.createElement('span');
            txt.className = 'mc-task-text';
            txt.textContent = text;
            const close = doc.createElement('button');
            close.className = 'mc-task-close';
            close.type = 'button';
            close.title = 'Dismiss';
            close.textContent = '✕';
            el.append(spin, txt);
            let time = null, timerId = null;
            const started = Date.now();
            const elapsed = () => `${((Date.now() - started) / 1000).toFixed(1)}s`;
            if (opts.timer) {
                time = doc.createElement('span');
                time.className = 'mc-task-time';
                time.textContent = '0.0s';
                el.appendChild(time);
                timerId = hostWin.setInterval(() => { time.textContent = elapsed(); }, 100);
            }
            el.appendChild(close);
            trays[type === 'update' ? 'refresh' : type].appendChild(el);
            trays[type === 'update' ? 'refresh' : type].scrollTop = 1e9;

            const b = {
                el, client, busy: true, orphan: false, keep: false, inPanel: false, links: [],
                mutating: opts.mutating !== undefined ? !!opts.mutating : (type === 'create' || type === 'delete'),
                autoText: opts.autoText || null, source: sourceOf(win), text,
            };
            client.boxes.add(b);
            const removeEl = () => { el.remove(); client.boxes.delete(b); b.client && b.client.boxes.delete(b); };
            el.addEventListener('animationend', e => { if (e.animationName === 'mc-task-out') removeEl(); });
            close.addEventListener('click', removeEl);

            const stop = () => {
                if (timerId) { hostWin.clearInterval(timerId); timerId = null; time.textContent = elapsed(); }
                el.classList.remove('busy');
                if (b.busy) {
                    b.busy = false;
                    if (b.client.awaiting === b) b.client.awaiting = null;
                    if (opts.onDone && b.client.alive && !b.orphan) { try { opts.onDone(); } catch (e) {} }
                }
                if (b.inPanel) updatePanel();
            };
            const setIcon = mark => {
                const icon = doc.createElement('span');
                icon.className = 'mc-task-icon';
                icon.textContent = mark;
                if (spin.isConnected) spin.replaceWith(icon);
                else { const old = el.querySelector('.mc-task-icon'); if (old) old.replaceWith(icon); }
            };
            Object.assign(b, {
                remove: removeEl,
                setText(s) { b.text = s; txt.textContent = s; },
                info(s) {
                    if (!b.busy && el.classList.contains('ok')) return;
                    stop();
                    el.classList.add('ok', 'info');
                    setIcon('ℹ');
                    txt.textContent = s;
                    el.classList.add('autofade');
                },
                dismiss() {
                    // the page ended it without a result; when it changes resources and is still running
                    // (orphaned), the result comes from its requests
                    stop();
                    el.classList.add('dismissed');
                },
                finish(ok, s, action) {
                    if (!b.busy && (el.classList.contains('ok') || el.classList.contains('fail'))) return;
                    stop();
                    el.classList.remove('info');
                    el.classList.add(ok ? 'ok' : 'fail');
                    setIcon(ok ? '✓' : '✕');
                    txt.textContent = s;
                    b.keep = !ok;
                    if (action && b.client.alive) {
                        const btn = doc.createElement('button');
                        btn.className = 'mc-task-action';
                        btn.type = 'button';
                        btn.textContent = action.label;
                        btn.addEventListener('click', e => { try { action.onClick(e); } catch (err) {} });
                        close.before(btn);
                        b.keep = true;
                    } else if (ok) {
                        el.classList.add('autofade');
                    }
                },
                // the page is gone before the result and the request is not known
                untracked() {
                    stop();
                    el.classList.add('info');
                    setIcon('?');
                    txt.textContent = `${baseText(b.text)} — page left before the result; check the list`;
                    b.keep = true;
                },
                // all linked requests answered: set the result (fetchWithProgress, or a page that is gone)
                check() {
                    if (!b.busy || !b.links.length || b.links.some(r => !r.settled)) return;
                    if (!b.autoText && !b.orphan) return;               // the page sets the result itself
                    const failed = b.links.filter(r => !r.ok);
                    const t = b.autoText || [baseText(b.text), doneText(b.text), verbText(b.text)];
                    if (!failed.length) { b.finish(true, t[1]); return; }
                    const r = failed[0];
                    // the page shows the details itself; a page that is gone cannot: the box shows them
                    const msg = b.orphan && r.message ? `: ${r.message}` : '';
                    b.finish(false, `${t[2]} failed (HTTP ${r.status || '-'}${msg})`);
                },
            });
            if (opts.link) { b.linkMode = opts.link; client.awaiting = b; }
            else if (b.mutating) {
                // a box made right after its requests (multiFanout): take the first one of its kind
                const want = type === 'create' ? ['POST', 'PUT'] : (type === 'delete' ? ['DELETE'] : null);
                const i = client.unlinked.findIndex(r => !want || want.includes(r.method));
                if (i >= 0) link(b, client.unlinked.splice(i, 1)[0]);
            }
            return b;
        }

        function link(b, req) {
            b.links.push(req);
            req.boxes.push(b);
            if (req.settled) b.check();
        }

        // a request of a page: linked to the box that waits for it, or kept for the box made next
        function track(client, method) {
            const req = { method, settled: false, ok: false, status: 0, message: '', boxes: [] };
            const a = client && client.awaiting;
            if (a && a.busy) {
                link(a, req);
                if (a.linkMode === 'one') client.awaiting = null;
            } else if (client && client.alive && method !== 'GET') {
                client.unlinked.push(req);
                hostWin.setTimeout(() => {
                    const i = client.unlinked.indexOf(req);
                    if (i >= 0) client.unlinked.splice(i, 1);
                }, 0);
            }
            return req;
        }

        function settle(req, status, bodyText, error) {
            req.settled = true;
            req.status = status;
            let result = null;
            try { result = bodyText ? JSON.parse(bodyText) : null; } catch (e) {}
            const failedResult = result && (result.Result === 'false' || result.Result === false);
            req.ok = !error && status >= 200 && status < 300 && !failedResult;
            req.message = error ? String(error.message || error)
                : (!req.ok ? String((result && (result.message || result.Message)) || bodyText || '').trim().slice(0, 300) : '');
            req.boxes.forEach(b => b.check());
        }

        // fetch() by the host: goes on when the page is left
        function hfetch(win, url, options) {
            const client = clientOf(win);
            const req = track(client, methodOf(url, options));
            const p = hostFetch(url, options);
            p.then(response => response.clone().text().then(t => settle(req, response.status, t), () => settle(req, response.status, '')),
                   error => settle(req, 0, '', error));
            return p;
        }

        // a request sent by the page itself (not by the host): its result while the page is shown
        function linkPromise(win, b, promise) {
            const req = track(null, 'GET');
            req.pageSide = true;
            link(b, req);
            if (b.client.awaiting === b && b.linkMode === 'one') b.client.awaiting = null;
            promise.then(response => response.clone().text().then(t => settle(req, response.status, t), () => settle(req, response.status, '')),
                         error => settle(req, 0, '', error));
        }

        // Several Spider API requests in one call (POST /spider/adminweb/fanout): the server sends them at the
        // same time and streams each response back as soon as it arrives. One Promise per request, in order.
        function fanout(win, requests, baseOptions) {
            const client = clientOf(win);
            const reqs = requests.map(r => track(client, String(r.method || 'GET').toUpperCase()));
            const pending = requests.map(() => {
                let resolve, reject;
                const promise = new hostWin.Promise((res, rej) => { resolve = res; reject = rej; });
                return { promise, resolve, reject, done: false };
            });
            const finish = (index, response, error, bodyText) => {
                const p = pending[index];
                if (!p || p.done) return;
                p.done = true;
                settle(reqs[index], response ? response.status : 0, bodyText || '', error);
                if (error) p.reject(error); else p.resolve(response);
            };
            const options = Object.assign({}, baseOptions || {}, {
                method: 'POST',
                headers: Object.assign({ 'Content-Type': 'application/json' }, (baseOptions && baseOptions.headers) || {}),
                body: JSON.stringify({
                    requests: requests.map((r, i) => ({ id: String(i), method: r.method, path: r.path, headers: r.headers, body: r.body }))
                })
            });
            hostFetch('/spider/adminweb/fanout', options)
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
                            const body = typeof r.body === 'string' ? r.body : (r.body == null ? '' : JSON.stringify(r.body));
                            finish(index, new Response(noBody ? null : body, {
                                status: r.status, headers: { 'Content-Type': 'application/json' }
                            }), null, body);
                        } else {
                            finish(index, null, new Error(r.error || 'the request could not be sent'));
                        }
                    }
                    if (done) break;
                }
                throw new Error('no response from the server');
            })
            .catch(error => pending.forEach((p, i) => finish(i, null, error)));
            return pending.map(p => p.promise);
        }

        function awaiting(win) {
            const c = clientOf(win);
            return !!(c && c.awaiting && c.awaiting.busy);
        }

        return { attach, detach, place, box, fetch: hfetch, fanout, linkPromise, awaiting };
    }

    // ======================================================================================================
    // this window: the host, or a page of the host
    // ======================================================================================================
    let host = null;
    if (window.TASK_HOST === true) {
        host = window.AdminTaskHost = createHost(window);
        return;   // the main page has no boxes of its own
    }
    try { if (window.top !== window && window.top.AdminTaskHost) host = window.top.AdminTaskHost; } catch (e) { host = null; }
    if (!host) host = createHost(window);   // opened alone: its own host

    host.attach(window);
    window.addEventListener('pagehide', () => host.detach(window));
    window.addEventListener('resize', () => host.place(window));

    const NOOP_BOX = { setText() {}, info() {}, finish() {}, dismiss() {} };
    window.taskBox = function (type, text, opts) {
        if (HIDDEN) {
            if (opts && opts.onDone) {
                const done = () => { try { opts.onDone(); } catch (e) {} };
                return { setText() {}, info: done, finish: done, dismiss: done };
            }
            return NOOP_BOX;
        }
        return host.box(window, type, text, opts);
    };

    // requests that change resources go through the host, so they go on when the page is left
    // (the page's own fetch wrapper, e.g. the one adding the auth header, is called before this one)
    const pageFetch = window.fetch.bind(window);
    window.fetch = function (url, options) {
        const method = methodOf(url, options);
        if (isSpiderUrl(url) && (method !== 'GET' || host.awaiting(window))) return host.fetch(window, url, options);
        return pageFetch(url, options);
    };

    // multiFanout of the multi-connection pages, sent by the host (with the page's auth header)
    window.taskFanout = function (requests) {
        let base = {};
        try { if (typeof window.createFetchOptions === 'function') base = window.createFetchOptions({}); } catch (e) {}
        return host.fanout(window, requests, base);
    };

    // ---- texts of the requests ----
    const RESOURCE_NAMES = {
        vpc: 'VPC', subnet: 'Subnet', securitygroup: 'SecurityGroup', rules: 'Rules', keypair: 'KeyPair',
        vm: 'VM', cspvm: 'VM', controlvm: 'VM', disk: 'Disk', myimage: 'MyImage', nlb: 'NLB', vmgroup: 'VM Group',
        vms: 'VMs', health: 'Health', cluster: 'Cluster', nodegroup: 'NodeGroup', rdbms: 'RDBMS', nic: 'NIC',
        privateip: 'Private IP', publicip: 'PublicIP', dnszone: 'DNS Zone', record: 'Record', tag: 'Tag',
        s3: 'Bucket', filesystem: 'FileSystem', connectionconfig: 'Connection', regsubnet: 'Subnet',
    };
    // path words that are actions on the resource before them
    const ACTIONS = {
        attach: ['Attaching', 'Attached', 'Attach'], detach: ['Detaching', 'Detached', 'Detach'],
        associate: ['Associating', 'Associated', 'Associate'], disassociate: ['Disassociating', 'Disassociated', 'Disassociate'],
        size: ['Resizing', 'Resized', 'Resize'],
    };
    const VERBS = {
        POST: ['Creating', 'Created', 'Create'], PUT: ['Updating', 'Updated', 'Update'],
        DELETE: ['Deleting', 'Deleted', 'Delete'], GET: ['Loading', 'Loaded', 'Load'],
    };
    const nameOf = word => RESOURCE_NAMES[word] || word;
    const dec = s => { try { return decodeURIComponent(s); } catch (e) { return s; } };

    // "Creating VPC vpc-01", "Deleting Subnet sn-01 of VPC vpc-01", "Attaching NIC nic-01", ...
    // Returns { type, text: [present, past, base] }
    function describe(url, options) {
        const method = methodOf(url, options);
        let path = String(url);
        try { path = new URL(path, location.href).pathname; } catch (e) { path = path.split('?')[0]; }
        const segs = path.replace(/^\/spider\/?/, '').split('/').filter(Boolean).map(dec);
        const query = (() => { try { return new URL(String(url), location.href).searchParams; } catch (e) { return new URLSearchParams(); } })();
        let bodyName = '';
        try {
            const body = options && typeof options.body === 'string' ? JSON.parse(options.body) : null;
            bodyName = (body && body.ReqInfo && (body.ReqInfo.Name || body.ReqInfo.Key)) || '';
        } catch (e) {}

        let verbs = VERBS[method] || VERBS.GET;
        let words;
        const last = segs[segs.length - 1];
        if (segs[0] === 'controlvm') {
            const action = query.get('action') || 'control';
            const cap = action.charAt(0).toUpperCase() + action.slice(1);
            const vm = `VM ${segs[1] || ''}`.trim();
            return { type: 'update', text: [`${cap} ${vm}`, `${cap} ${vm} done`, `${cap} ${vm}`] };
        }
        if (segs.length >= 2 && ACTIONS[last]) {
            verbs = ACTIONS[last];
            words = `${nameOf(segs[segs.length - 3] || segs[0])} ${segs[segs.length - 2] || ''}`;
        } else if (segs.length >= 3) {
            // sub resource: /<res>/<name>/<sub>[/<subName>]
            const sub = nameOf(segs[2]);
            const subName = segs[3] || bodyName;
            words = `${sub}${subName ? ' ' + subName : ''} of ${nameOf(segs[0])} ${segs[1]}`;
        } else {
            words = `${nameOf(segs[0] || 'resource')} ${segs[1] || bodyName}`.trim();
        }
        const type = method === 'DELETE' ? 'delete' : (method === 'POST' && !ACTIONS[last] ? 'create' : (method === 'GET' ? 'refresh' : 'update'));
        return { type, text: verbs.map(v => `${v} ${words}`.trim()) };
    }

    // Requests answered by the Spider server alone (its meta DB, no CSP call): fast, so no progress box.
    //   connection info (connectionconfig, driver, credential, region, cloudos), resource counts (count*),
    //   server status (healthcheck, readyz, endpointinfo) and the AdminWeb's own endpoints
    const SPIDER_ONLY = /^\/spider\/(connectionconfig|driver|credential|region|cloudos|count[a-z]*|healthcheck|readyz|endpointinfo|adminweb)(\/|$)/i;
    function spiderOnly(url) {
        let path = String(url);
        try { path = new URL(path, location.href).pathname; } catch (e) { path = path.split('?')[0]; }
        return SPIDER_ONLY.test(path);
    }
    window.isSpiderOnlyRequest = spiderOnly;

    // fetch() with a progress box per request (only for requests that call the CSP, see SPIDER_ONLY).
    // label: optional box text (default: from the method and path). The third argument may also be the
    // old keepAlive flag (true): it is ignored. The host sets the result of the box from the response.
    window.fetchWithProgress = function (url, options, label) {
        if (spiderOnly(url)) return window.fetch(url, options);
        const method = methodOf(url, options);
        const d = describe(url, options);
        const text = typeof label === 'string' && label ? [baseText(label), doneText(label), verbText(label)] : d.text;
        const box = window.taskBox(d.type, `${text[0]}…`, { timer: true, mutating: method !== 'GET', link: 'one', autoText: text });
        const promise = window.fetch(url, options);   // the page's wrapper (auth) -> the host (linked to the box)
        // a page whose fetch wrapper does not reach the host: the result while the page is shown
        if (box && box.links && !box.links.length) host.linkPromise(window, box, promise);
        return promise;
    };

    // a box type from its text: "Creating ..." -> create, "Deleting ..." -> delete, else refresh
    function typeOf(label) {
        if (/^\s*(creat|add|upload|insert|attach|regist)/i.test(label)) return 'create';
        if (/^\s*(delet|remov|unregist|detach|drop|empty)/i.test(label)) return 'delete';
        if (/^\s*(updat|enabl|disabl|resiz|associat|disassociat|start|stop|reboot|suspend|resume)/i.test(label)) return 'update';
        return 'refresh';
    }

    // a box around any work: showProgressBar('Creating Bucket b1...') ... hideProgressBar([ok, resultText])
    // hideProgressBar() without a result just closes the box (the page reports errors itself);
    // hideProgressBar(true|false[, text]) shows the result. Calls nest: each one ends the last running box.
    // The requests sent while it runs belong to it: when the page is left, they set its result.
    const openBoxes = [];
    window.showProgressBar = function (label) {
        const text = (typeof label === 'string' && label.trim()) ? baseText(label.trim()) : 'Processing';
        const type = typeOf(text);
        const box = window.taskBox(type, `${text}…`, { timer: true, mutating: type !== 'refresh', link: 'all' });
        openBoxes.push({ box, text });
    };
    window.hideProgressBar = function (ok, resultText) {
        const open = openBoxes.pop();
        if (!open) return;
        if (ok === false) open.box.finish(false, resultText || `${open.text} failed`);
        else if (ok === true) open.box.finish(true, resultText || doneText(open.text));
        else open.box.dismiss();
    };
})();
