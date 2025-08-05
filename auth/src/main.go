package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	router := echo.New()
	router.Use(middleware.Logger())

	// Protected route with middleware
	router.GET("/health", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"message": "Healthy"})
	})

	router.Logger.Fatal(router.Start(":8090"))
}
