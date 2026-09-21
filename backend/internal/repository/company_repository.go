package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/resume-platform/backend/internal/models"
)

type CompanyRepository struct {
	db *sql.DB
}

func NewCompanyRepository(db *sql.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

func (r *CompanyRepository) Create(ctx context.Context, c *models.Company) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO companies (user_id, company_name, website, description, logo_url, industry, company_size)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.UserID, c.CompanyName, c.Website, c.Description, c.LogoURL, c.Industry, c.CompanySize,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const companySelectCols = `co.id, co.user_id, co.company_name, co.website, co.description, co.logo_url, co.industry,
	co.company_size, co.created_at, co.updated_at, u.email`

func (r *CompanyRepository) FindByUserID(ctx context.Context, userID int64) (*models.Company, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+companySelectCols+` FROM companies co JOIN users u ON u.id = co.user_id WHERE co.user_id = ?`, userID)
	return scanCompany(row)
}

func (r *CompanyRepository) FindByID(ctx context.Context, id int64) (*models.Company, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+companySelectCols+` FROM companies co JOIN users u ON u.id = co.user_id WHERE co.id = ?`, id)
	return scanCompany(row)
}

func scanCompany(row *sql.Row) (*models.Company, error) {
	var c models.Company
	err := row.Scan(&c.ID, &c.UserID, &c.CompanyName, &c.Website, &c.Description, &c.LogoURL, &c.Industry,
		&c.CompanySize, &c.CreatedAt, &c.UpdatedAt, &c.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CompanyRepository) Update(ctx context.Context, c *models.Company) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE companies SET company_name = ?, website = ?, description = ?, logo_url = ?, industry = ?, company_size = ?
		 WHERE id = ?`,
		c.CompanyName, c.Website, c.Description, c.LogoURL, c.Industry, c.CompanySize, c.ID,
	)
	return err
}
