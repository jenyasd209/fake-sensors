package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	temperatureKey  = "avgTemperature"
	transparencyKey = "avgTransparency"

	minTemperature = 0
	maxTemperature = 1
)

var ErrNoSensorsInArea = errors.New("no sensors in this area")

type Storage struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewStorage(opts ...Option) (*Storage, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	db, err := connectToDb(options)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			db, _ := db.DB()
			db.Close()
		}
	}()

	err = db.AutoMigrate(Fish{}, Group{}, Sensor{}, Transparency{}, Temperature{}, CurrentStatistic{}, CurrentSensorFish{})
	if err != nil {
		return nil, err
	}

	redisClient, err := connectToRedis(options)
	if err != nil {
		return nil, err
	}

	return &Storage{
		db:    db,
		redis: redisClient,
	}, nil
}

func (s *Storage) Close() error {
	var resultError error

	if err := s.redis.Close(); err != nil {
		resultError = multierror.Append(resultError, err)
	}

	db, err := s.db.DB()
	if err != nil {
		resultError = multierror.Append(resultError, err)
		return resultError
	}

	if err = db.Close(); err != nil {
		resultError = multierror.Append(resultError, err)
	}

	return resultError
}

func (s *Storage) GetAllGroups() ([]*Group, error) {
	var groups []*Group
	res := s.db.Find(&groups)
	if res.Error != nil {
		return nil, res.Error
	}

	return groups, nil
}

func (s *Storage) GetAllSensors() ([]*Sensor, error) {
	var sensors []*Sensor
	res := s.db.Find(&sensors)
	if res.Error != nil {
		return nil, res.Error
	}

	return sensors, nil
}

