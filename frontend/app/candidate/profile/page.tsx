"use client";

import { useEffect, useState } from "react";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { candidateApi, ApiRequestError } from "@/lib/api";
import type { Candidate } from "@/types";

function ProfileContent() {
  const [candidate, setCandidate] = useState<Candidate | null>(null);
  const [values, setValues] = useState({ full_name: "", phone: "", location: "", headline: "", summary: "" });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    candidateApi
      .me()
      .then((c) => {
        setCandidate(c);
        setValues({
          full_name: c.full_name || "",
          phone: c.phone || "",
          location: c.location || "",
          headline: c.headline || "",
          summary: c.summary || "",
        });
      })
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load profile"))
      .finally(() => setLoading(false));
  }, []);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setSaving(true);
    try {
      const updated = await candidateApi.updateMe(values);
      setCandidate(updated);
      setSuccess("Profile updated.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to save profile");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="py-20 text-center text-sm text-ink/50">Loading...</div>;

  return (
    <div className="mx-auto max-w-xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">Your profile</h1>
      <p className="mt-1 text-sm text-ink/60">
        This information appears alongside your resume when companies review your application.
      </p>

      {candidate && (
        <div className="card mt-6 p-4">
          <div className="flex items-center justify-between text-sm">
            <span className="text-ink/60">Profile completion</span>
            <span className="font-semibold text-navy-900">{candidate.profile_completion}%</span>
          </div>
          <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-line">
            <div className="h-full rounded-full bg-amber-500" style={{ width: `${candidate.profile_completion}%` }} />
          </div>
        </div>
      )}

      <form onSubmit={onSubmit} className="mt-6 space-y-4" noValidate>
        {error && <Alert>{error}</Alert>}
        {success && <Alert tone="success">{success}</Alert>}

        <TextField
          label="Full name"
          value={values.full_name}
          onChange={(e) => setValues((v) => ({ ...v, full_name: e.target.value }))}
        />
        <TextField
          label="Phone"
          value={values.phone}
          onChange={(e) => setValues((v) => ({ ...v, phone: e.target.value }))}
        />
        <TextField
          label="Location"
          value={values.location}
          onChange={(e) => setValues((v) => ({ ...v, location: e.target.value }))}
        />
        <TextField
          label="Headline"
          placeholder="e.g. Backend Engineer focused on distributed systems"
          value={values.headline}
          onChange={(e) => setValues((v) => ({ ...v, headline: e.target.value }))}
        />
        <div>
          <label className="field-label">Summary</label>
          <textarea
            className="input-field min-h-[100px]"
            value={values.summary}
            onChange={(e) => setValues((v) => ({ ...v, summary: e.target.value }))}
          />
        </div>

        <button type="submit" disabled={saving} className="btn-primary">
          {saving ? "Saving..." : "Save profile"}
        </button>
      </form>
    </div>
  );
}

export default function CandidateProfilePage() {
  return (
    <Protected role="candidate">
      <Navbar />
      <ProfileContent />
    </Protected>
  );
}
