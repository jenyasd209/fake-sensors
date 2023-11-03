package api

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/db"

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
	routes *gin.Engine
	db     *db.Database
}

func DefaultApiServer(db *db.Database) *Server {
	server := &Server{
		routes: gin.Default(),
		db:     db,
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
		groupRecords := s.db.GetAllGroups()
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
		avg, err := s.db.GetAvgTransparency(context)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, AvgResponse{
			Average: strconv.FormatUint(uint64(avg), 10),
		})
	})

	groups.GET("/temperature/average", func(context *gin.Context) {
		avg, err := s.db.GetAvgTemperature(context)
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
			Species: getSpecies(s.db, groupName, 0),
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
			Species: getSpecies(s.db, groupName, count),
		})
	})
}

func (s *Server) initTemperatureRoutes() {
	s.routes.GET(temperatureRouteGroup, func(context *gin.Context) {})

	groups := s.routes.Group(temperatureRouteGroup)
	groups.GET("/min", func(context *gin.Context) {
		context.JSON(http.StatusOK, ValueResponse{
			Value: strconv.FormatFloat(s.db.GetMinTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
		})
	})

	groups.GET("/max", func(context *gin.Context) {
		context.JSON(http.StatusOK, ValueResponse{
			Value: strconv.FormatFloat(s.db.GetMaxTemperatureByRegion(parseCoordinates(context)...), 'f', 2, 64),
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

		opts := make([]db.ConditionOption, 0, 2)
		fromQ := context.DefaultQuery("from", "")
		if fromQ != "" {
			from, err := time.Parse(time.UnixDate, context.DefaultQuery("from", ""))
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
				return
			}

			opts = append(opts, db.WithFrom(from))
		}

		tillQ := context.DefaultQuery("from", "")
		if tillQ != "" {
			till, err := time.Parse(time.UnixDate, context.DefaultQuery("till", ""))
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
				return
			}

			opts = append(opts, db.WithTill(till))
		}

		avg, err := s.db.GetSensorAvgTemperature(letter, index, opts...)
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		context.JSON(http.StatusOK, AvgResponse{
			Average: strconv.FormatFloat(avg, 'f', 2, 64),
		})
	})
}

func getSpecies(db *db.Database, groupName string, limit int) []*Species {
	fishesRecords := db.GetSpecies(groupName, limit)
	species := make([]*Species, 0, len(fishesRecords))
	for _, fish := range fishesRecords {
		species = append(species, &Species{
			Name:  fish.Name,
			Count: strconv.FormatUint(fish.Count, 10),
		})
	}

	return species
}

func parseCoordinates(ctx *gin.Context) []db.CoordinateOption {
	opts := make([]db.CoordinateOption, 0, 6)

	xMin := ctx.Query("xMin")
	if xMin != "" {
		opts = append(opts, db.WithXMin(ctx.GetFloat64("xMin")))
	}
	xMax := ctx.Query("xMax")
	if xMax != "" {
		opts = append(opts, db.WithXMax(ctx.GetFloat64("xMax")))
	}

	yMin := ctx.Query("yMin")
	if yMin != "" {
		opts = append(opts, db.WithYMin(ctx.GetFloat64("yMin")))
	}
	yMax := ctx.Query("yMax")
	if yMax != "" {
		opts = append(opts, db.WithYMax(ctx.GetFloat64("yMax")))
	}

	zMin := ctx.Query("zMin")
	if zMin != "" {
		opts = append(opts, db.WithZMin(ctx.GetFloat64("zMin")))
	}
	zMax := ctx.Query("zMax")
	if zMax != "" {
		opts = append(opts, db.WithZMax(ctx.GetFloat64("zMax")))
	}

	return opts
}
