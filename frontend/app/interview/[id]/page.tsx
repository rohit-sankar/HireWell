"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Protected } from "@/components/layout/protected";
import { Navbar } from "@/components/layout/navbar";
import { Alert } from "@/components/ui/alert";
import { interviewsApi, ApiRequestError } from "@/lib/api";
import type { Interview, InterviewEvaluation, InterviewQuestion } from "@/types";

function InterviewContent() {
  const params = useParams();
  const interviewId = Number(params.id);

  const [questions, setQuestions] = useState<InterviewQuestion[]>([]);
  const [current, setCurrent] = useState(0);
  const [answer, setAnswer] = useState("");
  const [answered, setAnswered] = useState<Set<number>>(new Set());
  const [evaluation, setEvaluation] = useState<InterviewEvaluation | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    interviewsApi
      .get(interviewId)
      .then(async (res) => {
        setQuestions(res.questions);
        let iv: Interview = res.interview;
        if (iv.status === "pending") {
          iv = await interviewsApi.start(interviewId);
        }
        if (iv.status === "completed") {
          const result = await interviewsApi.result(interviewId);
          setEvaluation(result);
        }
      })
      .catch((err) => setError(err instanceof ApiRequestError ? err.message : "Failed to load interview"))
      .finally(() => setLoading(false));
  }, [interviewId]);

  const submitAnswer = async () => {
    if (!answer.trim() || !questions[current]) return;
    setSubmitting(true);
    setError(null);
    try {
      await interviewsApi.answer(interviewId, questions[current].id, answer);
      setAnswered((s) => new Set(s).add(questions[current].id));
      setAnswer("");
      if (current < questions.length - 1) {
        setCurrent((c) => c + 1);
      }
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to submit answer");
    } finally {
      setSubmitting(false);
    }
  };

  const finishInterview = async () => {
    setSubmitting(true);
    setError(null);
    try {
      const result = await interviewsApi.complete(interviewId);
      setEvaluation(result);
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to complete interview");
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) return <div className="py-20 text-center text-sm text-ink/50">Loading interview...</div>;
  if (error && !questions.length) return <div className="py-20 text-center text-sm text-red-600">{error}</div>;

  if (evaluation) {
    return (
      <div className="mx-auto max-w-2xl px-6 py-10">
        <h1 className="text-display-sm text-navy-950">Interview complete</h1>
        <div className="card mt-6 p-6 text-center">
          <div className="text-5xl font-extrabold text-navy-900">{Math.round(evaluation.overall_score)}</div>
          <div className="mt-1 text-sm text-ink/50">overall score</div>
        </div>
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-5">
          {[
            ["Technical", evaluation.technical_accuracy],
            ["Problem solving", evaluation.problem_solving],
            ["Depth", evaluation.depth_of_knowledge],
            ["Communication", evaluation.communication],
            ["Relevance", evaluation.relevance],
          ].map(([label, value]) => (
            <div key={label as string} className="card p-3 text-center">
              <div className="text-lg font-bold text-navy-900">{Math.round(value as number)}</div>
              <div className="text-xs text-ink/50">{label}</div>
            </div>
          ))}
        </div>
        {evaluation.strengths?.length > 0 && (
          <div className="card mt-4 p-5">
            <h2 className="text-sm font-semibold text-emerald-700">Strengths</h2>
            <ul className="mt-2 space-y-1 text-sm text-ink/70">
              {evaluation.strengths.map((s, i) => (
                <li key={i}>• {s}</li>
              ))}
            </ul>
          </div>
        )}
        {evaluation.weaknesses?.length > 0 && (
          <div className="card mt-4 p-5">
            <h2 className="text-sm font-semibold text-amber-600">Areas to improve</h2>
            <ul className="mt-2 space-y-1 text-sm text-ink/70">
              {evaluation.weaknesses.map((w, i) => (
                <li key={i}>• {w}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    );
  }

  const q = questions[current];
  const allAnswered = questions.length > 0 && questions.every((qq) => answered.has(qq.id));

  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <h1 className="text-display-sm text-navy-950">AI interview</h1>
      <p className="mt-1 text-sm text-ink/60">
        Every question below is grounded in this job&rsquo;s requirements and your own profile — nothing else.
      </p>

      {error && (
        <div className="mt-4">
          <Alert>{error}</Alert>
        </div>
      )}

      <div className="mt-6 flex gap-1.5">
        {questions.map((qq, i) => (
          <div
            key={qq.id}
            className={`h-1.5 flex-1 rounded-full ${
              answered.has(qq.id) ? "bg-emerald-500" : i === current ? "bg-navy-800" : "bg-line"
            }`}
          />
        ))}
      </div>

      {q ? (
        <div className="card mt-6 p-6">
          <span className="text-xs font-medium uppercase tracking-wide text-ink/40">
            Question {current + 1} of {questions.length} ·{" "}
            {q.based_on.replace(/^(job_skill|candidate_project|candidate_experience):/, "")}
          </span>
          <p className="mt-3 text-lg text-ink">{q.question_text}</p>

          <textarea
            className="input-field mt-4 min-h-[140px]"
            placeholder="Type your answer..."
            value={answer}
            onChange={(e) => setAnswer(e.target.value)}
            disabled={answered.has(q.id)}
          />

          <div className="mt-4 flex justify-between">
            <button
              onClick={() => setCurrent((c) => Math.max(0, c - 1))}
              disabled={current === 0}
              className="btn-secondary"
            >
              Previous
            </button>
            {answered.has(q.id) ? (
              <button
                onClick={() => setCurrent((c) => Math.min(questions.length - 1, c + 1))}
                className="btn-secondary"
              >
                Next
              </button>
            ) : (
              <button onClick={submitAnswer} disabled={submitting || !answer.trim()} className="btn-primary">
                {submitting ? "Submitting..." : "Submit answer"}
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="card mt-6 p-8 text-center text-sm text-ink/50">
          No questions were generated for this interview.
        </div>
      )}

      {allAnswered && (
        <button onClick={finishInterview} disabled={submitting} className="btn-accent mt-6 w-full">
          {submitting ? "Evaluating..." : "Finish interview & see results"}
        </button>
      )}
    </div>
  );
}

export default function InterviewPage() {
  return (
    <Protected role="candidate">
      <Navbar />
      <InterviewContent />
    </Protected>
  );
}
