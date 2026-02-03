package usecase

import (
	"context"
	"fmt"
	"strings"

	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

type AIUseCase struct {
	aiService usecasecontract.IAIService
}

// check if AIUseCase implement IAIUseCase
var _ usecasecontract.IAIUseCase = (*AIUseCase)(nil)

func NewAIUseCase(aiServ usecasecontract.IAIService) *AIUseCase {
	return &AIUseCase{
		aiService: aiServ,
	}
}

func (uc *AIUseCase) GenerateBlogContent(ctx context.Context, keywords string) (string, error) {
	if strings.TrimSpace(keywords) == "" {
		return "", fmt.Errorf("failed to generate content: empty keyword provided")
	}
	prompt := fmt.Sprintf("Generate a blog post of at least 300 words with a compelling title based on the following keywords: %s. The post should be well-structured and engaging.", keywords)

	// call the ai service to generate content
	generateContent, err := uc.aiService.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	return generateContent, nil

}

func (uc *AIUseCase) SuggestAndModifyContent(ctx context.Context, keywords string, blog string) (string, error) {
	if strings.TrimSpace(keywords) == "" {
		return "", fmt.Errorf("failed to generate content: empty keyword provided")
	}
	if strings.TrimSpace(blog) == "" {
		return "", fmt.Errorf("failed to modify content: original blog content is empty")
	}
	prompt := fmt.Sprintf(
		`You are a professional editor. 
Improve the following blog post using the keywords: "%s".
Your tasks:
- Rewrite the content to be clearer, more engaging, and well-structured
- Integrate the keywords naturally into the blog
- Ensure the tone is consistent and professional
- Enhance the title if needed
- Do not add unrelated information

Here is the original blog:
%s

Return only the revised blog content.`,
		keywords,
		blog,
	)
	// call the ai service to generate content
	modifiedContent, err := uc.aiService.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	return modifiedContent, nil

}

func (uc *AIUseCase) CensorAndCheckBlog(ctx context.Context, blog string) (string, error) {
	if strings.TrimSpace(blog) == "" {
		return "", fmt.Errorf("failed to check content: empty blog provided")
	}
	prompt := fmt.Sprintf(
		`You are a content moderator.
Review the following blog post and respond with "yes" if it is appropriate and follows community guidelines, or "no" if it contains inappropriate content or violates guidelines.

Here is the blog post:
%s

Respond only with "yes" or "no".`,
		blog,
	)
	// call the ai service to generate content
	feedback, err := uc.aiService.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}
	return feedback, nil

}
