-- Modify "agent_runs" table to record the exact workflow/skill versions consumed by a runtime snapshot.
ALTER TABLE "agent_runs"
  ADD COLUMN "skill_version_ids" text[] NULL,
  ADD COLUMN "workflow_version_id" uuid NULL;

ALTER TABLE "agent_runs"
  ADD CONSTRAINT "agent_runs_workflow_versions_agent_runs"
  FOREIGN KEY ("workflow_version_id") REFERENCES "workflow_versions" ("id")
  ON DELETE SET NULL;
