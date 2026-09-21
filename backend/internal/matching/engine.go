// Package matching implements the job-candidate matching engine described
// in the project spec: deterministic, explainable, reproducible scoring,
// with AI used only as an optional semantic-similarity assist layer (see
// ai.Provider.SemanticMatch) rather than as the source of truth. Calling
// Calculate() twice with the same inputs always produces the same output.
//
// SCORING FORMULA (documented, not a black box):
//
//	overall_score = skill_score * weight_skill
//	              + experience_score * weight_experience
//	              + education_score * weight_education
//	              + project_score * weight_project
//	              + certification_score * weight_certification
//
// Each component score is 0-100. Weights are per-job, configurable via
// job_match_weights (see models.MatchWeights), and default to:
// skill 0.40, experience 0.25, education 0.10, project 0.15, certification 0.10
// (documented default rationale: required technical fit matters most,
// experience is the next strongest signal, projects show applied skill,
// education/certifications are supporting signals -- these are starting
// defaults a company can tune per job, not claimed to be universally
// "correct").
package matching

import (
	"time"

	"github.com/resume-platform/backend/internal/models"
	"github.com/resume-platform/backend/internal/utils"
)

type Input struct {
	Job    models.JobFull
	Resume models.ResumeFull
	// SemanticExtra holds AI-suggested additional skill matches (candidate
	// skill <-> job skill pairs the deterministic slug matcher missed).
	// Optional: pass nil/empty to run purely deterministic matching.
	SemanticExtra []SemanticPair
}

type SemanticPair struct {
	JobSkillSlug       string
	CandidateSkillSlug string
	Confidence         float64
}

type SkillMatchDetail struct {
	SkillName  string `json:"skill_name"`
	Required   bool   `json:"required"`
	Matched    bool   `json:"matched"`
	MatchedVia string `json:"matched_via,omitempty"` // "exact" | "alias" | "semantic"
}

type Explanation struct {
	SkillDetails       []SkillMatchDetail `json:"skill_details"`
	RequiredMatched    int                `json:"required_matched"`
	RequiredTotal      int                `json:"required_total"`
	PreferredMatched   int                `json:"preferred_matched"`
	PreferredTotal     int                `json:"preferred_total"`
	CandidateYearsExp  float64            `json:"candidate_years_experience"`
	JobMinYears        int                `json:"job_min_years_required"`
	EducationSatisfied bool               `json:"education_satisfied"`
	MatchedCertCount   int                `json:"matched_certification_count"`
	RequiredCertTotal  int                `json:"required_certification_count"`
	RelevantProjects   []string           `json:"relevant_projects"`
}

type Result struct {
	SkillScore         float64
	ExperienceScore    float64
	EducationScore     float64
	ProjectScore       float64
	CertificationScore float64
	OverallScore       float64
	WeightsUsed        models.MatchWeights
	Explanation        Explanation
}