func (s *Storage) GetCurrentSpecies(group string, limit int, opts ...ConditionOption) ([]*Fish, error) {
	resField := "count"
	tx := s.db.Table(CurrentSensorFishTable).
		Select(FishTable+".name, SUM("+FishTable+".count) as "+resField).
		Joins("LEFT JOIN "+FishTable+" ON "+CurrentSensorFishTable+".fish_id = "+FishTable+".id").
		Joins("LEFT JOIN "+SensorTable+" ON "+FishTable+".sensor_id = "+SensorTable+".id").
		Joins("LEFT JOIN "+GroupTable+" ON "+SensorTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", group).
		Group(FishTable + ".name")

	for _, opt := range opts {
		opt(FishTable, tx)
	}

	if limit > 0 {
		tx.Order(resField + " desc").Limit(limit)
	}

	var fishes []*Fish
	if res := tx.Find(&fishes); res.Error != nil {
		return nil, res.Error
	}

	return fishes, nil
}

func (s *Storage) GetMaxTemperatureByRegion(opts ...CoordinateOption) (float64, error) {
	return s.getTemperatureByRegion(maxTemperature, opts...)
}

func (s *Storage) GetMinTemperatureByRegion(opts ...CoordinateOption) (float64, error) {
	return s.getTemperatureByRegion(minTemperature, opts...)
}

func (s *Storage) GetSensorAvgTemperature(groupName string, indexInGroup int, condOpts ...ConditionOption) (float64, error) {
	var avg float64
	tx := s.db.Table(TemperatureTable).
		Select("AVG("+TemperatureTable+".temperature) AS average_temp").
		Joins("LEFT JOIN "+SensorTable+" ON "+TemperatureTable+".sensor_id = "+SensorTable+".id").
		Joins("LEFT JOIN "+GroupTable+" ON "+SensorTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", groupName).
		Where(SensorTable+".index_in_group = ?", indexInGroup)

	for _, opt := range condOpts {
		opt(TemperatureTable, tx)
	}

	res := tx.Find(&avg)
	if res.Error != nil {
		return 0, res.Error
	}

	return avg, nil
}

func (s *Storage) GetAvgTemperature(ctx context.Context, group string) (float64, error) {
	return s.getAvg(ctx, group, temperatureKey, TemperatureTable, "temperature")
}

func (s *Storage) GetAvgTransparency(ctx context.Context, group string) (uint8, error) {
	avg, err := s.getAvg(ctx, group, transparencyKey, TransparencyTable, "transparency")
	return uint8(avg), err
}

func (s *Storage) CreateGroup(group *Group) error {
	return s.db.Create(group).Error
}

func (s *Storage) CreateSensor(sensor *Sensor) error {
	return s.db.Create(sensor).Error
}

func (s *Storage) CreateTemperature(temperature *Temperature) error {
	return s.db.Create(temperature).Error
}

func (s *Storage) CreateTransparency(transparency *Transparency) error {
	return s.db.Create(transparency).Error
}

func (s *Storage) CreateFish(fish *Fish) error {
	return s.db.Create(fish).Error
}

func (s *Storage) UpdateSensorData(sensor *Sensor, fishes []*Fish, temperature *Temperature, transparency *Transparency) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Create(fishes).Error; err != nil {
		tx.Rollback()
		return err
	}

	csfs := make([]*CurrentSensorFish, 0, len(fishes))
	for _, fish := range fishes {
		csfs = append(csfs, &CurrentSensorFish{
			SensorId: sensor.ID,
			FishId:   fish.ID,
		})
	}

	if err := tx.Exec("DELETE FROM "+CurrentSensorFishTable+" WHERE sensor_id = ?", sensor.ID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(csfs).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(temperature).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Create(transparency).Error; err != nil {
		tx.Rollback()
		return err
	}

	statistic := &CurrentStatistic{
		GroupId:        uint(sensor.GroupId),
		SensorId:       sensor.ID,
		TransparencyId: transparency.ID,
		TemperatureId:  temperature.ID,
	}

	if err := tx.Where("sensor_id = ?", sensor.ID).FirstOrCreate(statistic).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return tx.Error
}

func (s *Storage) InitSensorGroups(group *Group, sensors []*Sensor) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, sensor := range sensors {
		sensor.GroupId = uint64(group.ID)
		if err := tx.Create(sensor).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return tx.Error
}

func (s *Storage) getAvg(ctx context.Context, group, key, table, field string) (float64, error) {
	redisKey := key + group

	res := s.redis.Get(ctx, redisKey)
	if res.Err() != nil && res.Err() != redis.Nil {
		log.Printf("Error getting value by key %s: %s", key, res.Err())
	} else if res.Err() == nil {
		return strconv.ParseFloat(res.Val(), 64)
	}

	value, err := s.getAvgFromDb(group, table, field)
	if err != nil {
		return 0, err
	}

	err = s.redis.Set(ctx, redisKey, value, 10*time.Second).Err()
	if err != nil {
		log.Printf("Error setting value by key %s: %s", key, err)
	}

	return value, nil
}

func (s *Storage) getAvgFromDb(group, table, field string) (float64, error) {
	var avg sql.NullFloat64
	res := s.db.Table(CurrentStatisticTable).
		Select("AVG("+table+"."+field+")").
		Joins("LEFT JOIN "+table+" ON "+CurrentStatisticTable+"."+field+"_id = "+table+".id").
		Joins("LEFT JOIN "+GroupTable+" ON "+CurrentStatisticTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", group).
		Pluck("AVG(value)", &avg)

	if res.Error != nil {
		return 0, res.Error
	}

	if !avg.Valid {
		return 0, errors.New("average " + field + " for " + group + " not found")
	}

	return avg.Float64, nil
}

func (s *Storage) getTemperatureByRegion(v uint8, opts ...CoordinateOption) (float64, error) {
	exp := ""
	if v == minTemperature {
		exp = "MIN"
	} else if v == maxTemperature {
		exp = "MAX"
	} else {
		return 0, errors.New("bad request")
	}

	var t sql.NullFloat64
	tx := s.db.Table(CurrentStatisticTable).
		Select(exp + "(" + TemperatureTable + ".temperature) as res").
		Joins("LEFT JOIN " + TemperatureTable + " ON " + CurrentStatisticTable + ".temperature_id = " + TemperatureTable + ".id").
		Joins("LEFT JOIN " + SensorTable + " ON " + CurrentStatisticTable + ".sensor_id = " + SensorTable + ".id")

	for _, opt := range opts {
		opt(tx)
	}

	err := tx.Row().Scan(&t)
	if err != nil {
		return 0, err
	} else if !t.Valid {
		return 0, ErrNoSensorsInArea
	}

	return t.Float64, nil
}

func connectToDb(options *Options) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s sslmode=disable",
		options.dbHost,
		options.dbPort,
		options.dbUser,
		options.dbPassword,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	defer func() {
		db, _ := db.DB()
		db.Close()
	}()

	var dbExists bool
	db.Raw("SELECT EXISTS (SELECT datname FROM pg_database WHERE datname = ?)", options.dbName).Scan(&dbExists)

	if !dbExists {
		db.Exec("CREATE DATABASE " + options.dbName)
	}

	db, err = gorm.Open(postgres.Open(dsn+" dbname="+options.dbName), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to the database")
	}

	return gorm.Open(postgres.Open(dsn+" dbname="+options.dbName), &gorm.Config{})
}

func connectToRedis(options *Options) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{Addr: options.redisAddress})
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return redisClient, nil
}
