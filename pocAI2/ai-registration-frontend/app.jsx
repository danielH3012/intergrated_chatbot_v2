const { useState, useEffect, useMemo, useCallback, useRef } = React;

const REQUIRED_FIELDS = ["name", "category", "location"];
const ACCEPTED_FILE_TYPES = ["application/pdf", "image/jpeg", "image/jpg", "image/png"];
const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
const MAX_TEXT_LENGTH = 2000;

const FIELD_LABELS = {
  name: "Asset Name",
  category: "Category",
  brand: "Brand",
  model_type: "Model/Type",
  purchase_date: "Purchase Date",
  purchase_price: "Purchase Price",
  location: "Location",
};

const DRAFT_COLUMNS = ["name", "category", "brand", "model_type", "purchase_date", "purchase_price", "location"];

function formatCurrency(v) {
  if (v == null || v === "") return null;
  return "Rp" + Number(v).toLocaleString("id-ID");
}

function formatDate(v) {
  if (!v) return null;
  const d = new Date(v);
  if (isNaN(d)) return v;
  return d.toLocaleDateString("en-GB", { day: "2-digit", month: "short", year: "numeric" });
}

function isRowReady(fields) {
  return REQUIRED_FIELDS.every((k) => fields[k] && fields[k].value != null && fields[k].value !== "");
}

// ---------------------------------------------------------------------
// Toast
// ---------------------------------------------------------------------

function ToastHost({ toasts }) {
  return (
    <div className="toast-host">
      {toasts.map((t) => (
        <div key={t.id} className={"toast toast-" + (t.type || "default")}>
          {t.message}
        </div>
      ))}
    </div>
  );
}

function useToasts() {
  const [toasts, setToasts] = useState([]);
  const push = useCallback((message, type) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((t) => [...t, { id, message, type }]);
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), 3500);
  }, []);
  return { toasts, push };
}

function Spinner() {
  return <span className="spinner" aria-hidden="true" />;
}

function EmptyState({ icon, title, cta }) {
  return (
    <div className="empty-state">
      <i className={"ph " + icon} />
      <p>{title}</p>
      {cta}
    </div>
  );
}

// ---------------------------------------------------------------------
// Navigation Bar
// ---------------------------------------------------------------------

function TopNavBar({ user, onLogout, onGoList, onGoChat, page }) {
  const roleColors = {
    admin: { bg: "#f3e8ff", color: "#7e22ce", border: "#d8b4fe" },
    operator: { bg: "#dbeafe", color: "#1d4ed8", border: "#93c5fd" },
    viewer: { bg: "#ccfbf1", color: "#0f766e", border: "#5eead4" },
  };
  const roleStyle = roleColors[user?.role] || roleColors.operator;

  return (
    <header className="top-navbar">
      <div className="nav-brand" onClick={onGoList}>
        <div className="nav-brand-logo">
          <i className="ph ph-shield-check" />
        </div>
        <span>QTERA Asset Management</span>
      </div>
      {page === "chat" && (
        <button className="btn btn-ghost btn-sm nav-back-btn" onClick={onGoList}>
          <i className="ph ph-arrow-left" /> Back to Assets
        </button>
      )}

      <div className="nav-user-panel">
        {user ? (
          <>
            <div className="user-badge">
              <i className="ph ph-user-circle" style={{ fontSize: "18px" }} />
              <span>{user.username}</span>
              {user.company && (
                <span className="user-company-pill" title="Company">
                  <i className="ph ph-buildings" style={{ fontSize: "11px" }} /> {user.company}
                </span>
              )}
              {user.role && (
                <span
                  className="user-role-pill"
                  style={{
                    backgroundColor: roleStyle.bg,
                    color: roleStyle.color,
                    border: `1px solid ${roleStyle.border}`,
                  }}
                >
                  {user.role}
                </span>
              )}
            </div>
            <button className="btn btn-ghost btn-sm" onClick={onLogout} title="Sign Out">
              <i className="ph ph-sign-out" /> Logout
            </button>
          </>
        ) : (
          <span className="muted small">Not authenticated</span>
        )}
      </div>
    </header>
  );
}

