"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { authApi, setAccessToken } from "@/lib/api";
import { useAuth } from "@/providers/auth-provider";

export default function GoogleCallbackPage() {
  const router = useRouter();
  const { refreshUser } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // The backend redirects here with #access_token=... in the URL
    // fragment (never a query param), so it's never sent to any server or
    // logged -- only readable by this page's own JavaScript.
    const hash = new URLSearchParams(window.location.hash.slice(1));
    const token = hash.get("access_token");

    if (!token) {
      setError("Google sign-in did not return a valid session. Please try again.");
      return;
    }

    setAccessToken(token);
    refreshUser()
      .then(async () => {
        const me = await authApi.me();
        router.replace(me.role === "company" ? "/company/dashboard" : "/candidate/dashboard");
      })
      .catch(() => setError("We couldn't complete your sign-in. Please try again."));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center bg-paper px-6 text-center">
      {error ? (
        <div>
          <p className="text-sm text-red-600">{error}</p>
          <a href="/login" className="mt-4 inline-block text-sm font-medium text-navy-900 hover:underline">
            Back to login
          </a>
        </div>
      ) : (
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-navy-200 border-t-navy-800" />
      )}
    </div>
  );
}
