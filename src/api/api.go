package api

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const (
	groupNameParam = "groupName"
	codeNameParam  = "codeName"

	groupRouteGroup       = "/group/:groupName"
	temperatureRouteGroup = "/region/temperature"
)

var pattern = regexp.MustCompile("([a-zA-Z]+)([0-9]+)")

type Server struct {
	routes  *gin.Engine
	storage *storage.Storage
}

func DefaultApiServer(storage *storage.Storage) *Server {
	server := &Server{
		routes:  gin.Default(),
		storage: storage,
	}
	server.initRoutes()

	return server
}

func (s *Server) initRoutes() {
	s.initGroupRoutes()
	s.initTemperatureRoutes()
	s.initSensorRoutes()
}

func (s *Server) initGroupRoutes() {
	s.routes.GET(groupRouteGroup, func(context *gin.Context) {
		groupRecords := s.storage.GetAllGroups()
		groups := make([]string, len(groupRecords))

		for i, record := range groupRecords {
			groups[i] = record.Name
		}

		context.JSON(http.StatusOK, GroupsResponse{
			Groups: groups,
		})
	})

	groups := s.routes.Group(groupRouteGroup)

	groups.GET("/transparency/average", func(context *gin.Context) {
		avg, err := s.storage.GetAvgTransparency(context)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, AvgResponse{
			Average: strconv.FormatUint(uint64(avg), 10),
		})
	})

	groups.GET("/temperature/average", func(context *gin.Context) {
		avg, err := s.storage.GetAvgTemperature(context)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, AvgResponse{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})

	groups.GET("/species", func(context *gin.Context) {
		groupName := context.Param(groupNameParam)
		context.JSON(http.StatusOK, SpeciesResponse{
			Species: getSpecies(s.storage, groupName, 0),
		})
	})

	species := s.routes.Group("/species")
	species.GET("top/:n", func(context *gin.Context) {
		count, err := strconv.Atoi(context.Param("n"))
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		groupName := context.Param(groupNameParam)
		context.JSON(http.StatusOK, SpeciesResponse{
			Species: getSpecies(s.storage, groupName, count),
		})
	})
}

func (s *Server) initTemperatureRoutes() {
	s.routes.GET(temperatureRouteGroup, func(context *gin.Context) {})

	groups := s.routes.Group(temperatureRouteGroup)
	groups.GET("/min", func(context *gin.Context) {
		context.JSON(http.StatusOK, ValueResponse{
			Value: strconv.FormatFloat(s.storage.GetMinTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
		})
	})

	groups.GET("/max", func(context *gin.Context) {
		context.JSON(http.StatusOK, ValueResponse{
			Value: strconv.FormatFloat(s.storage.GetMaxTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
		})
	})
}

func (s *Server) initSensorRoutes() {
	s.routes.GET("/sensor/:codeName/temperature/average", func(context *gin.Context) {
		matches := pattern.FindStringSubmatch(context.Param(codeNameParam))
		if len(matches) < 3 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid codeName"})
		}

		letter := matches[1]
		index, err := strconv.Atoi(matches[2])
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
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

			opts = append(opts, storage.WithFrom(from))
		}

		tillQ := context.DefaultQuery("from", "")
		if tillQ != "" {
			till, err := time.Parse(time.UnixDate, context.DefaultQuery("till", ""))
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
				return
			}

			opts = append(opts, storage.WithTill(till))
		}

		avg, err := s.storage.GetSensorAvgTemperature(letter, index, opts...)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, AvgResponse{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})
}

func getSpecies(storage *storage.Storage, groupName string, limit int) []*Species {
	fishesRecords := storage.GetSpecies(groupName, limit)
	species := make([]*Species, 0, len(fishesRecords))
	for _, fish := range fishesRecords {
		species = append(species, &Species{
			Name:  fish.Name,
			Count: strconv.FormatUint(fish.Count, 10),
		})
	}

	return species
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
