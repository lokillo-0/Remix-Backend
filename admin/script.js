const BASE_URL = 'https://remixogfn.dev/rmx/server/api/v1';

if (window.location.pathname.endsWith('.html')) {
    window.history.replaceState(null, '', window.location.pathname.replace(/\.html$/, ''));
}

const storedKey = localStorage.getItem('adminKey');
if (!storedKey) {
    window.location.href = 'login.html';
}

document.querySelectorAll('.nav-links li').forEach(link => {
    link.addEventListener('click', () => {
        document.querySelectorAll('.nav-links li').forEach(l => l.classList.remove('active'));
        document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
        link.classList.add('active');
        document.getElementById(link.dataset.tab).classList.add('active');
        document.getElementById('page-title').textContent = link.textContent.trim();
        if (link.dataset.tab === 'manage-panel') fetchAccounts();
    });
});

document.getElementById('logout-btn').addEventListener('click', () => {
    localStorage.removeItem('adminKey');
    window.location.href = 'login.html';
});

function showToast(message, type) {
    if (!type) type = 'success';
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    toast.innerHTML = '<span>' + message + '</span>';
    container.appendChild(toast);
    setTimeout(() => toast.classList.add('show'), 10);
    setTimeout(() => { toast.classList.remove('show'); setTimeout(() => toast.remove(), 300); }, 3000);
}

async function apiCall(endpoint, method, data) {
    if (!method) method = 'POST';
    try {
        const options = { method: method, headers: { 'Content-Type': 'application/json', 'Authorization': storedKey } };
        if (data) options.body = JSON.stringify(data);
        const response = await fetch(BASE_URL + endpoint, options);
        if (!response.ok) {
            let errMsg = 'HTTP error! status: ' + response.status;
            try { const errData = await response.json(); if (errData.error) errMsg = errData.error; } catch (e) {}
            throw new Error(errMsg);
        }
        const result = await response.json();
        showToast('Operation successful!', 'success');
        return result;
    } catch (error) {
        showToast(error.message, 'error');
        console.error(error);
    }
}

async function wipeAllItems(btn) {
    const container = btn.closest('.inline-editor-container');
    const id = container ? container.dataset.accountId : null;
    if (!id) { showToast('Could not find account ID', 'error'); return; }
    if (btn.dataset.confirm !== 'yes') {
        btn.dataset.confirm = 'yes';
        btn.textContent = 'Click again to confirm';
        setTimeout(() => { btn.dataset.confirm = ''; btn.textContent = 'Wipe All Items'; }, 3000);
        return;
    }
    btn.dataset.confirm = '';
    btn.textContent = 'Wipe All Items';
    await apiCall('/admin/items/' + id, 'DELETE');
}

const shopEditor = document.getElementById('shop-json-editor');
const shopEditorStatus = document.getElementById('shop-editor-status');
const shopSaveButton = document.getElementById('btn-shop-save');
const shopReloadButton = document.getElementById('btn-shop-reload');

let shopOriginalContent = '';
let shopHasUnsavedChanges = false;
let shopRequestInFlight = false;

function setShopStatus(message, tone) {
    if (!shopEditorStatus) return;
    shopEditorStatus.textContent = message;
    shopEditorStatus.className = 'shop-editor-status';
    if (tone) shopEditorStatus.classList.add('is-' + tone);
}

function updateShopEditorButtons() {
    if (shopSaveButton) {
        shopSaveButton.disabled = shopRequestInFlight || !shopHasUnsavedChanges;
    }

    if (shopReloadButton) {
        shopReloadButton.disabled = shopRequestInFlight;
    }
}

function refreshShopEditorDirtyState() {
    if (!shopEditor) return;
    shopHasUnsavedChanges = shopEditor.value !== shopOriginalContent;
    if (shopHasUnsavedChanges) {
        setShopStatus('Unsaved changes', 'warning');
    } else if (!shopRequestInFlight) {
        setShopStatus('All changes saved', 'success');
    }
    updateShopEditorButtons();
}

function getErrorMessageFromResponse(data, fallback) {
    if (data && typeof data.error === 'string' && data.error.trim()) return data.error;
    if (data && typeof data.message === 'string' && data.message.trim()) return data.message;
    return fallback;
}

