package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ispo-schedule/internal/schedule"
)

// Teachers

func handleAdminListTeachers(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := repo.ListTeachers()
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateTeacher(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.Teacher
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		if err := repo.CreateTeacher(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "teachers", strconv.Itoa(req.ID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateTeacher(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.Teacher
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		row, err := repo.UpdateTeacher(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "teachers", strconv.Itoa(id), req)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteTeacher(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteTeacher(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "teachers", strconv.Itoa(id), gin.H{"id": id})
		c.Status(http.StatusNoContent)
	}
}

// Teacher subjects

func handleAdminListTeacherSubjects(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters schedule.TeacherSubjectFilters
		if v := c.Query("teacher_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher_id"})
				return
			}
			filters.TeacherID = &id
		}
		if v := c.Query("subject_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
				return
			}
			filters.SubjectID = &id
		}
		rows, err := repo.ListTeacherSubjects(filters)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateTeacherSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.TeacherSubject
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.TeacherID <= 0 || req.SubjectID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "teacher_id and subject_id required"})
			return
		}
		if err := repo.CreateTeacherSubject(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "teacher_subjects", strconv.Itoa(req.TeacherID)+"/"+strconv.Itoa(req.SubjectID), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminDeleteTeacherSubject(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		teacherID, err := strconv.Atoi(c.Param("teacher_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher_id"})
			return
		}
		subjectID, err := strconv.Atoi(c.Param("subject_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
			return
		}
		if err := repo.DeleteTeacherSubject(teacherID, subjectID); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "teacher_subjects", strconv.Itoa(teacherID)+"/"+strconv.Itoa(subjectID), gin.H{"teacher_id": teacherID, "subject_id": subjectID})
		c.Status(http.StatusNoContent)
	}
}

// Course assignments

func handleAdminListCourseAssignments(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filters schedule.CourseAssignmentFilters
		if v := c.Query("group_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
				return
			}
			filters.GroupID = &id
		}
		if v := c.Query("semester"); v != "" {
			i, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid semester"})
				return
			}
			vv := int16(i)
			filters.Semester = &vv
		}
		if v := c.Query("subject_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subject_id"})
				return
			}
			filters.SubjectID = &id
		}
		if v := c.Query("teacher_id"); v != "" {
			id, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid teacher_id"})
				return
			}
			filters.TeacherID = &id
		}

		rows, err := repo.ListCourseAssignments(filters)
		if err != nil {
			writeDBError(c, err)
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func handleAdminCreateCourseAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req schedule.CourseAssignment
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.GroupID <= 0 || req.Semester <= 0 || req.SubjectID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_id, semester, subject_id required"})
			return
		}
		if req.TeacherID != nil {
			if err := repo.EnsureTeacherSubjectAllowed(*req.TeacherID, req.SubjectID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		if err := repo.CreateCourseAssignment(&req); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "create", "course_assignments", strconv.FormatInt(req.ID, 10), req)
		c.JSON(http.StatusCreated, req)
	}
}

func handleAdminUpdateCourseAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req schedule.CourseAssignment
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		if req.GroupID <= 0 || req.Semester <= 0 || req.SubjectID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_id, semester, subject_id required"})
			return
		}
		if req.TeacherID != nil {
			if err := repo.EnsureTeacherSubjectAllowed(*req.TeacherID, req.SubjectID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		row, err := repo.UpdateCourseAssignment(id, &req)
		if err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "update", "course_assignments", strconv.FormatInt(id, 10), req)
		c.JSON(http.StatusOK, row)
	}
}

func handleAdminDeleteCourseAssignment(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := repo.DeleteCourseAssignment(id); err != nil {
			writeDBError(c, err)
			return
		}
		writeAudit(c, repo, "delete", "course_assignments", strconv.FormatInt(id, 10), gin.H{"id": id})
		c.Status(http.StatusNoContent)
	}
}
