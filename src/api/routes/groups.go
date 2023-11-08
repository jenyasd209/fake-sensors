package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jenyasd209/fake-sensors/src/storage"

	"github.com/gin-gonic/gin"
)

const (
	groupNameParam = "groupName"

	groupRouteGroup      = "/group/:" + groupNameParam
	groupAvgTransparency = "/transparency/average"
	groupAvgTemperature  = "/temperature/average"
	groupSpecies         = "/species"
	groupTopSpecies      = "/top/:n"
)

func RegisterGroupRoutes(router *Router) {
	router.routes.GET("/group", router.GetGroups)

	groups := router.routes.Group(groupRouteGroup)

	groups.GET(groupAvgTransparency, router.GetGroupAvgTransparency)
	groups.GET(groupAvgTemperature, router.GetGroupAvgTemperature)

	groups.GET(groupSpecies, router.GetGroupSpecies)
	species := groups.Group(groupSpecies)
	species.GET(groupTopSpecies, router.GetGroupTopSpecies)
}

// @Summary Get groups list
// @Description Get groups list
// @Produce json
// @Success 200 {object} Groups
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /group [get]
func (r *Router) GetGroups(context *gin.Context) {
	groupRecords, err := r.storage.GetAllGroups()
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	groups := make([]string, len(groupRecords))
	for i, record := range groupRecords {
		groups[i] = record.Name
	}

	context.JSON(http.StatusOK, Groups{
		Groups: groups,
	})
}

// @Summary Get current average transparency inside the group
// @Description Get the current average transparency within a group.
// @Produce json
// @Param groupName path string true "Group name"
// @Success 200 {object} Average
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /group/{groupName}/transparency/average [get]
func (r *Router) GetGroupAvgTransparency(context *gin.Context) {
	avg, err := r.storage.GetAvgTransparency(context, context.Param(groupNameParam))
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Average{
		Average: strconv.FormatUint(uint64(avg), 10),
	})
}

// @Summary Get current average temperature inside the group
// @Description Get the current average temperature within a group.
// @Produce json
// @Param groupName path string true "Group name"
// @Success 200 {object} Average
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /group/{groupName}/temperature/average [get]
func (r *Router) GetGroupAvgTemperature(context *gin.Context) {
	avg, err := r.storage.GetAvgTemperature(context, context.Param(groupNameParam))
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, Average{
		Average: strconv.FormatFloat(avg, 'f', 2, 64),
	})
}

// @Summary Get full list of species inside the group
// @Description Get full list of species (with counts) currently detected inside the group.
// @Produce json
// @Param groupName path string true "Group name"
// @Success 200 {object} SpeciesList
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /group/{groupName}/species [get]
func (r *Router) GetGroupSpecies(context *gin.Context) {
	species, err := getSpecies(r.storage, context.Param(groupNameParam), 0)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, SpeciesList{
		Species: species,
	})
}

// @Summary Get full list of N species inside the group
// @Description Get full list of N species (with counts) currently detected inside the group.
// @Produce json
// @Param groupName path string true "Group name"
// @Param n path int true "Count of species"
// @Param from query string false "From (UNIX timestamps)"
// @Param till query string false "Till (UNIX timestamps)"
// @Success 200 {object} SpeciesList
// @Failure 400 {object} ErrorResponse "error message"
// @Failure 500 {object} ErrorResponse "error message"
// @Router /group/{groupName}/species/top/:n [get]
func (r *Router) GetGroupTopSpecies(context *gin.Context) {
	count, err := strconv.Atoi(context.Param("n"))
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

	species, err := getSpecies(r.storage, context.Param(groupNameParam), count, opts...)
	if err != nil {
		context.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	context.JSON(http.StatusOK, SpeciesList{
		Species: species,
	})
}

func getSpecies(storage *storage.Storage, groupName string, top int, opts ...storage.ConditionOption) ([]*Species, error) {
	fishesRecords, err := storage.GetCurrentSpecies(groupName, top, opts...)
	if err != nil {
		return nil, err
	}

	species := make([]*Species, 0, len(fishesRecords))
	for _, fish := range fishesRecords {
		species = append(species, &Species{
			Name:  fish.Name,
			Count: strconv.FormatUint(fish.Count, 10),
		})
	}

	return species, nil
}
