package validators

type JobSkillRequest struct {
	Name       string  `json:"name" binding:"required"`
	IsRequired bool    `json:"is_required"`
	Weight     float64 `json:"weight"`
}

type CreateJobRequest struct {
	Title                  string            `json:"title" binding:"required,min=3,max=255"`
	Description            string            `json:"description" binding:"required,min=10"`
	Location               string            `json:"location"`
	EmploymentType         string            `json:"employment_type" binding:"omitempty,oneof=full_time part_time contract internship remote"`
	ExperienceMinYears     uint8             `json:"experience_min_years"`
	ExperienceMaxYears     *uint8            `json:"experience_max_years"`
	EducationRequirement   string            `json:"education_requirement"`
	InterviewQuestionCount uint8             `json:"interview_question_count"`
	Skills                 []JobSkillRequest `json:"skills"`
	Certifications         []string          `json:"certifications"`
}

type UpdateMatchWeightsRequest struct {
	WeightSkill         float64 `json:"weight_skill" binding:"required,gte=0,lte=1"`
	WeightExperience    float64 `json:"weight_experience" binding:"required,gte=0,lte=1"`
	WeightEducation     float64 `json:"weight_education" binding:"required,gte=0,lte=1"`
	WeightProject       float64 `json:"weight_project" binding:"required,gte=0,lte=1"`
	WeightCertification float64 `json:"weight_certification" binding:"required,gte=0,lte=1"`
}

type UpdateApplicationStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=under_review shortlisted rejected hired"`
}

type SubmitAnswerRequest struct {
	QuestionID int64  `json:"question_id" binding:"required"`
	Answer     string `json:"answer" binding:"required,min=1"`
}
