package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Auth struct{ secret []byte }

func NewAuth(secret string) *Auth { return &Auth{secret: []byte(secret)} }

func (a *Auth) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
			return a.secret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		sub, err := token.Claims.GetSubject()
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		id, err := primitive.ObjectIDFromHex(sub)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
			c.Abort()
			return
		}
		c.Set("userID", id)
		c.Next()
	}
}
