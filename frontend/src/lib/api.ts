const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

// fetch wrapper that attaches the access token, and on 401 tries once to refresh
// and retry; if that fails it clears tokens and sends the user back to /login.
async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (res.status === 401) {
    const refreshed = await tryRefreshToken();
    if (refreshed) {
      headers["Authorization"] = `Bearer ${localStorage.getItem("access_token")}`;
      const retryRes = await fetch(`${BASE_URL}${endpoint}`, {
        ...options,
        headers,
      });
      if (!retryRes.ok) {
        const err = await retryRes.json().catch(() => ({ error: "request failed" }));
        throw new Error(err.error || "request failed");
      }
      return retryRes.json();
    }

    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    window.location.href = "/login";
    throw new Error("unauthorized");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "request failed" }));
    throw new Error(err.error || "request failed");
  }

  return res.json();
}

// swaps the stored refresh token for a new access/refresh pair; returns false if it can't.
async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = localStorage.getItem("refresh_token");
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = await res.json();
    localStorage.setItem("access_token", data.data.access_token);
    localStorage.setItem("refresh_token", data.data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

export const api = {
  get: <T>(endpoint: string) => request<T>(endpoint),
  post: <T>(endpoint: string, body: unknown) =>
    request<T>(endpoint, { method: "POST", body: JSON.stringify(body) }),
  put: <T>(endpoint: string, body: unknown) =>
    request<T>(endpoint, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: "DELETE" }),
};
