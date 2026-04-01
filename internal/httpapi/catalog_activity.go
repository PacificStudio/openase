package httpapi

import "github.com/labstack/echo/v4"

func (s *Server) registerCatalogActivityRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/activity", s.listActivityEvents)
}
