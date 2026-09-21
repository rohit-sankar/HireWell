import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().min(1, "Email is required").email("Enter a valid email address"),
  password: z.string().min(1, "Password is required"),
});

export const registerCandidateSchema = z.object({
  full_name: z.string().min(2, "Enter your full name"),
  email: z.string().min(1, "Email is required").email("Enter a valid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
  phone: z.string().optional(),
});

export const registerCompanySchema = z.object({
  company_name: z.string().min(2, "Enter your company name"),
  email: z.string().min(1, "Email is required").email("Enter a valid email address"),
  password: z.string().min(8, "Password must be at least 8 characters"),
  website: z.string().optional(),
});

export const createJobSchema = z.object({
  title: z.string().min(3, "Title must be at least 3 characters"),
  description: z.string().min(20, "Description should be at least 20 characters"),
  location: z.string().optional(),
  employment_type: z.enum(["full_time", "part_time", "contract", "internship", "remote"]),
  experience_min_years: z.coerce.number().min(0).max(50),
  education_requirement: z.string().optional(),
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterCandidateInput = z.infer<typeof registerCandidateSchema>;
export type RegisterCompanyInput = z.infer<typeof registerCompanySchema>;
export type CreateJobInput = z.infer<typeof createJobSchema>;
