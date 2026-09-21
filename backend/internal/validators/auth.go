// Package validators holds request DTOs with gin/go-playground validator
// binding tags. Controllers bind directly into these structs via
// c.ShouldBindJSON, so invalid requests are rejected before any service
// code runs -- see middleware/error_handler.go for how binding errors are
// converted into the standard 422 VALIDATION_ERROR response.
package validators

type RegisterCandidateRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	FullName string `json:"full_name" binding:"required,min=2,max=255"`
	Phone    string `json:"phone" binding:"omitempty,max=50"`
}

type RegisterCompanyRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8,max=72"`
	CompanyName string `json:"company_name" binding:"required,min=2,max=255"`
	Website     string `json:"website" binding:"omitempty,max=255"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type GoogleCallbackRequest struct {
	Code string `form:"code" binding:"required"`
}
