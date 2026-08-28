package httpserver

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/metrics"
)

func metricsMiddleware(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()

	route := c.Route().Path
	if route == "" {
		route = c.Path()
	}
	status := c.Response().StatusCode()
	if fe, ok := err.(*fiber.Error); ok {
		status = fe.Code
	}

	metrics.RecordHTTPRequest(c.Method(), route, strconv.Itoa(status))
	metrics.HTTPRequestDuration.WithLabelValues(c.Method(), route).Observe(time.Since(start).Seconds())

	return err
}
