package postgres

import (
	"context"
	"errors"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

var ErrCompanyExclusion = errors.New("company exclusions allow at most 50 names, each containing 1–120 characters")

// Update only the exclusions under a row lock so separate card actions cannot lose each other.
func (s *Store) ExcludeCompany(ctx context.Context, id string) (domain.Preferences, error) {
	var pref domain.Preferences
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var company domain.Company
		if e := tx.First(&company, "id = ?", id).Error; e != nil {
			return e
		}
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pref, "id = ?", ProfileID).Error; e != nil {
			return e
		}
		name := strings.TrimSpace(company.Name)
		for _, existing := range pref.ExcludedCompanies {
			if strings.EqualFold(existing, name) {
				return nil
			}
		}
		if name == "" || len(name) > 120 || len(pref.ExcludedCompanies) >= 50 {
			return ErrCompanyExclusion
		}
		pref.ExcludedCompanies = append(pref.ExcludedCompanies, name)
		return tx.Model(&pref).Select("ExcludedCompanies").Updates(&pref).Error
	})
	return pref, err
}
