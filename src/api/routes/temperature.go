package routes

import (
	"net/http"
	"strconv"

	"github.com/jenyasd209/fake-sensors/src/api/response"
	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const temperatureRouteGroup = "/region/temperature"

func RegisterTemperatureRoutes(routes *gin.Engine, storage *storage.Storage) {
	routes.GET(temperatureRouteGroup, func(context *gin.Context) {})

	groups := routes.Group(temperatureRouteGroup)
	groups.GET("/min", func(context *gin.Context) {
		context.JSON(http.StatusOK, response.Value{
			Value: strconv.FormatFloat(storage.GetMinTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
		})
	})

	groups.GET("/max", func(context *gin.Context) {
		context.JSON(http.StatusOK, response.Value{
			Value: strconv.FormatFloat(storage.GetMaxTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
		})
	})
}

func parseCoordinates(ctx *gin.Context) []storage.CoordinateOption {
	opts := make([]storage.CoordinateOption, 0, 6)

	xMin := ctx.Query("xMin")
	if xMin != "" {
		opts = append(opts, storage.WithXMin(ctx.GetFloat64("xMin")))
	}
	xMax := ctx.Query("xMax")
	if xMax != "" {
		opts = append(opts, storage.WithXMax(ctx.GetFloat64("xMax")))
	}

	yMin := ctx.Query("yMin")
	if yMin != "" {
		opts = append(opts, storage.WithYMin(ctx.GetFloat64("yMin")))
	}
	yMax := ctx.Query("yMax")
	if yMax != "" {
		opts = append(opts, storage.WithYMax(ctx.GetFloat64("yMax")))
	}

	zMin := ctx.Query("zMin")
	if zMin != "" {
		opts = append(opts, storage.WithZMin(ctx.GetFloat64("zMin")))
	}
	zMax := ctx.Query("zMax")
	if zMax != "" {
		opts = append(opts, storage.WithZMax(ctx.GetFloat64("zMax")))
	}

	return opts
}