async function loadShopConfigIntoEditor() {
    if (!shopEditor) return;

    shopRequestInFlight = true;
    updateShopEditorButtons();
    setShopStatus('Loading assets/shop.json...', 'info');

    try {
        const response = await fetch(BASE_URL + '/admin/shop/config', {
            headers: { 'Authorization': storedKey }
        });

        let data = null;
        try {
            data = await response.json();
        } catch (e) {}

        if (!response.ok) {
            throw new Error(getErrorMessageFromResponse(data, 'Failed to load shop.json'));
        }

        const content = data && typeof data.content === 'string' ? data.content : '';
        shopOriginalContent = content;
        shopEditor.value = content;
        shopHasUnsavedChanges = false;
        setShopStatus('Loaded assets/shop.json', 'success');
    } catch (error) {
        setShopStatus(error.message || 'Failed to load shop.json', 'error');
        showToast(error.message || 'Failed to load shop.json', 'error');
    } finally {
        shopRequestInFlight = false;
        updateShopEditorButtons();
    }
}

async function saveShopConfigFromEditor() {
    if (!shopEditor || shopRequestInFlight || !shopHasUnsavedChanges) return;

    const content = shopEditor.value;
    try {
        JSON.parse(content);
    } catch (error) {
        setShopStatus('Invalid JSON. Fix syntax before saving.', 'error');
        showToast('Invalid JSON. Fix syntax before saving.', 'error');
        return;
    }

    shopRequestInFlight = true;
    updateShopEditorButtons();
    setShopStatus('Saving assets/shop.json...', 'info');

    try {
        const response = await fetch(BASE_URL + '/admin/shop/config', {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': storedKey,
            },
            body: JSON.stringify({ content: content }),
        });

        let data = null;
        try {
            data = await response.json();
        } catch (e) {}

        if (!response.ok) {
            throw new Error(getErrorMessageFromResponse(data, 'Failed to save shop.json'));
        }

        shopOriginalContent = content;
        shopHasUnsavedChanges = false;
        setShopStatus('Saved assets/shop.json', 'success');
        showToast('shop.json saved', 'success');
    } catch (error) {
        setShopStatus(error.message || 'Failed to save shop.json', 'error');
        showToast(error.message || 'Failed to save shop.json', 'error');
    } finally {
        shopRequestInFlight = false;
        updateShopEditorButtons();
    }
}

if (shopEditor) {
    shopEditor.addEventListener('input', refreshShopEditorDirtyState);
}

if (shopSaveButton) {
    shopSaveButton.addEventListener('click', saveShopConfigFromEditor);
}

if (shopReloadButton) {
    shopReloadButton.addEventListener('click', loadShopConfigIntoEditor);
}

if (shopEditor) {
    loadShopConfigIntoEditor();
}

const renderTarget = document.getElementById('accounts-render-target');
const editorTemplate = document.getElementById('manage-editor-template').innerHTML;

