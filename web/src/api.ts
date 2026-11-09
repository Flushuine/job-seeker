export async function api<T>(path: string, options: RequestInit = {}): Promise<T> { const res = await fetch('/api' + path, { ...options, headers: { ...(options.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }), Authorization: 'Bearer ' + (sessionStorage.getItem('hunter-token') || ''), ...options.headers } }); let body; try {
    body = await res.json();
}
catch {
    throw new Error('The API is unavailable. Check that the backend is running.');
} if (!res.ok)
    throw new Error(body.error?.message || 'Request failed'); return body as T; }
export const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) });
