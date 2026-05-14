package httpapi

import (
	"context"
	"net/http"
	"os"
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
		b, err := os.ReadFile("docs/openapi.yaml")
		if err != nil {
			abortWithError(c, http.StatusInternalServerError, "openapi_unavailable", "", "openapi spec unavailable")
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", b)
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
		v1.GET("/campuses", handleAdminListCampuses(deps.Repo))
		v1.GET("/location-types", handleAdminListLocationTypes(deps.Repo))

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
				adminRead.GET("/campuses", handleAdminListCampuses(deps.Repo))
				adminRead.GET("/location-types", handleAdminListLocationTypes(deps.Repo))
				adminRead.GET("/location-type-links", handleAdminListLocationTypeLinks(deps.Repo))
				adminRead.GET("/teachers", handleAdminListTeachers(deps.Repo))
				adminRead.GET("/teacher-subjects", handleAdminListTeacherSubjects(deps.Repo))
				adminRead.GET("/course-assignments", handleAdminListCourseAssignments(deps.Repo))
				adminRead.GET("/teacher-location-preferences", handleAdminListTeacherLocationPreferences(deps.Repo))
				adminRead.GET("/room-requests", handleAdminListRoomRequests(deps.Repo))
				adminRead.GET("/room-assignments", handleAdminListRoomAssignments(deps.Repo))
				adminRead.GET("/study-activities", handleAdminListStudyActivities(deps.Repo))
				adminRead.GET("/study-calendar-weeks", handleAdminListStudyCalendarWeeks(deps.Repo))
				adminRead.GET("/teacher-day-constraints", handleAdminListTeacherDayConstraints(deps.Repo))
				adminRead.GET("/calendar-day-constraints", handleAdminListCalendarDayConstraints(deps.Repo))
				adminRead.GET("/replacements", handleAdminListScheduleReplacements(deps.Repo))
				adminRead.GET("/location-availability/weeks", handleAdminListLocationWeekAvailability(deps.Repo))

				adminRead.GET("/db/schema", handleAdminDBSchema(deps.Repo.DB()))
				adminRead.GET("/specialties", handleAdminListSpecialties(deps.Repo))
				adminRead.GET("/curricula", handleAdminListCurricula(deps.Repo))
				adminRead.GET("/curricula/:id/calendars", handleAdminListAcademicCalendars(deps.Repo))
				adminRead.GET("/calendars/:id/weeks", handleAdminListAcademicCalendarWeeks(deps.Repo))
				adminRead.GET("/curricula/:id/items", handleAdminListCurriculumItems(deps.Repo))
				adminRead.GET("/curriculum-items/:id/allocations", handleAdminListCurriculumItemAllocations(deps.Repo))

				adminRead.GET("/schedule-lessons", handleAdminListScheduleLessons(deps.Repo))
				adminRead.GET("/schedule/view", handleAdminScheduleView(deps.ScheduleSvc))
				adminRead.GET("/schedule/explain", handleAdminExplainScheduleSlot(deps.ScheduleSvc, deps.Repo))
				adminRead.GET("/overrides", handleAdminListAppliedScheduleOverrides(deps.Repo))
				adminRead.GET("/reports/schedule-overrides", handleAdminListAppliedScheduleOverrides(deps.Repo))
				adminRead.GET("/day-events", handleAdminListDayEvents(deps.Repo))
			}

			adminDictWrite := admin.Group("")
			adminDictWrite.Use(requireAnyPermission(PermDictWrite))
			{
				adminDictWrite.POST("/groups", handleAdminCreateGroup(deps.Repo, deps.Push))
				adminDictWrite.PUT("/groups/:id", handleAdminUpdateGroup(deps.Repo, deps.Push))
				adminDictWrite.DELETE("/groups/:id", handleAdminDeleteGroup(deps.Repo, deps.Push))

				adminDictWrite.POST("/teachers", handleAdminCreateTeacher(deps.Repo))
				adminDictWrite.PUT("/teachers/:id", handleAdminUpdateTeacher(deps.Repo))
				adminDictWrite.DELETE("/teachers/:id", handleAdminDeleteTeacher(deps.Repo))

				adminDictWrite.POST("/teacher-subjects", handleAdminCreateTeacherSubject(deps.Repo))
				adminDictWrite.DELETE("/teacher-subjects/:teacher_id/:subject_id", handleAdminDeleteTeacherSubject(deps.Repo))
				adminDictWrite.POST("/study-activities", handleAdminCreateStudyActivity(deps.Repo))
				adminDictWrite.PUT("/study-activities/:id", handleAdminUpdateStudyActivity(deps.Repo))
				adminDictWrite.DELETE("/study-activities/:id", handleAdminDeleteStudyActivity(deps.Repo))

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

				adminDictWrite.POST("/campuses", handleAdminCreateCampus(deps.Repo))
				adminDictWrite.PUT("/campuses/:id", handleAdminUpdateCampus(deps.Repo))
				adminDictWrite.DELETE("/campuses/:id", handleAdminDeleteCampus(deps.Repo))

				adminDictWrite.POST("/location-types", handleAdminCreateLocationType(deps.Repo))
				adminDictWrite.PUT("/location-types/:id", handleAdminUpdateLocationType(deps.Repo))
				adminDictWrite.DELETE("/location-types/:id", handleAdminDeleteLocationType(deps.Repo))
				adminDictWrite.POST("/location-type-links", handleAdminCreateLocationTypeLink(deps.Repo))
				adminDictWrite.DELETE("/location-type-links/:location_id/:type_id", handleAdminDeleteLocationTypeLink(deps.Repo))
			}

			adminScheduleWrite := admin.Group("")
			adminScheduleWrite.Use(requireAnyPermission(PermScheduleWrite))
			{
				// Import is restricted by permission (admin by default).
				adminImport := adminScheduleWrite.Group("")
				adminImport.Use(requireAnyPermission(PermImport))
				{
					adminImport.POST(
						"/import/curriculum-items/csv",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportCurriculumItemsCSV(deps.Repo),
					)
					adminImport.POST(
						"/import/curriculum-items/xlsx",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportCurriculumItemsXLSX(deps.Repo),
					)
					adminImport.POST(
						"/import/plx-curriculum/xlsx",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportPLXCurriculumXLSX(deps.Repo),
					)
					adminImport.POST(
						"/import/study-calendar/csv",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportStudyCalendarCSV(deps.Repo, deps.Push),
					)
					adminImport.POST(
						"/import/study-calendar/xlsx",
						maxBodyBytesMiddleware(deps.Config.Server.AdminImportMaxBodyBytes),
						rateLimitMiddleware(rlStore, deps.Config.Server.RateLimit.AdminImport, "admin_import"),
						handleAdminImportStudyCalendarXLSX(deps.Repo, deps.Push),
					)
				}

				adminScheduleWrite.POST("/schedule-lessons", handleAdminCreateScheduleLesson(deps.ScheduleSvc, deps.Repo, deps.Push))
				adminScheduleWrite.PATCH("/schedule-lessons/:id", handleAdminUpdateScheduleLesson(deps.ScheduleSvc, deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/schedule-lessons/:id", handleAdminUpdateScheduleLesson(deps.ScheduleSvc, deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/schedule-lessons/:id", handleAdminDeleteScheduleLesson(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/schedule-lessons/:id/cancel", handleAdminCancelScheduleLesson(deps.ScheduleSvc, deps.Repo, deps.Push))

				adminScheduleWrite.POST("/course-assignments", handleAdminCreateCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/course-assignments/:id", handleAdminUpdateCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/course-assignments/:id", handleAdminDeleteCourseAssignment(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/course-assignments/publish", handleAdminPublishDraftCourseAssignments(deps.Repo, deps.Push))
				adminScheduleWrite.POST("/course-assignments/discard-drafts", handleAdminDiscardDraftCourseAssignments(deps.Repo))
				adminScheduleWrite.POST("/teacher-location-preferences", handleAdminCreateTeacherLocationPreference(deps.Repo))
				adminScheduleWrite.PUT("/teacher-location-preferences/:id", handleAdminUpdateTeacherLocationPreference(deps.Repo))
				adminScheduleWrite.DELETE("/teacher-location-preferences/:id", handleAdminDeleteTeacherLocationPreference(deps.Repo))
				adminScheduleWrite.POST("/room-requests", handleAdminCreateRoomRequest(deps.Repo))
				adminScheduleWrite.PUT("/room-requests/:id", handleAdminUpdateRoomRequest(deps.Repo))
				adminScheduleWrite.DELETE("/room-requests/:id", handleAdminDeleteRoomRequest(deps.Repo))
				adminScheduleWrite.POST("/room-assignments", handleAdminCreateRoomAssignment(deps.Repo))
				adminScheduleWrite.PUT("/room-assignments/:id", handleAdminUpdateRoomAssignment(deps.Repo))
				adminScheduleWrite.DELETE("/room-assignments/:id", handleAdminDeleteRoomAssignment(deps.Repo))
				adminScheduleWrite.PUT("/groups/:id/study-calendar-weeks", handleAdminUpsertStudyCalendarWeeks(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/study-calendar-weeks/:id", handleAdminDeleteStudyCalendarWeek(deps.Repo))

				adminScheduleWrite.POST("/teacher-day-constraints", handleAdminCreateTeacherDayConstraint(deps.Repo))
				adminScheduleWrite.PUT("/teacher-day-constraints/:id", handleAdminUpdateTeacherDayConstraint(deps.Repo))
				adminScheduleWrite.DELETE("/teacher-day-constraints/:id", handleAdminDeleteTeacherDayConstraint(deps.Repo))
				adminScheduleWrite.POST("/calendar-day-constraints", handleAdminCreateCalendarDayConstraint(deps.Repo, deps.Push))
				adminScheduleWrite.PATCH("/calendar-day-constraints/:id", handleAdminUpdateCalendarDayConstraint(deps.Repo, deps.Push))
				adminScheduleWrite.PUT("/calendar-day-constraints/:id", handleAdminUpdateCalendarDayConstraint(deps.Repo, deps.Push))
				adminScheduleWrite.DELETE("/calendar-day-constraints/:id", handleAdminDeleteCalendarDayConstraint(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/replacements", handleAdminCreateScheduleReplacement(deps.Repo))
				adminScheduleWrite.PUT("/replacements/:id", handleAdminUpdateScheduleReplacement(deps.Repo))
				adminScheduleWrite.DELETE("/replacements/:id", handleAdminDeleteScheduleReplacement(deps.Repo))
				adminScheduleWrite.PUT("/location-availability/weeks", handleAdminUpsertLocationWeekAvailability(deps.Repo))
				adminScheduleWrite.DELETE("/location-availability/weeks/:id", handleAdminDeleteLocationWeekAvailability(deps.Repo))
				adminScheduleWrite.POST("/location-availability/autofill", handleAdminAutofillLocations(deps.ScheduleSvc, deps.Repo, deps.Push))

				adminScheduleWrite.POST("/schedule-overrides/apply", handleAdminApplyScheduleOverride(deps.ScheduleSvc, deps.Repo, deps.Push))
				adminScheduleWrite.GET("/schedule/validate", handleAdminValidateSchedule(deps.ScheduleSvc))

				adminScheduleWrite.POST("/overlay", handleAdminUpsertOverlay(deps.Repo, deps.Push))

				adminScheduleWrite.POST("/day-events", handleAdminCreateDayEvent(deps.Repo))
				adminScheduleWrite.PUT("/day-events/:id", handleAdminUpdateDayEvent(deps.Repo))
				adminScheduleWrite.DELETE("/day-events/:id", handleAdminDeleteDayEvent(deps.Repo))
			}
		}
	}

	return r
}
