package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

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
	s.redis.Close()

	db, err := s.db.DB()
	if err != nil {
		return err
	}

	db.Close()
	return nil
}

func (s *Storage) GetAllGroups() ([]*Group, error) {
	var groups []*Group
	res := s.db.Find(&groups)
	if res.Error != nil {
		return nil, res.Error
	}

	return groups, nil
}

func (s *Storage) GetSpecies(group string, limit int, opts ...ConditionOption) ([]*Fish, error) {
	var fishes []*Fish
	tx := s.db.Table(FishTable).
		Select(FishTable+".*").
		Joins("LEFT JOIN "+FishTable+" ON "+FishTable+".group_id = "+GroupTable+".id").
		Where(GroupTable+".name = ?", group)

	for _, opt := range opts {
		opt(FishTable, "created_at", tx)
	}

	if limit > 0 {
		tx.Limit(limit)
	}

	res := tx.Find(&fishes)
	if res.Error != nil {
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
	tx := s.db.Table("groups").
		Select("AVG("+TemperatureTable+"temperatures.temperature) AS average_temp").
		Joins("LEFT JOIN"+SensorTable+" ON "+GroupTable+".id = "+SensorTable+".group_id").
		Joins("LEFT JOIN"+TemperatureTable+" ON "+SensorTable+".id = "+TemperatureTable+".sensor_id").
		Group("groups.name").
		Where(GroupTable+".name = ?", groupName).
		Where(SensorTable+".index_in_group = ?", indexInGroup)

	for _, opt := range condOpts {
		opt(TemperatureTable, "created_at", tx)
	}

	res := tx.Find(&avg)
	if res.Error != nil {
		return 0, res.Error
	}

	return avg, nil
}

func (s *Storage) GetAvgTemperature(ctx context.Context) (float64, error) {
	return s.getCachedAvg(ctx, temperatureKey, TemperatureTable, "temperature")
}

func (s *Storage) GetAvgTransparency(ctx context.Context) (uint8, error) {
	avg, err := s.getCachedAvg(ctx, transparencyKey, TransparencyTable, "transparency")
	return uint8(avg), err
}

func (s *Storage) CreateGroup(group *Group) error {
	result := s.db.Create(group)

	return result.Error
}

func (s *Storage) CreateSensor(sensor *Sensor) error {
	result := s.db.Create(sensor)

	return result.Error
}

func (s *Storage) CreateTemperature(temperature *Temperature) error {
	result := s.db.Create(temperature)

	return result.Error
}

func (s *Storage) CreateTransparency(transparency *Transparency) error {
	result := s.db.Create(transparency)

	return result.Error
}

func (s *Storage) CreateFish(fish *Fish) error {
	result := s.db.Create(fish)

	return result.Error
}

func (s *Storage) getCachedAvg(ctx context.Context, key, table, field string) (float64, error) {
	cachedData, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		value, err := s.getAvg(table, field)
		if err != nil {
			return 0, err
		}

		if err == redis.Nil {
			s.redis.Set(ctx, key, value, 10*time.Second)
		} else {
			// should log error
			return value, nil
		}
	}

	return strconv.ParseFloat(cachedData, 64)
}

func (s *Storage) getAvg(table, field string) (float64, error) {
	var average float64
	res := s.db.Table(table).Select("AVG("+field+") as average").Pluck("AVG(value)", &average)
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

	var t float64
	tx := s.db.Table(SensorTable).
		Select(exp + "(" + TemperatureTable + ".temperature) as average").
		Joins("LEFT JOIN " + TemperatureTable + " ON " + TemperatureTable + ".sensor_id = " + SensorTable + ".id")

	for _, opt := range opts {
		opt(tx)
	}

	res := tx.Find(&t)
	if res.Error != nil {
		return 0, res.Error
	}

	return t, nil
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
