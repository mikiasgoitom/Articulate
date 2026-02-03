package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

type EmailHandler struct {
	emailVerificationUC usecasecontract.IEmailVerificationUC
	userRepository      contract.IUserRepository
}

func NewEmailHandler(eu usecasecontract.IEmailVerificationUC, uc contract.IUserRepository) *EmailHandler {
	return &EmailHandler{
		emailVerificationUC: eu,
		userRepository:      uc,
	}
}

type requestEmailVerificationRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *EmailHandler) HandleRequestEmailVerification(ctx *gin.Context) {
	var req requestEmailVerificationRequest
	requestCtx := ctx.Request.Context()
	// parse the json req
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// fetch user base on the req.userid
	user, err := h.userRepository.GetUserByID(requestCtx, req.UserID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Invalid request body"})
		return
	}

	// send a email validation request
	if err = h.emailVerificationUC.RequestVerificationEmail(requestCtx, user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}
	// send a successfull message
	ctx.JSON(http.StatusOK, gin.H{"message": "Verification email sent successfully"})
}

func (h *EmailHandler) HandleVerifyEmailToken(ctx *gin.Context) {
	requestCtx := ctx.Request.Context()
	verifier := ctx.Query("verifier")
	plainToken := ctx.Query("token")

	if verifier == "" || plainToken == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing verifier or token"})
		return
	}

	// call the verify email token usecase
	user, err := h.emailVerificationUC.VerifyEmailToken(requestCtx, verifier, plainToken)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid token or expired token"})
		return
	}
	user.IsVerified = true
	user.IsActive = true
	// update the user
	if _, err := h.userRepository.UpdateUser(ctx, user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	// send a success message
	ctx.JSON(http.StatusOK, gin.H{"message": "Email verified successfully", "user": user})
	// redirect to success page
	ctx.Redirect(http.StatusFound, fmt.Sprintf("/email-verified-success?username=%s", user.Username))
}
