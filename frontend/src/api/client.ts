import createClient, { type Middleware } from "openapi-fetch";

import { readToken } from "../auth/tokenStore";
import { ApiError, NETWORK_ERROR, toApiError } from "./errors";
import type { paths } from "./schema.gen";

export const API_URL = (import.meta.env.VITE_API_URL ?? "http://localhost:8080").replace(
  /\/+$/,
  "",
);

type UnauthorizedListener = () => void;

const unauthorizedListeners = new Set<UnauthorizedListener>();

export function subscribeUnauthorized(listener: UnauthorizedListener): () => void {
  unauthorizedListeners.add(listener);
  return () => {
    unauthorizedListeners.delete(listener);
  };
}

const authMiddleware: Middleware = {
  onRequest({ request }) {
    const token = readToken();
    if (token !== null) {
      request.headers.set("Authorization", `Bearer ${token}`);
    }
    return request;
  },
  onResponse({ request, response }) {
    if (response.status === 401 && request.headers.has("Authorization")) {
      for (const listener of unauthorizedListeners) {
        listener();
      }
    }
    return undefined;
  },
};

export const api = createClient<paths>({
  baseUrl: API_URL,
  fetch: (request) => globalThis.fetch(request),
});

api.use(authMiddleware);

type ClientResult<T> =
  | { data: T; error?: never; response: Response }
  | { data?: never; error: unknown; response: Response };

export async function unwrap<T>(request: Promise<ClientResult<T>>): Promise<T> {
  let result: ClientResult<T>;
  try {
    result = await request;
  } catch (cause) {
    throw new ApiError(0, NETWORK_ERROR, "Network request failed", { cause });
  }
  if (!result.response.ok) {
    throw toApiError(result.response.status, result.error);
  }
  return result.data as T;
}
