package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testSensorGroup struct {
	group   *Group
	sensors []*Sensor
}

var testSensorGroups = []*testSensorGroup{
	{
		group: &Group{Name: "a"},
		sensors: []*Sensor{
			{
				IndexInGroup:   1,
				X:              1,
				Y:              2,
				Z:              3,
				DataOutputRate: time.Second * 10,
			},
			{
				IndexInGroup:   2,
				X:              4,
				Y:              5,
				Z:              6,
				DataOutputRate: time.Second * 10,
			},
		},
	},
	{
		group: &Group{Name: "b"},
		sensors: []*Sensor{
			{
				IndexInGroup:   1,
				X:              1,
				Y:              2,
				Z:              3,
				DataOutputRate: time.Second * 10,
			},
			{
				IndexInGroup:   2,
				X:              4,
				Y:              5,
				Z:              6,
				DataOutputRate: time.Second * 10,
			},
		},
	},
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

type StorageTestSuite struct {
	suite.Suite
	storage *Storage

	testSensorGroups []*testSensorGroup
}

func (s *StorageTestSuite) SetupSuite() {
	storage, err := connectToTestDb()
	s.NoError(err, err)
	s.NotNil(s)

	s.storage = storage
	s.testSensorGroups = testSensorGroups
}

func (s *StorageTestSuite) TearDownSuite() {
	s.storage.db.Migrator().DropTable(&Fish{}, &Group{}, &Sensor{}, &Temperature{}, &Transparency{})

	err := s.storage.Close()
	s.NoError(err, err)
}

func (s *StorageTestSuite) SetupTest() {
	for _, d := range testSensorGroups {
		err := s.storage.InitSensorGroups(d.group, d.sensors)
		s.NoError(err, err)
	}
}

func (s *StorageTestSuite) TearDownTest() {
	s.storage.db.Delete(&Fish{})
	s.storage.db.Delete(&Group{})
	s.storage.db.Delete(&Sensor{})
	s.storage.db.Delete(&Temperature{})
	s.storage.db.Delete(&Transparency{})
}

func (s *StorageTestSuite) TestInitSensorGroups(t *testing.T) {
	for _, d := range s.testSensorGroups {
		err := s.storage.InitSensorGroups(d.group, d.sensors)
		assert.NoError(t, err, err)
	}

	t.Run("GetGroups", func(t *testing.T) {
		groups, err := s.storage.GetAllGroups()
		require.NoError(t, err, err)
		require.Equal(t, len(s.testSensorGroups), len(groups))
		for i, d := range s.testSensorGroups {
			assert.Equal(t, d.group.Name, groups[i].Name)
		}
	})

	t.Run("GetSensors", func(t *testing.T) {
		expSensors := make([]*Sensor, 0, 4)
		for _, d := range s.testSensorGroups {
			expSensors = append(expSensors, d.sensors...)
		}

		sensors, err := s.storage.GetAllSensors()
		require.NoError(t, err, err)
		require.Equal(t, len(expSensors), len(sensors))
		for i, s := range expSensors {
			assertSensor(t, s, sensors[i])
		}
	})
}

func (s *StorageTestSuite) TestGetSpecies() {
	group := s.testSensorGroups[0].group
	sensor := s.testSensorGroups[0].sensors[0]

	fishes := []*Fish{
		{
			SensorId: uint64(sensor.ID),
			Name:     "FishA",
			Count:    1,
		},
		{
			SensorId: uint64(sensor.ID),
			Name:     "FishB",
			Count:    2,
		},
	}

	for _, fish := range fishes {
		err := s.storage.CreateFish(fish)
		s.Require().NoError(err, err)
	}

	s.T().Run("GetAllSpecies", func(t *testing.T) {
		species, err := s.storage.GetSpecies(group.Name, 0)
		s.Require().NoError(err, err)
		s.Equal(len(fishes), len(species))
	})

	s.T().Run("GetNSpecies", func(t *testing.T) {
		species, err := s.storage.GetSpecies(group.Name, 1)
		s.Require().NoError(err, err)
		s.Equal(1, len(species))
	})

	s.T().Run("FilterFromAndTill", func(t *testing.T) {
		time.Sleep(time.Second)
		from := time.Now()

		expFish := &Fish{
			SensorId: uint64(sensor.ID),
			Name:     "From",
			Count:    3,
		}
		err := s.storage.CreateFish(expFish)
		s.Require().NoError(err, err)

		till := time.Now()
		fish := &Fish{
			SensorId: uint64(sensor.ID),
			Name:     "Till",
			Count:    4,
		}
		err = s.storage.CreateFish(fish)
		s.Require().NoError(err, err)

		species, err := s.storage.GetSpecies(group.Name, 0, WithCreatedFrom(from), WithCreatedTill(till))
		s.Require().NoError(err, err)
		s.Require().Equal(1, len(species))
		assertFish(t, expFish, species[0])
	})
}

func (s *StorageTestSuite) TestGetSensorAvgTemperature() {
	group := s.testSensorGroups[0].group
	sensor := s.testSensorGroups[0].sensors[0]

	temperatures := []*Temperature{
		{
			SensorId:    uint64(sensor.ID),
			Temperature: 1,
		},
		{
			SensorId:    uint64(sensor.ID),
			Temperature: 2,
		},
		{
			SensorId:    uint64(sensor.ID),
			Temperature: 3,
		},
		{
			SensorId:    uint64(s.testSensorGroups[0].sensors[1].ID),
			Temperature: 4,
		},
	}

	for _, t := range temperatures {
		err := s.storage.CreateTemperature(t)
		s.Require().NoError(err, err)
	}

	expT := float64(0)
	for i := 0; i < len(temperatures)-1; i++ {
		expT += temperatures[i].Temperature
	}
	expT /= float64(len(temperatures) - 1)

	gotT, err := s.storage.GetSensorAvgTemperature(group.Name, int(sensor.IndexInGroup))
	s.NoError(err, err)
	s.Equal(expT, gotT)
}

func (s *StorageTestSuite) TestGetAvgTransparency() {
	group := s.testSensorGroups[0].group
	sensor := s.testSensorGroups[0].sensors[0]

	transparency := []*Transparency{
		{
			SensorId:     uint64(sensor.ID),
			Transparency: 11,
		},
		{
			SensorId:     uint64(sensor.ID),
			Transparency: 22,
		},
		{
			SensorId:     uint64(sensor.ID),
			Transparency: 33,
		},
		{
			SensorId:     uint64(s.testSensorGroups[0].sensors[1].ID),
			Transparency: 44,
		},
	}

	for _, t := range transparency {
		err := s.storage.CreateTransparency(t)
		s.Require().NoError(err, err)
	}

	expT := uint8(0)
	for i := 0; i < len(transparency); i++ {
		expT += transparency[i].Transparency
	}
	expT /= uint8(len(transparency))

	gotT, err := s.storage.GetAvgTransparency(context.TODO(), group.Name)
	s.NoError(err, err)
	s.Equal(expT, gotT)
}

func connectToTestDb() (*Storage, error) {
	return NewStorage(
		WithDbUser("postgres"),
		WithDbPassword("pwd"),
		WithDbPort("5432"),
		WithDbHost("0.0.0.0"),
		WithDbName("test"),
		WithRedisAddress("0.0.0.0:6379"),
	)
}

func assertSensor(t *testing.T, exp, got *Sensor) {
	assert.Equal(t, exp.IndexInGroup, got.IndexInGroup)
	assert.Equal(t, exp.X, got.X)
	assert.Equal(t, exp.Y, got.Y)
	assert.Equal(t, exp.Z, got.Z)
	assert.Equal(t, exp.DataOutputRate, got.DataOutputRate)
}

func assertFish(t *testing.T, exp, got *Fish) {
	assert.Equal(t, exp.SensorId, got.SensorId)
	assert.Equal(t, exp.Name, got.Name)
	assert.Equal(t, exp.Count, got.Count)
}
