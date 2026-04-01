-- Create "activity_events" table
CREATE TABLE "activity_events" (
  "id" uuid NOT NULL,
  "event_type" character varying NOT NULL,
  "message" text NULL,
  "metadata" jsonb NOT NULL,
  "created_at" timestamptz NOT NULL,
  "agent_id" uuid NULL,
  "project_id" uuid NOT NULL,
  "ticket_id" uuid NULL,
  PRIMARY KEY ("id")
);
-- Create index "activityevent_project_id_created_at" to table: "activity_events"
CREATE INDEX "activityevent_project_id_created_at" ON "activity_events" ("project_id", "created_at");
-- Create index "activityevent_ticket_id_created_at" to table: "activity_events"
CREATE INDEX "activityevent_ticket_id_created_at" ON "activity_events" ("ticket_id", "created_at");
-- Create "agent_providers" table
CREATE TABLE "agent_providers" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "adapter_type" character varying NOT NULL,
  "cli_command" character varying NOT NULL,
  "cli_args" text[] NULL,
  "auth_config" jsonb NOT NULL,
  "model_name" character varying NOT NULL,
  "model_temperature" double precision NOT NULL DEFAULT 0,
  "model_max_tokens" bigint NOT NULL DEFAULT 16384,
  "cost_per_input_token" numeric(18,8) NOT NULL DEFAULT 0,
  "cost_per_output_token" numeric(18,8) NOT NULL DEFAULT 0,
  "machine_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "agentprovider_organization_id_name" to table: "agent_providers"
CREATE UNIQUE INDEX "agentprovider_organization_id_name" ON "agent_providers" ("organization_id", "name");
-- Create "agents" table
CREATE TABLE "agents" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "status" character varying NOT NULL DEFAULT 'idle',
  "session_id" character varying NULL,
  "total_tokens_used" bigint NOT NULL DEFAULT 0,
  "total_tickets_completed" bigint NOT NULL DEFAULT 0,
  "last_heartbeat_at" timestamptz NULL,
  "current_ticket_id" uuid NULL,
  "provider_id" uuid NOT NULL,
  "project_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "agent_project_id_name" to table: "agents"
CREATE UNIQUE INDEX "agent_project_id_name" ON "agents" ("project_id", "name");
-- Create index "agent_project_id_status_last_heartbeat_at" to table: "agents"
CREATE INDEX "agent_project_id_status_last_heartbeat_at" ON "agents" ("project_id", "status", "last_heartbeat_at");
-- Create "organizations" table
CREATE TABLE "organizations" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "slug" character varying NOT NULL,
  "default_agent_provider_id" uuid NULL,
  PRIMARY KEY ("id")
);
-- Create index "organization_slug" to table: "organizations"
CREATE UNIQUE INDEX "organization_slug" ON "organizations" ("slug");
-- Create "project_repos" table
CREATE TABLE "project_repos" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "repository_url" character varying NOT NULL,
  "default_branch" character varying NOT NULL DEFAULT 'main',
  "clone_path" character varying NULL,
  "is_primary" boolean NOT NULL DEFAULT false,
  "labels" text[] NULL,
  "project_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "projectrepo_labels" to table: "project_repos"
CREATE INDEX "projectrepo_labels" ON "project_repos" USING GIN ("labels");
-- Create index "projectrepo_project_id_name" to table: "project_repos"
CREATE UNIQUE INDEX "projectrepo_project_id_name" ON "project_repos" ("project_id", "name");
-- Create "projects" table
CREATE TABLE "projects" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "slug" character varying NOT NULL,
  "description" text NULL,
  "status" character varying NOT NULL DEFAULT 'planning',
  "max_concurrent_agents" bigint NOT NULL DEFAULT 5,
  "organization_id" uuid NOT NULL,
  "default_agent_provider_id" uuid NULL,
  PRIMARY KEY ("id")
);
-- Create index "project_organization_id_slug" to table: "projects"
CREATE UNIQUE INDEX "project_organization_id_slug" ON "projects" ("organization_id", "slug");
-- Create "scheduled_jobs" table
CREATE TABLE "scheduled_jobs" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "cron_expression" character varying NOT NULL,
  "ticket_template" jsonb NOT NULL,
  "is_enabled" boolean NOT NULL DEFAULT true,
  "last_run_at" timestamptz NULL,
  "next_run_at" timestamptz NULL,
  "project_id" uuid NOT NULL,
  "workflow_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "scheduledjob_next_run_at" to table: "scheduled_jobs"
