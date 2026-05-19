/* SpiderWatch — Client-side JavaScript */

// ── Resources settings modal ──────────────────────────────────────────────
(function () {
  const btnOpen   = document.getElementById('btn-resources');
  const backdrop  = document.getElementById('modal-resources');
  const listEl    = document.getElementById('modal-resource-list');
  const btnCancel = document.getElementById('btn-resources-cancel');
  const btnSave   = document.getElementById('btn-resources-save');
  const btnAll    = document.getElementById('chk-resources-all');
  let chkCleanup = null;
  if (!btnOpen || !backdrop || !listEl || !btnCancel || !btnSave) return;

  function syncSelectAll() {
    if (!btnAll) return;
    const boxes = listEl.querySelectorAll('input[type="checkbox"]');
    btnAll.checked = boxes.length > 0 && Array.from(boxes).every(c => c.checked);
    btnAll.indeterminate = !btnAll.checked && Array.from(boxes).some(c => c.checked);
  }
  if (btnAll) {
    btnAll.addEventListener('change', () => {
      listEl.querySelectorAll('input[type="checkbox"]').forEach(c => c.checked = btnAll.checked);
    });
  }

  let currentEnabled = [];

  function buildCheckboxes(all, enabled) {
    listEl.innerHTML = '';
    all.forEach(kind => {
      const label = document.createElement('label');
      label.className = 'modal-resource-item';
      const chk = document.createElement('input');
      chk.type = 'checkbox';
      chk.name = 'resource';
      chk.value = kind;
      chk.checked = enabled.includes(kind);
      chk.addEventListener('change', syncSelectAll);
      label.appendChild(chk);
      label.appendChild(document.createTextNode(kind));
      listEl.appendChild(label);
    });
    syncSelectAll();
    // Cleanup checkbox below s3 (last resource)
    const cleanupLabel = document.createElement('label');
    cleanupLabel.className = 'modal-resource-item modal-cleanup-item';
    chkCleanup = document.createElement('input');
    chkCleanup.type = 'checkbox';
    chkCleanup.id = 'chk-cleanup';
    cleanupLabel.appendChild(chkCleanup);
    cleanupLabel.appendChild(document.createTextNode('Delete all'));
    listEl.appendChild(cleanupLabel);
  }

  async function openModal() {
    try {
      const [resRes, cleanupRes] = await Promise.all([
        fetch('/api/v1/config/resources'),
        fetch('/api/v1/config/cleanup'),
      ]);
      if (!resRes.ok) throw new Error('HTTP ' + resRes.status);
      if (!cleanupRes.ok) throw new Error('HTTP ' + cleanupRes.status);
      const data = await resRes.json();
      const cleanupData = await cleanupRes.json();
      currentEnabled = data.enabled || [];
      buildCheckboxes(data.all || [], currentEnabled);
      if (chkCleanup) chkCleanup.checked = !!cleanupData.cleanup;
      backdrop.classList.add('open');
    } catch (err) {
      showToast('✗ Failed to load resources: ' + err.message, 'fail');
    }
  }

  function closeModal() {
    backdrop.classList.remove('open');
  }

  btnOpen.addEventListener('click', openModal);
  btnCancel.addEventListener('click', closeModal);

  // Close on backdrop click (outside the modal box)
  backdrop.addEventListener('click', e => {
    if (e.target === backdrop) closeModal();
  });

  // Close on Escape key
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && backdrop.classList.contains('open')) closeModal();
  });

  btnSave.addEventListener('click', async () => {
    const checked = Array.from(listEl.querySelectorAll('input[name="resource"]:checked'))
      .map(c => c.value);
    const cleanupVal = chkCleanup ? chkCleanup.checked : true;
    btnSave.disabled = true;
    btnSave.textContent = '⟳ Saving…';
    try {
      const [resRes, cleanupRes] = await Promise.all([
        fetch('/api/v1/config/resources', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ resources: checked }),
        }),
        fetch('/api/v1/config/cleanup', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ cleanup: cleanupVal }),
        }),
      ]);
      if (resRes.ok && cleanupRes.ok) {
        closeModal();
        showToast('✓ Resources updated. Changes take effect on the next run.', 'ok');
      } else {
        const failed = !resRes.ok ? resRes : cleanupRes;
        const body = await failed.json().catch(() => ({ message: failed.statusText }));
        showToast('✗ Failed to save: ' + body.message, 'fail');
      }
    } catch (err) {
      showToast('✗ Network error: ' + err.message, 'fail');
    } finally {
      btnSave.disabled = false;
      btnSave.textContent = 'Save';
    }
  });
})();

