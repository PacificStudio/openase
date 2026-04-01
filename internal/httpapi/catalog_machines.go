package httpapi

import (
	"net/http"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerMachineRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/machines", s.listMachines)
	api.POST("/orgs/:orgId/machines", s.createMachine)
	api.GET("/machines/:machineId", s.getMachine)
	api.PATCH("/machines/:machineId", s.patchMachine)
	api.DELETE("/machines/:machineId", s.deleteMachine)
	api.POST("/machines/:machineId/test", s.testMachine)
	api.POST("/machines/:machineId/refresh-health", s.refreshMachineHealth)
	api.GET("/machines/:machineId/resources", s.getMachineResources)
}

func (s *Server) listMachines(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListMachines(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machines": mapMachineResponses(items),
	})
}

func (s *Server) createMachine(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var request domain.MachineInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateMachine(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateMachine(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) getMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) patchMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch machinePatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	input, err := parseMachinePatchRequest(machineID, current, patch)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateMachine(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) deleteMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) testMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, probe, err := s.catalog.TestMachineConnection(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
		"probe":   mapMachineProbeResponse(probe),
	})
}

func (s *Server) refreshMachineHealth(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.RefreshMachineHealth(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) getMachineResources(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine_id":               item.ID.String(),
		"status":                   item.Status.String(),
		"last_heartbeat_at":        timeToStringPointer(item.LastHeartbeatAt),
		"resources":                cloneMap(item.Resources),
		"environment_provisioning": domain.PlanMachineEnvironmentProvisioning(item),
	})
}
