-- Promote workflow status bindings from legacy single columns to required multi-status join tables.
CREATE TABLE "workflow_pickup_statuses" (
  "workflow_id" uuid NOT NULL,
  "ticket_status_id" uuid NOT NULL,
  PRIMARY KEY ("workflow_id", "ticket_status_id"),
  CONSTRAINT "workflow_pickup_statuses_workflow_id" FOREIGN KEY ("workflow_id") REFERENCES "workflows" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "workflow_pickup_statuses_ticket_status_id" FOREIGN KEY ("ticket_status_id") REFERENCES "ticket_status" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

CREATE TABLE "workflow_finish_statuses" (
  "workflow_id" uuid NOT NULL,
  "ticket_status_id" uuid NOT NULL,
  PRIMARY KEY ("workflow_id", "ticket_status_id"),
  CONSTRAINT "workflow_finish_statuses_workflow_id" FOREIGN KEY ("workflow_id") REFERENCES "workflows" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "workflow_finish_statuses_ticket_status_id" FOREIGN KEY ("ticket_status_id") REFERENCES "ticket_status" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

INSERT INTO "workflow_pickup_statuses" ("workflow_id", "ticket_status_id")
SELECT "id", "pickup_status_id"
FROM "workflows";

INSERT INTO "workflow_finish_statuses" ("workflow_id", "ticket_status_id")
SELECT "id", COALESCE("finish_status_id", "pickup_status_id")
FROM "workflows";

ALTER TABLE "workflows"
  DROP CONSTRAINT "workflows_ticket_status_pickup_workflows",
  DROP CONSTRAINT "workflows_ticket_status_finish_workflows",
  DROP COLUMN "pickup_status_id",
  DROP COLUMN "finish_status_id";
