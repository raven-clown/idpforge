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

	metrics.HTTPRequestsTotal.WithLabelValues(c.Method(), route, strconv.Itoa(status)).Inc()
	metrics.HTTPRequestDuration.WithLabelValues(c.Method(), route).Observe(time.Since(start).Seconds())

	return err
}
