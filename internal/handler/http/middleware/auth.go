package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/usecase"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

func AuthMiddleWare(jwtService usecase.JWTService, userUseCase usecasecontract.IUserUseCase) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower((parts[0])) != "bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header format"})
			return
		}
		tokenString := parts[1]

		claims, err := jwtService.ParseAccessToken(tokenString)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		ctx.Set("userID", claims.UserID)
		ctx.Set("userRole", claims.Role)

		ctx.Next()
	}
}