// Calculate runs the full deterministic scoring pipeline. It never calls
// out to the network -- any AI-assisted semantic matches must already be
// computed and passed in via Input.SemanticExtra.
func Calculate(in Input) Result {
	candidateSkillSlugs := map[string]bool{}
	for _, s := range in.Resume.Skills {
		candidateSkillSlugs[s.SkillSlug] = true
	}

	semanticBySlug := map[string]SemanticPair{}
	for _, p := range in.SemanticExtra {
		if candidateSkillSlugs[p.CandidateSkillSlug] {
			semanticBySlug[p.JobSkillSlug] = p
		}
	}

	var details []SkillMatchDetail
	requiredMatched, requiredTotal := 0, 0
	preferredMatched, preferredTotal := 0, 0
	var weightedRequiredHit, weightedRequiredTotal float64
	var weightedPreferredHit, weightedPreferredTotal float64

	for _, js := range in.Job.Skills {
		matched := false
		via := ""
		if candidateSkillSlugs[js.SkillSlug] {
			matched = true
			via = "exact"
		} else if sp, ok := semanticBySlug[js.SkillSlug]; ok && sp.Confidence >= 0.7 {
			matched = true
			via = "semantic"
		}

		details = append(details, SkillMatchDetail{
			SkillName: js.SkillName, Required: js.IsRequired, Matched: matched, MatchedVia: via,
		})

		w := js.Weight
		if w <= 0 {
			w = 1
		}
		if js.IsRequired {
			requiredTotal++
			weightedRequiredTotal += w
			if matched {
				requiredMatched++
				weightedRequiredHit += w
			}
		} else {
			preferredTotal++
			weightedPreferredTotal += w
			if matched {
				preferredMatched++
				weightedPreferredHit += w
			}
		}
	}

	// Skill score: required skills carry 75% of the skill component,
	// preferred skills 25% -- required-but-missing skills should hurt the
	// score far more than a missing "nice to have".
	requiredFraction := safeDiv(weightedRequiredHit, weightedRequiredTotal)
	preferredFraction := safeDiv(weightedPreferredHit, weightedPreferredTotal)
	skillScore := 0.0
	switch {
	case weightedRequiredTotal > 0 && weightedPreferredTotal > 0:
		skillScore = (requiredFraction*0.75 + preferredFraction*0.25) * 100
	case weightedRequiredTotal > 0:
		skillScore = requiredFraction * 100
	case weightedPreferredTotal > 0:
		skillScore = preferredFraction * 100
	default:
		skillScore = 100 // job specified no skills at all -- nothing to fail on
	}

	// Experience score: candidate's total distinct years of work experience
	// (computed from resume_experience date ranges, see sumYearsExperience)
	// compared against the job's stated minimum. Meeting or exceeding the
	// minimum scores 100; falling short scores proportionally, floored at 0.
	candidateYears := sumYearsExperience(in.Resume.Experience)
	minYears := float64(in.Job.ExperienceMinYears)
	experienceScore := 100.0
	if minYears > 0 {
		experienceScore = clamp((candidateYears/minYears)*100, 0, 100)
	}

	// Education score: binary satisfied/not-satisfied against the job's
	// free-text education requirement, using a simple substring/keyword
	// check against degree+field. This is intentionally conservative
	// (documented limitation): it does not attempt to rank degree
	// prestige or equivalence -- only whether the requirement's keywords
	// appear in the candidate's education records.
	educationScore, educationSatisfied := scoreEducation(in.Job.EducationRequirement, in.Resume.Education)

	// Certification score: fraction of job-required certifications present
	// in the candidate's resume (slug match).
	certScore, matchedCerts := scoreCertifications(in.Job.Certifications, in.Resume.Certifications)

	// Project score: fraction of required job skills that appear in at
	// least one candidate project's technology list -- rewards candidates
	// who have hands-on applied experience with the required stack, not
	// just a bare skills list.
	projectScore, relevantProjects := scoreProjects(in.Job.Skills, in.Resume.Projects)

	w := in.Job.MatchWeights
	overall := skillScore*w.WeightSkill +
		experienceScore*w.WeightExperience +
		educationScore*w.WeightEducation +
		projectScore*w.WeightProject +
		certScore*w.WeightCertification

	return Result{
		SkillScore:         round2(skillScore),
		ExperienceScore:    round2(experienceScore),
		EducationScore:     round2(educationScore),
		ProjectScore:       round2(projectScore),
		CertificationScore: round2(certScore),
		OverallScore:       round2(overall),
		WeightsUsed:        w,
		Explanation: Explanation{
			SkillDetails:       details,
			RequiredMatched:    requiredMatched,
			RequiredTotal:      requiredTotal,
			PreferredMatched:   preferredMatched,
			PreferredTotal:     preferredTotal,
			CandidateYearsExp:  round2(candidateYears),
			JobMinYears:        int(in.Job.ExperienceMinYears),
			EducationSatisfied: educationSatisfied,
			MatchedCertCount:   matchedCerts,
			RequiredCertTotal:  len(in.Job.Certifications),
			RelevantProjects:   relevantProjects,
		},
	}
}

// ExplanationToMap converts Explanation to the generic map the repository
// layer stores as JSON, keeping the models package free of a
// matching-package import (avoids an import cycle: matching depends on
// models, not the reverse).
func ExplanationToMap(e Explanation) models.JSONMap {
	return models.JSONMap{
		"skill_details":                e.SkillDetails,
		"required_matched":             e.RequiredMatched,
		"required_total":               e.RequiredTotal,
		"preferred_matched":            e.PreferredMatched,
		"preferred_total":              e.PreferredTotal,
		"candidate_years_experience":   e.CandidateYearsExp,
		"job_min_years_required":       e.JobMinYears,
		"education_satisfied":          e.EducationSatisfied,
		"matched_certification_count":  e.MatchedCertCount,
		"required_certification_count": e.RequiredCertTotal,
		"relevant_projects":            e.RelevantProjects,
	}
}

