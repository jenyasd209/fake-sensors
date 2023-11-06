package routes

import (
	"net/http"
	"strconv"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const temperatureRouteGroup = "/region/temperature"

func RegisterTemperatureRoutes(router *Router) {
	groups := router.routes.Group(temperatureRouteGroup)

	groups.GET("/min", router.GetMinTemperature)
	groups.GET("/max", router.GetMaxTemperature)
}

// @Summary Get current minimum temperature inside the region
// @Description Get current minimum temperature inside the region. Region here and below is an area represented by the range of coordinates
// @Produce json
// @Param minX path number false "minX" format(float)
// @Param maxX path number false "maxX" format(float)
// @Param minY path number false "minY" format(float)
// @Param maxY path number false "maxY" format(float)
// @Param minZ path number false "minZ" format(float)
// @Param maxZ path number false "maxZ" format(float)
// @Success 200 {object} Value
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /region/temperature/min [get]
func (r *Router) GetMinTemperature(context *gin.Context) {
	minT, err := r.storage.GetMinTemperatureByRegion(parseCoordinates(context)...)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Value{
		Value: strconv.FormatFloat(minT, 'f', 2, 64),
	})
}

// @Summary Get current maximum temperature inside the region
// @Description Get current maximum temperature inside the region. Region here and below is an area represented by the range of coordinates
// @Produce json
// @Param minX path number false "minX" format(float)
// @Param maxX path number false "maxX" format(float)
// @Param minY path number false "minY" format(float)
// @Param maxY path number false "maxY" format(float)
// @Param minZ path number false "minZ" format(float)
// @Param maxZ path number false "maxZ" format(float)
// @Success 200 {object} Value
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /region/temperature/max [get]
func (r *Router) GetMaxTemperature(context *gin.Context) {
	maxT, err := r.storage.GetMaxTemperatureByRegion(parseCoordinates(context)...)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Value{
		Value: strconv.FormatFloat(maxT, 'f', 2, 64),
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
