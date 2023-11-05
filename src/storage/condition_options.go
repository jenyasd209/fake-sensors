package storage

import (
	"time"

	"gorm.io/gorm"
)

type ConditionOption func(table string, tx *gorm.DB)

func WithCreatedFrom(from time.Time) ConditionOption {
	return func(table string, tx *gorm.DB) {
		tx.Where(table+".created_at >= ?", from)
	}
}

func WithCreatedTill(till time.Time) ConditionOption {
	return func(table string, tx *gorm.DB) {
		tx.Where(table+".created_at <= ?", till)
	}
}

func WithCreatedBetween(from time.Time, till time.Time) ConditionOption {
	return func(table string, tx *gorm.DB) {
		tx.Where(table+".created_at BETWEEN ? AND ?", from, till)
	}
}
