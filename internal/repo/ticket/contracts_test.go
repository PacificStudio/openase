package ticket_test

import (
	repository "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
)

var (
	_ ticketservice.ActivityRepository = (*repository.ActivityRepository)(nil)
	_ ticketservice.QueryRepository    = (*repository.QueryRepository)(nil)
	_ ticketservice.CommandRepository  = (*repository.CommandRepository)(nil)
	_ ticketservice.LinkRepository     = (*repository.LinkRepository)(nil)
	_ ticketservice.CommentRepository  = (*repository.CommentRepository)(nil)
	_ ticketservice.UsageRepository    = (*repository.UsageRepository)(nil)
	_ ticketservice.RuntimeRepository  = (*repository.RuntimeRepository)(nil)
)
