import type {
  ApiError,
  ApiSuccess,
  Application,
  Candidate,
  Company,
  Interview,
  InterviewEvaluation,
  InterviewQuestion,
  Job,
  MatchResult,
  RankedApplication,
  ResumeFull,
  User,
} from "@/types";

const API_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

// The access token lives only in memory (a module-level variable), never in
// localStorage/sessionStorage -- this is deliberate XSS hardening: a script
// injected into the page cannot read a token that isn't in any browser
// storage API. On page load, AuthProvider calls refresh() (which relies on
// the httpOnly refresh-token cookie) to silently re-hydrate this value.
let accessToken: string | null = null;
let refreshPromise: Promise<boolean> | null = null;
export function setAccessToken(token: string | null) {
  accessToken = token;
}
export function getAccessToken() {
  return accessToken;
}

export class ApiRequestError extends Error {
  code: string;
  status: number;
  details?: unknown;
  constructor(
    status: number,
    code: string,
    message: string,
    details?: unknown,
  ) {
    super(message);
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  formData?: FormData;
  skipAuthRetry?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {};
  if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;

  let bodyInit: BodyInit | undefined;
  if (opts.formData) {
    bodyInit = opts.formData; // browser sets multipart Content-Type + boundary
  } else if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    bodyInit = JSON.stringify(opts.body);
  }

  const res = await fetch(`${API_URL}${path}`, {
    method: opts.method || "GET",
    headers,
    body: bodyInit,
    credentials: "include", // send the httpOnly refresh_token cookie
  });

  // Access tokens are short-lived (15m default). On a 401 (excluding the
  // refresh call itself, to avoid an infinite loop), try one silent refresh
  // and retry the original request once -- this is what lets a user's
  // session survive well past the access token's expiry without them
  // noticing, using the refresh-token rotation flow.
  if (res.status === 401 && !opts.skipAuthRetry && path !== "/auth/refresh") {
    const refreshed = await tryRefresh();
    if (refreshed) {
      return request<T>(path, { ...opts, skipAuthRetry: true });
    }
  }

  const json = (await res.json().catch(() => null)) as
    | ApiSuccess<T>
    | ApiError
    | null;

  if (!res.ok || !json || json.success === false) {
    const err =
      json && json.success === false
        ? json.error
        : { code: "UNKNOWN", message: `Request failed (${res.status})` };
    throw new ApiRequestError(res.status, err.code, err.message, err.details);
  }

  return (json as ApiSuccess<T>).data;
}

async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise;

  refreshPromise = performRefresh();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

async function performRefresh(): Promise<boolean> {
  try {
    const res = await fetch(`${API_URL}/auth/refresh`, {
      method: "POST",
      credentials: "include",
    });
    const json = (await res.json().catch(() => null)) as
      | ApiSuccess<{ access_token: string }>
      | ApiError
      | null;
    if (res.ok && json && json.success) {
      setAccessToken(json.data.access_token);
      return true;
    }
  } catch {
    // network error during refresh -- treat as not-authenticated
  }
  setAccessToken(null);
  return false;
}

// --- auth ---------------------------------------------------------------

export interface AuthResponse {
  user: User;
  access_token: string;
  expires_at: string;
}

export const authApi = {
  registerCandidate: (input: {
    email: string;
    password: string;
    full_name: string;
    phone?: string;
  }) =>
    request<AuthResponse>("/auth/register/candidate", {
      method: "POST",
      body: input,
    }),
  registerCompany: (input: {
    email: string;
    password: string;
    company_name: string;
    website?: string;
  }) =>
    request<AuthResponse>("/auth/register/company", {
      method: "POST",
      body: input,
    }),
  login: (input: { email: string; password: string }) =>
    request<AuthResponse>("/auth/login", { method: "POST", body: input }),
  logout: () => request<null>("/auth/logout", { method: "POST" }),
  me: () => request<User>("/auth/me"),
  refresh: tryRefresh,
  googleLoginUrl: () => `${API_URL}/auth/google`,
};

// --- candidates -----------------------------------------------------------

