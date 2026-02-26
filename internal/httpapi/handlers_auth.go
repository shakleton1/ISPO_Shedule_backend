package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/schedule"
)

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Role        auth.Role `json:"role"`
}

func handleLogin(tokens *auth.TokenManager, repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Login == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "login and password required"})
			return
		}

		u, err := repo.GetUserByLogin(req.Login)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !auth.VerifyPassword(u.PasswordHash, req.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		now := time.Now().UTC()
		tok, exp, err := tokens.IssueAccessToken(u, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}

		c.JSON(http.StatusOK, loginResponse{AccessToken: tok, ExpiresAt: exp, Role: u.Role})
	}
}

func handleMe(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get(ctxUserKey)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		u := v.(*auth.User)
		// refresh from DB for latest role/group assignment
		fresh, err := repo.GetUserByID(u.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id":       fresh.ID,
			"login":    fresh.Login,
			"role":     fresh.Role,
			"group_id": fresh.GroupID,
			"subgroup": fresh.Subgroup,
		})
	}
}
