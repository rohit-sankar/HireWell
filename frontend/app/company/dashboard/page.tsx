"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { useAuth } from "@/providers/auth-provider";
import { companyApi, jobsApi, ApiRequestError } from "@/lib/api";
import type { Company, Job } from "@/types";

const statusStyles: Record<Job["status"], string> = {
  draft: "bg-paper text-ink/50",
  published: "bg-emerald-50 text-emerald-700",
  closed: "bg-amber-100 text-amber-700",
  archived: "bg-ink/5 text-ink/40",
};

function DashboardContent() {
  const { user } = useAuth();
  const [company, setCompany] = useState<Company | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([companyApi.me(), jobsApi.list({ mine: true })])
      .then(([c, j]) => {
        setCompany(c);
        setJobs(j);
      })
      .catch((err) =>
        setError(
          err instanceof ApiRequestError
            ? err.message
            : "Failed to load dashboard",
        ),
      )
      .finally(() => setLoading(false));
  }, []);

  const publish = async (id: number) => {
    setError(null);
    try {
      const updated = await jobsApi.publish(id);
      setJobs((js) => js.map((j) => (j.id === id ? updated : j)));
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : "Failed to publish job",
      );
    }
  };
  const close = async (id: number) => {
    setError(null);
    try {
      const updated = await jobsApi.close(id);
      setJobs((js) => js.map((j) => (j.id === id ? updated : j)));
    } catch (err) {
      setError(
        err instanceof ApiRequestError ? err.message : "Failed to close job",
      );
    }
  };

  if (loading)
    return (
      <div className="py-20 text-center text-sm text-ink/50">Loading...</div>
    );
  if (error || !company)
    return (
      <div className="py-20 text-center text-sm text-red-600">{error}</div>
    );

  return (
    <div className="mx-auto max-w-5xl px-6 py-10">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-display-sm text-navy-950">
            {company.company_name}
          </h1>
          <p className="mt-1 text-sm text-ink/60">{user?.email}</p>
        </div>
        <Link href="/company/jobs/new" className="btn-accent">
          Post a job
        </Link>
      </div>

      <h2 className="mt-10 text-lg font-semibold text-navy-950">
        Your job postings
      </h2>

      {jobs.length === 0 ? (
        <div className="card mt-4 p-8 text-center text-sm text-ink/50">
          You haven&rsquo;t posted any jobs yet.{" "}
          <Link
            href="/company/jobs/new"
            className="font-medium text-navy-900 hover:underline"
          >
            Create your first one
          </Link>
          .
        </div>
      ) : (
        <div className="card mt-4 divide-y divide-line">
          {jobs.map((job) => (
            <div
              key={job.id}
              className="flex items-center justify-between gap-4 px-5 py-4"
            >
              <div className="min-w-0">
                <Link
                  href={`/company/jobs/${job.id}/applicants`}
                  className="font-medium text-ink hover:underline"
                >
                  {job.title}
                </Link>
                <div className="mt-0.5 text-xs text-ink/50">
                  {job.employment_type.replace("_", " ")} ·{" "}
                  {job.location || "Location not set"}
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span
                  className={`rounded-full px-2.5 py-1 text-xs font-medium ${statusStyles[job.status]}`}
                >
                  {job.status}
                </span>
                {job.status === "draft" && (
                  <button
                    onClick={() => publish(job.id)}
                    className="btn-secondary !px-3 !py-1.5 text-xs"
                  >
                    Publish
                  </button>
                )}
                {job.status === "published" && (
                  <button
                    onClick={() => close(job.id)}
                    className="btn-secondary !px-3 !py-1.5 text-xs"
                  >
                    Close
                  </button>
                )}
                <Link
                  href={`/company/jobs/${job.id}/edit`}
                  className="text-sm font-medium text-ink/60 hover:underline"
                >
                  Edit
                </Link>
                <Link
                  href={`/company/jobs/${job.id}/applicants`}
                  className="text-sm font-medium text-navy-700 hover:underline"
                >
                  Applicants →
                </Link>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function CompanyDashboardPage() {
  return (
    <Protected role="company">
      <Navbar />
      <DashboardContent />
    </Protected>
  );
}
