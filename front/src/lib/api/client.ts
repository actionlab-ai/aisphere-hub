export type ApiResult<T> = { code?: number; message?: string; data?: T; error?: string } | T;

const TOKEN_KEY = 'aihub_console_token';
const REFRESH_KEY = 'aihub_console_refresh';

// AI Sphere integration toggle. When true the client will, on a 401
// without a local token, redirect to /v3/auth/aisphere/login so the
// platform can re-establish the session cookie. Set via
// NEXT_PUBLIC_AISPHERE_AUTH_ENABLED in the frontend env.
const AISPHERE_AUTH_ENABLED =
  process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === 'true' ||
  process.env.NEXT_PUBLIC_AISPHERE_AUTH_ENABLED === '1';

export function getToken(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(TOKEN_KEY) || '';
}

export function setTokens(accessToken: string, refreshToken?: string) {
  if (typeof window === 'undefined') return;
  localStorage.setItem(TOKEN_KEY, accessToken);
  if (refreshToken) localStorage.setItem(REFRESH_KEY, refreshToken);
}

export function clearTokens() {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export function getAccessSpace(): string {
  if (typeof window === 'undefined') return 'default';
  return localStorage.getItem('aihub_access_space') || 'default';
}

export function setAccessSpace(id: string) {
  if (typeof window === 'undefined') return;
  localStorage.setItem('aihub_access_space', id);
}

// redirectToAISphereLogin bounces the user to aisphere-auth via the
// SkillHub-side /v3/auth/aisphere/login redirect. We capture the current
// URL so aisphere-auth can return the user here after re-authenticating.
export function redirectToAISphereLogin() {
  if (typeof window === 'undefined') return;
  const here = window.location.origin + window.location.pathname + window.location.search;
  window.location.href = `/v3/auth/aisphere/login?redirect=${encodeURIComponent(here)}`;
}

export async function request<T>(url: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers || []);
  const token = getToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const res = await fetch(url, { ...init, headers });
  const contentType = res.headers.get('content-type') || '';

  if (!res.ok) {
    // 401 path: if we have no local bearer token AND the AI Sphere
    // integration is enabled, the session cookie is the only identity
    // we can rely on. Redirect through SkillHub to aisphere-auth so
    // the cookie gets refreshed, then come back here. Existing
    // token-based callers keep the legacy "throw and let the caller
    // show login" behavior.
    if (res.status === 401 && !token && AISPHERE_AUTH_ENABLED) {
      redirectToAISphereLogin();
      // Returning a never-resolving promise so callers don't try to
      // render an error toast while the redirect is in flight.
      return new Promise<T>(() => {});
    }
    let msg = `${res.status} ${res.statusText}`;
    try {
      if (contentType.includes('json')) {
        const j = await res.json();
        msg = j.message || j.error || msg;
      } else if (contentType.includes('text/html')) {
        // Don't try to parse HTML error responses - use status code
        msg = `API unavailable: ${res.status} ${res.statusText}`;
      } else {
        const text = await res.text();
        if (text.length < 200) msg = text;
      }
    } catch {
      /* ignore parse errors */
    }
    throw new Error(msg);
  }

  if (contentType.includes('application/zip') || contentType.includes('octet-stream')) {
    return (await res.blob()) as T;
  }

  if (!contentType.includes('json')) {
    // Some backends / proxies don't set Content-Type: application/json.
    // Try to parse as JSON first; if that fails, fall back to text.
    const text = await res.text();
    if (text && (text.trim().startsWith('{') || text.trim().startsWith('['))) {
      try {
        const json = JSON.parse(text);
        if (json && typeof json === 'object' && 'code' in json) {
          if (json.code !== 0) throw new Error(json.message || 'request failed');
          return json.data as T;
        }
        return (json.data ?? json) as T;
      } catch {
        // not valid JSON despite looking like it — return as text
      }
    }
    return text as T;
  }

  const json = await res.json();
  if (json && typeof json === 'object' && 'code' in json) {
    if (json.code !== 0) throw new Error(json.message || 'request failed');
    return json.data as T;
  }
  return (json.data ?? json) as T;
}

export function toQuery(params: Record<string, unknown>): string {
  const q = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
  });
  return q.toString();
}

export function asItems<T>(page: unknown): T[] {
  if (!page) return [];
  const p = page as Record<string, unknown>;
  return (
    (p.items as T[]) ||
    (p.pageItems as T[]) ||
    (p.list as T[]) ||
    (p.data as T[]) ||
    (Array.isArray(page) ? page : [])
  );
}
