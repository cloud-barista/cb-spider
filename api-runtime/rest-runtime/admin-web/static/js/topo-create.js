// Topology "add resource" helper for AdminWeb resource pages.
//
// The Topology page opens a resource page in a modal iframe with
//   ?topoCreate=<kind>&tok=<token>[&vpc=<name>][&subnet=<name>][&cluster=<name>]
// This script (loaded by every page that has a create form) then shows only that page's
// create form over the Topology, pre-fills the parent resource (VPC / Subnet / Cluster),
// and tells the Topology how it ended (postMessage, same origin):
//   topoCreateReady   - the form is shown
//   topoCreateStarted - the form was submitted ({names}); the Topology closes the dialog and keeps
//                       this page running hidden until the request answers
//   topoCreateResult  - every create request answered ({ok: [names], failed: [{name, msg}]})
//   topoCreateDone    - created (result details unknown)
//   topoCreateCancel  - the form was closed without creating anything
// Without ?topoCreate the script does nothing.
(function () {
    const q = new URLSearchParams(location.search);
    const kind = q.get('topoCreate');
    if (!kind || window.parent === window) return;
    const tok = q.get('tok') || '';
    const KEY = 'topoCreate:' + tok;
    const send = (type, extra) => {
        try { window.parent.postMessage(Object.assign({ type, tok, kind }, extra || {}), location.origin); } catch (e) {}
    };

    // overlay: the create form element, open: shows it, close: functions that hide it,
    // post: create request (method + path), selects: [select id, wanted option] to pre-fill in order
    const SPECS = {
        vpc:       { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/vpc$/] },
        subnet:    { overlay: 'subnet-overlay', open: p => showSubnetOverlay(p.get('vpc')), close: ['hideSubnetOverlay'],
                     post: ['POST', /^\/spider\/vpc\/[^/]+\/subnet$/] },
        sg:        { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/securitygroup$/], selects: p => [['vpcName', p.get('vpc')]] },
        keypair:   { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/keypair$/] },
        disk:      { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/disk$/] },
        vm:        { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/vm$/],
                     selects: p => [['vpcSelect', p.get('vpc')], ['subnetSelect', p.get('subnet')]] },
        publicip:  { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/publicip$/] },
        nlb:       { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/nlb$/], selects: p => [['vpcName', p.get('vpc')]] },
        cluster:   { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/cluster$/], selects: p => [['vpcSelect', p.get('vpc')]] },
        nodegroup: { overlay: 'overlayNewNodeGroup', open: p => showNewNodeGroupOverlay(p.get('cluster')),
                     close: ['hideNewNodeGroupOverlay'], post: ['POST', /^\/spider\/cluster\/[^/]+\/nodegroup$/] },
        rdbms:     { overlay: 'overlay', open: () => showOverlay(), close: ['hideOverlay'],
                     post: ['POST', /^\/spider\/rdbms$/], selects: p => [['vpcName', p.get('vpc')]] },
    };
    const spec = SPECS[kind];
    if (!spec) { send('topoCreateCancel'); return; }

    // Reloaded by the page after a successful create → report the results; nothing else to do.
    let saved = null;
    try { saved = sessionStorage.getItem(KEY); } catch (e) {}
    if (saved) {
        try { sessionStorage.removeItem(KEY); } catch (e) {}
        let r = null;
        try { r = JSON.parse(saved); } catch (e) {}
        if (r && Array.isArray(r.ok)) send('topoCreateResult', { ok: r.ok, failed: r.failed || [] });
        else send('topoCreateDone');
        return;
    }

    // Only the dialogs are shown: the page's own content is hidden and its background is
    // transparent, so the form appears over the Topology.
    const st = document.createElement('style');
    st.textContent = `
        html, body { background: transparent !important; }
        body > *:not(script):not(style) { visibility: hidden !important; }
        [id*="overlay" i], [class*="overlay" i], [id*="modal" i], [class*="modal" i] { visibility: visible !important; }
        [id*="progress" i] { display: none !important; }`;   /* the Topology shows its own status box */
    (document.head || document.documentElement).appendChild(st);

    // Create requests of this form (a page can send several, ex: "#" VMs)
    const res = { ok: [], failed: [] };
    let started = 0, settled = 0, finished = false;
    const save = () => { try { sessionStorage.setItem(KEY, JSON.stringify(res)); } catch (e) {} };
    const names = [];
    function settle() {
        settled++;
        if (settled === started) {
            finished = true;
            try { sessionStorage.removeItem(KEY); } catch (e) {}
            send('topoCreateResult', res);
        }
    }
    // Closing the form before submitting it cancels the dialog (after submitting, the result is sent)
    function finish() {
        if (finished || started) return;
        finished = true;
        send('topoCreateCancel');
    }
    function reqName(init) {
        try {
            const b = JSON.parse(init && init.body);
            return (b.ReqInfo && (b.ReqInfo.Name || b.ReqInfo.NameId)) || b.Name || '';
        } catch (e) { return ''; }
    }

    // Installed on 'load' (after every page script has run), so the page's own fetch wrapper and
    // form functions exist wherever this script is in the page.
    function installHooks() {
        // Watch the create request: a successful response marks the create as done
        // (the pages reload themselves right after; the reload is reported above).
        const origFetch = window.fetch;
        window.fetch = function (input, init) {
            const p = origFetch.apply(this, arguments);
            try {
                const method = String((init && init.method) || (input && input.method) || 'GET').toUpperCase();
                const path = new URL(typeof input === 'string' ? input : input.url, location.href).pathname;
                if (method === spec.post[0] && spec.post[1].test(path)) {
                    const name = reqName(init);
                    started++;
                    names.push(name);
                    // the form now runs in the background: its own error pop-ups are not shown
                    // (the Topology shows the result)
                    window.alert = m => console.log('topoCreate (alert suppressed):', m);
                    send('topoCreateStarted', { names: names.filter(Boolean) });
                    p.then(r => {
                        if (r && r.ok) {             // recorded at once: the page may reload right after
                            res.ok.push(name); save(); settle();
                            return;
                        }
                        const status = r ? 'HTTP ' + r.status : 'no response';
                        (r ? r.clone().text() : Promise.resolve('')).then(t => {
                            let msg = t;
                            try { const j = JSON.parse(t); msg = j.message || j.Message || t; } catch (e) {}
                            res.failed.push({ name, msg: String(msg || status).trim().slice(0, 400) });
                            save(); settle();
                        }, () => { res.failed.push({ name, msg: status }); save(); settle(); });
                    }, err => {
                        res.failed.push({ name, msg: (err && err.message) || 'request failed' });
                        save(); settle();
                    });
                }
            } catch (e) {}
            return p;
        };
        spec.close.forEach(fn => {
            const f = window[fn];
            if (typeof f !== 'function') return;
            window[fn] = function () {
                const r = f.apply(this, arguments);
                setTimeout(finish, 0);
                return r;
            };
        });
    }

    // Select the wanted option once the page has filled the list (matched by value, text, or the
    // name before " (CIDR)"), and keep it for a few seconds in case the page resets its form after
    // loading. Stops as soon as the user touches the list.
    function pick(id, want) {
        return new Promise(resolve => {
            if (!want) { resolve(); return; }
            const t0 = Date.now();
            let picked = false, userTouched = false;
            const match = sel => Array.from(sel.options).find(o =>
                o.value === want || o.text === want || o.text.split(' ')[0] === want);
            (function tick() {
                const sel = document.getElementById(id);
                if (sel && !sel._topoWatch) {
                    sel._topoWatch = true;
                    ['mousedown', 'keydown'].forEach(ev => sel.addEventListener(ev, () => { userTouched = true; }));
                }
                if (userTouched) { resolve(); return; }
                const opt = sel && match(sel);
                if (opt && sel.selectedIndex !== opt.index) {
                    sel.selectedIndex = opt.index;
                    sel.dispatchEvent(new Event('change', { bubbles: true }));
                }
                if (opt && !picked) { picked = true; resolve(); }
                const limit = picked ? 4000 : 20000;
                if (Date.now() - t0 < limit) setTimeout(tick, 200);
                else resolve();
            })();
        });
    }
    async function prefill() {
        const list = spec.selects ? spec.selects(q) : [];
        for (const [id, want] of list) await pick(id, want);
    }

    function start() {
        installHooks();
        try { spec.open(q); } catch (e) { console.error('topoCreate:', e); send('topoCreateCancel'); return; }
        const t0 = Date.now();
        (function waitShown() {
            const el = document.getElementById(spec.overlay);
            if (el && getComputedStyle(el).display !== 'none') { send('topoCreateReady'); prefill(); return; }
            if (Date.now() - t0 > 30000) { send('topoCreateReady'); return; }
            setTimeout(waitShown, 100);
        })();
    }
    if (document.readyState === 'complete') start();
    else window.addEventListener('load', start);
})();
