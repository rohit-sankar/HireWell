"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { Alert } from "@/components/ui/alert";
import { candidateApi, applicationsApi, interviewsApi, ApiRequestError } from "@/lib/api";
import type { Application, MatchResult } from "@/types";

function ApplicationsContent() {
  const [applications, setApplications] = useState<Application[]>([]);
  const [matches, setMatches] = useState<Record<number, MatchResult>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startingInterview, setStartingInterview] = useState<number | null>(null);

  useEffect(() => {
    candidateApi
      .myApplications()
      .then(async (apps) => {
        setApplications(apps);
        const results = await Promise.allSettled(apps.map((a) => applicationsApi.getMatch(a.id)));
        const byId: Record<number, MatchResult> = {};
        results.forEach((r, i) => {
          if (r.status === "fulfilled") byId[apps[i].id] = r.value;
        });
        setMatches(byId);
      })
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load applications"))
      .finally(() => setLoading(false));
  }, []);

  const startInterview = async (applicationId: number) => {
    setStartingInterview(applicationId);
    try {
      const interview = await interviewsApi.create(applicationId);
      window.location.href = `/interview/${interview.id}`;
    } catch (err) {
      alert(err instanceof ApiRequestError ? err.message : "Failed to start interview");
      setStartingInterview(null);
    }
  };

  if (loading) return <div className="py-20 text-center text-sm text-ink/50">Loading...</div>;

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">Your applications</h1>

      {error ? (
        <div className="mt-6">
          <Alert>{error}</Alert>
        </div>
      ) : applications.length === 0 ? (
        <div className="card mt-6 p-8 text-center text-sm text-ink/50">
          No applications yet.{" "}
          <Link href="/jobs" className="font-medium text-navy-900 hover:underline">
            Browse open roles
          </Link>
          .
        </div>
      ) : (
        <div className="card mt-6 divide-y divide-line">
          {applications.map((app) => {
            const match = matches[app.id];
            return (
              <div key={app.id} className="flex flex-wrap items-center justify-between gap-4 px-5 py-4">
                <div>
                  <div className="font-medium text-ink">{app.job_title}</div>
                  <div className="text-xs text-ink/50">
                    {app.company_name} · {app.status.replace("_", " ")}
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  {match && (
                    <div className="text-right">
                      <div className="text-lg font-bold text-navy-900">{Math.round(match.overall_score)}</div>
                      <div className="text-xs text-ink/40">match score</div>
                    </div>
                  )}
                  <button
                    onClick={() => startInterview(app.id)}
                    disabled={startingInterview === app.id}
                    className="btn-secondary !px-3 !py-1.5 text-xs"
                  >
                    {startingInterview === app.id ? "Starting..." : "Take AI interview"}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default function ApplicationsPage() {
  return (
    <Protected role="candidate">
      <Navbar />
      <ApplicationsContent />
    </Protected>
  );
}
