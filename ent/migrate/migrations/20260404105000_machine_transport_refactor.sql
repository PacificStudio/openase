ALTER TABLE "machines" ADD COLUMN "connection_mode" character varying NULL;
UPDATE "machines"
SET "connection_mode" = CASE
  WHEN "host" = 'local' THEN 'local'
  ELSE 'ssh'
END
WHERE "connection_mode" IS NULL;
ALTER TABLE "machines" ALTER COLUMN "connection_mode" SET NOT NULL;
ALTER TABLE "machines" ALTER COLUMN "connection_mode" SET DEFAULT 'ssh';

ALTER TABLE "machines" ADD COLUMN "transport_capabilities" text[] NULL;
ALTER TABLE "machines" ADD COLUMN "advertised_endpoint" character varying NULL;
ALTER TABLE "machines" ADD COLUMN "daemon_registered" boolean NOT NULL DEFAULT false;
ALTER TABLE "machines" ADD COLUMN "daemon_last_registered_at" timestamptz NULL;
ALTER TABLE "machines" ADD COLUMN "daemon_session_id" character varying NULL;
ALTER TABLE "machines" ADD COLUMN "daemon_session_state" character varying NOT NULL DEFAULT 'unknown';
ALTER TABLE "machines" ADD COLUMN "detected_os" character varying NOT NULL DEFAULT 'unknown';
ALTER TABLE "machines" ADD COLUMN "detected_arch" character varying NOT NULL DEFAULT 'unknown';
ALTER TABLE "machines" ADD COLUMN "detection_status" character varying NOT NULL DEFAULT 'unknown';
ALTER TABLE "machines" ADD COLUMN "channel_credential_kind" character varying NOT NULL DEFAULT 'none';
ALTER TABLE "machines" ADD COLUMN "channel_token_id" character varying NULL;
ALTER TABLE "machines" ADD COLUMN "channel_certificate_id" character varying NULL;
