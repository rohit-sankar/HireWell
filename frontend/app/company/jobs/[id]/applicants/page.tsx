"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { jobsApi, applicationsApi, ApiRequestError } from "@/lib/api";
import type { Job, RankedApplication } from "@/types";

function scoreColor(score: number) {
  if (score >= 75) return "text-emerald-600";
  if (score >= 50) return "text-amber-600";
  return "text-ink/50";
}

function ApplicantsContent() {
  const params = useParams();
  const jobId = Number(Array.isArray(params.id) ? params.id[0] : params.id);
  const validJobId = Number.isInteger(jobId) && jobId > 0;

  const [job, setJob] = useState<Job | null>(null);
  const [applicants, setApplicants] = useState<RankedApplication[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<number | null>(null);

  useEffect(() => {
    if (!validJobId) {
      setError("Invalid job link.");
      setLoading(false);
      return;
    }
    Promise.all([jobsApi.get(jobId), jobsApi.matches(jobId)])
      .then(([j, m]) => {
        setJob(j);
        setApplicants(m);
      })
      .catch((err) =>
        setError(
          err instanceof ApiRequestError
            ? err.message
            : "Failed to load applicants",
        ),
      )
      .finally(() => setLoading(false));
  }, [jobId, validJobId]);

  const updateStatus = async (
    applicationId: number,
    status: "shortlisted" | "rejected" | "under_review",
  ) => {
    setError(null);
    try {
      const updated = await applicationsApi.updateStatus(applicationId, status);
      setApplicants((apps) =>
        apps.map((a) =>
          a.id === applicationId ? { ...a, status: updated.status } : a,
        ),
      );
    } catch (err) {
      setError(
        err instanceof ApiRequestError
          ? err.message
          : "Failed to update applicant status",
      );
    }
  };

  if (loading)
    return (
      <div className="py-20 text-center text-sm text-ink/50">
        Loading applicants...
      </div>
    );
  if (error || !job)
    return (
      <div className="py-20 text-center text-sm text-red-600">{error}</div>
    );

  return (
    <div className="mx-auto max-w-5xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">{job.title}</h1>
      <p className="mt-1 text-sm text-ink/60">
        {applicants.length} applicant{applicants.length === 1 ? "" : "s"},
        ranked by match score
      </p>

      {applicants.length === 0 ? (
        <div className="card mt-6 p-8 text-center text-sm text-ink/50">
          No applicants yet.
        </div>
      ) : (
        <div className="card mt-6 divide-y divide-line">
          {applicants.map((app) => (
            <div key={app.id} className="px-5 py-4">
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <div className="font-medium text-ink">
                    {app.candidate_name}
                  </div>
                  <div className="mt-0.5 text-xs text-ink/50">
                    Applied {new Date(app.applied_at).toLocaleDateString()} ·{" "}
                    {app.status.replace("_", " ")}
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <div
                      className={`text-2xl font-bold ${scoreColor(app.MatchResult?.overall_score ?? 0)}`}
                    >
                      {app.MatchResult
                        ? Math.round(app.MatchResult.overall_score)
                        : "—"}
                    </div>
                    <button
                      onClick={() =>
                        setExpanded(expanded === app.id ? null : app.id)
                      }
                      className="text-xs font-medium text-navy-700 hover:underline"
                    >
                      {expanded === app.id
                        ? "Hide breakdown"
                        : "View breakdown"}
                    </button>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => updateStatus(app.id, "shortlisted")}
                      className="btn-secondary !px-3 !py-1.5 text-xs"
                      disabled={app.status === "shortlisted"}
                    >
                      Shortlist
                    </button>
                    <button
                      onClick={() => updateStatus(app.id, "rejected")}
                      className="btn-secondary !px-3 !py-1.5 text-xs"
                      disabled={app.status === "rejected"}
                    >
                      Reject
                    </button>
                  </div>
                </div>
              </div>

              {expanded === app.id && app.MatchResult && (
                <div className="mt-4 grid grid-cols-2 gap-3 rounded-md bg-paper p-4 text-sm md:grid-cols-5">
                  <ScoreCell
                    label="Skills"
                    value={app.MatchResult.skill_score}
                  />
                  <ScoreCell
                    label="Experience"
                    value={app.MatchResult.experience_score}
                  />
                  <ScoreCell
                    label="Education"
                    value={app.MatchResult.education_score}
                  />
                  <ScoreCell
                    label="Projects"
                    value={app.MatchResult.project_score}
                  />
                  <ScoreCell
                    label="Certifications"
                    value={app.MatchResult.certification_score}
                  />
                  <div className="col-span-2 mt-2 md:col-span-5">
                    <div className="mb-1.5 text-xs font-medium uppercase tracking-wide text-ink/40">
                      Required skills matched:{" "}
                      {app.MatchResult.explanation.required_matched}/
                      {app.MatchResult.explanation.required_total}
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {app.MatchResult.explanation.skill_details?.map((d) => (
                        <span
                          key={d.skill_name}
                          className={`rounded px-2 py-0.5 text-xs ${
                            d.matched
                              ? "bg-emerald-50 text-emerald-700"
                              : "bg-red-50 text-red-500"
                          }`}
                        >
                          {d.skill_name}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ScoreCell({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <div className="text-xs text-ink/40">{label}</div>
      <div className="font-semibold text-ink">{Math.round(value)}</div>
    </div>
  );
}

export default function ApplicantsPage() {
  return (
    <Protected role="company">
      <Navbar />
      <ApplicantsContent />
    </Protected>
  );
}
