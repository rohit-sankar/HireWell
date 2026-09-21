"use client";

import Link from "next/link";
import { useAuth } from "@/providers/auth-provider";

export function Navbar() {
  const { user, loading, logout } = useAuth();

  const dashboardHref = user?.role === "company" ? "/company/dashboard" : "/candidate/dashboard";

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-paper/90 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-7 w-7 items-center justify-center rounded-md bg-navy-900 text-sm font-bold text-amber-400">
            H
          </span>
          <span className="text-[15px] font-semibold tracking-tight text-navy-900">Hirewell</span>
        </Link>

        <nav className="hidden items-center gap-8 text-sm font-medium text-ink/70 md:flex">
          <Link href="/jobs" className="hover:text-navy-900">
            Browse jobs
          </Link>
          <Link href="/#how-it-works" className="hover:text-navy-900">
            How it works
          </Link>
          <Link href="/#for-companies" className="hover:text-navy-900">
            For companies
          </Link>
        </nav>

        <div className="flex items-center gap-3">
          {loading ? null : user ? (
            <>
              <Link href={dashboardHref} className="btn-secondary !px-4 !py-2 text-sm">
                Dashboard
              </Link>
              <Link
                href={user.role === "company" ? "/company/profile" : "/candidate/profile"}
                className="text-sm font-medium text-ink/60 hover:text-ink"
              >
                Profile
              </Link>
              <button onClick={logout} className="text-sm font-medium text-ink/60 hover:text-ink">
                Log out
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="text-sm font-medium text-ink/70 hover:text-navy-900">
                Log in
              </Link>
              <Link href="/register" className="btn-primary !px-4 !py-2 text-sm">
                Get started
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
