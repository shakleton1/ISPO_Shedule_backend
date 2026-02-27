package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type limitOffset struct {
	Limit  *int
	Offset *int
}

func parseLimitOffset(c *gin.Context, defaultLimit *int, maxLimit int) (limitOffset, bool) {
	var out limitOffset

	limitStr := strings.TrimSpace(c.Query("limit"))
	offsetStr := strings.TrimSpace(c.Query("offset"))

	if limitStr == "" {
		out.Limit = defaultLimit
	} else {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v <= 0 {
			writeError(c, http.StatusBadRequest, "validation_error", "limit", "limit must be a positive integer")
			return limitOffset{}, false
		}
		if maxLimit > 0 && v > maxLimit {
			v = maxLimit
		}
		out.Limit = &v
	}

	if offsetStr == "" {
		if out.Limit != nil {
			zero := 0
			out.Offset = &zero
		}
	} else {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			writeError(c, http.StatusBadRequest, "validation_error", "offset", "offset must be a non-negative integer")
			return limitOffset{}, false
		}
		out.Offset = &v
	}

	if out.Limit == nil {
		// pagination disabled: ignore offset if provided
		out.Offset = nil
		return out, true
	}
	if out.Offset == nil {
		zero := 0
		out.Offset = &zero
	}
	return out, true
}
