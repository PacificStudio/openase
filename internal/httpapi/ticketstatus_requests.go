package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawCreateTicketStatusRequest struct {
	Name          string `json:"name"`
	Stage         string `json:"stage"`
	Color         string `json:"color"`
	Icon          string `json:"icon"`
	Position      *int   `json:"position"`
	MaxActiveRuns *int   `json:"max_active_runs"`
	IsDefault     bool   `json:"is_default"`
	Description   string `json:"description"`
}

type rawUpdateTicketStatusRequest struct {
	Name          *string          `json:"name"`
	Stage         *string          `json:"stage"`
	Color         *string          `json:"color"`
	Icon          *string          `json:"icon"`
	Position      *int             `json:"position"`
	MaxActiveRuns nullableIntField `json:"max_active_runs"`
	IsDefault     *bool            `json:"is_default"`
	Description   *string          `json:"description"`
}

type nullableIntField struct {
	Set   bool
	Value *int
}

func (f *nullableIntField) UnmarshalJSON(data []byte) error {
	f.Set = true
	if isJSONNull(data) {
		f.Value = nil
		return nil
	}

	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

func parseCreateTicketStatusRequest(projectID uuid.UUID, raw rawCreateTicketStatusRequest) (ticketstatus.CreateInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ticketstatus.CreateInput{}, fmt.Errorf("name must not be empty")
	}

	color := strings.TrimSpace(raw.Color)
	if color == "" {
		return ticketstatus.CreateInput{}, fmt.Errorf("color must not be empty")
	}

	input := ticketstatus.CreateInput{
		ProjectID:     projectID,
		Name:          name,
		Stage:         ticketing.DefaultStatusStage,
		Color:         color,
		Icon:          strings.TrimSpace(raw.Icon),
		MaxActiveRuns: raw.MaxActiveRuns,
		IsDefault:     raw.IsDefault,
		Description:   strings.TrimSpace(raw.Description),
	}
	if strings.TrimSpace(raw.Stage) != "" {
		stage, err := ticketing.ParseStatusStage(raw.Stage)
		if err != nil {
			return ticketstatus.CreateInput{}, err
		}
		input.Stage = stage
	}
	if raw.Position != nil {
		if *raw.Position < 0 {
			return ticketstatus.CreateInput{}, fmt.Errorf("position must be greater than or equal to 0")
		}
		input.Position = ticketstatus.Some(*raw.Position)
	}
	if raw.MaxActiveRuns != nil && *raw.MaxActiveRuns <= 0 {
		return ticketstatus.CreateInput{}, fmt.Errorf("max_active_runs must be greater than 0")
	}

	return input, nil
}

func parseUpdateTicketStatusRequest(statusID uuid.UUID, raw rawUpdateTicketStatusRequest) (ticketstatus.UpdateInput, error) {
	input := ticketstatus.UpdateInput{StatusID: statusID}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return ticketstatus.UpdateInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = ticketstatus.Some(name)
	}

	if raw.Stage != nil {
		stage, err := ticketing.ParseStatusStage(*raw.Stage)
		if err != nil {
			return ticketstatus.UpdateInput{}, err
		}
		input.Stage = ticketstatus.Some(stage)
	}

	if raw.Color != nil {
		color := strings.TrimSpace(*raw.Color)
		if color == "" {
			return ticketstatus.UpdateInput{}, fmt.Errorf("color must not be empty")
		}
		input.Color = ticketstatus.Some(color)
	}

	if raw.Icon != nil {
		input.Icon = ticketstatus.Some(strings.TrimSpace(*raw.Icon))
	}

	if raw.Position != nil {
		if *raw.Position < 0 {
			return ticketstatus.UpdateInput{}, fmt.Errorf("position must be greater than or equal to 0")
		}
		input.Position = ticketstatus.Some(*raw.Position)
	}

	if raw.MaxActiveRuns.Set {
		if raw.MaxActiveRuns.Value != nil && *raw.MaxActiveRuns.Value <= 0 {
			return ticketstatus.UpdateInput{}, fmt.Errorf("max_active_runs must be greater than 0")
		}
		input.MaxActiveRuns = ticketstatus.Some(raw.MaxActiveRuns.Value)
	}

	if raw.IsDefault != nil {
		input.IsDefault = ticketstatus.Some(*raw.IsDefault)
	}

	if raw.Description != nil {
		input.Description = ticketstatus.Some(strings.TrimSpace(*raw.Description))
	}

	return input, nil
}

func parseProjectID(c echo.Context) (uuid.UUID, error) {
	projectID, err := uuid.Parse(c.Param("projectId"))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("projectId must be a valid UUID")
	}
	return projectID, nil
}

func parseStatusID(c echo.Context) (uuid.UUID, error) {
	statusID, err := uuid.Parse(c.Param("statusId"))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("statusId must be a valid UUID")
	}
	return statusID, nil
}

func isJSONNull(raw []byte) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
