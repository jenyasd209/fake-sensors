package storage

import (
	"time"

	"gorm.io/gorm"
)

type ConditionOption func(table, field string, tx *gorm.DB)

func WithFrom(from time.Time) ConditionOption {
	return func(table, field string, tx *gorm.DB) {
		tx.Where(table+"."+field+" >= ?", from)
	}
}

func WithTill(till time.Time) ConditionOption {
	return func(table, field string, tx *gorm.DB) {
		tx.Where(table+"."+field+" <= ?", till)
	}
}
