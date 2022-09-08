package main

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"net/http"
	"time"
)

func main() {
	// Echo instance
	e := echo.New()
	e.Logger.SetLevel(log.INFO)

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Route
	e.GET("/sleep", func(c echo.Context) (err error) {
		// This select make the trick of finish this request when the middleware timeouts
		select {
		case <-time.After(5 * time.Second):
			c.Logger().Info("Done")
			return c.JSON(http.StatusOK, "Done")
		case <-c.Request().Context().Done():
			c.Logger().Info("Timeout")
			return nil
		}

	}, timeoutMiddleware)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))

}

func timeoutMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Just to play easily with the middleware using a query parameter
		timeout := 2 * time.Second
		if t, err := time.ParseDuration(c.QueryParam("timeout")); err == nil {
			timeout = t
		}

		// This is the context that controls the timeout. Its parent is the original
		// http.Request context
		ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
		defer cancel() // releases resources if next(c) completes before timeout elapses

		// A channel and a goroutine to run next(c) and know if its ends
		done := make(chan error, 1)
		go func() {
			// This goroutine will not stop even this middleware timeouts,
			// unless someone in the next(c) call chain handle ctx.Done() properly
			c.SetRequest(c.Request().Clone(ctx))
			done <- next(c)
		}()

		// The real timeout logic
		select {
		case <-ctx.Done():
			return c.JSON(http.StatusGatewayTimeout, ctx.Err())
		case err := <-done:
			return err
		}
	}
}
