package generator

import (
	"math/rand"
	"sync"
	"time"

	"underwater-sensors/src/model"
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

type DataOption func(data *generatorRules)

func WithTransparency(t uint8) DataOption {
	return func(gd *generatorRules) {
		gd.transparencyInfelicity = t
	}
}

func WithGroupsCount(t uint16) DataOption {
	return func(gd *generatorRules) {
		if t <= uint16(len(greekLetters)) {
			gd.groupsCount = t
		}
	}
}

func WithSensorsCount(min, max uint16) DataOption {
	return func(gd *generatorRules) {
		if min > max {
			min, max = max, min
		}

		gd.minSensorsCount = min
		gd.maxSensorsCount = max
	}
}

func WithDataOutputRate(min, max uint) DataOption {
	return func(gd *generatorRules) {
		if min > max {
			min, max = max, min
		}

		gd.minDataOutputRate = min
		gd.maxDataOutputRate = max
	}
}

type regenerateNode struct {
	previous time.Time
	sensor   *model.Sensor
}

type Generator struct {
	rules *generatorRules

	listLock         sync.Mutex
	listToRegenerate []*regenerateNode
	regenerateCh     chan *regenerateNode

	done chan struct{}
}

func NewGenerator(opts ...DataOption) (*Generator, error) {
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
		listToRegenerate: make([]*regenerateNode, 0, rules.groupsCount*rules.maxSensorsCount),
		regenerateCh:     make(chan *regenerateNode, rules.groupsCount*rules.maxSensorsCount/2),
		done:             make(chan struct{}),
	}

	return generator, nil
}

func (g *Generator) Start() {
	g.generateGroups()
	go g.startMonitoring()
}

func (g *Generator) Stop() {
	g.done <- struct{}{}
}

func (g *Generator) generateGroups() model.SensorGroup {
	letters := shuffleArray(greekLetters)

	sensorGroupsLock := sync.Mutex{}
	sensorGroups := make(model.SensorGroup, g.rules.groupsCount)

	wg := sync.WaitGroup{}
	for i := 0; i < int(g.rules.groupsCount); i++ {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()

			sensorsCount := int(g.rules.minSensorsCount) + random.Intn(int(g.rules.maxSensorsCount-g.rules.minSensorsCount))
			for j := 0; j < sensorsCount; j++ {
				dataOutputRate := int(g.rules.minDataOutputRate) + random.Intn(int(g.rules.maxDataOutputRate-g.rules.minDataOutputRate))
				sensor := &model.Sensor{
					Codename: model.Codename{
						Group: l,
						Index: uint64(j),
					},
					Coordinate: model.Coordinate{
						X: random.Float64(),
						Y: random.Float64(),
						Z: random.Float64(),
					},
					DataOutputRate: time.Second * time.Duration(dataOutputRate),
					Temperature:    randomTemperature(),
					Transparency:   uint8(random.Intn(defaultTransparency)),
					FishList:       g.newRandomFishList(defaultFishListLength),
				}

				sensorGroupsLock.Lock()
				sensorGroups[l] = sensor
				sensorGroupsLock.Unlock()

				g.listLock.Lock()
				g.listToRegenerate = append(g.listToRegenerate, &regenerateNode{
					previous: time.Now(),
					sensor:   sensor,
				})
				g.listLock.Unlock()
			}
		}(letters[i])
	}

	wg.Wait()
	return sensorGroups
}

func (g *Generator) regenerateFishList() {
	for {
		select {
		case n, ok := <-g.regenerateCh:
			if !ok {
				return
			}

			n.sensor.Temperature = randomTemperature()
			n.sensor.Transparency = uint8(random.Intn(defaultTransparency))
			n.sensor.FishList = g.newRandomFishList(defaultFishListLength)
			n.previous = time.Now()
		}
	}
}

func (g *Generator) startMonitoring() {
	go g.regenerateFishList()

	for {
		sleep := time.Duration(0)
		select {
		case <-g.done:
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

func (g *Generator) newRandomFishList(count int) []*model.Fish {
	fishList := make([]*model.Fish, count)
	fishIndex := 0
	usedIndex := make(map[int]struct{})

	for i := 0; i < count; i++ {
		fishIndex = random.Intn(len(g.rules.fishNames) - 1)
		_, ok := usedIndex[fishIndex]
		if ok {
			i--
			continue
		}

		fishList[i] = &model.Fish{
			Name:  g.rules.fishNames[fishIndex],
			Count: uint64(random.Intn(defaultMaxFishCount)),
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