CREATE INDEX "scheduledjob_next_run_at" ON "scheduled_jobs" ("next_run_at") WHERE (is_enabled = true);
-- Create index "scheduledjob_project_id_name" to table: "scheduled_jobs"
CREATE UNIQUE INDEX "scheduledjob_project_id_name" ON "scheduled_jobs" ("project_id", "name");
-- Create "ticket_dependencies" table
CREATE TABLE "ticket_dependencies" (
  "id" uuid NOT NULL,
  "type" character varying NOT NULL DEFAULT 'blocks',
  "source_ticket_id" uuid NOT NULL,
  "target_ticket_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "ticketdependency_source_ticket_id" to table: "ticket_dependencies"
CREATE INDEX "ticketdependency_source_ticket_id" ON "ticket_dependencies" ("source_ticket_id");
-- Create index "ticketdependency_source_ticket_id_target_ticket_id_type" to table: "ticket_dependencies"
CREATE UNIQUE INDEX "ticketdependency_source_ticket_id_target_ticket_id_type" ON "ticket_dependencies" ("source_ticket_id", "target_ticket_id", "type");
-- Create index "ticketdependency_target_ticket_id_type" to table: "ticket_dependencies"
CREATE INDEX "ticketdependency_target_ticket_id_type" ON "ticket_dependencies" ("target_ticket_id", "type");
-- Create "ticket_external_links" table
CREATE TABLE "ticket_external_links" (
  "id" uuid NOT NULL,
  "link_type" character varying NOT NULL,
  "url" character varying NOT NULL,
  "external_id" character varying NOT NULL,
  "title" character varying NULL,
  "status" character varying NULL,
  "relation" character varying NOT NULL DEFAULT 'related',
  "created_at" timestamptz NOT NULL,
  "ticket_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "ticketexternallink_external_id" to table: "ticket_external_links"
CREATE INDEX "ticketexternallink_external_id" ON "ticket_external_links" ("external_id");
-- Create index "ticketexternallink_ticket_id_external_id" to table: "ticket_external_links"
CREATE UNIQUE INDEX "ticketexternallink_ticket_id_external_id" ON "ticket_external_links" ("ticket_id", "external_id");
-- Create "ticket_repo_scopes" table
CREATE TABLE "ticket_repo_scopes" (
  "id" uuid NOT NULL,
  "branch_name" character varying NOT NULL,
  "pull_request_url" character varying NULL,
  "pr_status" character varying NOT NULL DEFAULT 'none',
  "ci_status" character varying NOT NULL DEFAULT 'pending',
  "is_primary_scope" boolean NOT NULL DEFAULT false,
  "repo_id" uuid NOT NULL,
  "ticket_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "ticketreposcope_repo_id_branch_name" to table: "ticket_repo_scopes"
CREATE INDEX "ticketreposcope_repo_id_branch_name" ON "ticket_repo_scopes" ("repo_id", "branch_name");
-- Create index "ticketreposcope_ticket_id" to table: "ticket_repo_scopes"
CREATE INDEX "ticketreposcope_ticket_id" ON "ticket_repo_scopes" ("ticket_id");
-- Create index "ticketreposcope_ticket_id_repo_id" to table: "ticket_repo_scopes"
CREATE UNIQUE INDEX "ticketreposcope_ticket_id_repo_id" ON "ticket_repo_scopes" ("ticket_id", "repo_id");
-- Create "ticket_status" table
CREATE TABLE "ticket_status" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "color" character varying NOT NULL,
  "icon" character varying NULL,
  "position" bigint NOT NULL DEFAULT 0,
  "is_default" boolean NOT NULL DEFAULT false,
  "description" character varying NULL,
  "project_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "ticketstatus_project_id_name" to table: "ticket_status"
CREATE UNIQUE INDEX "ticketstatus_project_id_name" ON "ticket_status" ("project_id", "name");
-- Create index "ticketstatus_project_id_position" to table: "ticket_status"
CREATE INDEX "ticketstatus_project_id_position" ON "ticket_status" ("project_id", "position");
-- Create "tickets" table
CREATE TABLE "tickets" (
  "id" uuid NOT NULL,
  "identifier" character varying NOT NULL,
  "title" character varying NOT NULL,
  "description" text NULL,
  "priority" character varying NOT NULL DEFAULT 'medium',
  "type" character varying NOT NULL DEFAULT 'feature',
  "created_by" character varying NOT NULL,
  "external_ref" character varying NULL,
  "attempt_count" bigint NOT NULL DEFAULT 0,
  "consecutive_errors" bigint NOT NULL DEFAULT 0,
  "next_retry_at" timestamptz NULL,
  "retry_paused" boolean NOT NULL DEFAULT false,
  "pause_reason" character varying NULL,
  "stall_count" bigint NOT NULL DEFAULT 0,
  "retry_token" character varying NULL,
  "harness_version" bigint NOT NULL DEFAULT 0,
  "budget_usd" numeric(12,2) NOT NULL DEFAULT 0,
  "cost_tokens_input" bigint NOT NULL DEFAULT 0,
  "cost_tokens_output" bigint NOT NULL DEFAULT 0,
  "cost_amount" numeric(12,2) NOT NULL DEFAULT 0,
  "metadata" jsonb NOT NULL,
  "started_at" timestamptz NULL,
  "completed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "current_run_id" uuid NULL,
  "target_machine_id" uuid NULL,
  "project_id" uuid NOT NULL,
  "parent_ticket_id" uuid NULL,
  "status_id" uuid NOT NULL,
  "workflow_id" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "tickets_tickets_children" FOREIGN KEY ("parent_ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Create index "ticket_project_id_identifier" to table: "tickets"
CREATE UNIQUE INDEX "ticket_project_id_identifier" ON "tickets" ("project_id", "identifier");
-- Create index "ticket_project_id_external_ref" to table: "tickets"
CREATE INDEX "ticket_project_id_external_ref" ON "tickets" ("project_id", "external_ref");
-- Create index "ticket_project_id_status_id" to table: "tickets"
CREATE INDEX "ticket_project_id_status_id" ON "tickets" ("project_id", "status_id");
-- Create index "ticket_project_id_status_id_current_run_id_priority_created_at" to table: "tickets"
CREATE INDEX "ticket_project_id_status_id_current_run_id_priority_created_at" ON "tickets" ("project_id", "status_id", "current_run_id", "priority", "created_at");
-- Create "workflows" table
CREATE TABLE "workflows" (
  "id" uuid NOT NULL,
  "name" character varying NOT NULL,
  "type" character varying NOT NULL,
  "harness_path" character varying NOT NULL,
  "hooks" jsonb NOT NULL,
  "max_concurrent" bigint NOT NULL DEFAULT 3,
  "max_retry_attempts" bigint NOT NULL DEFAULT 3,
  "timeout_minutes" bigint NOT NULL DEFAULT 60,
  "stall_timeout_minutes" bigint NOT NULL DEFAULT 5,
  "version" bigint NOT NULL DEFAULT 1,
  "is_active" boolean NOT NULL DEFAULT true,
  "project_id" uuid NOT NULL,
  "pickup_status_id" uuid NOT NULL,
  "finish_status_id" uuid NULL,
  PRIMARY KEY ("id")
);
-- Create index "workflow_project_id_is_active" to table: "workflows"
CREATE INDEX "workflow_project_id_is_active" ON "workflows" ("project_id", "is_active");
-- Create index "workflow_project_id_name" to table: "workflows"
CREATE UNIQUE INDEX "workflow_project_id_name" ON "workflows" ("project_id", "name");
-- Modify "activity_events" table
ALTER TABLE "activity_events" ADD CONSTRAINT "activity_events_agents_activity_events" FOREIGN KEY ("agent_id") REFERENCES "agents" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "activity_events_projects_activity_events" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "activity_events_tickets_activity_events" FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "agent_providers" table
ALTER TABLE "agent_providers" ADD CONSTRAINT "agent_providers_machines_providers" FOREIGN KEY ("machine_id") REFERENCES "machines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "agent_providers_organizations_providers" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "agents" table
ALTER TABLE "agents" ADD CONSTRAINT "agents_agent_providers_agents" FOREIGN KEY ("provider_id") REFERENCES "agent_providers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "agents_projects_agents" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "agents_tickets_current_ticket" FOREIGN KEY ("current_ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "organizations" table
ALTER TABLE "organizations" ADD CONSTRAINT "organizations_agent_providers_default_agent_provider" FOREIGN KEY ("default_agent_provider_id") REFERENCES "agent_providers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "project_repos" table
ALTER TABLE "project_repos" ADD CONSTRAINT "project_repos_projects_repos" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "projects" table
ALTER TABLE "projects" ADD CONSTRAINT "projects_agent_providers_default_agent_provider" FOREIGN KEY ("default_agent_provider_id") REFERENCES "agent_providers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "projects_organizations_projects" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "scheduled_jobs" table
ALTER TABLE "scheduled_jobs" ADD CONSTRAINT "scheduled_jobs_projects_scheduled_jobs" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "scheduled_jobs_workflows_scheduled_jobs" FOREIGN KEY ("workflow_id") REFERENCES "workflows" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ticket_dependencies" table
ALTER TABLE "ticket_dependencies" ADD CONSTRAINT "ticket_dependencies_tickets_incoming_dependencies" FOREIGN KEY ("target_ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ticket_dependencies_tickets_outgoing_dependencies" FOREIGN KEY ("source_ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ticket_external_links" table
ALTER TABLE "ticket_external_links" ADD CONSTRAINT "ticket_external_links_tickets_external_links" FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ticket_repo_scopes" table
ALTER TABLE "ticket_repo_scopes" ADD CONSTRAINT "ticket_repo_scopes_project_repos_ticket_scopes" FOREIGN KEY ("repo_id") REFERENCES "project_repos" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "ticket_repo_scopes_tickets_repo_scopes" FOREIGN KEY ("ticket_id") REFERENCES "tickets" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ticket_status" table
ALTER TABLE "ticket_status" ADD CONSTRAINT "ticket_status_projects_statuses" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "tickets" table
ALTER TABLE "tickets" ADD CONSTRAINT "tickets_agent_runs_current_for_ticket" FOREIGN KEY ("current_run_id") REFERENCES "agent_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "tickets_machines_target_tickets" FOREIGN KEY ("target_machine_id") REFERENCES "machines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "tickets_projects_tickets" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "tickets_ticket_status_tickets" FOREIGN KEY ("status_id") REFERENCES "ticket_status" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "tickets_workflows_tickets" FOREIGN KEY ("workflow_id") REFERENCES "workflows" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Modify "workflows" table
ALTER TABLE "workflows" ADD CONSTRAINT "workflows_projects_workflows" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "workflows_ticket_status_finish_workflows" FOREIGN KEY ("finish_status_id") REFERENCES "ticket_status" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "workflows_ticket_status_pickup_workflows" FOREIGN KEY ("pickup_status_id") REFERENCES "ticket_status" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
