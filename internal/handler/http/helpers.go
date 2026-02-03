package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/handler/http/dto"
)

// ErrorHandler centralizes error handling for HTTP responses
func ErrorHandler(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, dto.ErrorResponse{Error: message})
}

// SuccessHandler centralizes success responses
func SuccessHandler(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// MessageHandler centralizes message responses
func MessageHandler(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, dto.MessageResponse{Message: message})
}

// BindAndValidate binds JSON request and validates it
func BindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}