// ── CSP selection modal ───────────────────────────────────────────────────
(function () {
  const btnOpen   = document.getElementById('btn-csps');
  const backdrop  = document.getElementById('modal-csps');
  const listEl    = document.getElementById('modal-csp-list');
  const btnCancel = document.getElementById('btn-csps-cancel');
  const btnSave   = document.getElementById('btn-csps-save');
  const btnAll    = document.getElementById('chk-csps-all');
  const btnNone   = btnAll; // same element
  if (!btnOpen || !backdrop || !listEl || !btnCancel || !btnSave) return;

  function syncSelectAll() {
    if (!btnAll) return;
    const boxes = listEl.querySelectorAll('input[type="checkbox"]');
    btnAll.checked = boxes.length > 0 && Array.from(boxes).every(c => c.checked);
    btnAll.indeterminate = !btnAll.checked && Array.from(boxes).some(c => c.checked);
  }
  if (btnAll) {
    btnAll.addEventListener('change', () => {
      listEl.querySelectorAll('input[type="checkbox"]').forEach(c => c.checked = btnAll.checked);
    });
  }

  async function openModal() {
    try {
      const res = await fetch('/api/v1/config/csps');
      if (!res.ok) throw new Error('HTTP ' + res.status);
      const data = await res.json();
      listEl.innerHTML = '';
      (data.csps || []).forEach(csp => {
        const label = document.createElement('label');
        label.className = 'modal-csp-item';
        const chk = document.createElement('input');
        chk.type = 'checkbox';
        chk.name = 'csp';
        chk.value = csp.name;
        chk.checked = csp.enabled;
        chk.addEventListener('change', syncSelectAll);
        const info = document.createElement('span');
        info.className = 'csp-item-info';
        const nameEl = document.createElement('span');
        nameEl.className = 'csp-item-name';
        nameEl.textContent = csp.name;
        const connEl = document.createElement('span');
        connEl.className = 'csp-item-conn';
        connEl.textContent = csp.connection;
        info.appendChild(nameEl);
        info.appendChild(connEl);
        label.appendChild(chk);
        label.appendChild(info);
        listEl.appendChild(label);
      });
      syncSelectAll();
      backdrop.classList.add('open');
    } catch (err) {
      showToast('✗ Failed to load CSPs: ' + err.message, 'fail');
    }
  }

  function closeModal() { backdrop.classList.remove('open'); }

  btnOpen.addEventListener('click', openModal);
  btnCancel.addEventListener('click', closeModal);
  backdrop.addEventListener('click', e => { if (e.target === backdrop) closeModal(); });
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && backdrop.classList.contains('open')) closeModal();
  });

  btnSave.addEventListener('click', async () => {
    const checked = Array.from(listEl.querySelectorAll('input[type="checkbox"]:checked'))
      .map(c => c.value);
    btnSave.disabled = true;
    btnSave.textContent = '⟳ Saving…';
    try {
      const res = await fetch('/api/v1/config/csps', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ csps: checked }),
      });
      if (res.ok) {
        closeModal();
        showToast('✓ CSP selection updated. Changes take effect on the next run.', 'ok');
      } else {
        const body = await res.json().catch(() => ({ message: res.statusText }));
        showToast('✗ Failed to save: ' + body.message, 'fail');
      }
    } catch (err) {
      showToast('✗ Network error: ' + err.message, 'fail');
    } finally {
      btnSave.disabled = false;
      btnSave.textContent = 'Save';
    }
  });
})();

// ── Footer clock ──────────────────────────────────────────────────────────
(function () {
  const el = document.getElementById('footer-time');
  if (!el) return;
  function tick() {
    const now = new Date();
    el.textContent = 'Updated: ' + now.toLocaleString('ko-KR', {
      timeZone: 'Asia/Seoul',
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
      hour12: false
    }) + ' KST';
  }
  tick();
  setInterval(tick, 1000);
})();

