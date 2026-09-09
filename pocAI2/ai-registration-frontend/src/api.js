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
  const token = getToken();
  const headers = {
    ...(options.headers || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return fetch(path, { ...options, headers });
}
