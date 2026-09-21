"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { useAuth } from "@/providers/auth-provider";
import { candidateApi, ApiRequestError } from "@/lib/api";
import type { Application, Candidate } from "@/types";

function DashboardContent() {
  const { user } = useAuth();
  const [candidate, setCandidate] = useState<Candidate | null>(null);
  const [applications, setApplications] = useState<Application[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([candidateApi.me(), candidateApi.myApplications()])
      .then(([c, apps]) => {
        setCandidate(c);
        setApplications(apps);
      })
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load dashboard"))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return <div className="py-20 text-center text-sm text-ink/50">Loading your dashboard...</div>;
  }
  if (error || !candidate) {
    return <div className="py-20 text-center text-sm text-red-600">{error || "Could not load profile."}</div>;
  }

  const statusCounts = applications.reduce<Record<string, number>>((acc, a) => {
    acc[a.status] = (acc[a.status] || 0) + 1;
    return acc;
  }, {});

  return (
    <div className="mx-auto max-w-5xl px-6 py-10">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-display-sm text-navy-950">Welcome back, {candidate.full_name.split(" ")[0]}</h1>
          <p className="mt-1 text-sm text-ink/60">{user?.email}</p>
        </div>
        <Link href="/candidate/profile" className="btn-secondary">
          Edit profile
        </Link>
      </div>

      <div className="mt-8 grid gap-4 md:grid-cols-3">
        <div className="card p-5">
          <div className="text-xs font-medium uppercase tracking-wide text-ink/40">Profile completion</div>
          <div className="mt-2 text-3xl font-bold text-navy-900">{candidate.profile_completion}%</div>
          <div className="mt-3 h-1.5 w-full overflow-hidden rounded-full bg-line">
            <div className="h-full rounded-full bg-amber-500" style={{ width: `${candidate.profile_completion}%` }} />
          </div>
        </div>
        <div className="card p-5">
          <div className="text-xs font-medium uppercase tracking-wide text-ink/40">Resume status</div>
          <div className="mt-2 text-3xl font-bold text-navy-900">
            {candidate.active_resume_id ? "Parsed" : "Not uploaded"}
          </div>
          <Link href="/candidate/resume" className="mt-3 inline-block text-sm font-medium text-navy-700 hover:underline">
            {candidate.active_resume_id ? "View & edit resume →" : "Upload your resume →"}
          </Link>
        </div>
        <div className="card p-5">
          <div className="text-xs font-medium uppercase tracking-wide text-ink/40">Applications</div>
          <div className="mt-2 text-3xl font-bold text-navy-900">{applications.length}</div>
          <div className="mt-3 text-sm text-ink/50">{statusCounts.shortlisted || 0} shortlisted</div>
        </div>
      </div>

      <div className="mt-10 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-navy-950">Recent applications</h2>
        <Link href="/jobs" className="text-sm font-medium text-navy-700 hover:underline">
          Browse jobs →
        </Link>
      </div>

      {applications.length === 0 ? (
        <div className="card mt-4 p-8 text-center text-sm text-ink/50">
          You haven&rsquo;t applied to any jobs yet.{" "}
          <Link href="/jobs" className="font-medium text-navy-900 hover:underline">
            Find your first opportunity
          </Link>
          .
        </div>
      ) : (
        <div className="card mt-4 divide-y divide-line">
          {applications.map((app) => (
            <div key={app.id} className="flex items-center justify-between px-5 py-4">
              <div>
                <div className="text-sm font-medium text-ink">{app.job_title}</div>
                <div className="text-xs text-ink/50">{app.company_name}</div>
              </div>
              <StatusBadge status={app.status} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: Application["status"] }) {
  const styles: Record<Application["status"], string> = {
    applied: "bg-navy-50 text-navy-700",
    under_review: "bg-amber-100 text-amber-600",
    shortlisted: "bg-emerald-50 text-emerald-700",
    rejected: "bg-red-50 text-red-600",
    hired: "bg-emerald-100 text-emerald-800",
  };
  const labels: Record<Application["status"], string> = {
    applied: "Applied",
    under_review: "Under review",
    shortlisted: "Shortlisted",
    rejected: "Not selected",
    hired: "Hired",
  };
  return <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${styles[status]}`}>{labels[status]}</span>;
}

export default function CandidateDashboardPage() {
  return (
    <Protected role="candidate">
      <Navbar />
      <DashboardContent />
    </Protected>
  );
}
