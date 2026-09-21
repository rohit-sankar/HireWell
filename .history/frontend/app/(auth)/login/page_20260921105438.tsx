"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { loginSchema } from "@/lib/validations";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { ApiRequestError, authApi } from "@/lib/api";

export default function LoginPage() {
  const { loginCandidate } = useAuth();
  const router = useRouter();
  const [values, setValues] = useState({ email: "", password: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);

    const parsed = loginSchema.safeParse(values);
    if (!parsed.success) {
      const fieldErrors: Record<string, string> = {};
      for (const issue of parsed.error.issues) {
        const key = issue.path[0] as string;
        if (!fieldErrors[key]) fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    setSubmitting(true);
    try {
      const user = await loginCandidate(
        parsed.data.email,
        parsed.data.password,
      );
      router.push(
        user.role === "company" ? "/company/dashboard" : "/candidate/dashboard",
      );
    } catch (err) {
      if (err instanceof ApiRequestError) setApiError(err.message);
      else setApiError("Something went wrong. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-paper px-6">
      <div className="w-full max-w-sm">
        <Link href="/" className="mb-8 flex items-center gap-2">
          <span className="flex h-7 w-7 items-center justify-center rounded-md bg-navy-900 text-sm font-bold text-amber-400">
            H
          </span>
          <span className="text-[15px] font-semibold tracking-tight text-navy-900">
            Hirewell
          </span>
        </Link>

        <h1 className="text-display-sm text-navy-950">Welcome back</h1>
        <p className="mt-2 text-sm text-ink/60">
          Log in to your candidate or company account.
        </p>

        <form onSubmit={onSubmit} className="mt-8 space-y-4" noValidate>
          {apiError && <Alert>{apiError}</Alert>}

          <TextField
            label="Email"
            type="email"
            autoComplete="email"
            value={values.email}
            onChange={(e) =>
              setValues((v) => ({ ...v, email: e.target.value }))
            }
            error={errors.email}
          />
          <TextField
            label="Password"
            type="password"
            autoComplete="current-password"
            value={values.password}
            onChange={(e) =>
              setValues((v) => ({ ...v, password: e.target.value }))
            }
            error={errors.password}
          />

          <button
            type="submit"
            disabled={submitting}
            className="btn-primary w-full"
          >
            {submitting ? "Logging in..." : "Log in"}
          </button>
        </form>

        <a
          href={authApi.googleLoginUrl()}
          className="btn-secondary mt-3 w-full"
        >
          Continue with Google
        </a>

        <p className="mt-6 text-center text-sm text-ink/60">
          Don&rsquo;t have an account?{" "}
          <Link
            href="/register"
            className="font-medium text-navy-900 hover:underline"
          >
            Sign up
          </Link>
        </p>
      </div>
    </div>
  );
}
