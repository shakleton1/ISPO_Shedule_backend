package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

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
	if len(deps.Config.Server.Proxy.TrustedProxies) > 0 {
		r.ForwardedByClientIP = true
		r.RemoteIPHeaders = []string{"X-Forwarded-For", "X-Real-IP"}
		_ = r.SetTrustedProxies(deps.Config.Server.Proxy.TrustedProxies)
	} else {
		// Security default: do not trust client-supplied forwarded IP headers.
		r.ForwardedByClientIP = false
		r.RemoteIPHeaders = nil
		_ = r.SetTrustedProxies([]string{})
	}
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		// Do not leak panic details to clients.
		_ = recovered
		abortWithError(c, http.StatusInternalServerError, "internal_error", "", "internal server error")
	}))
	r.Use(securityHeadersMiddleware())
	r.Use(corsMiddleware(deps.Config.Server.CORS))
	r.Use(requestIDMiddleware())
	if deps.Config.Server.Tracing.Enabled {
		r.Use(otelgin.Middleware(deps.Config.Server.Tracing.ServiceName))
	}
	r.Use(requestLoggingMiddleware())
	r.Use(metricsMiddleware(deps.DBPing))

	r.NoRoute(func(c *gin.Context) {
		abortWithError(c, http.StatusNotFound, "not_found", "", "not found")
	})
	r.NoMethod(func(c *gin.Context) {
		abortWithError(c, http.StatusMethodNotAllowed, "method_not_allowed", "", "method not allowed")
	})

	rlStore := newRateLimitStore(10 * time.Minute)

	// Prometheus metrics endpoint (обычно без auth, т.к. его дергает Prometheus).
	r.GET("/metrics", metricsHandler(deps.DBPing))

	// OpenAPI spec (YAML).
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Header("Content-Type", "application/yaml")
		c.File("docs/openapi.yaml")
	})

	// Swagger UI (loads spec from /openapi.yaml).
	swaggerUI := func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, `<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<title>ISPO Smart Schedule API Docs</title>
		<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
		<style>
			body { margin: 0; }
		</style>
	</head>
	<body>
		<div id="swagger-ui"></div>
		<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
		<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
		<script>
			window.onload = function () {
				window.ui = SwaggerUIBundle({
					url: '/openapi.yaml',
					dom_id: '#swagger-ui',
					deepLinking: true,
					presets: [
						SwaggerUIBundle.presets.apis,
						SwaggerUIStandalonePreset
					],
					layout: 'StandaloneLayout'
				});
			};
		</script>
	</body>
</html>`)
	}
	r.GET("/docs", swaggerUI)
	r.GET("/docs/", swaggerUI)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
		})
		v1.GET("/metrics/health", metricsHealthHandler(deps.DBPing))

		authGroup := v1.Group("/auth")
		{
			authGroup.POST(
				"/login",
				rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.Login, "auth_login"),
				handleLogin(deps.Tokens, deps.Repo, deps.Config.Auth.RefreshTokenTTL),
			)
			authGroup.POST(
				"/refresh",
				rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.Login, "auth_refresh"),
				handleRefresh(deps.Tokens, deps.Repo, deps.Config.Auth.RefreshTokenTTL),
			)
			authGroup.POST(
				"/logout",
				rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.Login, "auth_logout"),
				handleLogout(deps.Repo),
			)
			// /me requires JWT
			authGroup.GET("/me", authMiddleware(deps.Tokens, deps.Repo), handleMe(deps.Repo))
		}

		client := v1.Group("/schedule")
		{
			client.GET("/current", handleGetCurrentSchedule(deps.ScheduleSvc, deps.Repo))
			client.GET("/range", handleGetScheduleRange(deps.ScheduleSvc))
			client.GET("/version", handleGetScheduleVersion(deps.Repo))
			client.GET(
				"/pdf",
				rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.SchedulePDF, "schedule_pdf"),
				handleGetSchedulePDF(deps.ScheduleSvc, deps.Repo, pdfEngineAdapter{e: deps.PDF}),
			)
		}

		pushGroup := v1.Group("/push")
		{
			pushGroup.POST("/register", handlePushRegister(deps.Repo))
			pushGroup.POST("/unregister", handlePushUnregister(deps.Repo))
		}

		// Public dictionary endpoints for clients.
		v1.GET("/groups", handlePublicListGroups(deps.Repo))
		v1.GET("/subjects", handlePublicListSubjects(deps.Repo))
		v1.GET("/locations", handlePublicListLocations(deps.Repo))

		admin := v1.Group("/admin")
		admin.Use(adminGateMiddleware(deps.Config.Admin.APIKey, deps.Tokens, deps.Repo))
		{
			if deps.Config.Server.Debug.Pprof.Enabled {
				debug := admin.Group("/debug")
				debug.Use(requireAnyRole(auth.RoleAdmin))
				registerPprofRoutes(debug)
			}

			adminRead := admin.Group("")
			adminRead.Use(requireAnyPermission(PermAdminRead))
			{
				adminRead.GET("/groups", handleAdminListGroups(deps.Repo))
				adminRead.GET("/subjects", handleAdminListSubjects(deps.Repo))
				adminRead.GET("/locations", handleAdminListLocations(deps.Repo))
				adminRead.GET("/teachers", handleAdminListTeachers(deps.Repo))
				adminRead.GET("/teacher-subjects", handleAdminListTeacherSubjects(deps.Repo))
				adminRead.GET("/course-assignments", handleAdminListCourseAssignments(deps.Repo))

				adminRead.GET("/db/schema", handleAdminDBSchema(deps.Repo.DB()))
				adminRead.GET("/specialties", handleAdminListSpecialties(deps.Repo))
				adminRead.GET("/curricula", handleAdminListCurricula(deps.Repo))
				adminRead.GET("/curricula/:id/calendars", handleAdminListAcademicCalendars(deps.Repo))
				adminRead.GET("/calendars/:id/weeks", handleAdminListAcademicCalendarWeeks(deps.Repo))
				adminRead.GET("/curricula/:id/items", handleAdminListCurriculumItems(deps.Repo))
				adminRead.GET("/curriculum-items/:id/allocations", handleAdminListCurriculumItemAllocations(deps.Repo))

				adminRead.GET("/templates", handleAdminListTemplates(deps.Repo))
				adminRead.GET("/schedule/explain", handleAdminExplainScheduleSlot(deps.ScheduleSvc, deps.Repo))
				adminRead.GET("/overrides", handleAdminListOverrides(deps.Repo))
				adminRead.GET("/calendar-exceptions", handleAdminListCalendarExceptions(deps.Repo))
				adminRead.GET("/day-events", handleAdminListDayEvents(deps.Repo))
			}

			adminDictWrite := admin.Group("")
			adminDictWrite.Use(requireAnyPermission(PermDictWrite))
			{
				adminDictWrite.POST("/groups", handleAdminCreateGroup(deps.Repo))
				adminDictWrite.PUT("/groups/:id", handleAdminUpdateGroup(deps.Repo))
				adminDictWrite.DELETE("/groups/:id", handleAdminDeleteGroup(deps.Repo))

				adminDictWrite.POST("/teachers", handleAdminCreateTeacher(deps.Repo))
				adminDictWrite.PUT("/teachers/:id", handleAdminUpdateTeacher(deps.Repo))
				adminDictWrite.DELETE("/teachers/:id", handleAdminDeleteTeacher(deps.Repo))

				adminDictWrite.POST("/teacher-subjects", handleAdminCreateTeacherSubject(deps.Repo))
				adminDictWrite.DELETE("/teacher-subjects/:teacher_id/:subject_id", handleAdminDeleteTeacherSubject(deps.Repo))

				adminDictWrite.POST("/specialties", handleAdminCreateSpecialty(deps.Repo))
				adminDictWrite.PUT("/specialties/:id", handleAdminUpdateSpecialty(deps.Repo))
				adminDictWrite.DELETE("/specialties/:id", handleAdminDeleteSpecialty(deps.Repo))

				adminDictWrite.POST("/curricula", handleAdminCreateCurriculum(deps.Repo))
				adminDictWrite.PUT("/curricula/:id", handleAdminUpdateCurriculum(deps.Repo))
				adminDictWrite.DELETE("/curricula/:id", handleAdminDeleteCurriculum(deps.Repo))

				adminDictWrite.POST("/curricula/:id/calendars", handleAdminCreateAcademicCalendar(deps.Repo))
				adminDictWrite.DELETE("/calendars/:id", handleAdminDeleteAcademicCalendar(deps.Repo))
				adminDictWrite.PUT("/calendars/:id/weeks", handleAdminUpsertAcademicCalendarWeeks(deps.Repo))

				adminDictWrite.POST("/curricula/:id/items", handleAdminCreateCurriculumItem(deps.Repo))
				adminDictWrite.PUT("/curriculum-items/:id", handleAdminUpdateCurriculumItem(deps.Repo))
				adminDictWrite.DELETE("/curriculum-items/:id", handleAdminDeleteCurriculumItem(deps.Repo))
				adminDictWrite.PUT("/curriculum-items/:id/allocations", handleAdminUpsertCurriculumItemAllocations(deps.Repo))

				adminDictWrite.POST("/subjects", handleAdminCreateSubject(deps.Repo))
				adminDictWrite.PUT("/subjects/:id", handleAdminUpdateSubject(deps.Repo))
				adminDictWrite.DELETE("/subjects/:id", handleAdminDeleteSubject(deps.Repo))

				adminDictWrite.POST("/locations", handleAdminCreateLocation(deps.Repo))
				adminDictWrite.PUT("/locations/:id", handleAdminUpdateLocation(deps.Repo))
				adminDictWrite.DELETE("/locations/:id", handleAdminDeleteLocation(deps.Repo))
			}

			adminScheduleWrite := admin.Group("")
			adminScheduleWrite.Use(requireAnyPermission(PermScheduleWrite))
			{
				// Import is restricted by permission (admin by default).
				adminImport := adminScheduleWrite.Group("")
				adminImport.Use(requireAnyPermission(PermImport))
				{
					adminImport.POST(
						"/import/templates/csv",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportTemplatesCSV(deps.Repo, deps.Push),
					)
					adminImport.POST(
						"/import/templates/xlsx",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportTemplatesXLSX(deps.Repo, deps.Push),
					)
				}

				adminScheduleWrite.POST("/templates", handleAdminCreateTemplate(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/templates/:id", handleAdminUpdateTemplate(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/templates/:id", handleAdminDeleteTemplate(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/templates/publish", handleAdminPublishDraftTemplates(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/templates/discard-drafts", handleAdminDiscardDraftTemplates(deps.Repo))

				adminScheduleWrite.POST("/course-assignments", handleAdminCreateCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/course-assignments/:id", handleAdminUpdateCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/course-assignments/:id", handleAdminDeleteCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/course-assignments/publish", handleAdminPublishDraftCourseAssignments(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/course-assignments/discard-drafts", handleAdminDiscardDraftCourseAssignments(deps.Repo))

				adminScheduleWrite.POST("/override", handleAdminCreateOverride(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/overrides/:id", handleAdminUpdateOverride(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/overrides/:id", handleAdminDeleteOverride(deps.Repo, deps.Push))
				adminScheduleWrite.GET("/schedule/validate", handleAdminValidateSchedule(deps.ScheduleSvc))

				adminScheduleWrite.POST("/overrides/bulk", handleAdminBulkOverrides(deps.Repo))
				adminScheduleWrite.POST("/override/move", handleAdminMovePair(deps.ScheduleSvc, deps.Repo))

				adminScheduleWrite.POST("/overlay", handleAdminUpsertOverlay(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/calendar-exceptions", handleAdminUpsertCalendarException(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/calendar-exceptions/:date", handleAdminDeleteCalendarException(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/day-events", handleAdminCreateDayEvent(deps.Repo))
				adminScheduleWrite.PUT("/day-events/:id", handleAdminUpdateDayEvent(deps.Repo))
				adminScheduleWrite.DELETE("/day-events/:id", handleAdminDeleteDayEvent(deps.Repo))
			}
		}
	}

	return r
}
