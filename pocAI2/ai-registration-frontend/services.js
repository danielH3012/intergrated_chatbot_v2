// Clean direct API Client — All requests go directly to the backend with PostgreSQL persistence.

const API_BASE = (typeof window !== "undefined" && window.API_BASE) || "http://localhost:3000";

// Auth token storage helpers
function getAuthToken() {
  try {
    return localStorage.getItem("qtera_token") || "";
  } catch (e) {
    return "";
  }
}

function setAuthToken(token) {
  try {
    if (token) localStorage.setItem("qtera_token", token);
    else localStorage.removeItem("qtera_token");
  } catch (e) { }
}

function getStoredUser() {
  try {
    const u = localStorage.getItem("qtera_user");
    return u ? JSON.parse(u) : null;
  } catch (e) {
    return null;
  }
}

function setStoredUser(user) {
  try {
    if (user) localStorage.setItem("qtera_user", JSON.stringify(user));
    else localStorage.removeItem("qtera_user");
  } catch (e) { }
}

function getAuthHeaders(customHeaders = {}) {
  const headers = { ...customHeaders };
  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

// Helper for error handling
async function handleResponse(res) {
  if (!res.ok) {
    if (res.status === 401 && typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("qtera:unauthorized"));
    }
    const errBody = await res.json().catch(() => ({}));
    throw new Error(errBody.error || `Request failed with status ${res.status}`);
  }
  if (res.status === 204) {
    return { ok: true };
  }
  return await res.json();
}

// ---------------------------------------------------------------------
// Authentication API (users table)
// ---------------------------------------------------------------------

async function loginUser({ username, password }) {
  const res = await fetch(`${API_BASE}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  const data = await handleResponse(res);
  if (data && data.token) {
    setAuthToken(data.token);
    setStoredUser(data.user);
  }
  return data;
}

async function registerUser({ username, email, password, role, company, id_perusahaan }) {
  const res = await fetch(`${API_BASE}/api/auth/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, email, password, role, company, id_perusahaan }),
  });
  const data = await handleResponse(res);
  if (data && data.token) {
    setAuthToken(data.token);
    setStoredUser(data.user);
  }
  return data;
}

async function getMe() {
  const token = getAuthToken();
  if (!token) return null;
  const res = await fetch(`${API_BASE}/api/auth/me`, {
    headers: getAuthHeaders(),
  });
  const data = await handleResponse(res);
  if (data && data.user) {
    setStoredUser(data.user);
  }
  return data;
}

async function getPerusahaanList() {
  try {
    const res = await fetch(`${API_BASE}/api/perusahaan`);
    const data = await handleResponse(res);
    if (Array.isArray(data)) return data;
    if (data && Array.isArray(data.perusahaan)) return data.perusahaan;
    if (data && Array.isArray(data.data)) return data.data;
    return [];
  } catch (e) {
    console.warn("Failed to fetch perusahaan list:", e);
    return ["No Perusahaan Found"];
  }
}

function logoutUser() {
  setAuthToken("");
  setStoredUser(null);
}

// ---------------------------------------------------------------------
// Assets API
// ---------------------------------------------------------------------

