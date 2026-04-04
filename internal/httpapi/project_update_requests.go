package httpapi

import (
	"fmt"
	"strings"

	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	"github.com/google/uuid"
)

type rawCreateProjectUpdateThreadRequest struct {
	Status    string  `json:"status"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	CreatedBy *string `json:"created_by"`
}

type rawUpdateProjectUpdateThreadRequest struct {
	Status     string  `json:"status"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	EditedBy   *string `json:"edited_by"`
	EditReason *string `json:"edit_reason"`
}

type rawCreateProjectUpdateCommentRequest struct {
	Body      string  `json:"body"`
	CreatedBy *string `json:"created_by"`
}

type rawUpdateProjectUpdateCommentRequest struct {
	Body       string  `json:"body"`
	EditedBy   *string `json:"edited_by"`
	EditReason *string `json:"edit_reason"`
}

func parseCreateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	raw rawCreateProjectUpdateThreadRequest,
) (projectupdateservice.AddThreadInput, error) {
	status, err := parseProjectUpdateStatus(raw.Status)
	if err != nil {
		return projectupdateservice.AddThreadInput{}, err
	}
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return projectupdateservice.AddThreadInput{}, fmt.Errorf("title must not be empty")
	}
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return projectupdateservice.AddThreadInput{}, fmt.Errorf("body must not be empty")
	}

	input := projectupdateservice.AddThreadInput{
		ProjectID: projectID,
		Status:    status,
		Title:     title,
		Body:      body,
	}
	if raw.CreatedBy != nil {
		input.CreatedBy = strings.TrimSpace(*raw.CreatedBy)
	}
	return input, nil
}

func parseUpdateProjectUpdateThreadRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	raw rawUpdateProjectUpdateThreadRequest,
) (projectupdateservice.UpdateThreadInput, error) {
	status, err := parseProjectUpdateStatus(raw.Status)
	if err != nil {
		return projectupdateservice.UpdateThreadInput{}, err
	}
	title := strings.TrimSpace(raw.Title)
	if title == "" {
		return projectupdateservice.UpdateThreadInput{}, fmt.Errorf("title must not be empty")
	}
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return projectupdateservice.UpdateThreadInput{}, fmt.Errorf("body must not be empty")
	}

	input := projectupdateservice.UpdateThreadInput{
		ProjectID: projectID,
		ThreadID:  threadID,
		Status:    status,
		Title:     title,
		Body:      body,
	}
	if raw.EditedBy != nil {
		input.EditedBy = strings.TrimSpace(*raw.EditedBy)
	}
	if raw.EditReason != nil {
		input.EditReason = strings.TrimSpace(*raw.EditReason)
	}
	return input, nil
}

func parseCreateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	raw rawCreateProjectUpdateCommentRequest,
) (projectupdateservice.AddCommentInput, error) {
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return projectupdateservice.AddCommentInput{}, fmt.Errorf("body must not be empty")
	}
	input := projectupdateservice.AddCommentInput{
		ProjectID: projectID,
		ThreadID:  threadID,
		Body:      body,
	}
	if raw.CreatedBy != nil {
		input.CreatedBy = strings.TrimSpace(*raw.CreatedBy)
	}
	return input, nil
}

func parseUpdateProjectUpdateCommentRequest(
	projectID uuid.UUID,
	threadID uuid.UUID,
	commentID uuid.UUID,
	raw rawUpdateProjectUpdateCommentRequest,
) (projectupdateservice.UpdateCommentInput, error) {
	body := strings.TrimSpace(raw.Body)
	if body == "" {
		return projectupdateservice.UpdateCommentInput{}, fmt.Errorf("body must not be empty")
	}
	input := projectupdateservice.UpdateCommentInput{
		ProjectID: projectID,
		ThreadID:  threadID,
		CommentID: commentID,
		Body:      body,
	}
	if raw.EditedBy != nil {
		input.EditedBy = strings.TrimSpace(*raw.EditedBy)
	}
	if raw.EditReason != nil {
		input.EditReason = strings.TrimSpace(*raw.EditReason)
	}
	return input, nil
}

func parseProjectUpdateStatus(raw string) (projectupdateservice.Status, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(projectupdateservice.StatusOnTrack):
		return projectupdateservice.StatusOnTrack, nil
	case string(projectupdateservice.StatusAtRisk):
		return projectupdateservice.StatusAtRisk, nil
	case string(projectupdateservice.StatusOffTrack):
		return projectupdateservice.StatusOffTrack, nil
	default:
		return "", fmt.Errorf("status must be one of on_track, at_risk, off_track")
	}
}
