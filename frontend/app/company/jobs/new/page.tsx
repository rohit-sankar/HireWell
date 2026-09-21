"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";
import { createJobSchema } from "@/lib/validations";
import { jobsApi, ApiRequestError } from "@/lib/api";

interface SkillRow {
  name: string;
  is_required: boolean;
  weight: number;
}

function NewJobContent() {
  const router = useRouter();
  const [values, setValues] = useState({
    title: "",
    description: "",
    location: "",
    employment_type: "full_time",
    experience_min_years: "0",
    education_requirement: "",
  });
  const [skills, setSkills] = useState<SkillRow[]>([{ name: "", is_required: true, weight: 1 }]);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [apiError, setApiError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const updateSkill = (i: number, patch: Partial<SkillRow>) =>
    setSkills((s) => s.map((row, idx) => (idx === i ? { ...row, ...patch } : row)));

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setApiError(null);

    const parsed = createJobSchema.safeParse(values);
    if (!parsed.success) {
      const fe: Record<string, string> = {};
      for (const issue of parsed.error.issues) fe[issue.path[0] as string] = issue.message;
      setErrors(fe);
      return;
    }
    setErrors({});
    setSubmitting(true);
    try {
      const job = await jobsApi.create({
        ...parsed.data,
        skills: skills.filter((s) => s.name.trim() !== ""),
      });
      router.push(`/company/jobs/${job.id}/applicants`);
    } catch (err) {
      setApiError(err instanceof ApiRequestError ? err.message : "Failed to create job");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">Post a new job</h1>
      <p className="mt-1 text-sm text-ink/60">
        Define required and preferred skills precisely — this is exactly what the matching engine and AI interviewer
        will use.
      </p>

      <form onSubmit={onSubmit} className="mt-8 space-y-5" noValidate>
        {apiError && <Alert>{apiError}</Alert>}

        <TextField
          label="Job title"
          value={values.title}
          onChange={(e) => setValues((v) => ({ ...v, title: e.target.value }))}
          error={errors.title}
        />

        <div>
          <label className="field-label">Description</label>
          <textarea
            className="input-field min-h-[120px]"
            value={values.description}
            onChange={(e) => setValues((v) => ({ ...v, description: e.target.value }))}
          />
          {errors.description && <p className="mt-1.5 text-sm text-red-600">{errors.description}</p>}
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
            label="Education requirement (optional)"
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
                  placeholder="e.g. Go, MySQL, Docker"
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

        <button type="submit" disabled={submitting} className="btn-primary w-full">
          {submitting ? "Creating..." : "Create job (draft)"}
        </button>
        <p className="text-center text-xs text-ink/40">You can review and publish it from your dashboard.</p>
      </form>
    </div>
  );
}

export default function NewJobPage() {
  return (
    <Protected role="company">
      <Navbar />
      <NewJobContent />
    </Protected>
  );
}
