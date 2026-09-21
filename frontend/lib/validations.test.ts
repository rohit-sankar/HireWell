import { describe, it, expect } from "vitest";
import { loginSchema, registerCandidateSchema, registerCompanySchema, createJobSchema } from "@/lib/validations";

describe("loginSchema", () => {
  it("accepts a valid email and non-empty password", () => {
    const result = loginSchema.safeParse({ email: "jane@example.com", password: "anything" });
    expect(result.success).toBe(true);
  });

  it("rejects an invalid email", () => {
    const result = loginSchema.safeParse({ email: "not-an-email", password: "x" });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].path).toEqual(["email"]);
    }
  });

  it("rejects an empty password", () => {
    const result = loginSchema.safeParse({ email: "jane@example.com", password: "" });
    expect(result.success).toBe(false);
  });

  it("rejects a missing email entirely", () => {
    const result = loginSchema.safeParse({ password: "x" });
    expect(result.success).toBe(false);
  });
});

describe("registerCandidateSchema", () => {
  const valid = { full_name: "John Doe", email: "john@example.com", password: "SecurePass123" };

  it("accepts valid candidate registration input", () => {
    expect(registerCandidateSchema.safeParse(valid).success).toBe(true);
  });

  it("rejects a password under 8 characters", () => {
    const result = registerCandidateSchema.safeParse({ ...valid, password: "short" });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.some((i) => i.path[0] === "password")).toBe(true);
    }
  });

  it("rejects a name under 2 characters", () => {
    const result = registerCandidateSchema.safeParse({ ...valid, full_name: "J" });
    expect(result.success).toBe(false);
  });

  it("allows phone to be omitted (optional field)", () => {
    const { phone, ...withoutPhone } = { ...valid, phone: "555-1234" };
    expect(registerCandidateSchema.safeParse(withoutPhone).success).toBe(true);
  });
});

describe("registerCompanySchema", () => {
  it("accepts valid company registration input", () => {
    const result = registerCompanySchema.safeParse({
      company_name: "Acme Corp",
      email: "hr@acme.com",
      password: "SecurePass123",
    });
    expect(result.success).toBe(true);
  });

  it("rejects a company name under 2 characters", () => {
    const result = registerCompanySchema.safeParse({
      company_name: "A",
      email: "hr@acme.com",
      password: "SecurePass123",
    });
    expect(result.success).toBe(false);
  });
});

describe("createJobSchema", () => {
  const valid = {
    title: "Backend Developer",
    description: "This is a sufficiently long job description for validation purposes.",
    employment_type: "full_time" as const,
    experience_min_years: 2,
  };

  it("accepts a valid job posting", () => {
    expect(createJobSchema.safeParse(valid).success).toBe(true);
  });

  it("rejects a title shorter than 3 characters", () => {
    const result = createJobSchema.safeParse({ ...valid, title: "Go" });
    expect(result.success).toBe(false);
  });

  it("rejects a description under 20 characters", () => {
    const result = createJobSchema.safeParse({ ...valid, description: "too short" });
    expect(result.success).toBe(false);
  });

  it("rejects an invalid employment_type", () => {
    const result = createJobSchema.safeParse({ ...valid, employment_type: "gig_work" });
    expect(result.success).toBe(false);
  });

  it("coerces a numeric string for experience_min_years", () => {
    const result = createJobSchema.safeParse({ ...valid, experience_min_years: "3" });
    expect(result.success).toBe(true);
    if (result.success) expect(result.data.experience_min_years).toBe(3);
  });

  it("rejects negative experience years", () => {
    const result = createJobSchema.safeParse({ ...valid, experience_min_years: -1 });
    expect(result.success).toBe(false);
  });
});
