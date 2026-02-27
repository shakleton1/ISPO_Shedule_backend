package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func handleAdminExplainScheduleSlot(svc *schedule.Service, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.Atoi(c.Query("group_id"))
		if err != nil || groupID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_id required"})
			return
		}
		dateStr := c.Query("date")
		if dateStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date required"})
			return
		}
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		pair, err := strconv.Atoi(c.Query("pair_number"))
		if err != nil || pair <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pair_number required"})
			return
		}
		var subgroup *int16
		if v := c.Query("subgroup"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subgroup"})
				return
			}
			sg := int16(i)
			subgroup = &sg
		}

		out, err := svc.ExplainSlot(groupID, d, int16(pair), subgroup)
		if err != nil {
			// Service methods already return friendly errors.
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// include repo in signature to keep handler patterns consistent (and allow future additions)
		_ = repo
		c.JSON(http.StatusOK, out)
	}
}
