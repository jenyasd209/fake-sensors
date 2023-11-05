package routes

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/api/response"
	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const codeNameParam = "codeName"

var pattern = regexp.MustCompile("([a-zA-Z]+)([0-9]+)")

func RegisterSensorRoutes(routes *gin.Engine, s *storage.Storage) {
	routes.GET("/sensor/:"+codeNameParam+"/temperature/average", func(context *gin.Context) {
		matches := pattern.FindStringSubmatch(context.Param(codeNameParam))
		if len(matches) < 3 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid codeName"})
		}

		letter := matches[1]
		index, err := strconv.Atoi(matches[2])
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		opts := make([]storage.ConditionOption, 0, 2)
		fromQ := context.DefaultQuery("from", "")
		if fromQ != "" {
			from, err := time.Parse(time.UnixDate, context.DefaultQuery("from", ""))
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
				return
			}

			opts = append(opts, storage.WithCreatedFrom(from))
		}

		tillQ := context.DefaultQuery("from", "")
		if tillQ != "" {
			till, err := time.Parse(time.UnixDate, context.DefaultQuery("till", ""))
			if err != nil {
				context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
				return
			}

			opts = append(opts, storage.WithCreatedTill(till))
		}

		avg, err := s.GetSensorAvgTemperature(letter, index, opts...)
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		context.JSON(http.StatusOK, response.Average{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})
}
