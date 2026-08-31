-- 003_create_chat_messages.sql
-- ChatMessageModel was only ever created by the development AutoMigrate branch
-- in cmd/server/main.go, so "chat_messages" did not exist in any environment
-- that runs the SQL migrations instead. See KPD-61.
--
-- This is the working owner/runner chat that KPD-48 asks about when it decides
-- what to do with the empty service-chat scaffold. Whatever is decided there,
-- the table has to exist outside a developer laptop first.
--
-- Note the column name: the Go field is MsgType, mapped to message_type.
-- booking_id points at a booking owned by service-booking, in a different
-- database, so it carries no foreign key -- same convention as trip_tracks.

CREATE TABLE IF NOT EXISTS chat_messages (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id   UUID        NOT NULL,
    sender_id    UUID        NOT NULL,
    sender_role  VARCHAR(20) NOT NULL,
    message_type VARCHAR(20) NOT NULL CHECK (message_type IN ('text', 'image', 'quick_reply')),
    content      TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The chat pane reads one booking's messages oldest-first.
CREATE INDEX IF NOT EXISTS idx_chat_messages_booking ON chat_messages(booking_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_booking_created ON chat_messages(booking_id, created_at);