async function getAssets(params = {}) {
  const q = new URLSearchParams();
  if (params.search) q.append("search", params.search);
  if (params.sort) q.append("sort", params.sort);
  if (params.order) q.append("order", params.order);
  if (params.page) q.append("page", String(params.page));
  if (params.pageSize) q.append("pageSize", String(params.pageSize));

  if (params.category && params.category.length) {
    q.append("category", params.category.join(","));
  }
  if (params.brand && params.brand.length) {
    q.append("brand", params.brand.join(","));
  }
  if (params.location && params.location.length) {
    q.append("location", params.location.join(","));
  }

  const res = await fetch(`${API_BASE}/api/assets?${q.toString()}`, {
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

async function getAssetOptions() {
  const res = await fetch(`${API_BASE}/api/assets/options`, {
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

async function getAssetById(id) {
  const res = await fetch(`${API_BASE}/api/assets/${encodeURIComponent(id)}`, {
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

async function createAssetManual(data) {
  const res = await fetch(`${API_BASE}/api/assets`, {
    method: "POST",
    headers: getAuthHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify(data),
  });
  return await handleResponse(res);
}

async function updateAsset(id, data) {
  const res = await fetch(`${API_BASE}/api/assets/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: getAuthHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify(data),
  });
  return await handleResponse(res);
}

async function deleteAsset(id) {
  const res = await fetch(`${API_BASE}/api/assets/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

function downloadAssets(params = {}) {
  const q = new URLSearchParams();
  if (params.search) q.append("search", params.search);
  window.open(`${API_BASE}/api/assets/download?${q.toString()}`, "_blank");
}

function downloadAssetsCSV(assets = []) {
  if (!assets.length) return;
  const headers = ["Asset ID", "Name", "Category", "Brand", "Model/Type", "Purchase Date", "Purchase Price", "Location", "Created At"];
  const rows = assets.map((a) => [
    a.assetId || a.id || "",
    `"${(a.name || "").replace(/"/g, '""')}"`,
    `"${(a.category || "").replace(/"/g, '""')}"`,
    `"${(a.brand || "").replace(/"/g, '""')}"`,
    `"${(a.modelType || "").replace(/"/g, '""')}"`,
    a.purchaseDate || "",
    a.purchasePrice || "",
    `"${(a.location || "").replace(/"/g, '""')}"`,
    a.createdAt || "",
  ]);
  const csvContent = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.setAttribute("href", url);
  link.setAttribute("download", `Assets_${new Date().toISOString().slice(0, 10)}.csv`);
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}

// ---------------------------------------------------------------------
// AI Registration Sessions API
// ---------------------------------------------------------------------

async function startRegistrationSession({ file, rawText }) {
  const formData = new FormData();
  if (rawText) formData.append("rawText", rawText);
  if (file) formData.append("file", file);

  const res = await fetch(`${API_BASE}/api/ai-registration/sessions`, {
    method: "POST",
    headers: getAuthHeaders(),
    body: formData,
  });
  return await handleResponse(res);
}

async function getSession(sessionId) {
  const res = await fetch(`${API_BASE}/api/ai-registration/sessions/${encodeURIComponent(sessionId)}`, {
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

async function updateDraftField(sessionId, draftId, fieldKey, value) {
  const res = await fetch(`${API_BASE}/api/ai-registration/drafts/${encodeURIComponent(draftId)}`, {
    method: "PATCH",
    headers: getAuthHeaders({ "Content-Type": "application/json" }),
    body: JSON.stringify({
      fields: {
        [fieldKey]: { value, source: "edited", confidence: null },
      },
    }),
  });
  return await handleResponse(res);
}

async function discardDraft(sessionId, draftId) {
  const res = await fetch(`${API_BASE}/api/ai-registration/drafts/${encodeURIComponent(draftId)}`, {
    method: "DELETE",
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

async function confirmAll(sessionId) {
  const res = await fetch(`${API_BASE}/api/ai-registration/sessions/${encodeURIComponent(sessionId)}/confirm-all`, {
    method: "POST",
    headers: getAuthHeaders(),
  });
  return await handleResponse(res);
}

// Attach to window for global access
window.getAuthToken = getAuthToken;
window.setAuthToken = setAuthToken;
window.getStoredUser = getStoredUser;
window.loginUser = loginUser;
window.registerUser = registerUser;
window.getMe = getMe;
window.logoutUser = logoutUser;
window.getPerusahaanList = getPerusahaanList;

window.getAssets = getAssets;
window.getAssetOptions = getAssetOptions;
window.getAssetById = getAssetById;
window.createAssetManual = createAssetManual;
window.updateAsset = updateAsset;
window.deleteAsset = deleteAsset;
window.downloadAssets = downloadAssets;
window.downloadAssetsCSV = downloadAssetsCSV;
window.startRegistrationSession = startRegistrationSession;
window.getSession = getSession;
window.updateDraftField = updateDraftField;
window.discardDraft = discardDraft;
window.confirmAll = confirmAll;
