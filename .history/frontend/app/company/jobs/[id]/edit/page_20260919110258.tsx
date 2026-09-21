"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { jobsApi, ApiRequestError } from "@/lib/api";
import type { Job } from "@/types";

interface SkillRow {
  name: string;
  is_required: boolean;
  weight: number;
}

function EditJobContent() {
  const params = useParams();
  const router = useRouter();
  const jobId = Number(Array.isArray(params.id) ? params.id[0] : params.id);
  const validJobId = Number.isInteger(jobId) && jobId > 0;

  const [values, setValues] = useState({
    title: "",
    description: "",
    location: "",
    employment_type: "full_time",
    experience_min_years: "0",
    education_requirement: "",
  });
  const [skills, setSkills] = useState<SkillRow[]>([]);
  const [status, setStatus] = useState<Job["status"]>("draft");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    if (!validJobId) {
      setError("Invalid job link.");
      setLoading(false);
      return;
    }
    jobsApi
      .get(jobId)
      .then((job) => {
        setValues({
          title: job.title,
          description: job.description,
          location: job.location || "",
          employment_type: job.employment_type,
          experience_min_years: String(job.experience_min_years),
          education_requirement: job.education_requirement || "",
        });
        setSkills((job.skills || []).map((s) => ({ name: s.skill_name, is_required: s.is_required, weight: s.weight })));
        setStatus(job.status);
      })
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load job"))
      .finally(() => setLoading(false));
  }, [jobId, validJobId]);

  const updateSkill = (i: number, patch: Partial<SkillRow>) =>
    setSkills((s) => s.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setSaving(true);
    try {
      await jobsApi.update(jobId, {
        ...values,
        experience_min_years: Number(values.experience_min_years),
        skills: skills.filter((s) => s.name.trim() !== ""),
      });
      setSuccess("Job updated.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to update job");
    } finally {
      setSaving(false);
    }
  };

  const archive = async () => {
    if (!confirm("Archive this job? It will no longer be visible to candidates.")) return;
    try {
      await jobsApi.archive(jobId);
      router.push("/company/dashboard");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to archive job");
    }
  };

  if (loading) return <div className="py-20 text-center text-sm text-ink/50">Loading...</div>;

  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <div className="flex items-center justify-between">
        <h1 className="text-display-sm text-navy-950">Edit job</h1>
        <span className="rounded-full bg-paper px-2.5 py-1 text-xs font-medium text-ink/60">{status}</span>
      </div>

      <form onSubmit={onSubmit} className="mt-8 space-y-5" noValidate>
        {error && <Alert>{error}</Alert>}
        {success && <Alert tone="success">{success}</Alert>}

        <TextField
          label="Job title"
          value={values.title}
          onChange={(e) => setValues((v) => ({ ...v, title: e.target.value }))}
        />

        <div>
          <label className="field-label">Description</label>
          <textarea
            className="input-field min-h-[120px]"
            value={values.description}
            onChange={(e) => setValues((v) => ({ ...v, description: e.target.value }))}
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <TextField
            label="Location"
            value={values.location}
            onChange={(e) => setValues((v) => ({ ...v, location: e.target.value }))}
          />
          <div>
            <label className="field-label">Employment type</label>
            <select
              className="input-field"
              value={values.employment_type}
              onChange={(e) => setValues((v) => ({ ...v, employment_type: e.target.value }))}
            >
              <option value="full_time">Full-time</option>
              <option value="part_time">Part-time</option>
              <option value="contract">Contract</option>
              <option value="internship">Internship</option>
              <option value="remote">Remote</option>
            </select>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <TextField
            label="Minimum years of experience"
            type="number"
            min={0}
            value={values.experience_min_years}
            onChange={(e) => setValues((v) => ({ ...v, experience_min_years: e.target.value }))}
          />
          <TextField
            label="Education requirement"
            value={values.education_requirement}
            onChange={(e) => setValues((v) => ({ ...v, education_requirement: e.target.value }))}
          />
        </div>

        <div>
          <label className="field-label">Skills</label>
          <div className="space-y-2">
            {skills.map((skill, i) => (
              <div key={i} className="flex items-center gap-2">
                <input
                  className="input-field flex-1"
                  value={skill.name}
                  onChange={(e) => updateSkill(i, { name: e.target.value })}
                />
                <label className="flex items-center gap-1.5 whitespace-nowrap text-sm text-ink/60">
                  <input
                    type="checkbox"
                    checked={skill.is_required}
                    onChange={(e) => updateSkill(i, { is_required: e.target.checked })}
                  />
                  Required
                </label>
                <button
                  type="button"
                  onClick={() => setSkills((s) => s.filter((_, idx) => idx !== i))}
                  className="text-ink/30 hover:text-red-500"
                  aria-label="Remove skill"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={() => setSkills((s) => [...s, { name: "", is_required: true, weight: 1 }])}
            className="mt-2 text-sm font-medium text-navy-700 hover:underline"
          >
            + Add another skill
          </button>
        </div>

        <div className="flex items-center justify-between pt-2">
          <button type="submit" disabled={saving} className="btn-primary">
            {saving ? "Saving..." : "Save changes"}
          </button>
          {status !== "archived" && (
            <button type="button" onClick={archive} className="text-sm font-medium text-red-600 hover:underline">
              Archive this job
            </button>
          )}
        </div>
      </form>
    </div>
  );
}

export default function EditJobPage() {
  return (
    <Protected role="company">
      <Navbar />
      <EditJobContent />
    </Protected>
  );
}
