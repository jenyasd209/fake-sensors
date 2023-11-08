package generator

import (
	"context"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"gorm.io/gorm"
)

const (
	defaultTransparencyInfelicity = uint8(10)

	defaultMinSensorsCount = 2
	defaultMaxSensorsCount = 10

	defaultMinDataOutputRate = 120
	defaultMaxDataOutputRate = 1200

	defaultFishListLength = 10
	defaultMaxFishCount   = 10

	minTemperature = -273.17
	maxTemperature = 56.7

	defaultMinX = -1000.0
	defaultMaxX = 1000.0

	defaultMinY = -1000.0
	defaultMaxY = 1000.0

	defaultMinZ = -1000.0
	defaultMaxZ = 1000.0

	temperatureSum               = (minTemperature - maxTemperature) * -1
	tempPerPoint                 = temperatureSum / defaultMaxZ
	allowedTemperatureDifference = 3
)

var (
	greekLetters = []string{
		"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta",
		"theta", "iota", "kappa", "lambda", "mu", "nu", "xi", "omicron",
		"pi", "rho", "sigma", "tau", "upsilon", "phi", "chi", "psi", "omega",
	}

	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	maxProc = runtime.GOMAXPROCS(0)
)

type generatorRules struct {
	groupsCount                          uint16
	minSensorsCount, maxSensorsCount     uint16
	minDataOutputRate, maxDataOutputRate uint

	fishNames []string
}

func defaultGeneratorRules() *generatorRules {
	return &generatorRules{
		groupsCount:       uint16(len(greekLetters)),
		minSensorsCount:   defaultMinSensorsCount,
		maxSensorsCount:   defaultMaxSensorsCount,
		minDataOutputRate: defaultMinDataOutputRate,
		maxDataOutputRate: defaultMaxDataOutputRate,
		fishNames:         []string{},
	}
}

type regenerateNode struct {
	sensor *storage.Sensor

	previousUpdate time.Time

	nearestTransparency uint8
	currentTransparency uint8
}

type Generator struct {
	rules *generatorRules

	storage *storage.Storage

	listToRegenerate []*regenerateNode
	regenerateCh     chan *regenerateNode

	cancelFunc context.CancelFunc
}

func NewGenerator(storage *storage.Storage) (*Generator, error) {
	rules := defaultGeneratorRules()

	fishNames, err := ParseFishNames()
	if err != nil {
		return nil, err
	}
	rules.fishNames = fishNames

	generator := &Generator{
		rules:            rules,
		storage:          storage,
		listToRegenerate: make([]*regenerateNode, 0, rules.groupsCount*rules.maxSensorsCount),
		regenerateCh:     make(chan *regenerateNode, rules.groupsCount*rules.maxSensorsCount/2),
	}

	return generator, nil
}

func (g *Generator) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	g.cancelFunc = cancel

	groups, err := g.storage.GetAllGroups()
	if err != nil {
		return err
	}

	if len(groups) == 0 {
		g.generateSensorGroups()
	}

	err = g.prepareSensors()
	if err != nil {
		return err
	}

	g.startMonitoring(childCtx)
	return nil
}

func (g *Generator) Stop() {
	g.cancelFunc()
}

func (g *Generator) prepareSensors() error {
	sensors, err := g.storage.GetAllSensors()
	if err != nil {
		return err
	}

	for _, sensor := range sortSensors(sensors) {
		g.listToRegenerate = append(g.listToRegenerate, &regenerateNode{sensor: sensor})
	}

	return nil
}

func (g *Generator) generateSensorGroups() {
	letters := shuffleArray(greekLetters)

	wg := sync.WaitGroup{}
	sem := make(chan struct{}, maxProc)
	defer close(sem)

	for i := 0; i < int(g.rules.groupsCount); i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(l string) {
			defer wg.Done()
			defer func() { <-sem }()

			err := g.storage.InitSensorGroups(&storage.Group{Name: l}, g.generateSensors())
			if err != nil {
				log.Printf("cannot save %s group and sensors for this group: %s\n", l, err)
				return
			}
		}(letters[i])
	}

	wg.Wait()
}

