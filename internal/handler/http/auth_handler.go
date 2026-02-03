package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	UserUseCase usecasecontract.IUserUseCase
	BaseURL     string
}

func NewAuthHandler(uc usecasecontract.IUserUseCase, baseURL string) *AuthHandler {
	return &AuthHandler{
		UserUseCase: uc,
		BaseURL:     baseURL,
	}
}

type UserInfo struct {
	Email string
	Name  string
}

func (h *AuthHandler) googleOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  h.BaseURL + "/api/v1/auth/google/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (h *AuthHandler) HandleGoogleLogin(ctx *gin.Context) {
	b := make([]byte, 16)
	rand.Read(b)
	oauthStateString := base64.URLEncoding.EncodeToString(b)
	ctx.SetCookie("oauthState", oauthStateString, 300, "/", "localhost", false, true)

	url := h.googleOauthConfig().AuthCodeURL(oauthStateString)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *AuthHandler) HandleGoogleCallback(ctx *gin.Context) {
	state := ctx.Query("state")
	cookieState, err := ctx.Cookie("oauthState")

	if err != nil || state != cookieState {
		ctx.String(http.StatusUnauthorized, "invalid CSRF state token\n")
		return
	}
	ctx.SetCookie("oauthState", "", -1, "/", "localhost", false, true)

	code := ctx.Query("code")
	if code == "" {
		ctx.String(http.StatusBadRequest, "authorization code not provided")
		return
	}

	requestCtx := ctx.Request.Context()

	token, err := h.googleOauthConfig().Exchange(requestCtx, code)
	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("failed to exchange autherization for token: %v\n", err))
		return
	}

	client := h.googleOauthConfig().Client(requestCtx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get user info: %v", err))
		return
	}

	defer resp.Body.Close()

	var userInfo UserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("Failed to decode user info: %v\n", err))
	}

	nameParts := strings.Split(userInfo.Name, " ")
	var fName, lName string
	if len(nameParts) >= 1 {
		fName = nameParts[0]
	}
	if len(nameParts) == 2 {
		lName = nameParts[1]
	}

	accessToken, refershToken, err := h.UserUseCase.LoginWithOAuth(requestCtx, fName, lName, userInfo.Email)

	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("failed to login with OAuth: %v\n", err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":       "login successful",
		"access token":  accessToken,
		"refresh token": refershToken,
	})
}
