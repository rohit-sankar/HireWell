"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import type { Role } from "@/types";

/**
 * Wraps a page's content and enforces that a session exists and (optionally)
 * that the session's role matches. This is a client-side UX guard, not the
 * security boundary -- every protected API call is independently
 * authorized server-side (see backend middleware.RequireAuth /
 * middleware.RequireRole), so a bypass here can't expose real data.
 */
export function Protected({ role, children }: { role?: Role; children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    if (!user) {
      router.replace("/login");
      return;
    }
    if (role && user.role !== role) {
      router.replace(user.role === "company" ? "/company/dashboard" : "/candidate/dashboard");
    }
  }, [loading, user, role, router]);

  if (loading || !user || (role && user.role !== role)) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-navy-200 border-t-navy-800" />
      </div>
    );
  }

  return <>{children}</>;
}
