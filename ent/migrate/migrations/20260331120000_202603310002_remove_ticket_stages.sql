-- Collapse legacy ticket stage capacity and ordering onto ticket_status.
ALTER TABLE "ticket_status" ADD COLUMN IF NOT EXISTS "max_active_runs" bigint NULL;

UPDATE "ticket_status" AS status
SET "max_active_runs" = stage."max_active_runs"
FROM "ticket_stages" AS stage
WHERE status."stage_id" = stage."id"
  AND status."max_active_runs" IS NULL
  AND stage."max_active_runs" IS NOT NULL;

ALTER TABLE "ticket_status" DROP CONSTRAINT IF EXISTS "ticket_status_ticket_stages_statuses";
DROP INDEX IF EXISTS "ticketstatus_project_id_stage_id_position";
ALTER TABLE "ticket_status" DROP COLUMN IF EXISTS "stage_id";
DROP TABLE IF EXISTS "ticket_stages";
