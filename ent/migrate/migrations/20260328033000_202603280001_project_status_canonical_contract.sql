-- Align project.status storage and existing rows to the canonical PRD contract.
UPDATE "projects"
SET "status" = CASE "status"
  WHEN 'planning' THEN 'Planned'
  WHEN 'active' THEN 'In Progress'
  WHEN 'paused' THEN 'Canceled'
  WHEN 'archived' THEN 'Archived'
  ELSE "status"
END;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM "projects"
    WHERE "status" NOT IN ('Backlog', 'Planned', 'In Progress', 'Completed', 'Canceled', 'Archived')
  ) THEN
    RAISE EXCEPTION 'projects.status contains unsupported values after canonical migration';
  END IF;
END
$$;

ALTER TABLE "projects"
ALTER COLUMN "status" TYPE character varying,
ALTER COLUMN "status" SET DEFAULT 'Planned';
