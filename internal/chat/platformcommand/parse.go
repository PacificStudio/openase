package platformcommand

import (
	"fmt"
	"strings"
)

func ParseProposal(payload map[string]any) (Proposal, error) {
	if strings.TrimSpace(stringValue(payload["type"])) != ProposalType {
		return Proposal{}, fmt.Errorf("proposal type must be %q", ProposalType)
	}

	rawCommands, ok := payload["commands"].([]any)
	if !ok {
		return Proposal{}, fmt.Errorf("platform command proposal commands must be an array")
	}

	commands := make([]Command, 0, len(rawCommands))
	for index, rawCommand := range rawCommands {
		command, err := parseCommand(rawCommand)
		if err != nil {
			return Proposal{}, fmt.Errorf("parse command %d: %w", index, err)
		}
		commands = append(commands, command)
	}

	return Proposal{
		Summary:  strings.TrimSpace(stringValue(payload["summary"])),
		Commands: commands,
	}, nil
}

func parseCommand(value any) (Command, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return Command{}, fmt.Errorf("command must be an object")
	}
	name := CommandName(strings.TrimSpace(stringValue(object["command"])))
	if name == "" {
		return Command{}, fmt.Errorf("command name must not be empty")
	}
	args, ok := object["args"].(map[string]any)
	if !ok {
		return Command{}, fmt.Errorf("command args must be an object")
	}

	switch name {
	case CommandProjectUpdateCreate:
		content, err := requiredString(args, "content")
		if err != nil {
			return Command{}, err
		}
		project, _ := optionalString(args, "project")
		title, _ := optionalString(args, "title")
		status, _ := optionalString(args, "status")
		return Command{
			Name: name,
			Args: ProjectUpdateCreateArgs{
				Project: project,
				Content: content,
				Title:   title,
				Status:  status,
			},
		}, nil
	case CommandTicketUpdate:
		ticketRef, err := requiredString(args, "ticket")
		if err != nil {
			return Command{}, err
		}
		title := optionalStringPointer(args, "title")
		description := optionalStringPointer(args, "description")
		status := optionalStringPointer(args, "status")
		if title == nil && description == nil && status == nil {
			return Command{}, fmt.Errorf("ticket.update requires at least one mutable field")
		}
		return Command{
			Name: name,
			Args: TicketUpdateArgs{
				Ticket:      ticketRef,
				Title:       title,
				Description: description,
				Status:      status,
			},
		}, nil
	case CommandTicketCreate:
		title, err := requiredString(args, "title")
		if err != nil {
			return Command{}, err
		}
		project, _ := optionalString(args, "project")
		description, _ := optionalString(args, "description")
		status := optionalStringPointer(args, "status")
		parentTicket := optionalStringPointer(args, "parent_ticket")
		return Command{
			Name: name,
			Args: TicketCreateArgs{
				Project:      project,
				Title:        title,
				Description:  description,
				Status:       status,
				ParentTicket: parentTicket,
			},
		}, nil
	default:
		return Command{}, fmt.Errorf("command %q is unsupported", name)
	}
}

func requiredString(object map[string]any, key string) (string, error) {
	value := strings.TrimSpace(stringValue(object[key]))
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", key)
	}
	return value, nil
}

func optionalString(object map[string]any, key string) (string, bool) {
	value := strings.TrimSpace(stringValue(object[key]))
	if value == "" {
		return "", false
	}
	return value, true
}

func optionalStringPointer(object map[string]any, key string) *string {
	value, ok := optionalString(object, key)
	if !ok {
		return nil
	}
	return &value
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}