func (g *Generator) generateSensors() []*storage.Sensor {
	sensorsCount := int(g.rules.minSensorsCount) + random.Intn(int(g.rules.maxSensorsCount-g.rules.minSensorsCount))
	sensors := make([]*storage.Sensor, 0, sensorsCount)

	for i := 0; i < sensorsCount; i++ {
		dataOutputRate := int(g.rules.minDataOutputRate) + random.Intn(int(g.rules.maxDataOutputRate-g.rules.minDataOutputRate))
		sensors = append(sensors, &storage.Sensor{
			Model:          gorm.Model{},
			IndexInGroup:   uint64(i),
			X:              randomPoint(defaultMinX, defaultMaxX),
			Y:              randomPoint(defaultMinY, defaultMaxY),
			Z:              randomPoint(defaultMinZ, defaultMaxZ),
			DataOutputRate: time.Second * time.Duration(dataOutputRate),
		})
	}

	return sensors
}

func (g *Generator) regenerateData(ctx context.Context) {
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-g.regenerateCh:
			if !ok {
				return
			}

			transparency := &storage.Transparency{
				SensorId:     uint64(n.sensor.ID),
				Transparency: randomTransparency(n.nearestTransparency),
			}
			err = g.storage.UpdateSensorData(
				g.newRandomFishList(uint64(n.sensor.ID), defaultFishListLength),
				&storage.Temperature{
					SensorId:    uint64(n.sensor.ID),
					Temperature: randomTemperature(n.sensor.Z),
				},
				transparency,
			)
			if err != nil {
				log.Printf("cannot save temperature for %d sensor: %s\n", n.sensor.ID, err)
				continue
			}

			n.currentTransparency = transparency.Transparency
			n.previousUpdate = time.Now()
		}
	}
}

func (g *Generator) startMonitoring(ctx context.Context) {
	if maxProc > 2 {
		maxProc /= 2
	}

	for i := 0; i < maxProc; i++ {
		go g.regenerateData(ctx)
	}

	go func() {
		sleep := time.Duration(0)
		for {
			select {
			case <-ctx.Done():
				close(g.regenerateCh)
				return
			default:
				for i, node := range g.listToRegenerate {
					if i != 0 {
						node.nearestTransparency = g.listToRegenerate[i-1].currentTransparency
					}

					diff := time.Now().Sub(node.previousUpdate)
					if diff > node.sensor.DataOutputRate {
						g.regenerateCh <- node
						continue
					}

					newSleep := node.sensor.DataOutputRate - diff
					if newSleep < sleep {
						sleep = newSleep
					}
				}

				time.Sleep(sleep)
			}
		}
	}()
}

func (g *Generator) newRandomFishList(sensorId uint64, count int) []*storage.Fish {
	fishList := make([]*storage.Fish, count)
	fishIndex := 0
	usedIndex := make(map[int]struct{})

	for i := 0; i < count; i++ {
		fishIndex = random.Intn(len(g.rules.fishNames) - 1)
		_, ok := usedIndex[fishIndex]
		if ok {
			i--
			continue
		}

		fishList[i] = &storage.Fish{
			SensorId: sensorId,
			Name:     g.rules.fishNames[fishIndex],
			Count:    uint64(random.Intn(defaultMaxFishCount-1) + 1),
		}

		usedIndex[fishIndex] = struct{}{}
	}

	return fishList
}

func shuffleArray(array []string) []string {
	n := len(array)
	mixedArray := make([]string, n)
	copy(mixedArray, array)

	for i := n - 1; i > 0; i-- {
		j := random.Intn(i + 1)
		mixedArray[i], mixedArray[j] = mixedArray[j], mixedArray[i]
	}

	return mixedArray
}

func randomTemperature(z float64) float64 {
	t := maxTemperature - z*tempPerPoint

	minT := t - allowedTemperatureDifference
	maxT := t + allowedTemperatureDifference

	return minT + random.Float64()*(maxT-minT)
}

func randomTransparency(nearestT uint8) uint8 {
	min := uint8(0)
	max := nearestT + defaultTransparencyInfelicity

	if nearestT > 0 && nearestT >= defaultTransparencyInfelicity {
		min = nearestT - defaultTransparencyInfelicity
	}

	if max > 100 {
		max = 100
	}

	t := int(min) + random.Intn(int(max-min))
	return uint8(t)
}

func randomPoint(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}

	return min + random.Float64()*(max-min)
}
