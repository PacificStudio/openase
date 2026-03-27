package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type rawCreateTicketStageRequest struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Position      *int   `json:"position"`
	MaxActiveRuns *int   `json:"max_active_runs"`
	Description   string `json:"description"`
}

type rawUpdateTicketStageRequest struct {
	Name          *string          `json:"name"`
	Position      *int             `json:"position"`
	MaxActiveRuns nullableIntField `json:"max_active_runs"`
	Description   *string          `json:"description"`
}

type rawCreateTicketStatusRequest struct {
	StageID     *string `json:"stage_id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Icon        string  `json:"icon"`
	Position    *int    `json:"position"`
	IsDefault   bool    `json:"is_default"`
	Description string  `json:"description"`
}

type rawUpdateTicketStatusRequest struct {
	StageID     nullableStringField `json:"stage_id"`
	Name        *string             `json:"name"`
	Color       *string             `json:"color"`
	Icon        *string             `json:"icon"`
	Position    *int                `json:"position"`
	IsDefault   *bool               `json:"is_default"`
	Description *string             `json:"description"`
}

type nullableStringField struct {
	Set   bool
	Value *string
}

func (f *nullableStringField) UnmarshalJSON(data []byte) error {
	f.Set = true
	if isJSONNull(data) {
		f.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
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

func parseCreateTicketStageRequest(projectID uuid.UUID, raw rawCreateTicketStageRequest) (ticketstatus.CreateStageInput, error) {
	key := strings.TrimSpace(raw.Key)
	if key == "" {
		return ticketstatus.CreateStageInput{}, fmt.Errorf("key must not be empty")
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ticketstatus.CreateStageInput{}, fmt.Errorf("name must not be empty")
	}

	input := ticketstatus.CreateStageInput{
		ProjectID:     projectID,
		Key:           key,
		Name:          name,
		MaxActiveRuns: raw.MaxActiveRuns,
		Description:   strings.TrimSpace(raw.Description),
	}
	if raw.Position != nil {
		if *raw.Position < 0 {
			return ticketstatus.CreateStageInput{}, fmt.Errorf("position must be greater than or equal to 0")
		}
		input.Position = ticketstatus.Some(*raw.Position)
	}
	if raw.MaxActiveRuns != nil && *raw.MaxActiveRuns <= 0 {
		return ticketstatus.CreateStageInput{}, fmt.Errorf("max_active_runs must be greater than 0")
	}

	return input, nil
}

func parseUpdateTicketStageRequest(stageID uuid.UUID, raw rawUpdateTicketStageRequest) (ticketstatus.UpdateStageInput, error) {
	input := ticketstatus.UpdateStageInput{StageID: stageID}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return ticketstatus.UpdateStageInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = ticketstatus.Some(name)
	}

	if raw.Position != nil {
		if *raw.Position < 0 {
			return ticketstatus.UpdateStageInput{}, fmt.Errorf("position must be greater than or equal to 0")
		}
		input.Position = ticketstatus.Some(*raw.Position)
	}

	if raw.MaxActiveRuns.Set {
		if raw.MaxActiveRuns.Value != nil && *raw.MaxActiveRuns.Value <= 0 {
			return ticketstatus.UpdateStageInput{}, fmt.Errorf("max_active_runs must be greater than 0")
		}
		input.MaxActiveRuns = ticketstatus.Some(raw.MaxActiveRuns.Value)
	}

	if raw.Description != nil {
		input.Description = ticketstatus.Some(strings.TrimSpace(*raw.Description))
	}

	return input, nil
}

func parseCreateTicketStatusRequest(projectID uuid.UUID, raw rawCreateTicketStatusRequest) (ticketstatus.CreateInput, error) {
	stageID, err := parseOptionalStatusUUIDString("stage_id", raw.StageID)
	if err != nil {
		return ticketstatus.CreateInput{}, err
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return ticketstatus.CreateInput{}, fmt.Errorf("name must not be empty")
	}

	color := strings.TrimSpace(raw.Color)
	if color == "" {
		return ticketstatus.CreateInput{}, fmt.Errorf("color must not be empty")
	}

	input := ticketstatus.CreateInput{
		ProjectID:   projectID,
		StageID:     stageID,
		Name:        name,
		Color:       color,
		Icon:        strings.TrimSpace(raw.Icon),
		IsDefault:   raw.IsDefault,
		Description: strings.TrimSpace(raw.Description),
	}
	if raw.Position != nil {
		if *raw.Position < 0 {
			return ticketstatus.CreateInput{}, fmt.Errorf("position must be greater than or equal to 0")
		}
		input.Position = ticketstatus.Some(*raw.Position)
	}

	return input, nil
}

func parseUpdateTicketStatusRequest(statusID uuid.UUID, raw rawUpdateTicketStatusRequest) (ticketstatus.UpdateInput, error) {
	input := ticketstatus.UpdateInput{StatusID: statusID}

	if raw.StageID.Set {
		stageID, err := parseOptionalStatusUUIDString("stage_id", raw.StageID.Value)
		if err != nil {
			return ticketstatus.UpdateInput{}, err
		}
		input.StageID = ticketstatus.Some(stageID)
	}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return ticketstatus.UpdateInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = ticketstatus.Some(name)
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

func parseStageID(c echo.Context) (uuid.UUID, error) {
	stageID, err := uuid.Parse(c.Param("stageId"))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("stageId must be a valid UUID")
	}
	return stageID, nil
}

func parseOptionalStatusUUIDString(field string, raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, fmt.Errorf("%s must not be empty", field)
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", field)
	}
	return &parsed, nil
}

func isJSONNull(raw []byte) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
