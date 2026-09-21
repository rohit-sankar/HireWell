"use client";

import { useEffect, useRef, useState } from "react";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { Alert } from "@/components/ui/alert";
import { candidateApi, ApiRequestError } from "@/lib/api";
import type { ResumeFull } from "@/types";

function ResumeContent() {
  const [resume, setResume] = useState<ResumeFull | null>(null);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    candidateApi
      .getResume()
      .then(setResume)
      .catch((err) => {
        // 404 just means no resume uploaded yet -- not a real error
        if (!(err instanceof ApiRequestError && err.status === 404)) {
          setError(err instanceof ApiRequestError ? err.message : "Failed to load resume");
        }
      })
      .finally(() => setLoading(false));
  }, []);

  const onFileSelected = async (file: File) => {
    setError(null);
    setSuccess(null);
    setUploading(true);
    try {
      const parsed = await candidateApi.uploadResume(file);
      setResume(parsed);
      setSuccess("Resume uploaded and parsed successfully.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Upload failed. Please try again.");
    } finally {
      setUploading(false);
    }
  };

  const saveEdits = async (updated: ResumeFull) => {
    if (!resume) return;
    setError(null);
    try {
      const saved = await candidateApi.updateResume(resume.id, updated);
      setResume(saved);
      setSuccess("Changes saved.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to save changes");
    }
  };

  return (
    <div className="mx-auto max-w-3xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">Resume</h1>
      <p className="mt-1 text-sm text-ink/60">
        Upload a PDF, DOCX, or TXT resume. We&rsquo;ll extract your information automatically &mdash; review and
        correct anything below before applying to jobs.
      </p>

      {error && (
        <div className="mt-4">
          <Alert>{error}</Alert>
        </div>
      )}
      {success && (
        <div className="mt-4">
          <Alert tone="success">{success}</Alert>
        </div>
      )}

      <div className="card mt-6 flex flex-col items-center gap-3 p-8 text-center">
        <input
          ref={fileInputRef}
          type="file"
          accept=".pdf,.doc,.docx,.txt"
          className="hidden"
          onChange={(e) => e.target.files?.[0] && onFileSelected(e.target.files[0])}
        />
        <p className="text-sm text-ink/60">{resume ? "Replace your resume" : "No resume uploaded yet"}</p>
        <button onClick={() => fileInputRef.current?.click()} disabled={uploading} className="btn-primary">
          {uploading ? "Uploading & parsing..." : resume ? "Upload new resume" : "Upload resume"}
        </button>
        <p className="text-xs text-ink/40">PDF, DOC, DOCX, or TXT — up to 10MB</p>
      </div>

      {loading && <div className="mt-8 text-center text-sm text-ink/50">Loading...</div>}

      {resume && resume.status === "failed" && (
        <div className="mt-6">
          <Alert>Parsing failed: {resume.parse_error || "Unknown error"}. Please try uploading again.</Alert>
        </div>
      )}

      {resume && resume.status === "parsed" && <ResumeEditor resume={resume} onSave={saveEdits} />}
    </div>
  );
}

function ResumeEditor({ resume, onSave }: { resume: ResumeFull; onSave: (r: ResumeFull) => void }) {
  const [skills, setSkills] = useState(resume.skills || []);
  const [experience, setExperience] = useState(resume.experience || []);
  const [education, setEducation] = useState(resume.education || []);
  const [projects, setProjects] = useState(resume.projects || []);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    setSkills(resume.skills || []);
    setExperience(resume.experience || []);
    setEducation(resume.education || []);
    setProjects(resume.projects || []);
    setDirty(false);
  }, [resume]);

  return (
    <div className="mt-8 space-y-8">
      <section>
        <h2 className="mb-3 text-sm font-semibold text-navy-900">Skills</h2>
        <div className="flex flex-wrap gap-2">
          {skills.map((s, i) => (
            <span key={i} className="flex items-center gap-1.5 rounded-md border border-line bg-white px-2.5 py-1 text-xs font-medium text-ink/70">
              {s.skill_name}
              <button
                type="button"
                aria-label={`Remove ${s.skill_name}`}
                onClick={() => {
                  setSkills(skills.filter((_, idx) => idx !== i));
                  setDirty(true);
                }}
                className="text-ink/30 hover:text-red-500"
              >
                ×
              </button>
            </span>
          ))}
          {skills.length === 0 && <p className="text-sm text-ink/40">No skills extracted. Try re-uploading your resume.</p>}
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold text-navy-900">Experience</h2>
        {experience.length === 0 ? (
          <p className="text-sm text-ink/40">No experience entries extracted yet.</p>
        ) : (
          <div className="space-y-3">
            {experience.map((e, i) => (
              <div key={i} className="card p-4">
                <div className="font-medium text-ink">{e.role}</div>
                <div className="text-sm text-ink/60">{e.company}</div>
                <div className="mt-1 text-xs text-ink/40">
                  {e.start_date || "?"} — {e.is_current ? "Present" : e.end_date || "?"}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold text-navy-900">Education</h2>
        {education.length === 0 ? (
          <p className="text-sm text-ink/40">No education entries extracted yet.</p>
        ) : (
          <div className="space-y-3">
            {education.map((e, i) => (
              <div key={i} className="card p-4">
                <div className="font-medium text-ink">{e.institution}</div>
                <div className="text-sm text-ink/60">
                  {e.degree} {e.field ? `in ${e.field}` : ""}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-semibold text-navy-900">Projects</h2>
        {projects.length === 0 ? (
          <p className="text-sm text-ink/40">No projects extracted yet.</p>
        ) : (
          <div className="space-y-3">
            {projects.map((p, i) => (
              <div key={i} className="card p-4">
                <div className="font-medium text-ink">{p.name}</div>
                <div className="mt-1 flex flex-wrap gap-1.5">
                  {(p.technologies || []).map((t) => (
                    <span key={t} className="rounded bg-paper px-1.5 py-0.5 text-xs text-ink/50">
                      {t}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {dirty && (
        <button
          onClick={() =>
            onSave({
              ...resume,
              skills,
              experience,
              education,
              projects,
            })
          }
          className="btn-primary"
        >
          Save changes
        </button>
      )}
    </div>
  );
}

export default function ResumePage() {
  return (
    <Protected role="candidate">
      <Navbar />
      <ResumeContent />
    </Protected>
  );
}
