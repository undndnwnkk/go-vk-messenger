import { createContext } from "react";

import type { Credentials, User } from "../api/types";

export interface AuthContextValue {
  token: string | null;
  isAuthenticated: boolean;
  user: User | null;
  login: (credentials: Credentials, remember: boolean) => Promise<void>;
  register: (credentials: Credentials, remember: boolean) => Promise<void>;
  logout: () => void;
}

export const AuthContext = createContext<AuthContextValue | null>(null);
