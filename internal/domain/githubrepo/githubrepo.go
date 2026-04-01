package githubrepo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPrivate, VisibilityPublic:
		return true
	default:
		return false
	}
}

type NamespaceKind string

const (
	NamespaceKindUser         NamespaceKind = "user"
	NamespaceKindOrganization NamespaceKind = "organization"
)

type Namespace struct {
	Login string
	Kind  NamespaceKind
}

type Repository struct {
	ID            int64
	Name          string
	FullName      string
	Owner         string
	DefaultBranch string
	Visibility    Visibility
	Private       bool
	HTMLURL       string
	CloneURL      string
}

type RepositoryPage struct {
	Repositories []Repository
	NextCursor   string
}

type ListRepositoriesInput struct {
	ProjectID uuid.UUID
	Query     string
	Page      int
}

type CreateRepositoryInput struct {
	ProjectID   uuid.UUID
	Owner       string
	Name        string
	Description string
	Visibility  Visibility
	AutoInit    bool
}

type CreateRepositoryRequest struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	AutoInit    *bool  `json:"auto_init"`
}

func ParseListRepositories(projectID uuid.UUID, rawQuery string, rawCursor string) (ListRepositoriesInput, error) {
	page := 1
	cursor := strings.TrimSpace(rawCursor)
	if cursor != "" {
		parsedPage, err := strconv.Atoi(cursor)
		if err != nil || parsedPage < 1 {
			return ListRepositoriesInput{}, fmt.Errorf("cursor must be a positive integer")
		}
		page = parsedPage
	}

	return ListRepositoriesInput{
		ProjectID: projectID,
		Query:     strings.TrimSpace(rawQuery),
		Page:      page,
	}, nil
}

func ParseCreateRepository(projectID uuid.UUID, raw CreateRepositoryRequest) (CreateRepositoryInput, error) {
	owner := strings.TrimSpace(raw.Owner)
	if owner == "" {
		return CreateRepositoryInput{}, fmt.Errorf("owner must not be empty")
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return CreateRepositoryInput{}, fmt.Errorf("name must not be empty")
	}

	visibility := Visibility(strings.ToLower(strings.TrimSpace(raw.Visibility)))
	if !visibility.IsValid() {
		return CreateRepositoryInput{}, fmt.Errorf("visibility must be one of: private, public")
	}

	autoInit := true
	if raw.AutoInit != nil {
		autoInit = *raw.AutoInit
	}

	return CreateRepositoryInput{
		ProjectID:   projectID,
		Owner:       owner,
		Name:        name,
		Description: strings.TrimSpace(raw.Description),
		Visibility:  visibility,
		AutoInit:    autoInit,
	}, nil
}
