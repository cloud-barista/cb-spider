// Delete all the resources of a connection, in the order (and with the waits) of
// test/vm-cb-user-validation-cli/common/10.destroy.sh:
//   1. S3 buckets (force: with their objects)
//   2. in parallel:
//      [cluster] Clusters: NodeGroups first (type1 CSPs) => wait gone => Cluster => wait gone
//      [others]  NLB, MyImage, RDBMS (RDBMS: wait gone) => VMs => Disks
//   3. the rest (KeyPair, SG, VPC, ...): Spider Destroy API, retried while some resources remain
// Requests refused because a resource is still in use are retried (every 20s, up to 10 times).
//
//   destroyConnectionResources(conn, report) => Promise<{ ok, remained, log }>
//     report(text): called with the current step (for a progress box)
//
// The browser runs these steps: the page must stay open until they finish.
(function () {
    const RETRY_MAX = 10, RETRY_INTERVAL = 20000;      // delete retries (in-use, dependency)
    const WAIT_MAX = 120, WAIT_INTERVAL = 30000;        // waits until resources are gone (1 hour)
    const NG_STALL_LIMIT = 600000 / WAIT_INTERVAL;      // NodeGroups not being deleted for 10 minutes
    const DESTROY_TRIES = 3, DESTROY_INTERVAL = 60000;  // Spider Destroy API
    // CSPs whose Cluster cannot be deleted while NodeGroups are attached (delete the NodeGroups first).
    // Others (type2: Azure, GCP, IBM, NCP, NHN) delete the system NodeGroup with the Cluster.
    const NODEGROUP_FIRST_CSPS = ['AWS', 'ALIBABA', 'TENCENT'];

    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
    const enc = encodeURIComponent;

    // Spider API call => { status, ok, data, text }
    async function api(method, path, body) {
        const options = { method, headers: { 'Accept': 'application/json' } };
        if (body) {
            options.headers['Content-Type'] = 'application/json';
            options.body = JSON.stringify(body);
        }
        try {
            const response = await fetch('/spider/' + path, options);
            const text = await response.text();
            let data = null;
            try { data = JSON.parse(text); } catch (e) {}
            return { status: response.status, ok: response.ok, data, text };
        } catch (e) {
            return { status: 0, ok: false, data: null, text: e.message };
        }
    }

    function message(res) {
        const d = res.data || {};
        return String(d.message || d.Message || res.text || `HTTP ${res.status}`).slice(0, 200);
    }

    function isNotFound(res) {
        return res.status === 404 || /not exist|not found|does not exist|NotFound/i.test(res.text || '');
    }

    function names(list) {
        return (list || []).map(item => (item.IId && item.IId.NameId) || item.Name).filter(Boolean);
    }

    window.destroyConnectionResources = async function (conn, report) {
        const log = [];
        const remained = [];
        const say = text => { log.push(`[${new Date().toLocaleTimeString()}] ${text}`); };
        const step = text => { say(text); if (report) report(text); };
        const body = { ConnectionName: conn };

        // DELETE with retries while in use => 'ok' | 'skip' (not found) | 'fail'
        async function deleteWithRetry(path, label, max = RETRY_MAX) {
            for (let i = 1; i <= max; i++) {
                const res = await api('DELETE', path, body);
                if (res.ok) { say(`[OK]   delete ${label}`); return 'ok'; }
                if (isNotFound(res)) { say(`[SKIP] ${label} not found`); return 'skip'; }
                say(`  ... retry delete ${label} (${i}/${max}): ${message(res)}`);
                if (i < max) await sleep(RETRY_INTERVAL);
            }
            say(`[FAIL] delete ${label}`);
            remained.push(label);
            return 'fail';
        }

        // check() => 'done' | 'wait' | 'abort'
        async function waitUntil(label, check) {
            for (let i = 1; i <= WAIT_MAX; i++) {
                const state = await check();
                if (state === 'done') return true;
                if (state === 'abort') return false;
                say(`  ... waiting ${label} (${i}/${WAIT_MAX})`);
                await sleep(WAIT_INTERVAL);
            }
            say(`[FAIL] timeout waiting ${label}`);
            return false;
        }

        async function list(resource, key) {
            const res = await api('GET', `${resource}?ConnectionName=${enc(conn)}`);
            if (!res.ok) {
                say(`[SKIP] list ${resource}: ${message(res)}`);
                return [];
            }
            return names(res.data && res.data[key]);
        }

        // resources the driver does not support are skipped (no capability info: try them all)
        const capRes = await api('GET', `driver/capability?ConnectionName=${enc(conn)}`);
        const cap = capRes.ok && capRes.data ? capRes.data : {};
        const supported = key => cap[key] !== false;
        const connRes = await api('GET', `connectionconfig/${enc(conn)}`);
        const provider = String((connRes.data && connRes.data.ProviderName) || '').toUpperCase();

        // ---- 1. S3 buckets
        step('deleting S3 buckets');
        const s3 = await api('GET', `s3?ConnectionName=${enc(conn)}`);
        if (s3.ok && s3.data && s3.data.Buckets) {
            const buckets = names(s3.data.Buckets.Bucket);
            await Promise.all(buckets.map(b =>
                deleteWithRetry(`s3/${enc(b)}?force=true&ConnectionName=${enc(conn)}`, `S3 bucket: ${b}`, 3)));
        } else {
            say(`[SKIP] S3: ${s3.ok ? 'no bucket' : message(s3)}`);
        }

        // ---- 2-a. Clusters: NodeGroups => Cluster
        async function deleteCluster(name) {
            const get = () => api('GET', `cluster/${enc(name)}?ConnectionName=${enc(conn)}`);
            const res = await get();
            if (!res.ok) { say(`[SKIP] Cluster not found: ${name}`); return; }

            if (NODEGROUP_FIRST_CSPS.includes(provider)) {
                const ngs = names(res.data && res.data.NodeGroupList);
                const accepted = (await Promise.all(ngs.map(ng =>
                    deleteWithRetry(`cluster/${enc(name)}/nodegroup/${enc(ng)}`, `NodeGroup: ${name}/${ng}`)))).filter(r => r !== 'fail').length;
                if (ngs.length > 0 && accepted === 0) {
                    say('[WARN] NodeGroup delete requests all failed; try Cluster delete anyway');
                } else if (ngs.length > 0) {
                    // Deleting / Updating: in progress => keep waiting. Active / Inactive for 10 min: not progressing => stop
                    let stall = 0;
                    const gone = await waitUntil(`NodeGroups of ${name} gone`, async () => {
                        const r = await get();
                        if (isNotFound(r)) return 'done';
                        if (!r.ok) return 'wait';
                        const list = (r.data && r.data.NodeGroupList) || [];
                        if (list.length === 0) return 'done';
                        if (list.some(ng => ng.Status === 'Deleting' || ng.Status === 'Updating')) stall = 0;
                        else if (++stall >= NG_STALL_LIMIT) return 'abort';
                        return 'wait';
                    });
                    if (!gone) say('[WARN] NodeGroups still remain; try Cluster delete anyway');
                }
            }

            // retried: refused while NodeGroups are still being removed
            if (await deleteWithRetry(`cluster/${enc(name)}`, `Cluster: ${name}`) !== 'ok') return;
            const gone = await waitUntil(`Cluster ${name} gone`, async () => (isNotFound(await get()) ? 'done' : 'wait'));
            if (gone) say(`[OK]   Cluster deleted: ${name}`);
            else remained.push(`Cluster: ${name}`);
        }

        async function deleteClusters() {
            if (!supported('ClusterHandler')) return;
            const clusters = await list('cluster', 'cluster');
            if (clusters.length) step(`deleting ${clusters.length} Cluster(s) (NodeGroups first; this can take a while)`);
            await Promise.all(clusters.map(deleteCluster));
        }

        // ---- 2-b. NLB, MyImage, RDBMS => VMs => Disks
        async function deleteOthers() {
            const nlbs = supported('NLBHandler') ? await list('nlb', 'nlb') : [];
            const myImages = supported('MyImageHandler') ? await list('myimage', 'myImage') : [];
            const rdbmss = supported('RDBMSHandler') ? await list('rdbms', 'rdbms') : [];
            if (nlbs.length + myImages.length + rdbmss.length) {
                step(`deleting ${nlbs.length} NLB(s), ${myImages.length} MyImage(s), ${rdbmss.length} RDBMS(s)`);
            }
            await Promise.all([
                ...nlbs.map(n => deleteWithRetry(`nlb/${enc(n)}`, `NLB: ${n}`)),
                ...myImages.map(m => deleteWithRetry(`myimage/${enc(m)}`, `MyImage: ${m}`)),
                ...rdbmss.map(async d => {
                    if (await deleteWithRetry(`rdbms/${enc(d)}`, `RDBMS: ${d}`) !== 'ok') return;
                    // the VPC cannot be deleted while the RDBMS is being removed
                    const gone = await waitUntil(`RDBMS ${d} gone`, async () =>
                        (isNotFound(await api('GET', `rdbms/${enc(d)}?ConnectionName=${enc(conn)}`)) ? 'done' : 'wait'));
                    if (!gone) remained.push(`RDBMS: ${d}`);
                })
            ]);

            const vms = supported('VMHandler') ? await list('vm', 'vm') : [];
            if (vms.length) step(`terminating ${vms.length} VM(s)`);
            await Promise.all(vms.map(v => deleteWithRetry(`vm/${enc(v)}`, `VM: ${v}`)));

            // retried while the terminating VMs still hold them
            const disks = supported('DiskHandler') ? await list('disk', 'disk') : [];
            if (disks.length) step(`deleting ${disks.length} Disk(s)`);
            await Promise.all(disks.map(d => deleteWithRetry(`disk/${enc(d)}`, `Disk: ${d}`)));
        }

        await Promise.all([deleteClusters(), deleteOthers()]);

        // ---- 3. the rest (KeyPair, SG, VPC, ...): Spider Destroy API
        let destroyed = false, destroyRemained = [];
        for (let i = 1; i <= DESTROY_TRIES; i++) {
            step(`deleting KeyPairs, SecurityGroups, VPCs, ... (try ${i}/${DESTROY_TRIES})`);
            const res = await api('DELETE', 'destroy', body);
            if (res.ok && res.data && res.data.IsAllDestroyed) {
                say(`[OK]   all resources destroyed: ${conn}`);
                destroyed = true;
                break;
            }
            destroyRemained = [];
            ((res.data && res.data.DeletedAllListByResourceType) || []).forEach(t =>
                (t.RemainedErrorInfoList || []).forEach(e => destroyRemained.push(`${t.ResourceType}: ${e.Name} (${String(e.ErrorMsg).slice(0, 120)})`)));
            say(`  some resources remained: ${destroyRemained.join('; ') || message(res)}`);
            if (i < DESTROY_TRIES) {
                step(`some resources remained; retry after ${DESTROY_INTERVAL / 1000}s (${i}/${DESTROY_TRIES})`);
                await sleep(DESTROY_INTERVAL);
            }
        }
        if (!destroyed) remained.push(...(destroyRemained.length ? destroyRemained : ['KeyPair/SG/VPC: Destroy API failed']));

        return { ok: remained.length === 0, remained, log };
    };
})();
