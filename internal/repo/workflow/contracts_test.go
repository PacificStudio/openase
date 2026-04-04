package workflow_test

import (
	repository "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

var (
	_ workflowservice.ProjectValidationRepository    = (*repository.EntRepository)(nil)
	_ workflowservice.WorkflowRepository             = (*repository.EntRepository)(nil)
	_ workflowservice.WorkflowVersionRepository      = (*repository.EntRepository)(nil)
	_ workflowservice.SkillRepository                = (*repository.EntRepository)(nil)
	_ workflowservice.SkillVersionRepository         = (*repository.EntRepository)(nil)
	_ workflowservice.WorkflowSkillBindingRepository = (*repository.EntRepository)(nil)
	_ workflowservice.WorkflowRuntimeSnapshotReader  = (*repository.EntRepository)(nil)
	_ workflowservice.HarnessTemplateDataBuilder     = (*repository.EntRepository)(nil)
)
