package repository

import (
	"context"

	"golang-dh/internal/modules/company/dto"
	"golang-dh/pkg/database"
)

// CompanyRepository defines the data access contract for perusahaan entities.
type CompanyRepository interface {
	List(ctx context.Context) ([]dto.CompanyItem, error)
	GetByID(ctx context.Context, id int64) (*dto.CompanyItem, error)
	GetByName(ctx context.Context, name string) (*dto.CompanyItem, error)
}

type companyRepository struct {
	db database.PostgreSQLServicer
}

// NewCompanyRepository instantiates a new CompanyRepository.
func NewCompanyRepository(db database.PostgreSQLServicer) CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) List(ctx context.Context) ([]dto.CompanyItem, error) {
	rows, err := r.db.Query(ctx, `SELECT id_perusahaan, nama_perusahaan FROM public.perusahaan ORDER BY id_perusahaan ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.CompanyItem
	for rows.Next() {
		var item dto.CompanyItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if result == nil {
		result = []dto.CompanyItem{}
	}
	return result, nil
}

func (r *companyRepository) GetByID(ctx context.Context, id int64) (*dto.CompanyItem, error) {
	row := r.db.QueryRow(ctx, `SELECT id_perusahaan, nama_perusahaan FROM public.perusahaan WHERE id_perusahaan = $1 LIMIT 1`, id)
	var item dto.CompanyItem
	if err := row.Scan(&item.ID, &item.Name); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *companyRepository) GetByName(ctx context.Context, name string) (*dto.CompanyItem, error) {
	row := r.db.QueryRow(ctx, `SELECT id_perusahaan, nama_perusahaan FROM public.perusahaan WHERE LOWER(nama_perusahaan) = LOWER($1) LIMIT 1`, name)
	var item dto.CompanyItem
	if err := row.Scan(&item.ID, &item.Name); err != nil {
		return nil, err
	}
	return &item, nil
}
