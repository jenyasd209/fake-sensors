package model

import (
	"time"
)

type Fish struct {
	Name  string
	Count uint64
}

type FishArray []*Fish

type SensorGroup map[string]*Sensor

type Coordinate struct {
	X, Y, Z float64
}

type Codename struct {
	Group string
	Index uint64
}

type Sensor struct {
	Codename Codename
	Coordinate
	// seconds
	DataOutputRate time.Duration

	Temperature  float64
	Transparency uint8
	FishList     FishArray
}
