// Thin API wrapper used by Chat.jsx
// Provides apiFetch() with auto Bearer token injection and getToken()

export function getToken() {
  try {
    return localStorage.getItem('qtera_token') || '';
  } catch {
    return '';
  }
}

/**
 * Fetch wrapper that automatically adds Authorization header if a token exists.
 * @param {string} path - relative or absolute URL
 * @param {RequestInit} options - standard fetch options
 */
export async function apiFetch(path, options = {}) {
  const headers = {
    ...(options.headers || {}),
  };
  try {
    const u = localStorage.getItem('qtera_user');
    if (u) {
      const user = JSON.parse(u);
      if (user.id) headers['X-User-ID'] = String(user.id);
      if (user.role) headers['X-User-Role'] = user.role;
      if (user.username) headers['X-User-Name'] = user.username;
      if (user.company) headers['X-Company'] = user.company;
      if (user.id_perusahaan) headers['X-Company-ID'] = String(user.id_perusahaan);
    }
  } catch {}
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return fetch(path, { ...options, headers });
}
