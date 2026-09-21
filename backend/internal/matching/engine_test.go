package matching

import (
	"testing"

	"github.com/resume-platform/backend/internal/models"
)

func defaultWeights() models.MatchWeights {
	return models.MatchWeights{
		WeightSkill: 0.40, WeightExperience: 0.25, WeightEducation: 0.10,
		WeightProject: 0.15, WeightCertification: 0.10,
	}
}

func jobFixture() models.JobFull {
	return models.JobFull{
		Job: models.Job{
			ExperienceMinYears:   1,
			EducationRequirement: strPtr("Bachelor degree in Computer Science"),
		},
		Skills: []models.JobSkill{
			{SkillName: "Go", SkillSlug: "go", IsRequired: true, Weight: 2},
			{SkillName: "MySQL", SkillSlug: "mysql", IsRequired: true, Weight: 1.5},
			{SkillName: "REST API", SkillSlug: "rest-api", IsRequired: true, Weight: 1.5},
			{SkillName: "Docker", SkillSlug: "docker", IsRequired: false, Weight: 1},
			{SkillName: "Git", SkillSlug: "git", IsRequired: false, Weight: 1},
		},
		MatchWeights: defaultWeights(),
	}
}

func strPtr(s string) *string { return &s }

func TestCalculate_PerfectMatch(t *testing.T) {
	job := jobFixture()
	resume := models.ResumeFull{
		Skills: []models.ResumeSkill{
			{SkillName: "Go", SkillSlug: "go"},
			{SkillName: "MySQL", SkillSlug: "mysql"},
			{SkillName: "REST API", SkillSlug: "rest-api"},
			{SkillName: "Docker", SkillSlug: "docker"},
			{SkillName: "Git", SkillSlug: "git"},
		},
		Experience: []models.ResumeExperience{
			{Company: "Acme", Role: "Backend Dev", StartDate: strPtr("2020-01"), EndDate: strPtr("2023-01")},
		},
		Education: []models.ResumeEducation{
			{Institution: "State University", Degree: strPtr("Bachelor of Science"), Field: strPtr("Computer Science")},
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})

	if result.SkillScore != 100 {
		t.Errorf("expected skill score 100, got %v", result.SkillScore)
	}
	if result.EducationScore != 100 {
		t.Errorf("expected education score 100, got %v", result.EducationScore)
	}
	if !result.Explanation.EducationSatisfied {
		t.Errorf("expected education satisfied")
	}
	if result.Explanation.RequiredMatched != 3 || result.Explanation.RequiredTotal != 3 {
		t.Errorf("expected 3/3 required matched, got %d/%d", result.Explanation.RequiredMatched, result.Explanation.RequiredTotal)
	}
}

func TestCalculate_MissingRequiredSkillHurtsMoreThanPreferred(t *testing.T) {
	job := jobFixture()

	// candidate has ALL preferred skills but ZERO required skills
	resumeNoRequired := models.ResumeFull{
		Skills: []models.ResumeSkill{
			{SkillName: "Docker", SkillSlug: "docker"},
			{SkillName: "Git", SkillSlug: "git"},
		},
	}
	// candidate has ALL required skills but ZERO preferred skills
	resumeNoPreferred := models.ResumeFull{
		Skills: []models.ResumeSkill{
			{SkillName: "Go", SkillSlug: "go"},
			{SkillName: "MySQL", SkillSlug: "mysql"},
			{SkillName: "REST API", SkillSlug: "rest-api"},
		},
	}

	rNoRequired := Calculate(Input{Job: job, Resume: resumeNoRequired})
	rNoPreferred := Calculate(Input{Job: job, Resume: resumeNoPreferred})

	if rNoRequired.SkillScore >= rNoPreferred.SkillScore {
		t.Errorf("missing all required skills should score lower than missing all preferred skills: got %v vs %v",
			rNoRequired.SkillScore, rNoPreferred.SkillScore)
	}
	// documented formula: required carries 75%% of skill score
	if rNoPreferred.SkillScore != 75 {
		t.Errorf("expected 75 (100%% required * 0.75 weight + 0%% preferred), got %v", rNoPreferred.SkillScore)
	}
	if rNoRequired.SkillScore != 25 {
		t.Errorf("expected 25 (0%% required * 0.75 + 100%% preferred * 0.25), got %v", rNoRequired.SkillScore)
	}
}