// ---------------------------------------------------------------------
// Authentication Page (Login / Sign Up)
// ---------------------------------------------------------------------

function AuthPage({ onAuthSuccess, showToast }) {
  const [mode, setMode] = useState("login"); // "login" | "signup"
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState("operator");
  const [company, setCompany] = useState("");
  const [idPerusahaan, setIdPerusahaan] = useState("");
  const [companies, setCompanies] = useState([]);
  const [loadingCompanies, setLoadingCompanies] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (typeof getPerusahaanList === "function") {
      setLoadingCompanies(true);
      getPerusahaanList()
        .then((list) => {
          if (list && list.length) {
            setCompanies(list);
            const first = list[0];
            if (typeof first === "string") {
              setCompany((prev) => prev || first);
              setIdPerusahaan((prev) => prev || first);
            } else {
              setCompany((prev) => prev || (first.nama_perusahaan || first.name || first.id_perusahaan || first.id || ""));
              setIdPerusahaan((prev) => prev || (first.id_perusahaan || first.id || first.nama_perusahaan || first.name || ""));
            }
          }
        })
        .catch((err) => console.warn("Failed loading companies:", err))
        .finally(() => setLoadingCompanies(false));
    }
  }, []);

  function handleCompanyChange(e) {
    const selectedVal = e.target.value;
    setIdPerusahaan(selectedVal);
    const matched = companies.find((c) => {
      if (typeof c === "string") return c === selectedVal;
      return (c.id_perusahaan === selectedVal || c.id === selectedVal || c.nama_perusahaan === selectedVal || c.name === selectedVal);
    });
    if (matched) {
      if (typeof matched === "string") {
        setCompany(matched);
        setIdPerusahaan(matched);
      } else {
        setCompany(matched.nama_perusahaan || matched.name || selectedVal);
        setIdPerusahaan(matched.id_perusahaan || matched.id || selectedVal);
      }
    } else {
      setCompany(selectedVal);
    }
  }

  function quickFill(user, pass) {
    setUsername(user);
    setPassword(pass);
    setError("");
  }

  async function handleSubmit(e) {
    if (e) e.preventDefault();
    setError("");

    const trimmedUser = username.trim();
    if (!trimmedUser) {
      setError("Username or email is required");
      return;
    }
    if (!password) {
      setError("Password is required");
      return;
    }
    if (mode === "signup") {
      const trimmedEmail = email.trim();
      if (!trimmedEmail) {
        setError("Email address is required");
        return;
      }
      if (!trimmedEmail.includes("@") || !trimmedEmail.includes(".")) {
        setError("Please enter a valid email address");
        return;
      }
      if (password.length < 6) {
        setError("Password must be at least 6 characters long");
        return;
      }
    }

    setLoading(true);
    try {
      if (mode === "login") {
        const res = await loginUser({ username: trimmedUser, password });
        showToast(`Welcome back, ${res.user?.username || trimmedUser}!`, "default");
        onAuthSuccess(res.user);
      } else {
        const res = await registerUser({
          username: trimmedUser,
          email: email.trim(),
          password,
          role,
          company: company.trim(),
          id_perusahaan: Number(idPerusahaan) || 0,
        });
        showToast("Account created and signed in successfully!", "default");
        onAuthSuccess(res.user);
      }
    } catch (err) {
      setError(err.message || "Authentication failed. Please check your credentials.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-wrapper">
      <div className="auth-glow-1" />
      <div className="auth-glow-2" />

      <div className="auth-card">
        <div className="auth-header">
          <div className="auth-logo">
            <i className="ph ph-shield-check" />
          </div>
          <h2>QTERA Platform</h2>
          <p>{mode === "login" ? "Sign in with your JWT credentials to manage assets" : "Create a new account with role and company context"}</p>
        </div>

        <div className="auth-tabs">
          <button
            type="button"
            className={"auth-tab" + (mode === "login" ? " active" : "")}
            onClick={() => { setMode("login"); setError(""); }}
          >
            <i className="ph ph-sign-in" /> Sign In
          </button>
          <button
            type="button"
            className={"auth-tab" + (mode === "signup" ? " active" : "")}
            onClick={() => { setMode("signup"); setError(""); }}
          >
            <i className="ph ph-user-plus" /> Sign Up
          </button>
        </div>

        {error && (
          <div className="auth-error-banner">
            <i className="ph ph-warning-circle" style={{ fontSize: "16px", flexShrink: 0 }} />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="auth-form">
          <div className="form-row">
            <label>
              {mode === "login" ? "Username or Email" : "Username"} <span className="req">*</span>
            </label>
            <div className="input-icon-wrap">
              <i className="ph ph-user input-icon" />
              <input
                type="text"
                autoFocus
                placeholder={mode === "login" ? "Enter username or email" : "Choose username (e.g. operator1)"}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={loading}
                className="input-with-icon"
              />
            </div>
          </div>

          {mode === "signup" && (
            <div className="form-row">
              <label>Email Address <span className="req">*</span></label>
              <div className="input-icon-wrap">
                <i className="ph ph-envelope input-icon" />
                <input
                  type="email"
                  placeholder="name@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  disabled={loading}
                  className="input-with-icon"
                />
              </div>
            </div>
          )}

          <div className="form-row">
            <label>Password <span className="req">*</span></label>
            <div className="input-icon-wrap">
              <i className="ph ph-lock-key input-icon" />
              <input
                type={showPassword ? "text" : "password"}
                placeholder={mode === "signup" ? "At least 6 characters" : "••••••••"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={loading}
                className="input-with-icon input-with-toggle"
              />
              <button
                type="button"
                className="password-toggle-btn"
                onClick={() => setShowPassword((v) => !v)}
                title={showPassword ? "Hide password" : "Show password"}
                tabIndex={-1}
              >
                <i className={"ph " + (showPassword ? "ph-eye-slash" : "ph-eye")} />
              </button>
            </div>
          </div>

          {mode === "signup" && (
            <>
              <div className="form-row">
                <label>Company / Perusahaan <span className="req">*</span></label>
                <div className="input-icon-wrap">
                  <i className="ph ph-buildings input-icon" />
                  <select
                    value={idPerusahaan || company}
                    onChange={handleCompanyChange}
                    disabled={loading || loadingCompanies}
                    className="input-with-icon"
                  >
                    {companies.length === 0 && <option value="">Loading companies...</option>}
                    {companies.map((c) => {
                      const idVal = typeof c === "string" ? c : (c.id_perusahaan || c.id || c.nama_perusahaan || c.name);
                      const label = typeof c === "string" ? c : (c.nama_perusahaan || c.name || c.id_perusahaan || c.id);
                      return (
                        <option key={idVal} value={idVal}>
                          {label}
                        </option>
                      );
                    })}
                  </select>
                </div>
              </div>

              <div className="form-row">
                <label>Assigned Role</label>
                <div className="input-icon-wrap">
                  <i className="ph ph-identification-badge input-icon" />
                  <select
                    value={role}
                    onChange={(e) => setRole(e.target.value)}
                    disabled={loading}
                    className="input-with-icon"
                  >
                    <option value="operator">Operator (Manage &amp; Edit Assets)</option>
                    <option value="admin">Administrator (Full Access &amp; Delete)</option>
                    <option value="viewer">Viewer (Read-Only Access)</option>
                  </select>
                </div>
              </div>
            </>
          )}

          <button type="submit" className="btn btn-primary btn-full btn-auth-submit" disabled={loading}>
            {loading ? (
              <>
                <Spinner /> {mode === "login" ? "Authenticating..." : "Creating Account..."}
              </>
            ) : mode === "login" ? (
              <>
                <i className="ph ph-sign-in" /> Sign In
              </>
            ) : (
              <>
                <i className="ph ph-user-plus" /> Create Account
              </>
            )}
          </button>
        </form>

        {mode === "login" && (
          <div className="auth-quick-fill-section">
            <span className="auth-quick-fill-title">Quick Demo Login:</span>
            <div className="auth-quick-fill-chips">
              <button
                type="button"
                className="chip-quick-fill"
                onClick={() => quickFill("admin", "admin123")}
              >
                <i className="ph ph-shield-check" /> admin / admin123
              </button>
              <button
                type="button"
                className="chip-quick-fill"
                onClick={() => quickFill("operator", "operator123")}
              >
                <i className="ph ph-user" /> operator / operator123
              </button>
            </div>
          </div>
        )}

        <div className="auth-card-footer">
          {mode === "login" ? (
            <p>
              Don't have an account yet?{" "}
              <button
                type="button"
                className="link-btn"
                onClick={() => { setMode("signup"); setError(""); }}
              >
                Sign Up
              </button>
            </p>
          ) : (
            <p>
              Already have an account?{" "}
              <button
                type="button"
                className="link-btn"
                onClick={() => { setMode("login"); setError(""); }}
              >
                Sign In
              </button>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------
// Asset List Page
// ---------------------------------------------------------------------

function AssetListPage({ onRegister, onEdit, onChat, showToast, reloadKey }) {
  const [loading, setLoading] = useState(true);
  const [rows, setRows] = useState([]);
  const [pagination, setPagination] = useState({ page: 1, pageSize: 10, totalItems: 0, totalPages: 1 });
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("createdAt");
  const [order, setOrder] = useState("desc");
  const [filterOpen, setFilterOpen] = useState(false);
  const [category, setCategory] = useState([]);
  const [brand, setBrand] = useState([]);
  const [location, setLocation] = useState([]);
  const [colVisibility, setColVisibility] = useState({});
  const [colMenuOpen, setColMenuOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleting, setDeleting] = useState(false);

  const [options, setOptions] = useState({ categories: [], locations: [], brands: [] });

  const loadOptions = useCallback(() => {
    if (typeof getAssetOptions === "function") {
      getAssetOptions().then((res) => {
        if (res && res.categories) setOptions(res);
      }).catch((e) => console.warn("Failed to load options:", e));
    }
  }, []);

  useEffect(() => {
    loadOptions();
  }, [loadOptions, reloadKey]);

  const allColumns = ["assetId", "name", "category", "brand", "modelType", "purchaseDate", "purchasePrice", "location", "createdAt"];
  const nonHideable = ["assetId", "name"];
  const columnLabels = {
    assetId: "Asset ID", name: "Name", category: "Category", brand: "Brand", modelType: "Model/Type",
    purchaseDate: "Purchase Date", purchasePrice: "Purchase Price", location: "Location", createdAt: "Created At",
  };

  const load = useCallback((page = pagination.page) => {
    setLoading(true);
    getAssets({ search, category, brand, location, sort, order, page, pageSize: pagination.pageSize }).then((res) => {
      setRows(res.assets || []);
      setPagination(res.pagination || { page: 1, pageSize: 10, totalItems: 0, totalPages: 1 });
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [search, category, brand, location, sort, order, pagination.pageSize]);

  useEffect(() => { load(1); }, [search, category, brand, location, sort, order, pagination.pageSize, reloadKey]);

  function toggleSort(col) {
    if (sort === col) setOrder(order === "asc" ? "desc" : "asc");
    else { setSort(col); setOrder("asc"); }
  }

  function toggleMultiSelect(list, setList, value) {
    setList(list.includes(value) ? list.filter((v) => v !== value) : [...list, value]);
  }

  async function handleDownload() {
    const res = await getAssets({ search, category, brand, location, sort, order, pageSize: "all" });
    downloadAssetsCSV(res.assets || []);
  }

  async function handleDeleteConfirm() {
    setDeleting(true);
    await deleteAsset(deleteTarget.assetId || deleteTarget.id);
    setDeleting(false);
    setDeleteTarget(null);
    showToast("Asset deleted successfully.");
    load(1);
    loadOptions();
  }

  const isColVisible = (c) => nonHideable.includes(c) || colVisibility[c] !== false;
  const activeFilterCount = category.length + brand.length + location.length;

  const dynamicCategories = options.categories.length ? options.categories : [...new Set(rows.map((a) => a.category).filter(Boolean))];
  const dynamicLocations = options.locations.length ? options.locations : [...new Set(rows.map((a) => a.location).filter(Boolean))];
  const dynamicBrands = options.brands.length ? options.brands : [...new Set(rows.map((a) => a.brand).filter(Boolean))];

  return (
    <div className="page">
      <div className="page-header">
        <h1>Assets Inventory</h1>
        <div className="page-header-actions">
          <button className="btn btn-chat-ai" onClick={onChat} title="Ask QTERA AI">
            <i className="ph ph-chat-teardrop-dots" /> Chat AI
          </button>
          <button className="btn btn-primary" onClick={onRegister}>
            <i className="ph ph-plus" /> Register Asset
          </button>
        </div>
      </div>

      <div className="toolbar">
        <div className="search-box">
          <i className="ph ph-magnifying-glass" />
          <input
            placeholder="Search by name, asset ID, category, brand..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="toolbar-actions">
          <div className="dropdown-wrap">
            <button className={"icon-btn" + (activeFilterCount ? " active" : "")} onClick={() => setFilterOpen((v) => !v)} title="Filter">
              <i className="ph ph-funnel" />
              {activeFilterCount > 0 && <span className="count-badge">{activeFilterCount}</span>}
            </button>
            {filterOpen && (
              <div className="dropdown-panel filter-panel">
                <FilterGroup label="Category" options={dynamicCategories} selected={category} onToggle={(v) => toggleMultiSelect(category, setCategory, v)} />
                <FilterGroup label="Brand" options={dynamicBrands} selected={brand} onToggle={(v) => toggleMultiSelect(brand, setBrand, v)} />
                <FilterGroup label="Location" options={dynamicLocations} selected={location} onToggle={(v) => toggleMultiSelect(location, setLocation, v)} />
                <button className="btn btn-ghost btn-sm" onClick={() => { setCategory([]); setBrand([]); setLocation([]); }}>Clear all</button>
              </div>
            )}
          </div>
          <button className="icon-btn" onClick={handleDownload} title="Download CSV">
            <i className="ph ph-download-simple" />
          </button>
          <div className="dropdown-wrap">
            <button className="icon-btn" onClick={() => setColMenuOpen((v) => !v)} title="Column visibility">
              <i className="ph ph-dots-three-outline-vertical" />
            </button>
            {colMenuOpen && (
              <div className="dropdown-panel">
                {allColumns.map((c) => (
                  <label key={c} className={"col-toggle" + (nonHideable.includes(c) ? " disabled" : "")}>
                    <input
                      type="checkbox"
                      checked={isColVisible(c)}
                      disabled={nonHideable.includes(c)}
                      onChange={() => setColVisibility((cv) => ({ ...cv, [c]: cv[c] === false ? true : false }))}
                    />
                    {columnLabels[c]}
                  </label>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              {allColumns.filter(isColVisible).map((c) => (
                <th key={c} onClick={() => toggleSort(c)} className="sortable">
                  {columnLabels[c]}
                  {sort === c && <i className={"ph " + (order === "asc" ? "ph-caret-up" : "ph-caret-down")} />}
                </th>
              ))}
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {loading &&
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="skeleton-row">
                  {allColumns.filter(isColVisible).map((c) => <td key={c}><div className="skeleton-cell" /></td>)}
                  <td><div className="skeleton-cell" /></td>
                </tr>
              ))}
            {!loading && rows.map((a) => (
              <tr key={a.assetId || a.id}>
                {isColVisible("assetId") && <td className="mono">{a.assetId || a.id}</td>}
                {isColVisible("name") && <td className="strong">{a.name}</td>}
                {isColVisible("category") && <td>{a.category || "—"}</td>}
                {isColVisible("brand") && <td>{a.brand || "—"}</td>}
                {isColVisible("modelType") && <td>{a.modelType || "—"}</td>}
                {isColVisible("purchaseDate") && <td>{formatDate(a.purchaseDate) || "—"}</td>}
                {isColVisible("purchasePrice") && <td>{formatCurrency(a.purchasePrice) || "—"}</td>}
                {isColVisible("location") && <td>{a.location || "—"}</td>}
                {isColVisible("createdAt") && <td className="muted">{formatDate(a.createdAt)}</td>}
                <td className="row-actions">
                  <button className="icon-btn-sm" title="Edit" onClick={() => onEdit(a)}><i className="ph ph-pencil-simple" /></button>
                  <button className="icon-btn-sm danger" title="Delete" onClick={() => setDeleteTarget(a)}><i className="ph ph-trash" /></button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && rows.length === 0 && pagination.totalItems === 0 && !search && activeFilterCount === 0 && (
          <EmptyState icon="ph-package" title="No assets registered yet" cta={<button className="btn btn-primary" onClick={onRegister}>+ Register Asset</button>} />
        )}
        {!loading && rows.length === 0 && (search || activeFilterCount > 0) && (
          <EmptyState icon="ph-magnifying-glass" title="No matching assets found" />
        )}
      </div>

      {!loading && rows.length > 0 && (
        <div className="pagination-bar">
          <span className="muted">{pagination.totalItems} asset(s)</span>
          <div className="pagination-controls">
            <select value={pagination.pageSize} onChange={(e) => setPagination((p) => ({ ...p, pageSize: e.target.value === "all" ? "all" : Number(e.target.value) }))}>
              {[10, 25, 50, 100].map((n) => <option key={n} value={n}>{n} / page</option>)}
              <option value="all">All</option>
            </select>
            <button className="icon-btn-sm" disabled={pagination.page <= 1} onClick={() => load(pagination.page - 1)}><i className="ph ph-caret-left" /></button>
            <span>Page {pagination.page} of {pagination.totalPages}</span>
            <button className="icon-btn-sm" disabled={pagination.page >= pagination.totalPages} onClick={() => load(pagination.page + 1)}><i className="ph ph-caret-right" /></button>
          </div>
        </div>
      )}

      {deleteTarget && (
        <ConfirmDialog
          title="Delete Asset"
          body={`Are you sure you want to delete ${deleteTarget.name}? This action cannot be undone.`}
          confirmLabel="Delete"
          danger
          loading={deleting}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={handleDeleteConfirm}
        />
      )}
    </div>
  );
}

function FilterGroup({ label, options, selected, onToggle }) {
  return (
    <div className="filter-group">
      <div className="filter-group-label">{label}</div>
      {options.map((o) => (
        <label key={o} className="col-toggle">
          <input type="checkbox" checked={selected.includes(o)} onChange={() => onToggle(o)} />
          {o}
        </label>
      ))}
    </div>
  );
}

function ConfirmDialog({ title, body, confirmLabel, danger, loading, onCancel, onConfirm }) {
  return (
    <div className="modal-overlay">
      <div className="modal modal-sm">
        <h3>{title}</h3>
        <p>{body}</p>
        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel} disabled={loading}>Cancel</button>
          <button className={"btn " + (danger ? "btn-danger" : "btn-primary")} onClick={onConfirm} disabled={loading}>
            {loading ? <Spinner /> : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------
// Asset Form
// ---------------------------------------------------------------------

function AssetForm({ asset, onDone, onCancel, showToast }) {
  const isEdit = !!asset;
  const [form, setForm] = useState({
    name: asset?.name || "",
    category: asset?.category || "",
    brand: asset?.brand || "",
    modelType: asset?.modelType || "",
    purchaseDate: asset?.purchaseDate || "",
    purchasePrice: asset?.purchasePrice || "",
    location: asset?.location || "",
  });
  const [errors, setErrors] = useState({});
  const [saving, setSaving] = useState(false);
  const [backendOptions, setBackendOptions] = useState({ categories: [], locations: [], brands: [] });

  useEffect(() => {
    if (typeof getAssetOptions === "function") {
      getAssetOptions().then((res) => {
        if (res) setBackendOptions(res);
      }).catch(() => { });
    }
  }, []);

  function set(key, value) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  function validate() {
    const e = {};
    if (!form.name.trim()) e.name = "Asset Name must not be empty";
    if (!form.category.trim()) e.category = "Category must not be empty";
    if (!form.location.trim()) e.location = "Location must not be empty";
    setErrors(e);
    return Object.keys(e).length === 0;
  }

  async function handleSubmit() {
    if (!validate()) return;
    setSaving(true);
    const payload = {
      name: form.name.trim(),
      category: form.category.trim(),
      brand: form.brand.trim() || null,
      modelType: form.modelType.trim() || null,
      purchaseDate: form.purchaseDate || null,
      purchasePrice: form.purchasePrice ? String(form.purchasePrice) : null,
      location: form.location.trim(),
    };
    try {
      if (isEdit) {
        await updateAsset(asset.assetId || asset.id, payload);
        showToast("Asset updated successfully.");
      } else {
        await createAssetManual(payload);
        showToast("Asset registered successfully.");
      }
      onDone();
    } catch (err) {
      showToast(err.message || "Failed to save asset", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page page-narrow">
      <div className="page-header">
        <button className="link-back" onClick={onCancel}><i className="ph ph-arrow-left" /> Back</button>
      </div>
      <h1>{isEdit ? "Edit Asset" : "Register Asset"}</h1>

      <div className="form-card">
        {isEdit && (
          <FormRow label="Asset ID">
            <input value={asset.assetId || asset.id} disabled className="readonly" />
          </FormRow>
        )}
        <FormRow label="Asset Name" required error={errors.name}>
          <input value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="e.g. Dell Latitude 5450" />
        </FormRow>
        <FormRow label="Category" required error={errors.category}>
          <input list="category-options" value={form.category} onChange={(e) => set("category", e.target.value)} placeholder="e.g. IT Equipment" />
          <datalist id="category-options">
            {(backendOptions.categories || []).map((c) => <option key={c} value={c} />)}
          </datalist>
        </FormRow>
        <FormRow label="Brand">
          <input list="brand-options" value={form.brand} onChange={(e) => set("brand", e.target.value)} placeholder="e.g. Dell" />
          <datalist id="brand-options">
            {(backendOptions.brands || []).map((b) => <option key={b} value={b} />)}
          </datalist>
        </FormRow>
        <FormRow label="Model/Type">
          <input value={form.modelType} onChange={(e) => set("modelType", e.target.value)} placeholder="e.g. Latitude 5450" />
        </FormRow>
        <FormRow label="Purchase Date">
          <input type="date" value={form.purchaseDate} onChange={(e) => set("purchaseDate", e.target.value)} />
        </FormRow>
        <FormRow label="Purchase Price">
          <input type="number" value={form.purchasePrice} onChange={(e) => set("purchasePrice", e.target.value)} placeholder="e.g. 18500000" />
        </FormRow>
        <FormRow label="Location" required error={errors.location}>
          <input list="location-options" value={form.location} onChange={(e) => set("location", e.target.value)} placeholder="e.g. HQ Office" />
          <datalist id="location-options">
            {(backendOptions.locations || []).map((l) => <option key={l} value={l} />)}
          </datalist>
        </FormRow>
      </div>

      <div className="form-footer">
        <button className="btn btn-ghost" onClick={onCancel} disabled={saving}>Cancel</button>
        <button className="btn btn-primary" onClick={handleSubmit} disabled={saving}>
          {saving ? <Spinner /> : isEdit ? "Save Changes" : "Create Asset"}
        </button>
      </div>
    </div>
  );
}

function FormRow({ label, required, error, children }) {
  return (
    <div className={"form-row" + (error ? " has-error" : "")}>
      <label>{label} {required && <span className="req">*</span>}</label>
      {children}
      {error && <div className="field-error">{error}</div>}
    </div>
  );
}

// ---------------------------------------------------------------------
// Registration Flow
// ---------------------------------------------------------------------

function RegistrationFlow({ onDone, onManual, showToast }) {
  return (
    <div className="modal-overlay">
      <div className="modal">
        <h3>Register Asset</h3>
        <p className="muted">How do you want to add this asset?</p>
        <div className="method-cards">
          <button className="method-card" onClick={onManual}>
            <i className="ph ph-note-pencil" />
            <div><strong>Manual Form</strong><span>Fill the standard form directly</span></div>
          </button>
        </div>
        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onDone}>Cancel</button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------
// App Shell
// ---------------------------------------------------------------------

function App() {
  const [user, setUser] = useState(getStoredUser());
  const [page, setPage] = useState("list"); // list | register | form | chat
  const [editingAsset, setEditingAsset] = useState(null);
  const [reloadKey, setReloadKey] = useState(0);
  const { toasts, push } = useToasts();

  useEffect(() => {
    const token = getAuthToken();
    if (token) {
      getMe().then((res) => {
        if (res && res.user) setUser(res.user);
      }).catch(() => {
        // Token invalid
        logoutUser();
        setUser(null);
      });
    }

    function handleUnauthorized() {
      logoutUser();
      setUser(null);
      push("Session expired. Please sign in again.", "warning");
    }

    window.addEventListener("qtera:unauthorized", handleUnauthorized);
    return () => window.removeEventListener("qtera:unauthorized", handleUnauthorized);
  }, [push]);

  function handleLogout() {
    logoutUser();
    setUser(null);
    push("Logged out successfully.");
  }

  function goList() {
    setPage("list");
    setEditingAsset(null);
    setReloadKey((k) => k + 1);
  }

  if (!user) {
    return (
      <div className="app-shell">
        <AuthPage onAuthSuccess={(u) => { setUser(u); goList(); }} showToast={push} />
        <ToastHost toasts={toasts} />
      </div>
    );
  }

  return (
    <div className="app-shell">
      <TopNavBar user={user} onLogout={handleLogout} onGoList={goList} page={page} />

      {page === "list" && (
        <AssetListPage
          onRegister={() => setPage("register")}
          onEdit={(asset) => { setEditingAsset(asset); setPage("form"); }}
          onChat={() => setPage("chat")}
          showToast={push}
          reloadKey={reloadKey}
        />
      )}
      {page === "chat" && (
        <iframe
          src="/chat"
          className="chat-inline-frame"
          title="QTERA Security AI Chat"
          allow="microphone"
        />
      )}
      {page === "register" && (
        <RegistrationFlow
          onDone={goList}
          onManual={() => { setEditingAsset(null); setPage("form"); }}
          showToast={push}
        />
      )}
      {page === "form" && (
        <AssetForm asset={editingAsset} onDone={goList} onCancel={goList} showToast={push} />
      )}

      <ToastHost toasts={toasts} />
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
