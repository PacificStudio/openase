-- Create "ticket_stages" table
CREATE TABLE "ticket_stages" (
  "id" uuid NOT NULL,
  "key" character varying NOT NULL,
  "name" character varying NOT NULL,
  "position" bigint NOT NULL DEFAULT 0,
  "max_active_runs" bigint NULL,
  "description" character varying NULL,
  "project_id" uuid NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "ticketstage_project_id_key" to table: "ticket_stages"
CREATE UNIQUE INDEX "ticketstage_project_id_key" ON "ticket_stages" ("project_id", "key");
-- Create index "ticketstage_project_id_position" to table: "ticket_stages"
CREATE INDEX "ticketstage_project_id_position" ON "ticket_stages" ("project_id", "position");
-- Add stage_id column to "ticket_status" table
ALTER TABLE "ticket_status" ADD COLUMN "stage_id" uuid NULL;
-- Create index "ticketstatus_project_id_stage_id_position" to table: "ticket_status"
CREATE INDEX "ticketstatus_project_id_stage_id_position" ON "ticket_status" ("project_id", "stage_id", "position");
-- Populate default stage rows for projects that already have ticket statuses
INSERT INTO "ticket_stages" ("id", "project_id", "key", "name", "position", "max_active_runs", "description")
SELECT
  (
    substr(md5(projects.project_id::text || ':' || template.key), 1, 8) || '-' ||
    substr(md5(projects.project_id::text || ':' || template.key), 9, 4) || '-' ||
    substr(md5(projects.project_id::text || ':' || template.key), 13, 4) || '-' ||
    substr(md5(projects.project_id::text || ':' || template.key), 17, 4) || '-' ||
    substr(md5(projects.project_id::text || ':' || template.key), 21, 12)
  )::uuid,
  projects.project_id,
  template.key,
  template.name,
  template.position,
  NULL,
  template.description
FROM (
  SELECT DISTINCT "project_id"
  FROM "ticket_status"
) AS projects
CROSS JOIN (
  VALUES
    ('backlog', 'Backlog', 0, '积压阶段'),
    ('in_progress', 'In Progress', 1, '进行中阶段'),
    ('review', 'Review', 2, '审查阶段'),
    ('done', 'Done', 3, '收尾阶段')
) AS template("key", "name", "position", "description");
-- Backfill stage_id for seeded default statuses
UPDATE "ticket_status" AS status
SET "stage_id" = stage."id"
FROM "ticket_stages" AS stage
WHERE stage."project_id" = status."project_id"
  AND (
    (stage."key" = 'backlog' AND status."name" IN ('Backlog', 'Todo')) OR
    (stage."key" = 'in_progress' AND status."name" = 'In Progress') OR
    (stage."key" = 'review' AND status."name" = 'In Review') OR
    (stage."key" = 'done' AND status."name" IN ('Done', 'Cancelled'))
  );
-- Modify "ticket_stages" table
ALTER TABLE "ticket_stages" ADD CONSTRAINT "ticket_stages_projects_stages" FOREIGN KEY ("project_id") REFERENCES "projects" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Modify "ticket_status" table
ALTER TABLE "ticket_status" ADD CONSTRAINT "ticket_status_ticket_stages_statuses" FOREIGN KEY ("stage_id") REFERENCES "ticket_stages" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
