"use client";

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  useRef,
} from "react";
import { useRouter } from "next/navigation";
import { authApi, setAccessToken } from "@/lib/api";
import type { User } from "@/types";

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  loginCandidate: (email: string, password: string) => Promise<User>;
  loginCompany: (email: string, password: string) => Promise<User>;
  registerCandidate: (input: {
    email: string;
    password: string;
    full_name: string;
    phone?: string;
  }) => Promise<void>;
  registerCompany: (input: {
    email: string;
    password: string;
    company_name: string;
    website?: string;
  }) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();
  const authVersion = useRef(0);

  const hydrate = useCallback(async () => {
    const versionAtStart = authVersion.current;
    const ok = await authApi.refresh();
    if (versionAtStart !== authVersion.current) {
      setLoading(false);
      return;
    }
    if (ok) {
      try {
        const me = await authApi.me();
        if (versionAtStart === authVersion.current) setUser(me);
      } catch {
        if (versionAtStart === authVersion.current) setUser(null);
      }
    } else {
      setUser(null);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    hydrate();
  }, [hydrate]);

  const loginCandidate = async (email: string, password: string) => {
    authVersion.current += 1;
    const res = await authApi.login({ email, password });
    setAccessToken(res.access_token);
    setUser(res.user);
    setLoading(false);
    return res.user;
  };
  const loginCompany = loginCandidate; // same endpoint; role comes back on the user object

  const registerCandidate: AuthContextValue["registerCandidate"] = async (
    input,
  ) => {
    authVersion.current += 1;
    const res = await authApi.registerCandidate(input);
    setAccessToken(res.access_token);
    setUser(res.user);
    setLoading(false);
  };

  const registerCompany: AuthContextValue["registerCompany"] = async (
    input,
  ) => {
    authVersion.current += 1;
    const res = await authApi.registerCompany(input);
    setAccessToken(res.access_token);
    setUser(res.user);
    setLoading(false);
  };

  const logout = async () => {
    authVersion.current += 1;
    await authApi.logout().catch(() => {});
    setAccessToken(null);
    setUser(null);
    router.push("/");
  };

  const refreshUser = async () => {
    const me = await authApi.me();
    setUser(me);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        loginCandidate,
        loginCompany,
        registerCandidate,
        registerCompany,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
