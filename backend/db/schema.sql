-- This script sets up all necessary tables, indexes, and functions.

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =================================================================
-- Table: users
-- Stores user account information.
-- =================================================================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user', -- e.g., 'user', 'admin'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =================================================================
-- Table: tokens
-- Stores refresh tokens for persistent user sessions.
-- =================================================================
CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    value VARCHAR(255) NOT NULL, -- This should be a hash of the token
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =================================================================
-- Table: movies
-- Stores metadata about uploaded video files.
-- =================================================================
CREATE TABLE IF NOT EXISTS movies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    original_file_path VARCHAR(500) NOT NULL DEFAULT '',
    transcoded_file_path VARCHAR(500) NOT NULL DEFAULT '',
    hls_playlist_url VARCHAR(500) NOT NULL DEFAULT '',
    duration_seconds INTEGER NOT NULL DEFAULT 0,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream',
    status VARCHAR(50) NOT NULL DEFAULT 'processing',
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processing_ended_at TIMESTAMP WITH TIME ZONE
);

-- =================================================================
-- Table: rooms
-- Represents a watch party room created by a host.
-- =================================================================
CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =================================================================
-- Table: room_access
-- Manages user access permissions for specific rooms.
-- =================================================================
CREATE TABLE IF NOT EXISTS room_access (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    access_type VARCHAR(50) NOT NULL DEFAULT 'granted', -- e.g., 'granted', 'guest'
    status VARCHAR(20) NOT NULL DEFAULT 'granted', -- e.g., 'granted', 'pending'
    granted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, room_id)
);

-- =================================================================
-- Table: room_invitations
-- Stores email-based invitations for users to join rooms.
-- =================================================================
CREATE TABLE IF NOT EXISTS room_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    message TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =================================================================
-- Table: room_sessions
-- Stores persistent metadata for a watch party session (for history/audit).
-- Real-time sync state is handled by Redis.
-- =================================================================
CREATE TABLE IF NOT EXISTS room_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    host_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    movie_id UUID NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    session_name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(room_id, created_at)
);

-- =================================================================
-- Table: room_session_events
-- Audit log of user actions during a watch party session.
-- =================================================================
CREATE TABLE IF NOT EXISTS room_session_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES room_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL, -- 'join', 'leave', 'play', 'pause', 'seek'
    event_data JSONB,
    video_time REAL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- =================================================================
-- Table: guest_access_requests
-- Stores requests from unauthenticated guests to join a room.
-- =================================================================
CREATE TABLE IF NOT EXISTS guest_access_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    guest_name VARCHAR(255) NOT NULL,
    request_message TEXT,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'approved', 'denied'
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE
);

-- =================================================================
-- Table: guest_sessions
-- Stores temporary session tokens for approved guests.
-- =================================================================
CREATE TABLE IF NOT EXISTS guest_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    guest_name VARCHAR(255) NOT NULL,
    session_token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    approved_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =================================================================
-- Indexes for Performance
-- =================================================================
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_tokens_value ON tokens(value);
CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_movies_uploaded_by ON movies(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_movies_status ON movies(status);
CREATE INDEX IF NOT EXISTS idx_rooms_movie_id ON rooms(movie_id);
CREATE INDEX IF NOT EXISTS idx_rooms_host_id ON rooms(host_id);
CREATE INDEX IF NOT EXISTS idx_room_access_room_id ON room_access(room_id);
CREATE INDEX IF NOT EXISTS idx_room_invitations_room_id ON room_invitations(room_id);
CREATE INDEX IF NOT EXISTS idx_room_invitations_token ON room_invitations(token);
CREATE INDEX IF NOT EXISTS idx_room_invitations_email ON room_invitations(email);
CREATE INDEX IF NOT EXISTS idx_room_invitations_expires_at ON room_invitations(expires_at);
CREATE INDEX IF NOT EXISTS idx_room_sessions_room_id ON room_sessions(room_id);
CREATE INDEX IF NOT EXISTS idx_room_sessions_host_id ON room_sessions(host_id);
CREATE INDEX IF NOT EXISTS idx_room_sessions_created_at ON room_sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_room_session_events_session_id ON room_session_events(session_id);
CREATE INDEX IF NOT EXISTS idx_room_session_events_user_id ON room_session_events(user_id);
CREATE INDEX IF NOT EXISTS idx_room_session_events_event_type ON room_session_events(event_type);
CREATE INDEX IF NOT EXISTS idx_room_session_events_timestamp ON room_session_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_guest_requests_room ON guest_access_requests(room_id);
CREATE INDEX IF NOT EXISTS idx_guest_requests_status ON guest_access_requests(status);
CREATE INDEX IF NOT EXISTS idx_guest_sessions_room ON guest_sessions(room_id);
CREATE INDEX IF NOT EXISTS idx_guest_sessions_token ON guest_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_guest_sessions_expires ON guest_sessions(expires_at);

-- =================================================================
-- Helper Functions
-- =================================================================

-- Function to log an event for a given session.
CREATE OR REPLACE FUNCTION log_session_event(
    p_session_id UUID,
    p_user_id UUID,
    p_event_type VARCHAR(50),
    p_event_data JSONB DEFAULT NULL,
    p_video_time REAL DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    event_id UUID;
BEGIN
    INSERT INTO room_session_events (session_id, user_id, event_type, event_data, video_time)
    VALUES (p_session_id, p_user_id, p_event_type, p_event_data, p_video_time)
    RETURNING id INTO event_id;
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;

-- Function to mark a room session as ended.
CREATE OR REPLACE FUNCTION end_room_session(p_session_id UUID)
RETURNS void AS $$
BEGIN
    UPDATE room_sessions
    SET ended_at = NOW()
    WHERE id = p_session_id AND ended_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- =================================================================
-- Seed Data
-- =================================================================

INSERT INTO users (email, password_hash, role)
VALUES ('marcellus@c3llus.dev', '$2a$10$vz3m3NI53x6g4ynoGXMMk.6kufWPCWm/Tzo6I1L9XRom1AItzFvpS', 'admin')
ON CONFLICT (email) DO NOTHING;

-- =================================================================
-- Post-creation Comments
-- =================================================================
COMMENT ON DATABASE watch_party IS 'Database for the Watch Party application. Real-time sync is handled by Redis, while PostgreSQL manages persistent data.';
COMMENT ON TABLE room_sessions IS 'Session metadata for video watching sessions. Real-time sync state moved to Redis.';
COMMENT ON TABLE room_session_events IS 'Audit log of user actions during video sessions. Real-time events handled via Redis.';