func TestCalculate_NoSkillsAtAll(t *testing.T) {
	job := jobFixture()
	resume := models.ResumeFull{}

	result := Calculate(Input{Job: job, Resume: resume})
	if result.SkillScore != 0 {
		t.Errorf("expected skill score 0 for a candidate with no matching skills, got %v", result.SkillScore)
	}
	for _, d := range result.Explanation.SkillDetails {
		if d.Matched {
			t.Errorf("no skill should be marked matched, got %+v", d)
		}
	}
}

func TestCalculate_JobWithNoSkillsRequiresNothing(t *testing.T) {
	job := models.JobFull{Job: models.Job{}, MatchWeights: defaultWeights()}
	resume := models.ResumeFull{}

	result := Calculate(Input{Job: job, Resume: resume})
	if result.SkillScore != 100 {
		t.Errorf("a job with no skill requirements should score candidates 100 on skills, got %v", result.SkillScore)
	}
}

func TestCalculate_ExperienceScoreProportionalBelowMinimum(t *testing.T) {
	job := jobFixture()
	job.ExperienceMinYears = 4 // require 4 years

	resume := models.ResumeFull{
		Experience: []models.ResumeExperience{
			{Company: "Acme", Role: "Dev", StartDate: strPtr("2022-01"), EndDate: strPtr("2024-01")}, // 2 years
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})
	// 2 years / 4 years required = 50%
	if result.ExperienceScore < 45 || result.ExperienceScore > 55 {
		t.Errorf("expected ~50 experience score for 2/4 years, got %v", result.ExperienceScore)
	}
}

func TestCalculate_ExperienceScoreCapsAt100(t *testing.T) {
	job := jobFixture()
	job.ExperienceMinYears = 1

	resume := models.ResumeFull{
		Experience: []models.ResumeExperience{
			{Company: "Acme", Role: "Dev", StartDate: strPtr("2010-01"), EndDate: strPtr("2024-01")}, // 14 years
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})
	if result.ExperienceScore != 100 {
		t.Errorf("experience score should cap at 100 even when candidate wildly exceeds minimum, got %v", result.ExperienceScore)
	}
}

func TestCalculate_OverlappingExperienceNotDoubleCounted(t *testing.T) {
	job := jobFixture()
	job.ExperienceMinYears = 2

	// two overlapping jobs covering the same 2020-2022 window should count
	// as 2 years total, not 4.
	resume := models.ResumeFull{
		Experience: []models.ResumeExperience{
			{Company: "A", Role: "Dev", StartDate: strPtr("2020-01"), EndDate: strPtr("2022-01")},
			{Company: "B", Role: "Contractor", StartDate: strPtr("2020-06"), EndDate: strPtr("2021-06")},
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})
	if result.ExperienceScore != 100 {
		t.Errorf("overlapping experience spans should merge to ~2 years (100%% of 2yr minimum), got score %v (years=%v)",
			result.ExperienceScore, result.Explanation.CandidateYearsExp)
	}
	if result.Explanation.CandidateYearsExp > 2.1 {
		t.Errorf("expected merged experience ~2 years, got %v (overlap was double-counted)", result.Explanation.CandidateYearsExp)
	}
}

func TestCalculate_CertificationScore(t *testing.T) {
	job := jobFixture()
	job.Certifications = []models.JobCertification{
		{CertificationName: "AWS Certified Developer", CertificationSlug: "aws-certified-developer"},
		{CertificationName: "CKA", CertificationSlug: "cka"},
	}

	resume := models.ResumeFull{
		Certifications: []models.ResumeCertification{
			{Name: "AWS Certified Developer", NameSlug: "aws-certified-developer"},
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})
	if result.CertificationScore != 50 {
		t.Errorf("expected 50 (1 of 2 required certs), got %v", result.CertificationScore)
	}
	if result.Explanation.MatchedCertCount != 1 {
		t.Errorf("expected matched cert count 1, got %d", result.Explanation.MatchedCertCount)
	}
}

