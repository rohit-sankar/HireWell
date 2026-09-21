"use client";

import { createContext, useContext, useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { authApi, setAccessToken } from "@/lib/api";
import type { User } from "@/types";

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  loginCandidate: (email: string, password: string) => Promise<User>;
  loginCompany: (email: string, password: string) => Promise<User>;
  registerCandidate: (input: { email: string; password: string; full_name: string; phone?: string }) => Promise<void>;
  registerCompany: (input: { email: string; password: string; company_name: string; website?: string }) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  const hydrate = useCallback(async () => {
    const ok = await authApi.refresh();
    if (ok) {
      try {
        const me = await authApi.me();
        setUser(me);
      } catch {
        setUser(null);
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
    const res = await authApi.login({ email, password });
    setAccessToken(res.access_token);
    setUser(res.user);
    return res.user;
  };
  const loginCompany = loginCandidate; // same endpoint; role comes back on the user object

  const registerCandidate: AuthContextValue["registerCandidate"] = async (input) => {
    const res = await authApi.registerCandidate(input);
    setAccessToken(res.access_token);
    setUser(res.user);
  };

  const registerCompany: AuthContextValue["registerCompany"] = async (input) => {
    const res = await authApi.registerCompany(input);
    setAccessToken(res.access_token);
    setUser(res.user);
  };

  const logout = async () => {
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
      value={{ user, loading, loginCandidate, loginCompany, registerCandidate, registerCompany, logout, refreshUser }}
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
