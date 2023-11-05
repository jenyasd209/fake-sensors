package generator

type DataOption func(data *generatorRules)

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