// ── Toast helper ──────────────────────────────────────────────────────────
function showToast(msg, type) {
  const el = document.getElementById('toast');
  if (!el) return;
  el.textContent = msg;
  el.className = 'toast toast-visible toast-' + (type || 'ok');
  clearTimeout(el._timer);
  el._timer = setTimeout(() => {
    el.className = 'toast';
  }, 4000);
}

// ── Run-Now button ────────────────────────────────────────────────────────
// Handled inside the running-state IIFE below so setTestRunning() is in scope.

// ── Poll for run completion, then reload page ──────────────────────────────
function pollForCompletion(intervalMs) {
  const timer = setInterval(async () => {
    try {
      const res = await fetch('/api/v1/runs/latest');
      if (!res.ok) return;
      const data = await res.json();
      if (data && (data.status === 'DONE' || data.status === 'FAILED' || data.status === 'STOPPED')) {
        clearInterval(timer);
        showToast('✓ Run completed (' + data.status + '). Reloading…', 'ok');
        setTimeout(() => location.reload(), 1500);
      }
    } catch (_) { /* network hiccup, keep polling */ }
  }, intervalMs);
}

// ── Live board: poll /board during a run and update the board in-place ────
function pollBoard(intervalMs) {
  const timer = setInterval(async () => {
    const board = document.getElementById('live-board');
    if (!board) { clearInterval(timer); return; }
    try {
      const res = await fetch('/board');
      if (!res.ok) return;
      board.innerHTML = await res.text();
    } catch (_) { /* network hiccup */ }
  }, intervalMs);
  return timer;
}

