package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	notificationdomain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	notificationservice "github.com/BetterAndBetterII/openase/internal/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestHTTPAPIErrorMappings(t *testing.T) {
	t.Run("agent platform", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unavailable", err: agentplatform.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
			{name: "invalid token", err: agentplatform.ErrInvalidToken, wantStatus: http.StatusUnauthorized, wantCode: "INVALID_AGENT_TOKEN"},
			{name: "token not found", err: agentplatform.ErrTokenNotFound, wantStatus: http.StatusUnauthorized, wantCode: "AGENT_TOKEN_NOT_FOUND"},
			{name: "token expired", err: agentplatform.ErrTokenExpired, wantStatus: http.StatusUnauthorized, wantCode: "AGENT_TOKEN_EXPIRED"},
			{name: "invalid scope", err: agentplatform.ErrInvalidScope, wantStatus: http.StatusForbidden, wantCode: "AGENT_SCOPE_INVALID"},
			{name: "agent not found", err: agentplatform.ErrAgentNotFound, wantStatus: http.StatusUnauthorized, wantCode: "AGENT_NOT_FOUND"},
			{name: "project mismatch", err: agentplatform.ErrProjectMismatch, wantStatus: http.StatusUnauthorized, wantCode: "AGENT_PROJECT_MISMATCH"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeAgentPlatformError(c, testCase.err)
				})
				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
			})
		}
	})

	t.Run("notification", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unavailable", err: notificationservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
			{name: "organization missing", err: notificationservice.ErrOrganizationNotFound, wantStatus: http.StatusNotFound, wantCode: "ORGANIZATION_NOT_FOUND"},
			{name: "project missing", err: notificationservice.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
			{name: "channel missing", err: notificationservice.ErrChannelNotFound, wantStatus: http.StatusNotFound, wantCode: "CHANNEL_NOT_FOUND"},
			{name: "channel conflict", err: notificationservice.ErrDuplicateChannelName, wantStatus: http.StatusConflict, wantCode: "CHANNEL_NAME_CONFLICT"},
			{name: "rule missing", err: notificationservice.ErrRuleNotFound, wantStatus: http.StatusNotFound, wantCode: "RULE_NOT_FOUND"},
			{name: "rule conflict", err: notificationservice.ErrDuplicateRuleName, wantStatus: http.StatusConflict, wantCode: "RULE_NAME_CONFLICT"},
			{name: "channel project mismatch", err: notificationservice.ErrChannelProjectMismatch, wantStatus: http.StatusBadRequest, wantCode: "CHANNEL_PROJECT_MISMATCH"},
			{name: "invalid config", err: notificationservice.ErrInvalidChannelConfig, wantStatus: http.StatusBadRequest, wantCode: "INVALID_CHANNEL_CONFIG"},
			{name: "unsupported type", err: notificationdomain.ErrChannelTypeUnsupported, wantStatus: http.StatusBadRequest, wantCode: "CHANNEL_TYPE_UNSUPPORTED"},
			{name: "adapter unavailable", err: notificationservice.ErrAdapterUnavailable, wantStatus: http.StatusBadRequest, wantCode: "ADAPTER_UNAVAILABLE"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeNotificationError(c, testCase.err)
				})
				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
			})
		}
	})

	t.Run("scheduled job", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unavailable", err: scheduledjobservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
			{name: "project missing", err: scheduledjobservice.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
			{name: "workflow missing", err: scheduledjobservice.ErrWorkflowNotFound, wantStatus: http.StatusNotFound, wantCode: "WORKFLOW_NOT_FOUND"},
			{name: "job missing", err: scheduledjobservice.ErrScheduledJobNotFound, wantStatus: http.StatusNotFound, wantCode: "SCHEDULED_JOB_NOT_FOUND"},
			{name: "job conflict", err: scheduledjobservice.ErrScheduledJobConflict, wantStatus: http.StatusConflict, wantCode: "SCHEDULED_JOB_CONFLICT"},
			{name: "status missing", err: scheduledjobservice.ErrStatusNotFound, wantStatus: http.StatusBadRequest, wantCode: "STATUS_NOT_FOUND"},
			{name: "invalid cron", err: scheduledjobservice.ErrInvalidCronExpression, wantStatus: http.StatusBadRequest, wantCode: "INVALID_CRON_EXPRESSION"},
			{name: "invalid template", err: scheduledjobservice.ErrInvalidTicketTemplate, wantStatus: http.StatusBadRequest, wantCode: "INVALID_TICKET_TEMPLATE"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeScheduledJobError(c, testCase.err)
				})
				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
			})
		}
	})

	t.Run("ticket", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unavailable", err: ticketservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
			{name: "project missing", err: ticketservice.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
			{name: "ticket missing", err: ticketservice.ErrTicketNotFound, wantStatus: http.StatusNotFound, wantCode: "TICKET_NOT_FOUND"},
			{name: "ticket conflict", err: ticketservice.ErrTicketConflict, wantStatus: http.StatusConflict, wantCode: "TICKET_CONFLICT"},
			{name: "comment missing", err: ticketservice.ErrCommentNotFound, wantStatus: http.StatusNotFound, wantCode: "COMMENT_NOT_FOUND"},
			{name: "dependency missing", err: ticketservice.ErrDependencyNotFound, wantStatus: http.StatusNotFound, wantCode: "DEPENDENCY_NOT_FOUND"},
			{name: "external link missing", err: ticketservice.ErrExternalLinkNotFound, wantStatus: http.StatusNotFound, wantCode: "EXTERNAL_LINK_NOT_FOUND"},
			{name: "status missing", err: ticketservice.ErrStatusNotFound, wantStatus: http.StatusBadRequest, wantCode: "STATUS_NOT_FOUND"},
			{name: "status blocked", err: ticketservice.ErrStatusNotAllowed, wantStatus: http.StatusBadRequest, wantCode: "STATUS_NOT_ALLOWED"},
			{name: "workflow missing", err: ticketservice.ErrWorkflowNotFound, wantStatus: http.StatusBadRequest, wantCode: "WORKFLOW_NOT_FOUND"},
			{name: "target machine missing", err: ticketservice.ErrTargetMachineNotFound, wantStatus: http.StatusBadRequest, wantCode: "TARGET_MACHINE_NOT_FOUND"},
			{name: "parent missing", err: ticketservice.ErrParentTicketNotFound, wantStatus: http.StatusBadRequest, wantCode: "PARENT_TICKET_NOT_FOUND"},
			{name: "dependency conflict", err: ticketservice.ErrDependencyConflict, wantStatus: http.StatusConflict, wantCode: "DEPENDENCY_CONFLICT"},
			{name: "external link conflict", err: ticketservice.ErrExternalLinkConflict, wantStatus: http.StatusConflict, wantCode: "EXTERNAL_LINK_CONFLICT"},
			{name: "invalid dependency", err: ticketservice.ErrInvalidDependency, wantStatus: http.StatusBadRequest, wantCode: "INVALID_DEPENDENCY"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeTicketError(c, testCase.err)
				})
				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
			})
		}
	})

	t.Run("workflow", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "unavailable", err: workflowservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
			{name: "project missing", err: workflowservice.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
			{name: "workflow missing", err: workflowservice.ErrWorkflowNotFound, wantStatus: http.StatusNotFound, wantCode: "WORKFLOW_NOT_FOUND"},
			{name: "status missing", err: workflowservice.ErrStatusNotFound, wantStatus: http.StatusBadRequest, wantCode: "STATUS_NOT_FOUND"},
			{name: "agent missing", err: workflowservice.ErrAgentNotFound, wantStatus: http.StatusBadRequest, wantCode: "AGENT_NOT_FOUND"},
			{name: "pickup status conflict", err: workflowservice.ErrPickupStatusConflict, wantStatus: http.StatusConflict, wantCode: "PICKUP_STATUS_CONFLICT"},
			{name: "workflow status binding overlap", err: workflowservice.ErrWorkflowStatusBindingOverlap, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_STATUS_BINDING_OVERLAP"},
			{name: "workflow name conflict", err: workflowservice.ErrWorkflowNameConflict, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_NAME_CONFLICT"},
			{name: "workflow harness path conflict", err: workflowservice.ErrWorkflowHarnessPathConflict, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_HARNESS_PATH_CONFLICT"},
			{name: "workflow referenced by tickets", err: workflowservice.ErrWorkflowReferencedByTickets, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_REFERENCED_BY_TICKETS"},
			{name: "workflow referenced by scheduled jobs", err: workflowservice.ErrWorkflowReferencedByScheduledJobs, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_REFERENCED_BY_SCHEDULED_JOBS"},
			{name: "workflow conflict", err: workflowservice.ErrWorkflowConflict, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_CONFLICT"},
			{name: "workflow in use", err: workflowservice.ErrWorkflowInUse, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_IN_USE"},
			{name: "skill missing", err: workflowservice.ErrSkillNotFound, wantStatus: http.StatusNotFound, wantCode: "SKILL_NOT_FOUND"},
			{name: "skill invalid", err: workflowservice.ErrSkillInvalid, wantStatus: http.StatusBadRequest, wantCode: "INVALID_SKILL"},
			{name: "harness invalid", err: workflowservice.ErrHarnessInvalid, wantStatus: http.StatusBadRequest, wantCode: "INVALID_HARNESS"},
			{name: "hook invalid", err: workflowservice.ErrHookConfigInvalid, wantStatus: http.StatusBadRequest, wantCode: "INVALID_WORKFLOW_HOOKS"},
			{name: "hook blocked", err: workflowservice.ErrWorkflowHookBlocked, wantStatus: http.StatusConflict, wantCode: "WORKFLOW_HOOK_BLOCKED"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeWorkflowError(c, testCase.err)
				})
				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.err.Error())
			})
		}
	})
}

func TestWorkflowRequestParsersCoverTrimmedAndInvalidInputs(t *testing.T) {
	workflowID := uuid.New()
	agentID := uuid.New()
	statusA := uuid.New()
	statusB := uuid.New()
	name := "  updated workflow  "
	workflowType := "  custom "
	harnessPath := "  .openase/harnesses/custom.md  "
	hooks := map[string]any{"workflow_hooks": map[string]any{"on_activate": []map[string]any{{"cmd": "echo ok"}}}}
	maxConcurrent := 7
	maxRetryAttempts := 4
	timeoutMinutes := 12
	stallTimeoutMinutes := 3
	isActive := false
	pickup := []string{statusA.String()}
	finish := []string{statusB.String()}

	input, err := parseUpdateWorkflowRequest(workflowID, "", rawUpdateWorkflowRequest{
		AgentID:             stringPointer(agentID.String()),
		Name:                &name,
		Type:                &workflowType,
		HarnessPath:         &harnessPath,
		Hooks:               &hooks,
		MaxConcurrent:       &maxConcurrent,
		MaxRetryAttempts:    &maxRetryAttempts,
		TimeoutMinutes:      &timeoutMinutes,
		StallTimeoutMinutes: &stallTimeoutMinutes,
		IsActive:            &isActive,
		PickupStatusIDs:     &pickup,
		FinishStatusIDs:     &finish,
	})
	if err != nil {
		t.Fatalf("parseUpdateWorkflowRequest() error = %v", err)
	}
	if input.WorkflowID != workflowID {
		t.Fatalf("WorkflowID = %s, want %s", input.WorkflowID, workflowID)
	}
	if !input.AgentID.Set || input.AgentID.Value != agentID {
		t.Fatalf("AgentID = %+v, want %s", input.AgentID, agentID)
	}
	if !input.Name.Set || input.Name.Value != "updated workflow" {
		t.Fatalf("Name = %+v", input.Name)
	}
	if !input.Type.Set || input.Type.Value.String() != "custom" {
		t.Fatalf("Type = %+v", input.Type)
	}
	if _, err := parseUpdateWorkflowRequest(workflowID, "", rawUpdateWorkflowRequest{
		RoleSlug: stringPointer("new-role-slug"),
	}); err == nil || !strings.Contains(err.Error(), "role_slug cannot be updated") {
		t.Fatalf("parseUpdateWorkflowRequest(role slug) error = %v", err)
	}
	if !input.HarnessPath.Set || input.HarnessPath.Value != ".openase/harnesses/custom.md" {
		t.Fatalf("HarnessPath = %+v", input.HarnessPath)
	}
	if !input.Hooks.Set || len(input.Hooks.Value) != 1 {
		t.Fatalf("Hooks = %+v", input.Hooks)
	}
	if !input.MaxConcurrent.Set || input.MaxConcurrent.Value != maxConcurrent {
		t.Fatalf("MaxConcurrent = %+v", input.MaxConcurrent)
	}
	if !input.MaxRetryAttempts.Set || input.MaxRetryAttempts.Value != maxRetryAttempts {
		t.Fatalf("MaxRetryAttempts = %+v", input.MaxRetryAttempts)
	}
	if !input.TimeoutMinutes.Set || input.TimeoutMinutes.Value != timeoutMinutes {
		t.Fatalf("TimeoutMinutes = %+v", input.TimeoutMinutes)
	}
	if !input.StallTimeoutMinutes.Set || input.StallTimeoutMinutes.Value != stallTimeoutMinutes {
		t.Fatalf("StallTimeoutMinutes = %+v", input.StallTimeoutMinutes)
	}
	if !input.IsActive.Set || input.IsActive.Value {
		t.Fatalf("IsActive = %+v", input.IsActive)
	}
	pickupIDs := input.PickupStatusIDs.Value.IDs()
	if !input.PickupStatusIDs.Set || len(pickupIDs) != 1 || pickupIDs[0] != statusA {
		t.Fatalf("PickupStatusIDs = %+v", input.PickupStatusIDs)
	}
	finishIDs := input.FinishStatusIDs.Value.IDs()
	if !input.FinishStatusIDs.Set || len(finishIDs) != 1 || finishIDs[0] != statusB {
		t.Fatalf("FinishStatusIDs = %+v", input.FinishStatusIDs)
	}

	for _, testCase := range []struct {
		name string
		raw  rawUpdateWorkflowRequest
		want string
	}{
		{name: "invalid agent id", raw: rawUpdateWorkflowRequest{AgentID: stringPointer("bad")}, want: "agent_id must be a valid UUID"},
		{name: "empty name", raw: rawUpdateWorkflowRequest{Name: stringPointer("   ")}, want: "name must not be empty"},
		{name: "invalid type", raw: rawUpdateWorkflowRequest{Type: stringPointer(" \n ")}, want: "type must not be empty"},
		{name: "invalid max concurrent", raw: rawUpdateWorkflowRequest{MaxConcurrent: intPointer(-1)}, want: "max_concurrent must be greater than or equal to zero"},
		{name: "invalid max retry", raw: rawUpdateWorkflowRequest{MaxRetryAttempts: intPointer(-1)}, want: "max_retry_attempts must be greater than or equal to zero"},
		{name: "invalid timeout", raw: rawUpdateWorkflowRequest{TimeoutMinutes: intPointer(0)}, want: "timeout_minutes must be greater than zero"},
		{name: "invalid stall timeout", raw: rawUpdateWorkflowRequest{StallTimeoutMinutes: intPointer(0)}, want: "stall_timeout_minutes must be greater than zero"},
		{name: "invalid pickup", raw: rawUpdateWorkflowRequest{PickupStatusIDs: &[]string{"bad"}}, want: "pickup_status_ids must be a valid UUID"},
		{name: "invalid finish", raw: rawUpdateWorkflowRequest{FinishStatusIDs: &[]string{"bad"}}, want: "finish_status_ids must be a valid UUID"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := parseUpdateWorkflowRequest(workflowID, "", testCase.raw); err == nil || err.Error() == "" || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("parseUpdateWorkflowRequest() error = %v, want substring %q", err, testCase.want)
			}
		})
	}

	defaultConcurrency, err := parseConcurrencyLimit("max_concurrent", nil)
	if err != nil || defaultConcurrency != 0 {
		t.Fatalf("parseConcurrencyLimit(nil) = (%d, %v), want (0, nil)", defaultConcurrency, err)
	}
	if got, err := parseConcurrencyLimit("max_concurrent", intPointer(0)); err != nil || got != 0 {
		t.Fatalf("parseConcurrencyLimit(0) = (%d, %v), want (0, nil)", got, err)
	}
	if _, err := parseConcurrencyLimit("max_concurrent", intPointer(-1)); err == nil {
		t.Fatal("parseConcurrencyLimit() expected error for negative value")
	}
	defaultNonNegative, err := parseMaxRetryAttempts(nil, 2)
	if err != nil || defaultNonNegative != 2 {
		t.Fatalf("parseMaxRetryAttempts(nil) = (%d, %v), want (2, nil)", defaultNonNegative, err)
	}
	if _, err := parseMaxRetryAttempts(intPointer(-1), 2); err == nil {
		t.Fatal("parseMaxRetryAttempts() expected error for negative")
	}
}