async function fetchAccounts() {
    renderTarget.innerHTML = '<div style="padding: 2rem; text-align: center; color: var(--text-secondary);">Loading accounts...</div>';
    try {
        const response = await fetch(BASE_URL + '/admin/accounts/all', { headers: { 'Authorization': storedKey } });
        if (!response.ok) throw new Error('Failed to fetch');
        const accounts = await response.json();

        renderTarget.innerHTML = '';
        if (!accounts || accounts.length === 0) {
            renderTarget.innerHTML = '<div style="padding: 2rem; text-align: center; color: var(--text-secondary);">No accounts found.</div>';
            return;
        }

        accounts.forEach(function(acc) {
            const row = document.createElement('div');
            row.className = 'account-row';
            const initials = (acc.displayName || '?').charAt(0).toUpperCase();
            const avatarInner = acc.avatar ? '<img src="' + acc.avatar + '" style="width:34px;height:34px;border-radius:50%;object-fit:cover;">' : initials;
            const nameWithStatus = (acc.displayName || 'No Name') + (acc.banned ? ' [BANNED]' : '');
            row.innerHTML =
                '<div class="account-info-col">' +
                    '<div class="account-avatar">' + avatarInner + '</div>' +
                    '<div class="account-info-text">' +
                        '<span class="account-name-val">' + nameWithStatus + '</span>' +
                        '<span class="account-id-val">' + acc.id + '</span>' +
                    '</div>' +
                '</div>' +
                '<div class="account-email-col">' +
                    '<span class="account-email-val">' + (acc.email || 'No Email') + '</span>' +
                '</div>' +
                '<div class="account-actions-col">' +
                    '<button class="btn-edit-row btn-edit" data-id="' + acc.id + '" data-banned="' + (acc.banned ? 'true' : 'false') + '">Edit</button>' +
                '</div>';
            renderTarget.appendChild(row);
        });

        document.querySelectorAll('.btn-edit').forEach(function(btn) {
            btn.addEventListener('click', function(e) {
                document.querySelectorAll('.inline-editor-container').forEach(function(el) { el.remove(); });
                const id = e.target.dataset.id;
                const isBanned = e.target.dataset.banned === 'true';
                const row = e.target.closest('.account-row');
                const container = document.createElement('div');
                container.className = 'inline-editor-container';
                container.style.gridColumn = '1 / -1';
                container.style.padding = '0 1.5rem';
                container.innerHTML = editorTemplate;
                container.dataset.accountId = id;
                row.parentNode.insertBefore(container, row.nextSibling);

                const banButton = container.querySelector('.btn-ban-account');
                if (isBanned) {
                    banButton.textContent = 'Unban Account';
                    banButton.classList.remove('danger');
                    banButton.classList.add('outline');
                }

                container.querySelector('.btn-update-name').addEventListener('click', async function() {
                    const val = container.querySelector('.manage-new-name').value;
                    if (val) await apiCall('/account/management/displayName/' + id, 'POST', { displayName: val });
                });
                container.querySelector('.btn-update-email').addEventListener('click', async function() {
                    const val = container.querySelector('.manage-new-email').value;
                    if (val) await apiCall('/account/management/email/' + id, 'POST', { new_email: val });
                });
                container.querySelector('.btn-update-pass').addEventListener('click', async function() {
                    const val = container.querySelector('.manage-new-pass').value;
                    if (val) await apiCall('/account/management/password/' + id, 'POST', { password: val });
                });
                container.querySelector('.btn-give-vbucks').addEventListener('click', async function() {
                    const amt = container.querySelector('.manage-vbucks-amount').value;
                    if (!amt) return showToast('Enter an amount', 'error');
                    await apiCall('/admin/vbucks/' + id + '/' + amt, 'POST', {});
                });
                container.querySelector('.btn-give-item').addEventListener('click', async function() {
                    const tpl = container.querySelector('.manage-item-template').value;
                    if (!tpl) return showToast('Enter a Template ID', 'error');
                    await apiCall('/admin/grant/' + id + '/' + encodeURIComponent(tpl), 'POST', {});
                });
                container.querySelector('.btn-grant-locker').addEventListener('click', async function() {
                    if (confirm('Give full locker to this account?')) await apiCall('/admin/fulllocker/' + id, 'POST', {});
                });
                container.querySelector('.btn-revoke-locker').addEventListener('click', async function() {
                    if (confirm('Revoke full locker from this account?')) await apiCall('/admin/fulllocker/' + id, 'DELETE');
                });
                container.querySelector('.btn-remove-battlepass').addEventListener('click', async function() {
                    const season = prompt('Enter season number to remove battle pass from:');
                    if (!season) return;
                    if (confirm('Remove battle pass for season ' + season + ' from this account?')) {
                        await apiCall('/admin/battlepass/' + id + '/' + season, 'DELETE');
                    }
                });
                container.querySelector('.btn-ban-account').addEventListener('click', async function() {
                    if (isBanned) {
                        if (!confirm('Unban this account and restore access?')) return;
                        await apiCall('/admin/unban/' + id, 'POST', {});
                        fetchAccounts();
                        return;
                    }

                    if (!confirm('Ban this account permanently? This blocks login and API access.')) return;

                    const reasonInput = prompt('Ban reason (optional):', 'Banned by admin panel');
                    if (reasonInput === null) return;

                    const reason = reasonInput.trim();
                    const payload = reason ? { reason: reason } : {};
                    await apiCall('/admin/ban/' + id, 'POST', payload);
                    fetchAccounts();
                });
                container.querySelector('.btn-delete-account').addEventListener('click', async function() {
                    if (confirm('Permanently delete this account?')) {
                        await apiCall('/admin/account/' + id, 'DELETE');
                        fetchAccounts();
                    }
                });
            });
        });
    } catch (err) {
        renderTarget.innerHTML = '<div style="padding: 2rem; text-align: center; color: #ff5555;">Error loading accounts. Ensure backend endpoint is deployed.</div>';
        console.error(err);
    }
}

document.getElementById('btn-refresh-accounts').addEventListener('click', fetchAccounts);

document.getElementById('accounts-search').addEventListener('input', function() {
    const q = this.value.toLowerCase();
    document.querySelectorAll('.account-row').forEach(function(row) {
        const name = (row.querySelector('.account-name-val') || {}).textContent || '';
        const id = (row.querySelector('.account-id-val') || {}).textContent || '';
        const email = (row.querySelector('.account-email-val') || {}).textContent || '';
        row.style.display = (name + id + email).toLowerCase().includes(q) ? '' : 'none';
    });
});
