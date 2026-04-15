ALTER TABLE "machines" ADD COLUMN "reachability_mode" character varying NULL;
ALTER TABLE "machines" ADD COLUMN "execution_mode" character varying NULL;

UPDATE "machines"
SET
  "reachability_mode" = CASE
    WHEN "connection_mode" = 'local' THEN 'local'
    WHEN "connection_mode" = 'ws_reverse' THEN 'reverse_connect'
    ELSE 'direct_connect'
  END,
  "execution_mode" = CASE
    WHEN "connection_mode" = 'local' THEN 'local_process'
    ELSE 'websocket'
  END
WHERE "reachability_mode" IS NULL
   OR "execution_mode" IS NULL;