func TestCatalogConflictMessageStripsConflictPrefix(t *testing.T) {
	err := errors.New(catalogservice.ErrConflict.Error() + ": repo already exists")
	if got := catalogConflictMessage(err); got != "repo already exists" {
		t.Fatalf("catalogConflictMessage() = %q, want %q", got, "repo already exists")
	}
	if got := catalogConflictMessage(errors.New("another problem")); got != "another problem" {
		t.Fatalf("catalogConflictMessage() passthrough = %q", got)
	}
}

func TestScheduledJobRequestParsersAndTicketEventCoverage(t *testing.T) {
	t.Run("scheduled job request parsers", func(t *testing.T) {
		projectID := uuid.New()
		jobID := uuid.New()
		disabled := false
		enabled := true
		name := "  nightly scan  "
		cronExpression := " 0 1 * * * "
		template := map[string]any{
			"title":       "Nightly",
			"description": "Run checks",
			"status":      "Todo",
			"priority":    "high",
			"type":        "feature",
		}

		createInput, err := parseCreateScheduledJobRequest(projectID, rawCreateScheduledJobRequest{
			Name:           name,
			CronExpression: cronExpression,
			TicketTemplate: template,
			IsEnabled:      &disabled,
		})
		if err != nil {
			t.Fatalf("parseCreateScheduledJobRequest() error = %v", err)
		}
		if createInput.ProjectID != projectID || createInput.Name != "nightly scan" || createInput.CronExpression != "0 1 * * *" || createInput.IsEnabled {
			t.Fatalf("parseCreateScheduledJobRequest() = %+v", createInput)
		}
		if createInput.TicketTemplate.Title != "Nightly" || createInput.TicketTemplate.Priority != "high" {
			t.Fatalf("parseCreateScheduledJobRequest().TicketTemplate = %+v", createInput.TicketTemplate)
		}

		updateInput, err := parseUpdateScheduledJobRequest(jobID, rawUpdateScheduledJobRequest{
			Name:           &name,
			CronExpression: &cronExpression,
			TicketTemplate: &template,
			IsEnabled:      &enabled,
		})
		if err != nil {
			t.Fatalf("parseUpdateScheduledJobRequest() error = %v", err)
		}
		if updateInput.JobID != jobID || !updateInput.Name.Set || updateInput.Name.Value != "nightly scan" || !updateInput.CronExpression.Set || updateInput.CronExpression.Value != "0 1 * * *" {
			t.Fatalf("parseUpdateScheduledJobRequest() = %+v", updateInput)
		}
		if !updateInput.TicketTemplate.Set || updateInput.TicketTemplate.Value.Title != "Nightly" || !updateInput.IsEnabled.Set || !updateInput.IsEnabled.Value {
			t.Fatalf("parseUpdateScheduledJobRequest().TicketTemplate/IsEnabled = %+v %+v", updateInput.TicketTemplate, updateInput.IsEnabled)
		}

		for _, testCase := range []struct {
			name string
			fn   func() error
			want string
		}{
			{
				name: "create blank name",
				fn: func() error {
					_, err := parseCreateScheduledJobRequest(projectID, rawCreateScheduledJobRequest{
						Name:           " ",
						CronExpression: "0 1 * * *",
						TicketTemplate: map[string]any{"title": "Nightly", "status": "Todo"},
					})
					return err
				},
				want: "name must not be empty",
			},
			{
				name: "create blank cron",
				fn: func() error {
					_, err := parseCreateScheduledJobRequest(projectID, rawCreateScheduledJobRequest{
						Name:           "nightly",
						CronExpression: " ",
						TicketTemplate: map[string]any{"title": "Nightly", "status": "Todo"},
					})
					return err
				},
				want: "cron_expression must not be empty",
			},
			{
				name: "create missing status",
				fn: func() error {
					_, err := parseCreateScheduledJobRequest(projectID, rawCreateScheduledJobRequest{
						Name:           "nightly",
						CronExpression: "0 1 * * *",
						TicketTemplate: map[string]any{"title": "Nightly"},
					})
					return err
				},
				want: "ticket_template.status must not be empty",
			},
			{
				name: "update invalid template",
				fn: func() error {
					badTemplate := map[string]any{"title": 1}
					_, err := parseUpdateScheduledJobRequest(jobID, rawUpdateScheduledJobRequest{TicketTemplate: &badTemplate})
					return err
				},
				want: "ticket template is invalid",
			},
			{
				name: "update missing status",
				fn: func() error {
					templateWithoutStatus := map[string]any{"title": "Nightly"}
					_, err := parseUpdateScheduledJobRequest(jobID, rawUpdateScheduledJobRequest{TicketTemplate: &templateWithoutStatus})
					return err
				},
				want: "ticket_template.status must not be empty",
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				if err := testCase.fn(); err == nil || !strings.Contains(err.Error(), testCase.want) {
					t.Fatalf("error = %v, want substring %q", err, testCase.want)
				}
			})
		}
	})

	t.Run("ticket events", func(t *testing.T) {
		server := NewServer(
			config.ServerConfig{Port: 40023},
			config.GitHubConfig{},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			eventinfra.NewChannelBus(),
			nil,
			nil,
			nil,
			nil,
			nil,
		)

		ticket := ticketservice.Ticket{
			ID:         uuid.New(),
			ProjectID:  uuid.New(),
			StatusID:   uuid.New(),
			Identifier: "ASE-278",
			Title:      "Coverage rollout",
			Priority:   "high",
			Type:       "feature",
			CreatedBy:  "user:test",
			CreatedAt:  time.Date(2026, time.March, 27, 12, 0, 0, 0, time.UTC),
		}

		streamCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		bus := server.events.(*eventinfra.ChannelBus)
		stream, err := bus.Subscribe(streamCtx, ticketEventsTopic)
		if err != nil {
			t.Fatalf("Subscribe() error = %v", err)
		}

		if err := server.publishTicketEvent(context.Background(), ticketUpdatedEventType, ticket); err != nil {
			t.Fatalf("publishTicketEvent() error = %v", err)
		}

		select {
		case event := <-stream:
			if event.Topic != ticketEventsTopic || event.Type != provider.MustParseEventType(ticketUpdatedEventType.String()) {
				t.Fatalf("ticket event = %+v", event)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for ticket event")
		}

		if err := server.publishTicketEvent(context.Background(), activityevent.TypeUnknown, ticket); err == nil || !strings.Contains(err.Error(), "parse ticket activity event type") {
			t.Fatalf("publishTicketEvent(empty type) error = %v", err)
		}

		closedBus := eventinfra.NewChannelBus()
		if err := closedBus.Close(); err != nil {
			t.Fatalf("closedBus.Close() error = %v", err)
		}
		server.events = closedBus
		if err := server.publishTicketEvent(context.Background(), ticketCreatedEventType, ticket); err == nil || !strings.Contains(err.Error(), "publish ticket event") {
			t.Fatalf("publishTicketEvent(closed bus) error = %v", err)
		}

		server.events = nil
		if err := server.publishTicketEvent(context.Background(), ticketCreatedEventType, ticket); err != nil {
			t.Fatalf("publishTicketEvent(nil bus) error = %v", err)
		}
	})
}

