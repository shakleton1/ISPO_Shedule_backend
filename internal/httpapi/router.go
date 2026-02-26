package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/config"
	"ispo-schedule/internal/pdf"
	"ispo-schedule/internal/push"
	"ispo-schedule/internal/schedule"
)

type RouterDeps struct {
	Config      *config.Config
	ScheduleSvc *schedule.Service
	Repo        *schedule.Repository
	PDF         *pdf.Engine
	Tokens      *auth.TokenManager
	DBPing      func(context.Context) error
	Push        *push.Service
}

func NewRouter(deps RouterDeps) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLoggingMiddleware())
	r.Use(metricsMiddleware(deps.DBPing))

	// Prometheus metrics endpoint (обычно без auth, т.к. его дергает Prometheus).
	r.GET("/metrics", metricsHandler(deps.DBPing))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
		})
		v1.GET("/metrics/health", metricsHealthHandler(deps.DBPing))

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

		pushGroup := v1.Group("/push")
		{
			pushGroup.POST("/register", handlePushRegister(deps.Repo))
			pushGroup.POST("/unregister", handlePushUnregister(deps.Repo))
		}

		admin := v1.Group("/admin")
		admin.Use(adminGateMiddleware(deps.Config.Admin.APIKey, deps.Tokens, deps.Repo))
		{
			adminRead := admin.Group("")
			adminRead.Use(requireAnyRole(auth.RoleAdmin, auth.RoleDispatcher, auth.RoleViewer))
			{
				adminRead.GET("/groups", handleAdminListGroups(deps.Repo))
				adminRead.GET("/subjects", handleAdminListSubjects(deps.Repo))
				adminRead.GET("/locations", handleAdminListLocations(deps.Repo))

				adminRead.GET("/templates", handleAdminListTemplates(deps.Repo))
				adminRead.GET("/overrides", handleAdminListOverrides(deps.Repo))
				adminRead.GET("/calendar-exceptions", handleAdminListCalendarExceptions(deps.Repo))
			}

			adminDictWrite := admin.Group("")
			adminDictWrite.Use(requireAnyRole(auth.RoleAdmin))
			{
				adminDictWrite.POST("/groups", handleAdminCreateGroup(deps.Repo))
				adminDictWrite.PUT("/groups/:id", handleAdminUpdateGroup(deps.Repo))
				adminDictWrite.DELETE("/groups/:id", handleAdminDeleteGroup(deps.Repo))

				adminDictWrite.POST("/subjects", handleAdminCreateSubject(deps.Repo))
				adminDictWrite.PUT("/subjects/:id", handleAdminUpdateSubject(deps.Repo))
				adminDictWrite.DELETE("/subjects/:id", handleAdminDeleteSubject(deps.Repo))

				adminDictWrite.POST("/locations", handleAdminCreateLocation(deps.Repo))
				adminDictWrite.PUT("/locations/:id", handleAdminUpdateLocation(deps.Repo))
				adminDictWrite.DELETE("/locations/:id", handleAdminDeleteLocation(deps.Repo))
			}

			adminScheduleWrite := admin.Group("")
			adminScheduleWrite.Use(requireAnyRole(auth.RoleAdmin, auth.RoleDispatcher))
			{
				adminScheduleWrite.POST("/import/templates/csv", handleAdminImportTemplatesCSV(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/import/templates/xlsx", handleAdminImportTemplatesXLSX(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/templates", handleAdminCreateTemplate(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/templates/:id", handleAdminUpdateTemplate(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/templates/:id", handleAdminDeleteTemplate(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/override", handleAdminCreateOverride(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/overrides/:id", handleAdminUpdateOverride(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/overrides/:id", handleAdminDeleteOverride(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/overlay", handleAdminUpsertOverlay(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/calendar-exceptions", handleAdminUpsertCalendarException(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/calendar-exceptions/:date", handleAdminDeleteCalendarException(deps.Repo, deps.Push))
			}
		}
	}

	return r
}
