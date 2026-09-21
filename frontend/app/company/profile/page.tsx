"use client";

import { useEffect, useState } from "react";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { companyApi, ApiRequestError } from "@/lib/api";

function CompanyProfileContent() {
  const [values, setValues] = useState({
    company_name: "",
    website: "",
    description: "",
    industry: "",
    company_size: "",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    companyApi
      .me()
      .then((c) =>
        setValues({
          company_name: c.company_name || "",
          website: c.website || "",
          description: c.description || "",
          industry: c.industry || "",
          company_size: c.company_size || "",
        })
      )
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load company profile"))
      .finally(() => setLoading(false));
  }, []);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setSaving(true);
    try {
      await companyApi.updateMe(values);
      setSuccess("Company profile updated.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to save company profile");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="py-20 text-center text-sm text-ink/50">Loading...</div>;

  return (
    <div className="mx-auto max-w-xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">Company profile</h1>
      <p className="mt-1 text-sm text-ink/60">This appears on every job posting you publish.</p>

      <form onSubmit={onSubmit} className="mt-6 space-y-4" noValidate>
        {error && <Alert>{error}</Alert>}
        {success && <Alert tone="success">{success}</Alert>}

        <TextField
          label="Company name"
          value={values.company_name}
          onChange={(e) => setValues((v) => ({ ...v, company_name: e.target.value }))}
        />
        <TextField
          label="Website"
          value={values.website}
          onChange={(e) => setValues((v) => ({ ...v, website: e.target.value }))}
        />
        <div className="grid grid-cols-2 gap-4">
          <TextField
            label="Industry"
            value={values.industry}
            onChange={(e) => setValues((v) => ({ ...v, industry: e.target.value }))}
          />
          <TextField
            label="Company size"
            placeholder="e.g. 11-50 employees"
            value={values.company_size}
            onChange={(e) => setValues((v) => ({ ...v, company_size: e.target.value }))}
          />
        </div>
        <div>
          <label className="field-label">Description</label>
          <textarea
            className="input-field min-h-[120px]"
            value={values.description}
            onChange={(e) => setValues((v) => ({ ...v, description: e.target.value }))}
          />
        </div>

        <button type="submit" disabled={saving} className="btn-primary">
          {saving ? "Saving..." : "Save company profile"}
        </button>
      </form>
    </div>
  );
}

export default function CompanyProfilePage() {
  return (
    <Protected role="company">
      <Navbar />
      <CompanyProfileContent />
    </Protected>
  );
}
