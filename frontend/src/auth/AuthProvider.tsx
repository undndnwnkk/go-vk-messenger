import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { api, subscribeUnauthorized, unwrap } from "../api/client";
import { meQuery } from "../api/queries";
import type { Credentials, TokenResponse } from "../api/types";
import { AuthContext, type AuthContextValue } from "./authContext";
import { clearToken, readToken, writeToken } from "./tokenStore";

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [token, setToken] = useState<string | null>(readToken);

  const logout = useCallback(() => {
    clearToken();
    setToken(null);
    queryClient.clear();
  }, [queryClient]);

  useEffect(() => subscribeUnauthorized(logout), [logout]);

  const startSession = useCallback(
    (response: TokenResponse, remember: boolean) => {
      queryClient.clear();
      writeToken(response.access_token, remember);
      setToken(response.access_token);
    },
    [queryClient],
  );

  const login = useCallback(
    async (credentials: Credentials, remember: boolean) => {
      const response = await unwrap(api.POST("/api/v1/auth/login", { body: credentials }));
      startSession(response, remember);
    },
    [startSession],
  );

  const register = useCallback(
    async (credentials: Credentials, remember: boolean) => {
      const response = await unwrap(api.POST("/api/v1/auth/register", { body: credentials }));
      startSession(response, remember);
    },
    [startSession],
  );

  const { data: user } = useQuery({ ...meQuery, enabled: token !== null });

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      isAuthenticated: token !== null,
      user: token === null ? null : (user ?? null),
      login,
      register,
      logout,
    }),
    [token, user, login, register, logout],
  );

  return <AuthContext value={value}>{children}</AuthContext>;
}
