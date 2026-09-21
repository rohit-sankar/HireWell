import Link from "next/link";
import { Navbar } from "@/components/layout/navbar";
import { Pipeline } from "@/components/landing/pipeline";

export default function LandingPage() {
  return (
    <div>
      <Navbar />

      {/* Hero */}
      <section className="mx-auto grid max-w-6xl gap-12 px-6 pb-20 pt-16 md:grid-cols-2 md:items-center md:pt-24">
        <div>
          <p className="mb-5 text-sm font-medium text-navy-600">For candidates and hiring teams</p>
          <h1 className="text-display-md text-navy-950 md:text-display-lg">
            Hiring, grounded in what candidates actually know.
          </h1>
          <p className="mt-6 max-w-lg text-lg leading-relaxed text-ink/70">
            Hirewell turns resumes into structured profiles, scores candidates against a job&rsquo;s real
            requirements with an explainable formula, and runs AI interviews that only ever ask about skills the
            job needs and the candidate actually has.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link href="/register" className="btn-accent">
              Create your profile
            </Link>
            <Link href="/#for-companies" className="btn-secondary">
              Hiring? Start here
            </Link>
          </div>
          <p className="mt-6 text-sm text-ink/50">No credit card required. Free while in beta.</p>
        </div>

        <Pipeline />
      </section>

      {/* Candidate workflow */}
      <section className="border-t border-line bg-white py-20">
        <div className="mx-auto max-w-6xl px-6">
          <div className="grid gap-12 md:grid-cols-2">
            <div>
              <h2 className="text-display-sm text-navy-950">Your resume becomes a real profile, not a keyword bag</h2>
              <p className="mt-4 max-w-md text-ink/70">
                Upload a PDF, DOCX, or TXT resume. Hirewell extracts your education, experience, projects, skills,
                and certifications into a structured record you can review and correct &mdash; your edits become the
                source of truth, not the AI&rsquo;s first guess.
              </p>
              <ul className="mt-6 space-y-3 text-sm text-ink/70">
                <li className="flex gap-3">
                  <span className="mt-0.5 h-1.5 w-1.5 flex-none rounded-full bg-amber-500" />
                  Personal info, education, experience, projects, skills, and certifications &mdash; extracted
                  separately, not dumped as raw text.
                </li>
                <li className="flex gap-3">
                  <span className="mt-0.5 h-1.5 w-1.5 flex-none rounded-full bg-amber-500" />
                  Every field is editable. Nothing is locked to what the AI first produced.
                </li>
                <li className="flex gap-3">
                  <span className="mt-0.5 h-1.5 w-1.5 flex-none rounded-full bg-amber-500" />
                  One profile, apply to as many jobs as you want.
                </li>
              </ul>
            </div>
            <div className="card p-6">
              <div className="mb-4 flex items-center justify-between border-b border-line pb-4">
                <span className="text-sm font-semibold text-navy-900">Extracted skills</span>
                <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-600">
                  parsed
                </span>
              </div>
              <div className="flex flex-wrap gap-2">
                {["Go", "MySQL", "REST APIs", "Docker", "Git", "PostgreSQL"].map((skill) => (
                  <span key={skill} className="rounded-md border border-line bg-paper px-2.5 py-1 text-xs font-medium text-ink/70">
                    {skill}
                  </span>
                ))}
              </div>
              <div className="mt-6 space-y-2 border-t border-line pt-4 text-sm">
                <div className="flex justify-between">
                  <span className="text-ink/50">Experience</span>
                  <span className="font-medium text-ink">3 roles extracted</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-ink/50">Projects</span>
                  <span className="font-medium text-ink">2 projects extracted</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-ink/50">Profile completion</span>
                  <span className="font-medium text-ink">85%</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Matching explanation */}
      <section id="how-it-works" className="border-t border-line bg-paper py-20">
        <div className="mx-auto max-w-6xl px-6">
          <h2 className="text-display-sm text-navy-950">An explainable score, not a black box</h2>
          <p className="mt-4 max-w-2xl text-ink/70">
            Every match score is computed from a documented formula, not a single opaque AI call. You can see
            exactly which required skills matched, how experience and education were weighed, and why the number is
            what it is.
          </p>

          <div className="mt-10 grid gap-6 md:grid-cols-3">
            <div className="card p-6 md:col-span-2">
              <div className="mb-4 text-sm font-semibold text-navy-900">Score breakdown</div>
              <div className="space-y-3">
                {[
                  { label: "Required skills matched", value: 90, weight: "40%" },
                  { label: "Experience vs. minimum", value: 75, weight: "25%" },
                  { label: "Relevant projects", value: 100, weight: "15%" },
                  { label: "Education match", value: 100, weight: "10%" },
                  { label: "Certifications", value: 50, weight: "10%" },
                ].map((row) => (
                  <div key={row.label}>
                    <div className="mb-1 flex justify-between text-xs text-ink/60">
                      <span>
                        {row.label} <span className="text-ink/35">&middot; weight {row.weight}</span>
                      </span>
                      <span className="font-medium text-ink">{row.value}</span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-line">
                      <div className="h-full rounded-full bg-navy-800" style={{ width: `${row.value}%` }} />
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className="card flex flex-col items-center justify-center p-6 text-center">
              <div className="text-5xl font-extrabold text-navy-900">83</div>
              <div className="mt-1 text-sm text-ink/50">overall match score</div>
              <div className="mt-4 rounded-md bg-amber-100 px-3 py-1.5 text-xs font-medium text-amber-600">
                Strong match
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Interview explanation */}
      <section className="border-t border-line bg-white py-20">
        <div className="mx-auto max-w-6xl px-6">
          <div className="grid gap-12 md:grid-cols-2 md:items-center">
            <div>
              <h2 className="text-display-sm text-navy-950">Interviews grounded in what&rsquo;s actually there</h2>
              <p className="mt-4 max-w-md text-ink/70">
                The AI interviewer only asks about skills that appear in <em>both</em> the job&rsquo;s requirements
                and the candidate&rsquo;s own profile. If a job requires React but the candidate never listed it,
                Hirewell won&rsquo;t suddenly run a React interview &mdash; every question is checked against that
                rule before it&rsquo;s ever shown.
              </p>
            </div>
            <div className="card divide-y divide-line">
              {[
                { q: "Tell me about a time you used Go to solve a real problem.", tag: "job requirement · Go" },
                { q: "Walk me through the architecture of your AI Resume Parser project.", tag: "candidate project" },
                { q: "At Example Technologies, what was your biggest contribution as a Backend Developer?", tag: "candidate experience" },
              ].map((item) => (
                <div key={item.q} className="p-5">
                  <p className="text-sm text-ink">{item.q}</p>
                  <span className="mt-2 inline-block rounded-md bg-paper px-2 py-0.5 text-xs font-medium text-ink/50">
                    {item.tag}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* For companies */}
      <section id="for-companies" className="border-t border-line bg-navy-950 py-20 text-white">
        <div className="mx-auto max-w-6xl px-6">
          <div className="grid gap-12 md:grid-cols-2">
            <div>
              <h2 className="text-display-sm">Post a job, get ranked candidates</h2>
              <p className="mt-4 max-w-md text-white/60">
                Define required and preferred skills, experience level, and education requirements. Every applicant
                is scored and ranked automatically, with the full breakdown available &mdash; no more scanning
                résumés by hand.
              </p>
              <div className="mt-8 flex flex-wrap gap-3">
                <Link href="/register" className="btn-accent">
                  Post your first job
                </Link>
              </div>
            </div>
            <div className="space-y-3">
              {[
                { name: "John Doe", score: 83, status: "Shortlisted" },
                { name: "Priya Nair", score: 76, status: "Under review" },
                { name: "Marco Lindt", score: 61, status: "Applied" },
              ].map((c) => (
                <div key={c.name} className="flex items-center justify-between rounded-md border border-white/10 bg-white/5 px-4 py-3">
                  <div>
                    <div className="text-sm font-medium">{c.name}</div>
                    <div className="text-xs text-white/40">{c.status}</div>
                  </div>
                  <div className="text-lg font-bold text-amber-400">{c.score}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="bg-paper py-20">
        <div className="mx-auto max-w-3xl px-6 text-center">
          <h2 className="text-display-sm text-navy-950">Start with your resume or your first job posting.</h2>
          <p className="mt-4 text-ink/60">Free during beta. Set up in under five minutes.</p>
          <div className="mt-8 flex flex-wrap justify-center gap-3">
            <Link href="/register" className="btn-accent">
              Create a free account
            </Link>
          </div>
        </div>
      </section>

      <footer className="border-t border-line bg-white py-10">
        <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-6 text-sm text-ink/50 md:flex-row">
          <span>&copy; {new Date().getFullYear()} Hirewell. All rights reserved.</span>
          <div className="flex gap-6">
            <Link href="/jobs" className="hover:text-ink">
              Browse jobs
            </Link>
            <Link href="/login" className="hover:text-ink">
              Log in
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
