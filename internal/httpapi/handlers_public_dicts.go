package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handlePublicListGroups(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var (
			rows []schedule.Group
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListGroupsPaged(p.Limit, p.Offset)
		} else {
			rows, err = repo.ListGroups()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]groupDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toGroupDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handlePublicListSubjects(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var (
			rows []schedule.Subject
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListSubjectsPaged(p.Limit, p.Offset)
		} else {
			rows, err = repo.ListSubjects()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]subjectDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toSubjectDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handlePublicListLocations(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var (
			rows []schedule.Location
			err  error
		)
		if p.Limit != nil {
			rows, err = repo.ListLocationsPaged(p.Limit, p.Offset)
		} else {
			rows, err = repo.ListLocations()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]locationDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toLocationDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}