// ── Running state + Spider state ─────────────────────────────────────────
(function () {
  const banner = document.getElementById('running-banner');
  const btnTrigger     = document.getElementById('btn-trigger');
  const btnCleanupOnly = document.getElementById('btn-cleanup-only');
  const btnStopRun     = document.getElementById('btn-stop-run');
  const btnStart = document.getElementById('btn-spider-start');
  const btnStop  = document.getElementById('btn-spider-stop');

  let testRunning    = !!window.__isRunning;
  let spiderRunning  = !!window.__isSpiderRunning;  // set from server-rendered value
  let externalSpider = !!window.__isExternalSpider; // true when external_url is configured

  function applyButtonState() {
    if (btnTrigger) {
      btnTrigger.disabled    = testRunning;
      btnTrigger.textContent = testRunning ? '⟳ Running…' : '▶ Run Now';
    }
    if (btnCleanupOnly) btnCleanupOnly.disabled = testRunning;
    if (btnStopRun) {
      btnStopRun.style.display = testRunning ? '' : 'none';
      btnStopRun.disabled      = false;
    }
    // Start/Stop Spider: always disabled in external mode (server not managed by SpiderWatch)
    if (btnStart) btnStart.disabled = externalSpider || testRunning || spiderRunning;
    if (btnStop)  btnStop.disabled  = externalSpider || testRunning || !spiderRunning;
  }

  function setTestRunning(running) {
    testRunning = running;
    if (banner) {
      if (running) banner.classList.remove('hidden');
      else         banner.classList.add('hidden');
    }
    // Update the server-rendered status badge dynamically.
    const badge = document.querySelector('.page-header .run-meta .badge');
    if (badge) {
      if (running) {
        badge.className = 'badge badge-status-running';
        badge.textContent = 'RUNNING';
      }
      // On completion, location.reload() will re-render with the final status.
    }
    applyButtonState();
  }

  function setSpiderRunning(running) {
    spiderRunning = running;
    applyButtonState();
  }

  async function refreshSpiderState() {
    try {
      const res = await fetch('/api/v1/spider/status');
      if (res.ok) {
        const d = await res.json();
        externalSpider = !!d.external;
        setSpiderRunning(!!d.running);
      }
    } catch (_) {}
  }

  // Initialise from server-rendered values (dashboard) or fetch once (other pages)
  setTestRunning(testRunning);
  if (testRunning) {
    pollForCompletion(5000);
    pollBoard(3000);
  }
  if (typeof window.__isSpiderRunning !== 'undefined') {
    setSpiderRunning(!!window.__isSpiderRunning);
  } else {
    refreshSpiderState(); // one-time fetch on non-dashboard pages
  }

  // ── Run Now button ────────────────────────────────────────────────────
  if (btnTrigger) {
    btnTrigger.addEventListener('click', async () => {
      btnTrigger.disabled = true;
      btnTrigger.textContent = '⟳ Starting…';
      let triggered = false;
      try {
        const res = await fetch('/api/v1/runs/trigger', { method: 'POST' });
        if (res.status === 202) {
          triggered = true;
          setTestRunning(true);
          showToast('✓ Run triggered! The page will refresh soon.', 'ok');
          pollForCompletion(6000);
          pollBoard(3000);
        } else if (res.status === 409) {
          showToast('⚠ A run is already in progress.', 'fail');
        } else {
          const body = await res.text().catch(() => '');
          showToast('✗ Failed to trigger run: ' + body, 'fail');
        }
      } catch (err) {
        showToast('✗ Network error: ' + err.message, 'fail');
      } finally {
        // Only reset button if trigger did not succeed — setTestRunning(true)
        // already set the correct disabled/label state for a running test.
        if (!triggered) {
          btnTrigger.disabled = false;
          btnTrigger.textContent = '▶ Run Now';
        }
      }
    });
  }

  // ── Cleanup Only button ──────────────────────────────────────────────
  if (btnCleanupOnly) {
    btnCleanupOnly.addEventListener('click', async () => {
      btnCleanupOnly.disabled = true;
      btnCleanupOnly.textContent = '⟳ Starting…';
      let triggered = false;
      try {
        const res = await fetch('/api/v1/runs/cleanup', { method: 'POST' });
        if (res.status === 202) {
          triggered = true;
          setTestRunning(true);
          showToast('✓ Cleanup-only run triggered! The page will refresh soon.', 'ok');
          pollForCompletion(6000);
          pollBoard(3000);
        } else if (res.status === 409) {
          showToast('⚠ A run is already in progress.', 'fail');
        } else {
          const body = await res.text().catch(() => '');
          showToast('✗ Failed to trigger cleanup: ' + body, 'fail');
        }
      } catch (err) {
        showToast('✗ Network error: ' + err.message, 'fail');
      } finally {
        if (!triggered) {
          btnCleanupOnly.disabled = false;
          btnCleanupOnly.textContent = '🧹 Clean up';
        }
      }
    });
  }

  // ── Stop Run button ───────────────────────────────────────────────────
  if (btnStopRun) {
    btnStopRun.addEventListener('click', async () => {
      btnStopRun.disabled = true;
      btnStopRun.textContent = '⟳ Stopping…';
      try {
        const res = await fetch('/api/v1/runs/stop', { method: 'POST' });
        if (res.status === 202) {
          showToast('⏹ Stop signal sent. Waiting for current operation to finish…', 'ok');
          // Keep polling — the run will transition to STOPPED when done.
        } else if (res.status === 409) {
          showToast('No run is currently in progress.', 'fail');
          setTestRunning(false);
        } else {
          const body = await res.json().catch(() => ({ message: res.statusText }));
          showToast('✗ Failed to stop run: ' + body.message, 'fail');
        }
      } catch (err) {
        showToast('✗ Network error: ' + err.message, 'fail');
      } finally {
        btnStopRun.textContent = '⏹ Stop';
        // Keep disabled until the run actually finishes (poll will reload page).
      }
    });
  }

  // ── Start Spider button ───────────────────────────────────────────────
  if (btnStart) {
    btnStart.addEventListener('click', async () => {
      btnStart.disabled = true;
      btnStart.textContent = '⟳ Starting…';
      try {
        const res = await fetch('/api/v1/spider/start', { method: 'POST' });
        if (res.ok) {
          setSpiderRunning(true);
          showToast('✓ Spider started. Connect at localhost:1024', 'ok');
        } else {
          const body = await res.json().catch(() => ({ message: res.statusText }));
          showToast('✗ Failed to start Spider: ' + body.message, 'fail');
          await refreshSpiderState();
        }
      } catch (err) {
        showToast('✗ Network error: ' + err.message, 'fail');
        await refreshSpiderState();
      } finally {
        btnStart.textContent = 'Start Spider';
      }
    });
  }

  // ── Stop Spider button ────────────────────────────────────────────────
  if (btnStop) {
    btnStop.addEventListener('click', async () => {
      btnStop.disabled = true;
      btnStop.textContent = '⟳ Stopping…';
      try {
        const res = await fetch('/api/v1/spider/stop', { method: 'POST' });
        if (res.ok) {
          setSpiderRunning(false);
          showToast('✓ Spider stopped.', 'ok');
        } else {
          const body = await res.json().catch(() => ({ message: res.statusText }));
          showToast('✗ Failed to stop Spider: ' + body.message, 'fail');
          await refreshSpiderState();
        }
      } catch (err) {
        showToast('✗ Network error: ' + err.message, 'fail');
        await refreshSpiderState();
      } finally {
        btnStop.textContent = 'Stop Spider';
      }
    });
  }
})();

