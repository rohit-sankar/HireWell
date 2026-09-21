"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Navbar } from "@/components/layout/navbar";
import { jobsApi, ApiRequestError } from "@/lib/api";
import type { Job } from "@/types";

const employmentLabels: Record<string, string> = {
  full_time: "Full-time",
  part_time: "Part-time",
  contract: "Contract",
  internship: "Internship",
  remote: "Remote",
};

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const t = setTimeout(() => {
      setLoading(true);
      setError(null);
      jobsApi
        .list({ search: search || undefined })
        .then(setJobs)
        .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load jobs"))
        .finally(() => setLoading(false));
    }, 250);
    return () => clearTimeout(t);
  }, [search]);

  return (
    <div>
      <Navbar />
      <div className="mx-auto max-w-4xl px-6 py-10">
        <h1 className="text-display-sm text-navy-950">Open roles</h1>
        <p className="mt-1 text-sm text-ink/60">Every job below is scored against your profile once you apply.</p>

        <input
          type="search"
          placeholder="Search by title or keyword..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="input-field mt-6"
        />

        {error && <p className="mt-6 text-sm text-red-600">{error}</p>}
        {loading && <p className="mt-6 text-sm text-ink/50">Loading jobs...</p>}

        {!loading && jobs.length === 0 && (
          <div className="card mt-6 p-8 text-center text-sm text-ink/50">No open roles match your search right now.</div>
        )}

        <div className="mt-6 space-y-3">
          {jobs.map((job) => (
            <Link
              key={job.id}
              href={`/jobs/${job.id}`}
              className="card block p-5 transition-colors hover:border-navy-300"
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="font-semibold text-navy-950">{job.title}</div>
                  <div className="mt-0.5 text-sm text-ink/60">{job.company_name}</div>
                </div>
                <span className="whitespace-nowrap rounded-full bg-paper px-2.5 py-1 text-xs font-medium text-ink/60">
                  {employmentLabels[job.employment_type] || job.employment_type}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap gap-2 text-xs text-ink/50">
                {job.location && <span>{job.location}</span>}
                {job.experience_min_years > 0 && <span>· {job.experience_min_years}+ yrs experience</span>}
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