func TestCalculate_ProjectScoreRewardsAppliedSkills(t *testing.T) {
	job := jobFixture() // required: go, mysql, rest-api

	resumeWithProject := models.ResumeFull{
		Projects: []models.ResumeProject{
			{Name: "API Service", Technologies: models.JSONList{"Go", "MySQL", "REST API"}},
		},
	}
	resumeWithoutProject := models.ResumeFull{}

	withProject := Calculate(Input{Job: job, Resume: resumeWithProject})
	withoutProject := Calculate(Input{Job: job, Resume: resumeWithoutProject})

	if withProject.ProjectScore != 100 {
		t.Errorf("expected 100 project score when project covers all required skills, got %v", withProject.ProjectScore)
	}
	if withoutProject.ProjectScore != 0 {
		t.Errorf("expected 0 project score with no projects, got %v", withoutProject.ProjectScore)
	}
}

func TestCalculate_OverallScoreFormula(t *testing.T) {
	// Directly verify the documented formula:
	// overall = skill*0.4 + experience*0.25 + education*0.1 + project*0.15 + cert*0.1
	job := jobFixture()
	resume := models.ResumeFull{
		Skills: []models.ResumeSkill{
			{SkillName: "Go", SkillSlug: "go"}, {SkillName: "MySQL", SkillSlug: "mysql"}, {SkillName: "REST API", SkillSlug: "rest-api"},
		},
	}

	result := Calculate(Input{Job: job, Resume: resume})
	expected := result.SkillScore*0.40 + result.ExperienceScore*0.25 + result.EducationScore*0.10 +
		result.ProjectScore*0.15 + result.CertificationScore*0.10
	expected = round2(expected)

	if result.OverallScore != expected {
		t.Errorf("overall score %v does not match documented formula result %v", result.OverallScore, expected)
	}
}

func TestCalculate_Reproducible(t *testing.T) {
	job := jobFixture()
	resume := models.ResumeFull{
		Skills: []models.ResumeSkill{{SkillName: "Go", SkillSlug: "go"}},
	}

	r1 := Calculate(Input{Job: job, Resume: resume})
	r2 := Calculate(Input{Job: job, Resume: resume})

	if r1.OverallScore != r2.OverallScore || r1.SkillScore != r2.SkillScore {
		t.Errorf("Calculate() must be deterministic/reproducible for identical inputs, got %v vs %v", r1, r2)
	}
}

func TestCalculate_SemanticMatchAssist(t *testing.T) {
	job := jobFixture() // requires "go" among others
	resume := models.ResumeFull{
		Skills: []models.ResumeSkill{{SkillName: "Programming in Go-like languages", SkillSlug: "programming-in-go-like-languages"}},
	}

	withoutSemantic := Calculate(Input{Job: job, Resume: resume})
	withSemantic := Calculate(Input{
		Job: job, Resume: resume,
		SemanticExtra: []SemanticPair{
			{JobSkillSlug: "go", CandidateSkillSlug: "programming-in-go-like-languages", Confidence: 0.9},
		},
	})

	if withSemantic.SkillScore <= withoutSemantic.SkillScore {
		t.Errorf("a high-confidence semantic match should improve skill score: %v vs %v", withoutSemantic.SkillScore, withSemantic.SkillScore)
	}
}

func TestCalculate_LowConfidenceSemanticMatchIgnored(t *testing.T) {
	job := jobFixture()
	resume := models.ResumeFull{
		Skills: []models.ResumeSkill{{SkillName: "Something Unrelated", SkillSlug: "something-unrelated"}},
	}

	result := Calculate(Input{
		Job: job, Resume: resume,
		SemanticExtra: []SemanticPair{
			{JobSkillSlug: "go", CandidateSkillSlug: "something-unrelated", Confidence: 0.3}, // below 0.7 threshold
		},
	})

	for _, d := range result.Explanation.SkillDetails {
		if d.SkillName == "Go" && d.Matched {
			t.Errorf("low-confidence semantic match (0.3) should not count as a match")
		}
	}
}