export const candidateApi = {
  me: () => request<Candidate>("/candidates/me"),
  updateMe: (input: Partial<Candidate>) =>
    request<Candidate>("/candidates/me", { method: "PUT", body: input }),
  getResume: () => request<ResumeFull>("/candidates/me/resume"),
  uploadResume: (file: File) => {
    const fd = new FormData();
    fd.append("resume", file);
    return request<ResumeFull>("/candidates/me/resume", {
      method: "POST",
      formData: fd,
    });
  },
  updateResume: (resumeId: number, data: Partial<ResumeFull>) =>
    request<ResumeFull>(`/candidates/me/resume/${resumeId}`, {
      method: "PUT",
      body: data,
    }),
  myApplications: async () =>
    (await request<Application[] | null>("/candidates/me/applications")) ?? [],
};

// --- companies ------------------------------------------------------------

export const companyApi = {
  me: () => request<Company>("/companies/me"),
  updateMe: (input: Partial<Company>) =>
    request<Company>("/companies/me", { method: "PUT", body: input }),
};

// --- jobs -------------------------------------------------------------------

export interface CreateJobPayload {
  title: string;
  description: string;
  location?: string;
  employment_type?: string;
  experience_min_years?: number;
  experience_max_years?: number;
  education_requirement?: string;
  interview_question_count?: number;
  skills?: { name: string; is_required: boolean; weight: number }[];
  certifications?: string[];
}

export const jobsApi = {
  list: async (
    params: { search?: string; page?: number; mine?: boolean } = {},
  ) => {
    const q = new URLSearchParams();
    if (params.search) q.set("search", params.search);
    if (params.page) q.set("page", String(params.page));
    if (params.mine) q.set("mine", "true");
    const qs = q.toString();
    return (await request<Job[] | null>(`/jobs${qs ? `?${qs}` : ""}`)) ?? [];
  },
  get: (id: number) => request<Job>(`/jobs/${id}`),
  create: (input: CreateJobPayload) =>
    request<Job>("/jobs", { method: "POST", body: input }),
  update: (id: number, input: Partial<CreateJobPayload>) =>
    request<Job>(`/jobs/${id}`, { method: "PUT", body: input }),
  publish: (id: number) =>
    request<Job>(`/jobs/${id}/publish`, { method: "POST" }),
  close: (id: number) => request<Job>(`/jobs/${id}/close`, { method: "POST" }),
  archive: (id: number) =>
    request<Job>(`/jobs/${id}/archive`, { method: "POST" }),
  remove: (id: number) => request<null>(`/jobs/${id}`, { method: "DELETE" }),
  applications: async (id: number) =>
    (await request<Application[] | null>(`/jobs/${id}/applications`)) ?? [],
  matches: async (id: number) =>
    (await request<RankedApplication[] | null>(`/jobs/${id}/matches`)) ?? [],
};

// --- applications ---------------------------------------------------------

export const applicationsApi = {
  apply: (jobId: number) =>
    request<Application>(`/jobs/${jobId}/apply`, { method: "POST" }),
  get: (id: number) => request<Application>(`/applications/${id}`),
  updateStatus: (id: number, status: Application["status"]) =>
    request<Application>(`/applications/${id}/status`, {
      method: "PUT",
      body: { status },
    }),
  getMatch: (id: number) => request<MatchResult>(`/applications/${id}/match`),
};

// --- interviews -------------------------------------------------------------

export const interviewsApi = {
  create: (applicationId: number) =>
    request<Interview>("/interviews", {
      method: "POST",
      body: { application_id: applicationId },
    }),
  get: (id: number) =>
    request<{ interview: Interview; questions: InterviewQuestion[] }>(
      `/interviews/${id}`,
    ),
  start: (id: number) =>
    request<Interview>(`/interviews/${id}/start`, { method: "POST" }),
  answer: (id: number, questionId: number, answer: string) =>
    request<null>(`/interviews/${id}/answer`, {
      method: "POST",
      body: { question_id: questionId, answer },
    }),
  complete: (id: number) =>
    request<InterviewEvaluation>(`/interviews/${id}/complete`, {
      method: "POST",
    }),
  result: (id: number) =>
    request<InterviewEvaluation>(`/interviews/${id}/result`),
};
