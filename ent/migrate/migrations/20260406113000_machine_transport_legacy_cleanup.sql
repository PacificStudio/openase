UPDATE "machines"
SET "connection_mode" = 'ws_listener'
WHERE "connection_mode" = 'ssh';
