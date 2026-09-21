"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { Navbar } from "@/components/layout/navbar";
import { Alert } from "@/components/ui/alert";
import { useAuth } from "@/providers/auth-provider";
import { applicationsApi, jobsApi, ApiRequestError } from "@/lib/api";
import type { Job } from "@/types";

export default function JobDetailPage() {
  const params = useParams();
  const jobId = Number(Array.isArray(params.id) ? params.id[0] : params.id);
  const validJobId = Number.isInteger(jobId) && jobId > 0;
  const router = useRouter();
  const { user } = useAuth();

  const [job, setJob] = useState<Job | null>(null);
  const [loading, setLoading] = useState(true);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [applied, setApplied] = useState(false);

  useEffect(() => {
    if (!validJobId) {
      setError("Invalid job link.");
      setLoading(false);
      return;
    }
    jobsApi
      .get(jobId)
      .then(setJob)
      .catch((err) =>
        setError(
          err instanceof ApiRequestError ? err.message : "Failed to load job",
        ),
      )
      .finally(() => setLoading(false));
  }, [jobId, validJobId]);

  const onApply = async () => {
    if (!user) {
      router.push("/register");
      return;
    }
    setError(null);
    setApplying(true);
    try {
      await applicationsApi.apply(jobId);
      setApplied(true);
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : "Failed to apply",
      );
    } finally {
      setApplying(false);
    }
  };

  if (loading)
    return (
      <div className="py-20 text-center text-sm text-ink/50">Loading...</div>
    );
  if (!job)
    return (
      <div className="py-20 text-center text-sm text-red-600">
        {error || "Job not found."}
      </div>
    );

  return (
    <div>
      <Navbar />
      <div className="mx-auto max-w-3xl px-6 py-10">
        <h1 className="text-display-sm text-navy-950">{job.title}</h1>
        <p className="mt-1 text-ink/60">
          {job.company_name} {job.location && `· ${job.location}`}
        </p>

        {error && (
          <div className="mt-4">
            <Alert>{error}</Alert>
          </div>
        )}
        {applied && (
          <div className="mt-4">
            <Alert tone="success">
              Application submitted! We&rsquo;ve calculated your match score —
              check your{" "}
              <a href="/candidate/dashboard" className="font-medium underline">
                dashboard
              </a>
              .
            </Alert>
          </div>
        )}

        <div className="mt-6 flex flex-wrap gap-2">
          {job.skills?.map((s) => (
            <span
              key={s.skill_name}
              className={`rounded-md px-2.5 py-1 text-xs font-medium ${
                s.is_required
                  ? "bg-navy-900 text-white"
                  : "border border-line bg-white text-ink/60"
              }`}
            >
              {s.skill_name} {s.is_required ? "" : "(preferred)"}
            </span>
          ))}
        </div>

        <div className="card mt-6 p-6">
          <h2 className="text-sm font-semibold text-navy-900">
            About this role
          </h2>
          <p className="mt-2 whitespace-pre-line text-sm leading-relaxed text-ink/70">
            {job.description}
          </p>

          <dl className="mt-6 grid grid-cols-2 gap-4 border-t border-line pt-4 text-sm">
            <div>
              <dt className="text-ink/40">Employment type</dt>
              <dd className="mt-0.5 font-medium text-ink">
                {job.employment_type.replace("_", " ")}
              </dd>
            </div>
            <div>
              <dt className="text-ink/40">Experience required</dt>
              <dd className="mt-0.5 font-medium text-ink">
                {job.experience_min_years}+ years
              </dd>
            </div>
            {job.education_requirement && (
              <div className="col-span-2">
                <dt className="text-ink/40">Education</dt>
                <dd className="mt-0.5 font-medium text-ink">
                  {job.education_requirement}
                </dd>
              </div>
            )}
          </dl>
        </div>

        {user?.role !== "company" && (
          <button
            onClick={onApply}
            disabled={applying || applied}
            className="btn-accent mt-6"
          >
            {applied
              ? "Applied"
              : applying
                ? "Submitting..."
                : user
                  ? "Apply now"
                  : "Sign up to apply"}
          </button>
        )}
      </div>
    </div>
  );
}