func TestCatalogHelpersAndOpenAPIBuilderCoverage(t *testing.T) {
	t.Run("catalog error mapping", func(t *testing.T) {
		for _, testCase := range []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
			wantBody   string
		}{
			{name: "invalid input", err: fmt.Errorf("%w: machine_id must reference an existing machine", catalogservice.ErrInvalidInput), wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST", wantBody: "machine_id must reference an existing machine"},
			{name: "not found", err: catalogservice.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "RESOURCE_NOT_FOUND", wantBody: "resource not found"},
			{name: "organization slug conflict", err: catalogdomain.ErrOrganizationSlugConflict, wantStatus: http.StatusConflict, wantCode: "ORGANIZATION_SLUG_CONFLICT", wantBody: "Organization slug already exists."},
			{name: "repo scope conflict", err: catalogdomain.ErrTicketRepoScopeConflict, wantStatus: http.StatusConflict, wantCode: "TICKET_REPO_SCOPE_CONFLICT", wantBody: "Repository is already attached to this ticket."},
			{name: "generic conflict", err: catalogservice.ErrConflict, wantStatus: http.StatusConflict, wantCode: "RESOURCE_CONFLICT", wantBody: "resource conflict"},
			{name: "probe failed", err: fmt.Errorf("%w: ssh handshake failed", catalogservice.ErrMachineProbeFailed), wantStatus: http.StatusBadGateway, wantCode: "MACHINE_PROBE_FAILED", wantBody: "ssh handshake failed"},
			{name: "testing unavailable", err: fmt.Errorf("%w: machine tests disabled", catalogservice.ErrMachineTestingUnavailable), wantStatus: http.StatusServiceUnavailable, wantCode: "MACHINE_TESTING_UNAVAILABLE", wantBody: "machine tests disabled"},
			{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantBody: "internal server error"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				rec := invokeAPIErrorWriter(t, func(c echo.Context) error {
					return writeCatalogError(c, testCase.err)
				})

				assertAPIErrorResponse(t, rec, testCase.wantStatus, testCase.wantCode, testCase.wantBody)
			})
		}
	})

	t.Run("decode json rejects unknown fields and multiple values", func(t *testing.T) {
		for _, testCase := range []struct {
			name     string
			body     string
			wantBody string
		}{
			{name: "unknown field", body: `{"name":"ok","extra":true}`, wantBody: "invalid JSON body"},
			{name: "multiple values", body: `{"name":"ok"}{"name":"again"}`, wantBody: "multiple JSON values are not allowed"},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testCase.body))
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				var payload struct {
					Name string `json:"name"`
				}
				err := decodeJSON(c, &payload)
				if !errors.Is(err, errAPIResponseCommitted) {
					t.Fatalf("decodeJSON() error = %v, want errAPIResponseCommitted", err)
				}
				if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), testCase.wantBody) {
					t.Fatalf("decodeJSON() response = %d %s, want body containing %q", rec.Code, rec.Body.String(), testCase.wantBody)
				}
			})
		}
	})

	t.Run("parse uuid path param", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		validRec := httptest.NewRecorder()
		validCtx := e.NewContext(req, validRec)
		validCtx.SetParamNames("projectId")
		validID := uuid.New()
		validCtx.SetParamValues(validID.String())
		parsed, err := parseUUIDPathParam(validCtx, "projectId")
		if err != nil || parsed != validID {
			t.Fatalf("parseUUIDPathParam(valid) = %s, %v; want %s, nil", parsed, err, validID)
		}

		invalidRec := httptest.NewRecorder()
		invalidCtx := e.NewContext(req, invalidRec)
		invalidCtx.SetParamNames("projectId")
		invalidCtx.SetParamValues("bad")
		if _, err := parseUUIDPathParam(invalidCtx, "projectId"); !errors.Is(err, errAPIResponseCommitted) {
			t.Fatalf("parseUUIDPathParam(invalid) error = %v, want errAPIResponseCommitted", err)
		}
		if invalidRec.Code != http.StatusBadRequest || !strings.Contains(invalidRec.Body.String(), "projectId must be a valid UUID") {
			t.Fatalf("parseUUIDPathParam(invalid) response = %d %s", invalidRec.Code, invalidRec.Body.String())
		}
	})

	t.Run("openapi builder helpers", func(t *testing.T) {
		builder := openAPISpecBuilder{
			doc: &openapi3.T{
				Components: &openapi3.Components{Schemas: openapi3.Schemas{}},
			},
		}

		jsonResponse, err := builder.jsonResponse("Agent", struct {
			ID string `json:"id"`
		}{})
		if err != nil {
			t.Fatalf("jsonResponse() error = %v", err)
		}
		if jsonResponse.Content["application/json"] == nil {
			t.Fatalf("jsonResponse() missing application/json content: %+v", jsonResponse.Content)
		}

		errorResponse, err := builder.errorResponse(http.StatusConflict)
		if err != nil {
			t.Fatalf("errorResponse() error = %v", err)
		}
		if errorResponse.Content["application/json"] == nil {
			t.Fatalf("errorResponse() missing application/json content: %+v", errorResponse.Content)
		}

		schemaRef, err := builder.schemaRef(func() {})
		if err != nil || schemaRef != nil {
			t.Fatalf("schemaRef(func) = %+v, %v; want nil, nil", schemaRef, err)
		}
	})
}

func TestServerRunReturnsStartupFailure(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Host: "127.0.0.1", Port: -1, ShutdownTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	err := server.Run(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want startup error")
	}
}

func invokeAPIErrorWriter(t *testing.T, fn func(echo.Context) error) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := fn(c); err != nil {
		t.Fatalf("error writer returned error: %v", err)
	}

	return rec
}

func assertAPIErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string, wantMessage string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, wantStatus, rec.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if payload["code"] != wantCode || payload["message"] != wantMessage {
		t.Fatalf("payload = %+v, want code=%q message=%q", payload, wantCode, wantMessage)
	}
}
