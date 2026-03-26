-- Backfill strategy:
-- 1. Ensure current_run_id exists and becomes the only active-claim pointer.
-- 2. Preserve rows that already reference an AgentRun.
-- 3. Legacy assigned_agent_id values cannot be mapped to a specific AgentRun, so dropping the column safely releases those stale claims.

ALTER TABLE "tickets" ADD COLUMN IF NOT EXISTS "current_run_id" uuid NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'tickets_agent_runs_current_for_ticket'
  ) THEN
    ALTER TABLE "tickets"
      ADD CONSTRAINT "tickets_agent_runs_current_for_ticket"
      FOREIGN KEY ("current_run_id") REFERENCES "agent_runs" ("id")
      ON UPDATE NO ACTION
      ON DELETE SET NULL;
  END IF;
END
$$;

DROP INDEX IF EXISTS "ticket_project_id_status_id_assigned_agent_id_priority_created_";
CREATE INDEX IF NOT EXISTS "ticket_project_id_status_id_current_run_id_priority_created_at"
  ON "tickets" ("project_id", "status_id", "current_run_id", "priority", "created_at");

ALTER TABLE "tickets" DROP CONSTRAINT IF EXISTS "tickets_agents_assigned_tickets";
ALTER TABLE "tickets" DROP COLUMN IF EXISTS "assigned_agent_id";
