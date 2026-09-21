"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { registerCandidateSchema, registerCompanySchema } from "@/lib/validations";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { ApiRequestError } from "@/lib/api";

type Role = "candidate" | "company";

export default function RegisterPage() {
  const { registerCandidate, registerCompany } = useAuth();
  const router = useRouter();
  const [role, setRole] = useState<Role>("candidate");
  const [values, setValues] = useState({ name: "", email: "", password: "", extra: "" });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);
    setErrors({});

    if (role === "candidate") {
      const parsed = registerCandidateSchema.safeParse({
        full_name: values.name,
        email: values.email,
        password: values.password,
        phone: values.extra,
      });
      if (!parsed.success) {
        const fe: Record<string, string> = {};
        for (const issue of parsed.error.issues) { const k = issue.path[0] as string; if (!fe[k]) fe[k] = issue.message; }
        setErrors(fe);
        return;
      }
      setSubmitting(true);
      try {
        await registerCandidate(parsed.data);
        router.push("/candidate/dashboard");
      } catch (err) {
        setApiError(err instanceof ApiRequestError ? err.message : "Something went wrong. Please try again.");
      } finally {
        setSubmitting(false);
      }
    } else {
      const parsed = registerCompanySchema.safeParse({
        company_name: values.name,
        email: values.email,
        password: values.password,
        website: values.extra,
      });
      if (!parsed.success) {
        const fe: Record<string, string> = {};
        for (const issue of parsed.error.issues) { const k = issue.path[0] as string; if (!fe[k]) fe[k] = issue.message; }
        setErrors(fe);
        return;
      }
      setSubmitting(true);
      try {
        await registerCompany(parsed.data);
        router.push("/company/dashboard");
      } catch (err) {
        setApiError(err instanceof ApiRequestError ? err.message : "Something went wrong. Please try again.");
      } finally {
        setSubmitting(false);
      }
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-paper px-6 py-12">
      <div className="w-full max-w-sm">
        <Link href="/" className="mb-8 flex items-center gap-2">
          <span className="flex h-7 w-7 items-center justify-center rounded-md bg-navy-900 text-sm font-bold text-amber-400">
            H
          </span>
          <span className="text-[15px] font-semibold tracking-tight text-navy-900">Hirewell</span>
        </Link>

        <h1 className="text-display-sm text-navy-950">Create your account</h1>

        <div className="mt-6 grid grid-cols-2 gap-2 rounded-md border border-line bg-white p-1">
          {(["candidate", "company"] as const).map((r) => (
            <button
              key={r}
              type="button"
              onClick={() => setRole(r)}
              className={`rounded-[6px] py-2 text-sm font-medium transition-colors ${
                role === r ? "bg-navy-900 text-white" : "text-ink/60 hover:text-ink"
              }`}
            >
              {r === "candidate" ? "I'm a candidate" : "I'm hiring"}
            </button>
          ))}
        </div>

        <form onSubmit={onSubmit} className="mt-6 space-y-4" noValidate>
          {apiError && <Alert>{apiError}</Alert>}

          <TextField
            label={role === "candidate" ? "Full name" : "Company name"}
            value={values.name}
            onChange={(e) => setValues((v) => ({ ...v, name: e.target.value }))}
            error={errors.full_name || errors.company_name}
          />
          <TextField
            label="Email"
            type="email"
            autoComplete="email"
            value={values.email}
            onChange={(e) => setValues((v) => ({ ...v, email: e.target.value }))}
            error={errors.email}
          />
          <TextField
            label="Password"
            type="password"
            autoComplete="new-password"
            value={values.password}
            onChange={(e) => setValues((v) => ({ ...v, password: e.target.value }))}
            error={errors.password}
          />
          <TextField
            label={role === "candidate" ? "Phone (optional)" : "Website (optional)"}
            value={values.extra}
            onChange={(e) => setValues((v) => ({ ...v, extra: e.target.value }))}
            error={errors.phone || errors.website}
          />

          <button type="submit" disabled={submitting} className="btn-primary w-full">
            {submitting ? "Creating account..." : "Create account"}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-ink/60">
          Already have an account?{" "}
          <Link href="/login" className="font-medium text-navy-900 hover:underline">
            Log in
          </Link>
        </p>
      </div>
    </div>
  );
}