// ── GitHub Issue Modal ────────────────────────────────────────────────────
(function () {
  const backdrop    = document.getElementById('modal-gh-issue');
  const metaEl      = document.getElementById('gh-issue-meta');
  const titleInput  = document.getElementById('gh-issue-title');
  const bodyTA      = document.getElementById('gh-issue-body');
  const previewPane = document.getElementById('gh-preview-pane');
  const tabWrite    = document.getElementById('gh-tab-write');
  const tabPreview  = document.getElementById('gh-tab-preview');
  const btnCancel   = document.getElementById('btn-gh-cancel');
  const btnSubmit   = document.getElementById('btn-gh-submit');
  if (!backdrop || !titleInput || !bodyTA || !btnCancel || !btnSubmit) return;

  // ── Write / Preview tab switching ──
  function showWriteTab() {
    tabWrite.classList.add('active');
    tabPreview.classList.remove('active');
    bodyTA.hidden = false;
    previewPane.hidden = true;
  }
  function showPreviewTab() {
    tabPreview.classList.add('active');
    tabWrite.classList.remove('active');
    bodyTA.hidden = true;
    previewPane.hidden = false;
    const md = bodyTA.value || '*No content*';
    previewPane.innerHTML = (typeof marked !== 'undefined')
      ? marked.parse(md, { breaks: true, gfm: true })
      : '<pre style="white-space:pre-wrap">' + md.replace(/</g, '&lt;') + '</pre>';
  }
  if (tabWrite)   tabWrite.addEventListener('click', showWriteTab);
  if (tabPreview) tabPreview.addEventListener('click', showPreviewTab);

  let currentRunID   = '';
  let currentCSP     = '';
  let currentResource = '';
  let currentBtn     = null; // the "File Issue" button that opened the modal

  async function openIssueModal(btn) {
    currentRunID    = btn.dataset.run;
    currentCSP      = btn.dataset.csp;
    currentResource = btn.dataset.resource;
    currentBtn      = btn;

    // Show loading state
    titleInput.value = '⟳ Loading draft…';
    bodyTA.value     = '';
    btnSubmit.disabled = true;
    showWriteTab();
    backdrop.classList.add('open');

    if (metaEl) {
      metaEl.innerHTML =
        '<span class="muted">CSP: <strong>' + currentCSP + '</strong> &nbsp;|&nbsp; ' +
        'Resource: <strong>' + currentResource + '</strong></span>';
    }

    try {
      const url = '/api/v1/runs/' + encodeURIComponent(currentRunID) +
        '/issue-draft?csp=' + encodeURIComponent(currentCSP) +
        '&resource=' + encodeURIComponent(currentResource);
      const res = await fetch(url);
      if (!res.ok) {
        const body = await res.json().catch(() => ({ message: res.statusText }));
        throw new Error(body.message || res.statusText);
      }
      const draft = await res.json();
      titleInput.value   = draft.title || '';
      bodyTA.value       = draft.body  || '';
      btnSubmit.disabled = false;
    } catch (err) {
      titleInput.value = '';
      bodyTA.value     = '⚠ Failed to load draft: ' + err.message;
      showToast('✗ Failed to load issue draft: ' + err.message, 'fail');
    }
  }

  function closeModal() {
    backdrop.classList.remove('open');
    showWriteTab();
    currentBtn = null;
  }

  // Open modal when any "File Issue" button is clicked (event delegation on document)
  document.addEventListener('click', e => {
    const btn = e.target.closest('.btn-gh-issue');
    if (btn) {
      e.preventDefault();
      openIssueModal(btn);
    }
  });

  btnCancel.addEventListener('click', closeModal);
  backdrop.addEventListener('click', e => { if (e.target === backdrop) closeModal(); });
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && backdrop.classList.contains('open')) closeModal();
  });

  btnSubmit.addEventListener('click', async () => {
    const title = titleInput.value.trim();
    const body  = bodyTA.value.trim();
    if (!title) {
      showToast('✗ Issue title cannot be empty.', 'fail');
      return;
    }

    btnSubmit.disabled = true;
    btnSubmit.textContent = '⟳ Submitting…';

    try {
      const res = await fetch(
        '/api/v1/runs/' + encodeURIComponent(currentRunID) + '/issue',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            csp:      currentCSP,
            resource: currentResource,
            title:    title,
            body:     body,
          }),
        }
      );
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({ message: res.statusText }));
        throw new Error(errBody.message || res.statusText);
      }
      const data = await res.json();
      showToast('✓ Issue #' + data.issue_number + ' filed successfully!', 'ok');

      // Replace the "File Issue" button with a link to the new issue, without reload.
      if (currentBtn) {
        const link = document.createElement('a');
        link.className  = 'issue-link';
        link.href       = data.issue_url;
        link.target     = '_blank';
        link.rel        = 'noopener';
        link.textContent = '#' + data.issue_number;
        currentBtn.replaceWith(link);
      }

      closeModal();
    } catch (err) {
      showToast('✗ Failed to file issue: ' + err.message, 'fail');
    } finally {
      btnSubmit.disabled = false;
      btnSubmit.textContent = 'Submit Issue';
    }
  });
})();

