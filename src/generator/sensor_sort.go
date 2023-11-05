package generator

import (
	"math"
	"sort"

	"github.com/jenyasd209/fake-sensors/src/storage"
)

type Coordinate struct {
	X, Y, Z float64
}

func distance(c1, c2 *Coordinate) float64 {
	dx := c2.X - c1.X
	dy := c2.Y - c1.Y
	dz := c2.Z - c1.Z

	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

type SortedSensors struct {
	sensors []*storage.Sensor
}

func (bd *SortedSensors) Len() int {
	return len(bd.sensors)
}

func (bd *SortedSensors) Swap(i, j int) {
	bd.sensors[i], bd.sensors[j] = bd.sensors[j], bd.sensors[i]
}

func (bd *SortedSensors) Less(i, j int) bool {
	totalDistI := 0.0
	totalDistJ := 0.0

	for k := range bd.sensors {
		if k != i {
			totalDistI += distance(
				&Coordinate{
					X: bd.sensors[i].X,
					Y: bd.sensors[i].Y,
					Z: bd.sensors[i].Z,
				},
				&Coordinate{
					X: bd.sensors[k].X,
					Y: bd.sensors[k].Y,
					Z: bd.sensors[k].Z,
				})
		}
		if k != j {
			totalDistJ += distance(
				&Coordinate{
					X: bd.sensors[j].X,
					Y: bd.sensors[j].Y,
					Z: bd.sensors[j].Z,
				},
				&Coordinate{
					X: bd.sensors[k].X,
					Y: bd.sensors[k].Y,
					Z: bd.sensors[k].Z,
				})
		}
	}

	return totalDistI < totalDistJ
}

func sortSensors(sensors []*storage.Sensor) []*storage.Sensor {
	sorted := &SortedSensors{sensors: sensors}
	sort.Sort(sorted)

	return sorted.sensors
}
