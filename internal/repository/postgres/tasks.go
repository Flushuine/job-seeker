package postgres

import (
	"context"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (s *Store) Enqueue(ctx context.Context, kind, id string, capacity int) (domain.AnalysisTask, error) {
	var task domain.AnalysisTask
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(735016)").Error; e != nil {
			return e
		}
		e := tx.Where("kind = ? AND resource_id = ?", kind, id).First(&task).Error
		if e != nil && e != gorm.ErrRecordNotFound {
			return e
		}
		if task.Status == "queued" || task.Status == "running" {
			return nil
		}
		var n int64
		if e = tx.Model(&domain.AnalysisTask{}).Where("status IN ?", []string{"queued", "running"}).Count(&n).Error; e != nil {
			return e
		}
		if n >= int64(capacity) {
			return ErrLimit
		}
		task = domain.AnalysisTask{Base: base(task.ID), Kind: kind, ResourceID: id, Status: "queued"}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "kind"}, {Name: "resource_id"}}, DoUpdates: clause.Assignments(map[string]any{"status": "queued", "error": "", "attempts": 0, "updated_at": time.Now()})}).Create(&task).Error
	})
	return task, e
}
func (s *Store) Task(ctx context.Context, id string) (domain.AnalysisTask, error) {
	var t domain.AnalysisTask
	e := s.DB.WithContext(ctx).First(&t, "id = ?", id).Error
	return t, e
}
func (s *Store) NextTask(ctx context.Context) (*domain.AnalysisTask, error) {
	var task domain.AnalysisTask
	e := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("status = ?", "queued").Order("created_at").Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).First(&task).Error; e != nil {
			return e
		}
		return tx.Model(&task).Updates(map[string]any{"status": "running", "attempts": gorm.Expr("attempts+1"), "updated_at": time.Now()}).Error
	})
	if e == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &task, e
}
func (s *Store) FinishTask(ctx context.Context, id string, err error) error {
	status := "completed"
	message := ""
	if err != nil {
		status = "failed"
		message = err.Error()
	}
	return s.DB.WithContext(ctx).Model(&domain.AnalysisTask{}).Where("id = ?", id).Updates(map[string]any{"status": status, "error": message, "updated_at": time.Now()}).Error
}
func (s *Store) RecoverTasks(ctx context.Context) error {
	return s.DB.WithContext(ctx).Model(&domain.AnalysisTask{}).Where("status = ?", "running").Updates(map[string]any{"status": "queued", "error": "Recovered after application restart", "updated_at": time.Now()}).Error
}
func (s *Store) SaveEvidence(ctx context.Context, v *domain.Evidence) error {
	v.Base = base("")
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "company_id"}, {Name: "source_url"}}, DoUpdates: clause.AssignmentColumns([]string{"summary", "category", "confidence", "published_at", "retrieved_at", "source_name", "updated_at"})}).Create(v).Error; e != nil {
			return e
		}
		return tx.Model(&domain.Research{}).Where("company_id = ?", v.CompanyID).Updates(map[string]any{"expires_at": time.Now(), "interpretation": nil}).Error
	})
}
func (s *Store) SaveInterpretation(ctx context.Context, id string, v map[string]any) error {
	b := domain.Research{Interpretation: v}
	r := s.DB.WithContext(ctx).Model(&domain.Research{}).Where("company_id = ?", id).Select("Interpretation").Updates(&b)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return fmt.Errorf("collect company research before interpretation")
	}
	return nil
}
