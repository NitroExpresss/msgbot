package server

import (
	"github.com/labstack/echo/v4"

	"gitlab.com/faemproject/backend/faem/pkg/web"
	"gitlab.com/faemproject/backend/faem/services/msgbot/handler"
)

const (
	apiPrefix = "/api/v2"
)

type Rest struct {
	Router  *echo.Echo
	Handler *handler.Handler
}

// Route defines all the application rest endpoints
func (r *Rest) Route() {
	web.UseHealthCheck(r.Router)

	//g := r.Router.Group(apiPrefix)
	//g.GET("/hello", r.Hello)
	//g.GET("/version", r.CurrentVersion)
}
