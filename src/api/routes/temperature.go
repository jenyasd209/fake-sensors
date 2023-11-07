package routes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const temperatureRouteGroup = "/region/temperature"

var ErrBadCoordinate = errors.New("bad coordinate")

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
	opts, err := parseCoordinates(context)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	minT, err := r.storage.GetMinTemperatureByRegion(opts...)
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
	opts, err := parseCoordinates(context)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	maxT, err := r.storage.GetMaxTemperatureByRegion(opts...)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Value{
		Value: strconv.FormatFloat(maxT, 'f', 2, 64),
	})
}

func parseCoordinates(ctx *gin.Context) ([]storage.CoordinateOption, error) {
	opts := make([]storage.CoordinateOption, 0, 6)

	xMin, ok, err := parseFloat64Query(ctx, "xMin")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithXMin(xMin))
	}

	xMax, ok, err := parseFloat64Query(ctx, "xMax")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithXMax(xMax))
	}

	yMin, ok, err := parseFloat64Query(ctx, "yMin")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithYMin(yMin))
	}

	yMax, ok, err := parseFloat64Query(ctx, "yMax")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithYMax(yMax))
	}

	zMin, ok, err := parseFloat64Query(ctx, "zMin")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithZMin(zMin))
	}

	zMax, ok, err := parseFloat64Query(ctx, "zMax")
	if err != nil {
		return nil, err
	} else if ok {
		opts = append(opts, storage.WithZMax(zMax))
	}

	return opts, nil
}

func parseFloat64Query(ctx *gin.Context, arg string) (float64, bool, error) {
	v, ok := ctx.GetQuery(arg)
	if !ok {
		return 0, false, nil
	}

	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false, errors.New(ErrBadCoordinate.Error() + ": " + arg)
	}

	return f, true, nil
}
