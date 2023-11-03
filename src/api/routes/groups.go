package routes

import (
	"net/http"
	"strconv"

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
		groupRecords := s.GetAllGroups()
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
		avg, err := s.GetAvgTransparency(context)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, response.Average{
			Average: strconv.FormatUint(uint64(avg), 10),
		})
	})

	groups.GET("/temperature/average", func(context *gin.Context) {
		avg, err := s.GetAvgTemperature(context)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, response.Average{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})

	groups.GET("/species", func(context *gin.Context) {
		groupName := context.Param(groupNameParam)
		context.JSON(http.StatusOK, response.SpeciesList{
			Species: getSpecies(s, groupName, 0),
		})
	})

	species := routes.Group("/species")
	species.GET("top/:n", func(context *gin.Context) {
		count, err := strconv.Atoi(context.Param("n"))
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		groupName := context.Param(groupNameParam)
		context.JSON(http.StatusOK, response.SpeciesList{
			Species: getSpecies(s, groupName, count),
		})
	})
}

func getSpecies(storage *storage.Storage, groupName string, limit int) []*response.Species {
	fishesRecords := storage.GetSpecies(groupName, limit)
	species := make([]*response.Species, 0, len(fishesRecords))
	for _, fish := range fishesRecords {
		species = append(species, &response.Species{
			Name:  fish.Name,
			Count: strconv.FormatUint(fish.Count, 10),
		})
	}

	return species
}