// ── Run history: multi-select delete ─────────────────────────────────────
(function () {
  const table   = document.getElementById('history-table');
  const toolbar = document.getElementById('history-toolbar');
  const chkAll  = document.getElementById('chk-all');
  const selCount = document.getElementById('history-sel-count');
  const btnDel  = document.getElementById('btn-delete-selected');
  if (!table || !toolbar || !chkAll || !btnDel) return;

  function getRowChks() {
    return Array.from(table.querySelectorAll('tbody .row-chk'));
  }

  function updateToolbar() {
    const checked = getRowChks().filter(c => c.checked);
    const n = checked.length;
    if (n > 0) {
      toolbar.classList.add('visible');
      selCount.textContent = n + ' selected';
    } else {
      toolbar.classList.remove('visible');
      selCount.textContent = '';
    }
    // Sync select-all checkbox state
    const all = getRowChks();
    chkAll.indeterminate = n > 0 && n < all.length;
    chkAll.checked = all.length > 0 && n === all.length;
  }

  // Select-all checkbox
  chkAll.addEventListener('change', () => {
    getRowChks().forEach(c => {
      c.checked = chkAll.checked;
      c.closest('tr').classList.toggle('row-selected', chkAll.checked);
    });
    updateToolbar();
  });

  // Individual row checkboxes (event delegation)
  table.querySelector('tbody').addEventListener('change', e => {
    if (!e.target.classList.contains('row-chk')) return;
    e.target.closest('tr').classList.toggle('row-selected', e.target.checked);
    updateToolbar();
  });

  // Delete selected
  btnDel.addEventListener('click', async () => {
    const checked = getRowChks().filter(c => c.checked);
    if (checked.length === 0) return;
    if (!confirm(`Delete ${checked.length} run(s)? This cannot be undone.`)) return;

    btnDel.disabled = true;
    btnDel.textContent = '⟳ Deleting…';
    let failed = 0;

    await Promise.all(checked.map(async chk => {
      const id = chk.value;
      try {
        const res = await fetch('/api/v1/runs/' + encodeURIComponent(id), { method: 'DELETE' });
        if (res.ok || res.status === 404) {
          chk.closest('tr').remove();
        } else {
          failed++;
        }
      } catch (_) {
        failed++;
      }
    }));

    // Reset select-all after removal
    chkAll.checked = false;
    chkAll.indeterminate = false;
    toolbar.classList.remove('visible');
    selCount.textContent = '';
    btnDel.disabled = false;
    btnDel.textContent = 'Delete selected';

    if (failed > 0) {
      showToast(`✗ ${failed} run(s) could not be deleted.`, 'fail');
    } else {
      showToast(`✓ ${checked.length} run(s) deleted.`, 'ok');
    }

    // Update the recorded count in the page header
    const remaining = table.querySelectorAll('tbody tr').length;
    const countEl = document.querySelector('.page-header .muted');
    if (countEl) countEl.textContent = remaining + ' runs recorded';
  });
})();
