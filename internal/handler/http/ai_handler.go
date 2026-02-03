package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

// implement the struct
type AIHandler struct {
	AIUseCase usecasecontract.IAIUseCase
}

// call the factory
func NewAIHandler(aiuc usecasecontract.IAIUseCase) *AIHandler {
	return &AIHandler{
		AIUseCase: aiuc,
	}
}

type GenerateBlogRequest struct {
	Keywords string `json:"keywords" binding:"required"`
}
type SuggestAndModifyRequest struct {
	Keywords string `json:"keywords" binding:"required"`
	Blog     string `json:"blog" binding:"required"`
}

// implement the handlebloggeneration
func (h *AIHandler) HandleBlogContentGeneration(ctx *gin.Context) {
	requestCtx := ctx.Request.Context()
	var req GenerateBlogRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to read the generate request: %v", err)})
		return
	}
	generatedBlog, err := h.AIUseCase.GenerateBlogContent(requestCtx, req.Keywords)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate blog content: %v", err)})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "successfully generated blog\n" + generatedBlog})

}

// implement the handlesuggestionandmodification
func (h *AIHandler) HandleSuggestAndModifyContent(ctx *gin.Context) {
	requestCtx := ctx.Request.Context()
	var req SuggestAndModifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to read the generate request: %v", err)})
		return
	}
	generatedBlog, err := h.AIUseCase.SuggestAndModifyContent(requestCtx, req.Keywords, req.Blog)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to generate blog content: %v", err)})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "successfully generated blog\n" + generatedBlog})
}
