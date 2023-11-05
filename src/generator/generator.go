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
	defaultTransparencyInfelicity = 10

	defaultMinSensorsCount = 2
	defaultMaxSensorsCount = 10

	defaultMinDataOutputRate = 10
	defaultMaxDataOutputRate = 20

	defaultFishListLength = 10
	defaultMaxFishCount   = 10

	defaultTransparency = 10

	minTemperature = -273.17
	maxTemperature = 56.7
)

var (
	greekLetters = []string{
		"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta",
		"Theta", "Iota", "Kappa", "Lambda", "Mu", "Nu", "Xi", "Omicron",
		"Pi", "Rho", "Sigma", "Tau", "Upsilon", "Phi", "Chi", "Psi", "Omega",
	}

	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	maxProc = runtime.GOMAXPROCS(0)
)

type generatorRules struct {
	transparencyInfelicity               uint8
	groupsCount                          uint16
	minSensorsCount, maxSensorsCount     uint16
	minDataOutputRate, maxDataOutputRate uint

	fishNames []string
}

func defaultGeneratorRules() *generatorRules {
	return &generatorRules{
		transparencyInfelicity: defaultTransparencyInfelicity,
		groupsCount:            uint16(len(greekLetters)),
		minSensorsCount:        defaultMinSensorsCount,
		maxSensorsCount:        defaultMaxSensorsCount,
		minDataOutputRate:      defaultMinDataOutputRate,
		maxDataOutputRate:      defaultMaxDataOutputRate,
		fishNames:              []string{},
	}
}

type regenerateNode struct {
	previous time.Time
	sensor   *storage.Sensor
}

type Generator struct {
	rules *generatorRules

	storage *storage.Storage

	listLock         sync.Mutex
	listToRegenerate []*regenerateNode

	regenerateCh chan *regenerateNode

	cancelFunc context.CancelFunc
}

func NewGenerator(storage *storage.Storage, opts ...DataOption) (*Generator, error) {
	rules := defaultGeneratorRules()
	for _, opt := range opts {
		opt(rules)
	}

	fishNames, err := ParseFishNames()
	if err != nil {
		return nil, err
	}
	rules.fishNames = fishNames

	generator := &Generator{
		rules:            rules,
		storage:          storage,
		listLock:         sync.Mutex{},
		listToRegenerate: make([]*regenerateNode, 0, rules.groupsCount*rules.maxSensorsCount),
		regenerateCh:     make(chan *regenerateNode, rules.groupsCount*rules.maxSensorsCount/2),
	}

	return generator, nil
}

func (g *Generator) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)
	g.cancelFunc = cancel

	groups, _ := g.storage.GetAllGroups()
	if len(groups) == 0 {
		g.generateGroups()
	} else {
		sensors, err := g.storage.GetAllSensors()
		if err != nil {
			return err
		}

		for _, sensor := range sensors {
			g.listToRegenerate = append(g.listToRegenerate, &regenerateNode{sensor: sensor})
		}
	}

	go g.startMonitoring(childCtx)

	return nil
}

func (g *Generator) Stop() {
	g.cancelFunc()
}

func (g *Generator) generateGroups() {
	letters := shuffleArray(greekLetters)
	wg := sync.WaitGroup{}

	sem := make(chan struct{}, maxProc)
	defer close(sem)

	for i := 0; i < int(g.rules.groupsCount); i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(l string) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			err := g.storage.CreateGroup(&storage.Group{Name: l})
			if err != nil {
				log.Printf("cannot save %s group\n", l)
				return
			}

			g.generateSensors(l)
		}(letters[i])
	}

	wg.Wait()
}

func (g *Generator) generateSensors(l string) {
	sensorsCount := int(g.rules.minSensorsCount) + random.Intn(int(g.rules.maxSensorsCount-g.rules.minSensorsCount))
	for i := 0; i < sensorsCount; i++ {
		dataOutputRate := int(g.rules.minDataOutputRate) + random.Intn(int(g.rules.maxDataOutputRate-g.rules.minDataOutputRate))
		sensor := &storage.Sensor{
			Model:          gorm.Model{},
			IndexInGroup:   uint64(i),
			X:              random.Float64(),
			Y:              random.Float64(),
			Z:              random.Float64(),
			DataOutputRate: time.Second * time.Duration(dataOutputRate),
		}

		err := g.storage.CreateSensor(sensor)
		if err != nil {
			log.Printf("cannot save %d sensor in the %s group\n", i, l)
			return
		}

		g.listLock.Lock()
		g.listToRegenerate = append(g.listToRegenerate, &regenerateNode{
			previous: time.Now(),
			sensor:   sensor,
		})
		g.listLock.Unlock()
	}
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

			newTemperature := randomTemperature()
			newTransparency := uint8(random.Intn(defaultTransparency))
			newFishList := g.newRandomFishList(uint64(n.sensor.ID), defaultFishListLength)

			err = g.storage.CreateTemperature(&storage.Temperature{
				SensorId:    uint64(n.sensor.ID),
				Temperature: newTemperature,
			})
			if err != nil {
				log.Printf("cannot save temperature for %d sensor", n.sensor.ID)
				continue
			}

			err = g.storage.CreateTransparency(&storage.Transparency{
				SensorId:     uint64(n.sensor.ID),
				Transparency: newTransparency,
			})
			if err != nil {
				log.Printf("cannot save transparency for %d sensor", n.sensor.ID)
				continue
			}

			for _, fish := range newFishList {
				err = g.storage.CreateFish(fish)
				if err != nil {
					log.Printf("cannot save fish for %d sensor", n.sensor.ID)
					continue
				}
			}

			n.previous = time.Now()
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

	for {
		sleep := time.Duration(0)
		select {
		case <-ctx.Done():
			close(g.regenerateCh)
			return
		default:
			for _, node := range g.listToRegenerate {
				diff := time.Now().Sub(node.previous)
				if diff > node.sensor.DataOutputRate {
					g.regenerateCh <- node
				} else {
					newSleep := node.sensor.DataOutputRate - diff
					if newSleep < sleep {
						sleep = newSleep
					}
				}
			}

			time.Sleep(sleep)
		}
	}
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
			Count:    uint64(random.Intn(defaultMaxFishCount)),
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

func randomTemperature() float64 {
	return minTemperature + rand.Float64()*(maxTemperature-minTemperature)
}
