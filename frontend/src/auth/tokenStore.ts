const TOKEN_KEY = "plover.access_token";

function attempt<T>(action: () => T, fallback: T): T {
  try {
    return action();
  } catch {
    return fallback;
  }
}

export function readToken(): string | null {
  return attempt(() => localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY), null);
}

export function clearToken(): void {
  attempt(() => {
    localStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(TOKEN_KEY);
  }, undefined);
}

export function writeToken(token: string, remember: boolean): void {
  clearToken();
  attempt(() => {
    (remember ? localStorage : sessionStorage).setItem(TOKEN_KEY, token);
  }, undefined);
}
