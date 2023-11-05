package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/api/response"
	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const (
	groupNameParam = "groupName"

	groupRouteGroup = "/group/:" + groupNameParam
)

func RegisterGroupRoutes(routes *gin.Engine, s *storage.Storage) {
	routes.GET(groupRouteGroup, func(context *gin.Context) {
		groupRecords, err := s.GetAllGroups()
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		groups := make([]string, len(groupRecords))

		for i, record := range groupRecords {
			groups[i] = record.Name
		}

		context.JSON(http.StatusOK, response.Groups{
			Groups: groups,
		})
	})

	groups := routes.Group(groupRouteGroup)
	groups.GET("/transparency/average", func(context *gin.Context) {
		avg, err := s.GetAvgTransparency(context, context.Param(groupNameParam))
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		context.JSON(http.StatusOK, response.Average{
			Average: strconv.FormatUint(uint64(avg), 10),
		})
	})

	groups.GET("/temperature/average", func(context *gin.Context) {
		avg, err := s.GetAvgTemperature(context, context.Param(groupNameParam))
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		context.JSON(http.StatusOK, response.Average{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})

	groups.GET("/species", func(context *gin.Context) {
		species, err := getSpecies(s, context.Param(groupNameParam), 0)
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		context.JSON(http.StatusOK, response.SpeciesList{
			Species: species,
		})
	})

	species := routes.Group("/species")
	species.GET("top/:n", func(context *gin.Context) {
		count, err := strconv.Atoi(context.Param("n"))
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
				context.JSON(http.StatusBadRequest, response.Error{Error: "Invalid date format"})
				return
			}

			opts = append(opts, storage.WithCreatedTill(till))
		}

		species, err := getSpecies(s, context.Param(groupNameParam), count, opts...)
		if err != nil {
			context.JSON(http.StatusInternalServerError, response.Error{Error: err.Error()})
			return
		}

		context.JSON(http.StatusOK, response.SpeciesList{
			Species: species,
		})
	})
}

func getSpecies(storage *storage.Storage, groupName string, limit int, opts ...storage.ConditionOption) ([]*response.Species, error) {
	fishesRecords, err := storage.GetSpecies(groupName, limit, opts...)
	if err != nil {
		return nil, err
	}

	species := make([]*response.Species, 0, len(fishesRecords))
	for _, fish := range fishesRecords {
		species = append(species, &response.Species{
			Name:  fish.Name,
			Count: strconv.FormatUint(fish.Count, 10),
		})
	}

	return species, nil
}