func WeightsToMap(w models.MatchWeights) models.JSONMap {
	return models.JSONMap{
		"weight_skill":         w.WeightSkill,
		"weight_experience":    w.WeightExperience,
		"weight_education":     w.WeightEducation,
		"weight_project":       w.WeightProject,
		"weight_certification": w.WeightCertification,
	}
}

// --- helpers ----------------------------------------------------------------

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// sumYearsExperience parses YYYY[-MM] style start/end dates from experience
// entries and sums the total distinct time span covered (overlapping ranges
// are merged, not double-counted) -- a simple, documented, testable
// heuristic rather than a black box.
func sumYearsExperience(exp []models.ResumeExperience) float64 {
	type span struct{ start, end float64 }
	var spans []span
	for _, e := range exp {
		start := parseYearFraction(derefStr(e.StartDate))
		end := parseYearFraction(derefStr(e.EndDate))
		if e.IsCurrent || end == 0 {
			end = currentYearFraction()
		}
		if start == 0 || end <= start {
			continue
		}
		spans = append(spans, span{start, end})
	}
	if len(spans) == 0 {
		return 0
	}
	for i := 0; i < len(spans); i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[i].start <= spans[j].end && spans[j].start <= spans[i].end {
				if spans[j].start < spans[i].start {
					spans[i].start = spans[j].start
				}
				if spans[j].end > spans[i].end {
					spans[i].end = spans[j].end
				}
				spans = append(spans[:j], spans[j+1:]...)
				j--
			}
		}
	}
	total := 0.0
	for _, s := range spans {
		total += s.end - s.start
	}
	return total
}

func scoreEducation(requirement *string, education []models.ResumeEducation) (float64, bool) {
	if requirement == nil || derefStr(requirement) == "" {
		return 100, true
	}
	reqLower := toLower(derefStr(requirement))
	for _, e := range education {
		combined := toLower(derefStr(e.Degree) + " " + derefStr(e.Field))
		if containsAnyWord(combined, reqLower) {
			return 100, true
		}
	}
	if len(education) > 0 {
		return 40, false
	}
	return 0, false
}

func scoreCertifications(required []models.JobCertification, have []models.ResumeCertification) (float64, int) {
	if len(required) == 0 {
		return 100, 0
	}
	haveSlugs := map[string]bool{}
	for _, c := range have {
		haveSlugs[c.NameSlug] = true
	}
	matched := 0
	for _, r := range required {
		if haveSlugs[r.CertificationSlug] {
			matched++
		}
	}
	return (float64(matched) / float64(len(required))) * 100, matched
}

func scoreProjects(jobSkills []models.JobSkill, projects []models.ResumeProject) (float64, []string) {
	var required []models.JobSkill
	for _, s := range jobSkills {
		if s.IsRequired {
			required = append(required, s)
		}
	}
	if len(required) == 0 {
		return 100, nil
	}

	coveredSlugs := map[string]bool{}
	var relevant []string
	for _, p := range projects {
		projectRelevant := false
		for _, tech := range p.Technologies {
			techStr, ok := tech.(string)
			if !ok {
				continue
			}
			techSlug := utils.CanonicalSkillSlug(techStr)
			for _, rs := range required {
				if rs.SkillSlug == techSlug {
					coveredSlugs[rs.SkillSlug] = true
					projectRelevant = true
				}
			}
		}
		if projectRelevant {
			relevant = append(relevant, p.Name)
		}
	}

	return (float64(len(coveredSlugs)) / float64(len(required))) * 100, relevant
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toLower(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func containsAnyWord(haystack, needleSentence string) bool {
	word := ""
	for i := 0; i <= len(needleSentence); i++ {
		if i == len(needleSentence) || needleSentence[i] == ' ' {
			if len(word) > 3 && contains(haystack, word) {
				return true
			}
			word = ""
		} else {
			word += string(needleSentence[i])
		}
	}
	return false
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func parseYearFraction(s string) float64 {
	if len(s) < 4 {
		return 0
	}
	year := 0
	for i := 0; i < 4; i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0
		}
		year = year*10 + int(s[i]-'0')
	}
	month := 1
	if len(s) >= 7 && s[4] == '-' {
		m := 0
		valid := true
		for i := 5; i < 7; i++ {
			if s[i] < '0' || s[i] > '9' {
				valid = false
				break
			}
			m = m*10 + int(s[i]-'0')
		}
		if valid && m >= 1 && m <= 12 {
			month = m
		}
	}
	return float64(year) + float64(month-1)/12
}

func currentYearFraction() float64 {
	now := time.Now()
	return float64(now.Year()) + float64(int(now.Month())-1)/12
}
