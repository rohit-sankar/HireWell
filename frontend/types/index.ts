export type Role = "candidate" | "company" | "admin";

export interface User {
  id: number;
  email: string;
  role: Role;
  is_active: boolean;
  email_verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface Candidate {
  id: number;
  user_id: number;
  full_name: string;
  phone?: string | null;
  location?: string | null;
  links?: Record<string, unknown>;
  headline?: string | null;
  summary?: string | null;
  active_resume_id?: number | null;
  profile_completion: number;
  created_at: string;
  updated_at: string;
}

export interface Company {
  id: number;
  user_id: number;
  company_name: string;
  website?: string | null;
  description?: string | null;
  logo_url?: string | null;
  industry?: string | null;
  company_size?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ResumeEducation {
  id?: number;
  institution: string;
  degree?: string | null;
  field?: string | null;
  start_date?: string | null;
  end_date?: string | null;
  gpa?: string | null;
}

export interface ResumeExperience {
  id?: number;
  company: string;
  role: string;
  start_date?: string | null;
  end_date?: string | null;
  is_current: boolean;
  description?: string | null;
  technologies: string[];
  responsibilities: string[];
}

export interface ResumeProject {
  id?: number;
  name: string;
  description?: string | null;
  technologies: string[];
  role?: string | null;
  links?: Record<string, unknown>;
}

export interface ResumeSkill {
  id?: number;
  skill_name: string;
  skill_slug?: string;
  category: string;
}

export interface ResumeCertification {
  id?: number;
  name: string;
  issuer?: string | null;
  issue_date?: string | null;
  credential_id?: string | null;
}

export interface ResumeFull {
  id: number;
  candidate_id: number;
  original_filename: string;
  status: "uploaded" | "processing" | "parsed" | "failed";
  parse_error?: string | null;
  uploaded_at: string;
  parsed_at?: string | null;
  education: ResumeEducation[] | null;
  experience: ResumeExperience[] | null;
  projects: ResumeProject[] | null;
  skills: ResumeSkill[] | null;
  certifications: ResumeCertification[] | null;
}

export interface JobSkill {
  id?: number;
  skill_name: string;
  is_required: boolean;
  weight: number;
}

export interface Job {
  id: number;
  company_id: number;
  title: string;
  description: string;
  location?: string | null;
  employment_type: string;
  experience_min_years: number;
  experience_max_years?: number | null;
  education_requirement?: string | null;
  status: "draft" | "published" | "closed" | "archived";
  interview_question_count: number;
  company_name?: string;
  created_at: string;
  published_at?: string | null;
  skills?: JobSkill[];
  certifications?: { certification_name: string }[];
}

export interface Application {
  id: number;
  job_id: number;
  candidate_id: number;
  resume_id: number;
  status: "applied" | "under_review" | "shortlisted" | "rejected" | "hired";
  applied_at: string;
  job_title?: string;
  company_name?: string;
  candidate_name?: string;
}

export interface MatchResult {
  id: number;
  application_id: number;
  skill_score: number;
  experience_score: number;
  education_score: number;
  project_score: number;
  certification_score: number;
  overall_score: number;
  explanation: {
    skill_details: { skill_name: string; required: boolean; matched: boolean; matched_via?: string }[];
    required_matched: number;
    required_total: number;
    preferred_matched: number;
    preferred_total: number;
    candidate_years_experience: number;
    job_min_years_required: number;
    education_satisfied: boolean;
    relevant_projects: string[] | null;
  };
}

export interface RankedApplication extends Application {
  MatchResult?: MatchResult;
}

export interface Interview {
  id: number;
  application_id: number;
  status: "pending" | "in_progress" | "completed";
  started_at?: string | null;
  completed_at?: string | null;
}

export interface InterviewQuestion {
  id: number;
  interview_id: number;
  question_text: string;
  based_on: string;
  order_index: number;
}

export interface InterviewEvaluation {
  id: number;
  interview_id: number;
  technical_accuracy: number;
  problem_solving: number;
  depth_of_knowledge: number;
  communication: number;
  relevance: number;
  overall_score: number;
  strengths: string[];
  weaknesses: string[];
  recommendation?: string | null;
}

export interface ApiSuccess<T> {
  success: true;
  data: T;
  message?: string;
  meta?: Record<string, unknown>;
}

export interface ApiError {
  success: false;
  error: { code: string; message: string; details?: unknown };
}
