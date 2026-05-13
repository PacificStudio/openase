package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	presetdomain "github.com/BetterAndBetterII/openase/internal/domain/projectpreset"
	presetservice "github.com/BetterAndBetterII/openase/internal/projectpreset"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawApplyProjectPresetRequest struct {
	WorkflowAgentBindings []rawProjectPresetAgentBinding `json:"workflow_agent_bindings"`
}

type rawProjectPresetAgentBinding struct {
	WorkflowKey string `json:"workflow_key"`
	AgentID     string `json:"agent_id"`
}

type projectPresetApplyResponse struct {
	Result presetdomain.ApplyResult `json:"result"`
}

func (s *Server) registerProjectPresetRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/pipeline-presets", s.handleListProjectPresets)
	api.POST("/projects/:projectId/pipeline-presets/:presetKey/apply", s.handleApplyProjectPreset)
}

func (s *Server) handleListProjectPresets(c echo.Context) error {
	if s.projectPresetService == nil {
		return writeProjectPresetError(c, presetservice.ErrUnavailable)
	}
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	catalog, err := s.projectPresetService.List(c.Request().Context(), projectID)
	if err != nil {
		return writeProjectPresetError(c, err)
	}
	return c.JSON(http.StatusOK, catalog)
}

func (s *Server) handleApplyProjectPreset(c echo.Context) error {
	if s.projectPresetService == nil {
		return writeProjectPresetError(c, presetservice.ErrUnavailable)
	}
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	presetKey := strings.TrimSpace(c.Param("presetKey"))
	if presetKey == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PRESET_KEY", "presetKey must not be empty")
	}
	var raw rawApplyProjectPresetRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseApplyProjectPresetRequest(projectID, presetKey, actorFromWritePrincipal(c), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	result, err := s.projectPresetService.Apply(c.Request().Context(), input)
	if err != nil {
		return writeProjectPresetError(c, err)
	}
	return c.JSON(http.StatusOK, projectPresetApplyResponse{Result: result})
}

func parseApplyProjectPresetRequest(
	projectID uuid.UUID,
	presetKey string,
	appliedBy string,
	raw rawApplyProjectPresetRequest,
) (presetdomain.ApplyInput, error) {
	bindings := make([]presetdomain.WorkflowAgentBinding, 0, len(raw.WorkflowAgentBindings))
	for index, item := range raw.WorkflowAgentBindings {
		agentID, err := parseUUIDString(fmt.Sprintf("workflow_agent_bindings[%d].agent_id", index), item.AgentID)
		if err != nil {
			return presetdomain.ApplyInput{}, err
		}
		bindings = append(bindings, presetdomain.WorkflowAgentBinding{
			WorkflowKey: strings.TrimSpace(item.WorkflowKey),
			AgentID:     agentID,
		})
	}
	return presetdomain.ApplyInput{
		ProjectID:     projectID,
		PresetKey:     strings.TrimSpace(presetKey),
		AppliedBy:     strings.TrimSpace(appliedBy),
		AgentBindings: bindings,
	}, nil
}

func writeProjectPresetError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, presetservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, presetdomain.ErrPresetNotFound):
		return writeAPIError(c, http.StatusNotFound, "PRESET_NOT_FOUND", err.Error())
	case errors.Is(err, presetdomain.ErrActiveTicketsPresent):
		return writeAPIError(c, http.StatusConflict, "ACTIVE_TICKETS_PRESENT", err.Error())
	case errors.Is(err, presetdomain.ErrAgentBindingRequired):
		return writeAPIError(c, http.StatusBadRequest, "AGENT_BINDING_REQUIRED", err.Error())
	case errors.Is(err, presetdomain.ErrAgentBindingInvalid):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_AGENT_BINDING", err.Error())
	case errors.Is(err, presetdomain.ErrWorkflowRoleConflict):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_ROLE_CONFLICT", err.Error())
	case errors.Is(err, ticketstatus.ErrUnavailable),
		errors.Is(err, ticketstatus.ErrProjectNotFound),
		errors.Is(err, ticketstatus.ErrStatusNotFound),
		errors.Is(err, ticketstatus.ErrDuplicateStatusName),
		errors.Is(err, ticketstatus.ErrDefaultStatusStage),
		errors.Is(err, ticketstatus.ErrCannotDeleteLastStatus),
		errors.Is(err, ticketstatus.ErrDefaultStatusRequired):
		return writeTicketStatusError(c, err)
	case errors.Is(err, workflowservice.ErrUnavailable),
		errors.Is(err, workflowservice.ErrProjectNotFound),
		errors.Is(err, workflowservice.ErrWorkflowNotFound),
		errors.Is(err, workflowservice.ErrStatusNotFound),
		errors.Is(err, workflowservice.ErrAgentNotFound),
		errors.Is(err, workflowservice.ErrPickupStatusConflict),
		errors.Is(err, workflowservice.ErrWorkflowStatusBindingOverlap),
		errors.Is(err, workflowservice.ErrWorkflowNameConflict),
		errors.Is(err, workflowservice.ErrWorkflowHarnessPathConflict),
		errors.Is(err, workflowservice.ErrWorkflowReferencedByTickets),
		errors.Is(err, workflowservice.ErrWorkflowReferencedByScheduledJobs),
		errors.Is(err, workflowservice.ErrWorkflowConflict),
		errors.Is(err, workflowservice.ErrWorkflowInUse),
		errors.Is(err, workflowservice.ErrWorkflowReplacementRequired),
		errors.Is(err, workflowservice.ErrWorkflowActiveAgentRuns),
		errors.Is(err, workflowservice.ErrWorkflowHistoricalAgentRuns),
		errors.Is(err, workflowservice.ErrWorkflowReplacementInvalid),
		errors.Is(err, workflowservice.ErrWorkflowReplacementNotFound),
		errors.Is(err, workflowservice.ErrWorkflowReplacementProjectMismatch),
		errors.Is(err, workflowservice.ErrWorkflowReplacementInactive),
		errors.Is(err, workflowservice.ErrSkillNotFound),
		errors.Is(err, workflowservice.ErrSkillInvalid),
		errors.Is(err, workflowservice.ErrHarnessInvalid),
		errors.Is(err, workflowservice.ErrHookConfigInvalid),
		errors.Is(err, workflowservice.ErrWorkflowHookBlocked):
		return writeWorkflowError(c, err)
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
