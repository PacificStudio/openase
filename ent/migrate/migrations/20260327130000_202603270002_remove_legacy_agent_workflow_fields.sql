-- Drop PRD-retired compatibility columns that should no longer appear in runtime/API contracts.
ALTER TABLE "agents" DROP COLUMN IF EXISTS "workspace_path";
ALTER TABLE "workflows" DROP COLUMN IF EXISTS "required_machine_labels";
