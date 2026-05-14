package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

func parseQueryInt(c *gin.Context, name string) (*int, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		writeValidationError(c, name, "invalid "+name)
		return nil, false
	}
	return &i, true
}

func parseQueryInt64(c *gin.Context, name string) (*int64, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		writeValidationError(c, name, "invalid "+name)
		return nil, false
	}
	return &i, true
}

func parseQueryInt16(c *gin.Context, name string) (*int16, bool) {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil, true
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		writeValidationError(c, name, "invalid "+name)
		return nil, false
	}
	out := int16(i)
	return &out, true
}

func handleAdminListCampuses(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.Campus
		var err error
		if page.Limit != nil {
			rows, err = repo.ListCampusesPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListCampuses()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]campusDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toCampusDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateCampus(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Campus
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateCampus(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "campuses", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toCampusDTO(req))
	}
}

func handleAdminUpdateCampus(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.Campus
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateCampus(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "campuses", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toCampusDTO(*row))
	}
}

func handleAdminDeleteCampus(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteCampus(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "campuses", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListLocationTypes(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		var rows []schedule.LocationType
		var err error
		if page.Limit != nil {
			rows, err = repo.ListLocationTypesPaged(page.Limit, page.Offset)
		} else {
			rows, err = repo.ListLocationTypes()
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]locationTypeDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toLocationTypeDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateLocationType(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.LocationType
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateLocationType(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "location_types", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, toLocationTypeDTO(req))
	}
}

func handleAdminUpdateLocationType(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.LocationType
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateLocationType(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "location_types", strconv.Itoa(id), row)
		c.JSON(http.StatusOK, toLocationTypeDTO(*row))
	}
}

func handleAdminDeleteLocationType(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteLocationType(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "location_types", strconv.Itoa(id), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListLocationTypeLinks(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID, ok := parseQueryInt(c, "location_id")
		if !ok {
			return
		}
		typeID, ok := parseQueryInt(c, "type_id")
		if !ok {
			return
		}
		rows, err := repo.ListLocationTypeLinks(schedule.LocationTypeLinkFilters{LocationID: locationID, TypeID: typeID})
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]locationTypeLinkDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toLocationTypeLinkDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateLocationTypeLink(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.LocationTypeLink
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateLocationTypeLink(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "location_type_links", strconv.Itoa(req.LocationID)+":"+strconv.Itoa(req.TypeID), req)
		c.JSON(http.StatusCreated, toLocationTypeLinkDTO(req))
	}
}

func handleAdminDeleteLocationTypeLink(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		locationID, err := strconv.Atoi(c.Param("location_id"))
		if err != nil {
			writeValidationError(c, "location_id", "invalid location_id")
			return
		}
		typeID, err := strconv.Atoi(c.Param("type_id"))
		if err != nil {
			writeValidationError(c, "type_id", "invalid type_id")
			return
		}
		if err := repo.DeleteLocationTypeLink(locationID, typeID); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "location_type_links", strconv.Itoa(locationID)+":"+strconv.Itoa(typeID), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListTeacherLocationPreferences(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		teacherID, ok := parseQueryInt(c, "teacher_id")
		if !ok {
			return
		}
		locationID, ok := parseQueryInt(c, "location_id")
		if !ok {
			return
		}
		filters := schedule.TeacherLocationPreferenceFilters{TeacherID: teacherID, LocationID: locationID}
		var rows []schedule.TeacherLocationPreference
		var err error
		if page.Limit != nil {
			rows, err = repo.ListTeacherLocationPreferencesPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListTeacherLocationPreferences(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]teacherLocationPreferenceDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toTeacherLocationPreferenceDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateTeacherLocationPreference(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.TeacherLocationPreference
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateTeacherLocationPreference(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "teacher_location_preferences", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toTeacherLocationPreferenceDTO(req))
	}
}

func handleAdminUpdateTeacherLocationPreference(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.TeacherLocationPreference
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateTeacherLocationPreference(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "teacher_location_preferences", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toTeacherLocationPreferenceDTO(*row))
	}
}

func handleAdminDeleteTeacherLocationPreference(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteTeacherLocationPreference(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "teacher_location_preferences", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListRoomRequests(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		teacherID, ok := parseQueryInt(c, "teacher_id")
		if !ok {
			return
		}
		subjectID, ok := parseQueryInt(c, "subject_id")
		if !ok {
			return
		}
		groupID, ok := parseQueryInt(c, "group_id")
		if !ok {
			return
		}
		semester, ok := parseQueryInt16(c, "semester")
		if !ok {
			return
		}
		status := stringPtrFromQuery(c, "status")
		filters := schedule.RoomRequestFilters{TeacherID: teacherID, SubjectID: subjectID, GroupID: groupID, Semester: semester, Status: status}
		var rows []schedule.RoomRequest
		var err error
		if page.Limit != nil {
			rows, err = repo.ListRoomRequestsPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListRoomRequests(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]roomRequestDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toRoomRequestDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func stringPtrFromQuery(c *gin.Context, name string) *string {
	v := strings.TrimSpace(c.Query(name))
	if v == "" {
		return nil
	}
	return &v
}

func handleAdminCreateRoomRequest(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.RoomRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateRoomRequest(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "room_requests", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toRoomRequestDTO(req))
	}
}

func handleAdminUpdateRoomRequest(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.RoomRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateRoomRequest(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "room_requests", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toRoomRequestDTO(*row))
	}
}

func handleAdminDeleteRoomRequest(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteRoomRequest(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "room_requests", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminListRoomAssignments(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, ok := parseLimitOffset(c, nil, 500)
		if !ok {
			return
		}
		lessonID, ok := parseQueryInt64(c, "schedule_lesson_id")
		if !ok {
			return
		}
		locationID, ok := parseQueryInt(c, "location_id")
		if !ok {
			return
		}
		var status *schedule.EntityStatus
		if s := strings.TrimSpace(c.Query("status")); s != "" {
			v := schedule.EntityStatus(s)
			status = &v
		}
		filters := schedule.RoomAssignmentFilters{
			ScheduleLessonID: lessonID,
			LocationID:       locationID,
			Status:           status,
		}
		var rows []schedule.RoomAssignment
		var err error
		if page.Limit != nil {
			rows, err = repo.ListRoomAssignmentsPaged(filters, page.Limit, page.Offset)
		} else {
			rows, err = repo.ListRoomAssignments(filters)
		}
		if err != nil {
			writeDBError(c, err)
			return
		}
		out := make([]roomAssignmentDTO, 0, len(rows))
		for _, r := range rows {
			out = append(out, toRoomAssignmentDTO(r))
		}
		c.JSON(http.StatusOK, out)
	}
}

func handleAdminCreateRoomAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.RoomAssignment
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		if err := repo.CreateRoomAssignment(&req); err != nil {
			writeDBError(c, err)
			return
		}
		_, _ = bumpScheduleVersionAndGet(repo)
		writeAudit(c, repo, "create", "room_assignments", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, toRoomAssignmentDTO(req))
	}
}

func handleAdminUpdateRoomAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		var req schedule.RoomAssignment
		if err := c.ShouldBindJSON(&req); err != nil {
			writeInvalidJSON(c)
			return
		}
		row, err := repo.UpdateRoomAssignment(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		_, _ = bumpScheduleVersionAndGet(repo)
		writeAudit(c, repo, "update", "room_assignments", strconv.FormatInt(id, 10), row)
		c.JSON(http.StatusOK, toRoomAssignmentDTO(*row))
	}
}

func handleAdminDeleteRoomAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			writeValidationError(c, "id", "invalid id")
			return
		}
		if err := repo.DeleteRoomAssignment(id); err != nil {
			writeDBError(c, err)
			return
		}
		_, _ = bumpScheduleVersionAndGet(repo)
		writeAudit(c, repo, "delete", "room_assignments", strconv.FormatInt(id, 10), nil)
		c.Writer.WriteHeader(http.StatusNoContent)
	}
}
