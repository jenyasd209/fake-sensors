package routes

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const (
	codeNameParam = "codeName"

	sensorAvgTemperature = "/sensor/:" + codeNameParam + "/temperature/average"
)

var pattern = regexp.MustCompile("([a-zA-Z]+)([0-9]+)")

func RegisterSensorRoutes(router *Router) {
	router.routes.GET(sensorAvgTemperature, router.GetSensorAvgTemperature)
}

// @Summary Get average temperature detected by a particular sensor
// @Description Get average temperature detected by a particular sensor between the specified date/time pairs (UNIX timestamps)
// @Produce json
// @Param codeName path string true "sensor code name"
// @Param from query string false "From (UNIX timestamps)"
// @Param till query string false "Till (UNIX timestamps)"
// @Success 200 {object} Average
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /sensor/{codeName}/temperature/average [get]
func (r *Router) GetSensorAvgTemperature(context *gin.Context) {
	matches := pattern.FindStringSubmatch(context.Param(codeNameParam))
	if len(matches) < 3 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid codeName"})
	}

	letter := matches[1]
	index, err := strconv.Atoi(matches[2])
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	opts := make([]storage.ConditionOption, 0, 2)
	fromQ := context.Query("from")
	if fromQ != "" {
		from, err := time.Parse(time.UnixDate, fromQ)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
			return
		}
		opts = append(opts, storage.WithCreatedFrom(from))
	}

	tillQ := context.Query("till")
	if tillQ != "" {
		till, err := time.Parse(time.UnixDate, tillQ)
		if err != nil {
			context.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid date format"})
			return
		}
		opts = append(opts, storage.WithCreatedTill(till))
	}

	avg, err := r.storage.GetSensorAvgTemperature(letter, index, opts...)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Average{
		Average: strconv.FormatFloat(avg, 'f', 2, 64),
	})
}
