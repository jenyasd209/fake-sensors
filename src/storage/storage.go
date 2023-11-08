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

	err = db.AutoMigrate(Fish{}, Group{}, Sensor{}, Transparency{}, Temperature{})
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

func (s *Storage) GetSpecies(group string, limit int, opts ...ConditionOption) ([]*Fish, error) {
	var fishes []*Fish

	tx := s.db.
		Select(FishTable+".*").
		Joins("LEFT JOIN "+SensorTable+" ON "+FishTable+".sensor_id = "+SensorTable+".id").
		Joins("LEFT JOIN "+GroupTable+" ON "+SensorTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", group)

	for _, opt := range opts {
		opt(FishTable, tx)
	}

	if limit > 0 {
		tx.Limit(limit)
	}

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

func (s *Storage) UpdateSensorData(fishes []*Fish, temperature *Temperature, transparency *Transparency) error {
	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	created := time.Now()
	for _, fish := range fishes {
		fish.CreatedAt = created
		fish.UpdatedAt = created
		if err := tx.Create(fish).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	temperature.CreatedAt = created
	temperature.UpdatedAt = created
	if err := tx.Create(temperature).Error; err != nil {
		tx.Rollback()
		return err
	}

	transparency.CreatedAt = created
	transparency.UpdatedAt = created
	if err := tx.Create(transparency).Error; err != nil {
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

	created := time.Now()

	group.CreatedAt = created
	group.UpdatedAt = created

	if err := tx.Create(group).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, sensor := range sensors {
		sensor.GroupId = uint64(group.ID)
		sensor.CreatedAt = created
		sensor.UpdatedAt = created

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
	var average float64
	res := s.db.Table(table).
		Select("AVG("+table+"."+field+") AS average").
		Joins("LEFT JOIN "+SensorTable+" ON "+table+".sensor_id = "+SensorTable+".id").
		Joins("LEFT JOIN "+GroupTable+" ON "+SensorTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", group).
		Pluck("AVG(value)", &average)

	if res.Error != nil {
		return 0, res.Error
	}

	return average, nil
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
	tx := s.db.Table(TemperatureTable).
		Select(exp + "(" + TemperatureTable + ".temperature) as res").
		Joins("LEFT JOIN " + SensorTable + " ON " + TemperatureTable + ".sensor_id = " + SensorTable + ".id")

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
