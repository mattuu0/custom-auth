package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SessionValidator defines the interface for session validation
type SessionValidator interface {
	ValidateSession(token string) (bool, error)
}

// HankoSessionValidator implements SessionValidator
type HankoSessionValidator struct {
	apiURL string
}

// ValidationResponse represents the Hanko API response
type ValidationResponse struct {
	IsValid        bool      `json:"is_valid"`
	ExpirationTime string    `json:"expiration_time"`
	UserID         string    `json:"user_id"`
	Claims         Claims    `json:"claims"`
}

// Claims represents the claims section of the validation response
type Claims struct {
	Subject    string   `json:"subject"`
	IssuedAt   string   `json:"issued_at"`
	Expiration string   `json:"expiration"`
	Audience   []string `json:"audience"`
	Issuer     string   `json:"issuer"`
	Email      Email    `json:"email"`
	SessionID  string   `json:"session_id"`
}

// Email represents the email information in claims
type Email struct {
	Address     string `json:"address"`
	IsPrimary   bool   `json:"is_primary"`
	IsVerified  bool   `json:"is_verified"`
}

func NewHankoSessionValidator(apiURL string) *HankoSessionValidator {
	return &HankoSessionValidator{apiURL: apiURL}
}

func (v *HankoSessionValidator) ValidateSession(token string) (ValidationResponse, error) {
	payload := strings.NewReader(fmt.Sprintf(`{"session_token":"%s"}`, token))

	req, err := http.NewRequest(http.MethodPost, v.apiURL+"/sessions/validate", payload)
	if err != nil {
		return ValidationResponse{}, fmt.Errorf("Failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ValidationResponse{}, fmt.Errorf("Failed to send request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ValidationResponse{}, fmt.Errorf("Failed to read response: %w", err)
	}

	var validationRes ValidationResponse
	if err := json.Unmarshal(body, &validationRes); err != nil {
		return ValidationResponse{}, fmt.Errorf("Failed to parse response: %w", err)
	}

	return validationRes, nil
}

func (validator HankoSessionValidator) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		// Extract session token from cookie
		cookie, err := ctx.Cookie("hanko")
		if err != nil {
			return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		}

		// Validate the session token
		response, err := validator.ValidateSession(cookie.Value)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
		}

		if !response.IsValid {
			return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		}

		log.Println(response.Claims.Subject)
		log.Println(response.Claims.Audience)

		return next(ctx)
	}
}

func main() {
	validator := NewHankoSessionValidator("http://hanko:8000")

	router := echo.New()
	router.Use(middleware.Logger())

	// Protected route with middleware
	router.GET("/protected", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"message": "Protected route"})
	}, validator.AuthMiddleware)

	router.Logger.Fatal(router.Start(":8090"))
}
