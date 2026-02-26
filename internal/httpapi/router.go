package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/config"
	"ispo-schedule/internal/pdf"
	"ispo-schedule/internal/schedule"
)

type RouterDeps struct {
	Config      *config.Config
	ScheduleSvc *schedule.Service
	Repo        *schedule.Repository
	PDF         *pdf.Engine
	Tokens      *auth.TokenManager
}

func NewRouter(deps RouterDeps) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
		})

		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", handleLogin(deps.Tokens, deps.Repo))
			// /me requires JWT
			authGroup.GET("/me", authMiddleware(deps.Tokens, deps.Repo), handleMe(deps.Repo))
		}

		client := v1.Group("/schedule")
		{
			client.GET("/current", handleGetCurrentSchedule(deps.ScheduleSvc, deps.Repo))
			client.GET("/range", handleGetScheduleRange(deps.ScheduleSvc))
			client.GET("/version", handleGetScheduleVersion(deps.Repo))
			client.GET("/pdf", handleGetSchedulePDF(deps.ScheduleSvc, deps.Repo, pdfEngineAdapter{e: deps.PDF}))
		}

		admin := v1.Group("/admin")
		admin.Use(adminGateMiddleware(deps.Config.Admin.APIKey, deps.Tokens, deps.Repo))
		{
			admin.GET("/groups", handleAdminListGroups(deps.Repo))
			admin.POST("/groups", handleAdminCreateGroup(deps.Repo))
			admin.PUT("/groups/:id", handleAdminUpdateGroup(deps.Repo))
			admin.DELETE("/groups/:id", handleAdminDeleteGroup(deps.Repo))

			admin.GET("/subjects", handleAdminListSubjects(deps.Repo))
			admin.POST("/subjects", handleAdminCreateSubject(deps.Repo))
			admin.PUT("/subjects/:id", handleAdminUpdateSubject(deps.Repo))
			admin.DELETE("/subjects/:id", handleAdminDeleteSubject(deps.Repo))

			admin.GET("/locations", handleAdminListLocations(deps.Repo))
			admin.POST("/locations", handleAdminCreateLocation(deps.Repo))
			admin.PUT("/locations/:id", handleAdminUpdateLocation(deps.Repo))
			admin.DELETE("/locations/:id", handleAdminDeleteLocation(deps.Repo))

			admin.GET("/templates", handleAdminListTemplates(deps.Repo))
			admin.POST("/templates", handleAdminCreateTemplate(deps.Repo))
			admin.PUT("/templates/:id", handleAdminUpdateTemplate(deps.Repo))
			admin.DELETE("/templates/:id", handleAdminDeleteTemplate(deps.Repo))

			admin.GET("/overrides", handleAdminListOverrides(deps.Repo))
			admin.POST("/override", handleAdminCreateOverride(deps.Repo))
			admin.PUT("/overrides/:id", handleAdminUpdateOverride(deps.Repo))
			admin.DELETE("/overrides/:id", handleAdminDeleteOverride(deps.Repo))

			admin.POST("/overlay", handleAdminUpsertOverlay(deps.Repo))

			admin.GET("/calendar-exceptions", handleAdminListCalendarExceptions(deps.Repo))
			admin.POST("/calendar-exceptions", handleAdminUpsertCalendarException(deps.Repo))
			admin.DELETE("/calendar-exceptions/:date", handleAdminDeleteCalendarException(deps.Repo))
		}
	}

	return r
}
