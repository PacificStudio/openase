ALTER TABLE "tickets" ADD COLUMN "archived" boolean NOT NULL DEFAULT false;

UPDATE "tickets" AS t
SET "archived" = true
FROM "ticket_status" AS s
WHERE t."status_id" = s."id"
  AND lower(trim(s."name")) = 'archived';

CREATE INDEX "ticket_project_id_archived_created_at" ON "tickets" ("project_id", "archived", "created_at");
