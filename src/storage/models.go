package storage

import (
	"time"

	"gorm.io/gorm"
)

const (
	FishTable         = "fish"
	GroupTable        = "groups"
	SensorTable       = "sensors"
	TemperatureTable  = "temperatures"
	TransparencyTable = "transparencies"

	CurrentStatisticTable  = "current_statistics"
	CurrentSensorFishTable = "current_sensor_fishes"
)

type Fish struct {
	gorm.Model

	SensorId uint64
	Name     string
	Count    uint64
}

type Group struct {
	gorm.Model

	Name string
}

type Sensor struct {
	gorm.Model

	GroupId      uint64
	IndexInGroup uint64

	X, Y, Z float64

	DataOutputRate time.Duration
}

type Temperature struct {
	gorm.Model

	SensorId    uint64
	Temperature float64
}

type Transparency struct {
	gorm.Model

	SensorId     uint64
	Transparency uint8
}

type CurrentStatistic struct {
	gorm.Model

	GroupId        uint
	SensorId       uint
	TransparencyId uint
	TemperatureId  uint
}

type CurrentSensorFish struct {
	gorm.Model

	SensorId uint
	FishId   uint
}
