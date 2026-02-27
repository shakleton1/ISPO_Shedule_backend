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
	AccessToken      string    `json:"access_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	Role             auth.Role `json:"role"`
}

func handleLogin(tokens *auth.TokenManager, repo *schedule.Repository, refreshTTL time.Duration) gin.HandlerFunc {
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

		refreshRaw, err := auth.GenerateRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}
		h, err := auth.HashRefreshToken(refreshRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}
		refreshExp := now.Add(refreshTTL)
		if _, err := repo.CreateRefreshToken(u.ID, h, refreshExp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		// Minimal audit: do not log password.
		writeAudit(c, repo, "login", "auth", u.Login, gin.H{"login": u.Login, "role": u.Role})
		c.JSON(http.StatusOK, loginResponse{AccessToken: tok, ExpiresAt: exp, RefreshToken: refreshRaw, RefreshExpiresAt: refreshExp, Role: u.Role})
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken      string    `json:"access_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

func handleRefresh(tokens *auth.TokenManager, repo *schedule.Repository, refreshTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req refreshRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
			return
		}
		h, err := auth.HashRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid refresh_token"})
			return
		}

		row, err := repo.GetRefreshTokenByHash(h)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh_token"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		now := time.Now().UTC()
		if row.RevokedAt != nil {
			// Reuse detection: a revoked token being presented again -> revoke all active tokens.
			_ = repo.RevokeAllRefreshTokensForUser(row.UserID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh_token"})
			return
		}
		if now.After(row.ExpiresAt) {
			_ = repo.RevokeRefreshToken(row.ID, nil)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh_token expired"})
			return
		}

		u, err := repo.GetUserByID(row.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unknown user"})
			return
		}

		access, accessExp, err := tokens.IssueAccessToken(u, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}
		newRaw, err := auth.GenerateRefreshToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}
		newHash, err := auth.HashRefreshToken(newRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
			return
		}
		newExp := now.Add(refreshTTL)
		newRow, err := repo.CreateRefreshToken(u.ID, newHash, newExp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		_ = repo.RevokeRefreshToken(row.ID, &newRow.ID)

		writeAudit(c, repo, "refresh", "auth", u.Login, gin.H{"login": u.Login})
		c.JSON(http.StatusOK, refreshResponse{AccessToken: access, ExpiresAt: accessExp, RefreshToken: newRaw, RefreshExpiresAt: newExp})
	}
}

func handleLogout(repo *schedule.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req refreshRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
			return
		}
		h, err := auth.HashRefreshToken(req.RefreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid refresh_token"})
			return
		}
		row, err := repo.GetRefreshTokenByHash(h)
		if err != nil {
			// Always return 204 to avoid leaking token existence.
			c.Status(http.StatusNoContent)
			return
		}
		_ = repo.RevokeRefreshToken(row.ID, nil)
		writeAudit(c, repo, "logout", "auth", "refresh_token", nil)
		c.Status(http.StatusNoContent)
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
