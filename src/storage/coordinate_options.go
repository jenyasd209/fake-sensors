package storage

import (
	"gorm.io/gorm"
)

type CoordinateOption func(tx *gorm.DB)

func WithXMin(xMin float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".x >= ?", xMin)
	}
}

func WithXMax(xMax float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".x <= ?", xMax)
	}
}

func WithYMin(yMin float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".y >= ?", yMin)
	}
}

func WithYMax(yMax float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".y <= ?", yMax)
	}
}

func WithZMin(zMin float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".z >= ?", zMin)
	}
}

func WithZMax(zMax float64) CoordinateOption {
	return func(tx *gorm.DB) {
		tx.Where(SensorTable+".z <= ?", zMax)
	}
}
