package postgres

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (s *Store) PatchCompanyProfile(ctx context.Context, id string, updates domain.CompanyProfile) (domain.Company, error) {
	var c domain.Company
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&c, "id = ?", id).Error; e != nil {
			return e
		}
		profile, e := domain.MergeCompanyProfile(c.Profile, updates, time.Now())
		if e != nil {
			return e
		}
		c.Profile = profile
		return tx.Model(&c).Select("Profile").Updates(&c).Error
	})
	return c, e
}
