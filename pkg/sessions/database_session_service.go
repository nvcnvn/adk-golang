// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sessions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nvcnvn/adk-golang/pkg/events"
)

// DatabaseConfig contains configuration for the database connection
type DatabaseConfig struct {
	// Driver is the database driver to use (e.g., "postgres", "mysql", "sqlite3")
	Driver string
	// DSN is the data source name or connection string
	DSN string
	// MaxConnections is the maximum number of connections to keep open
	MaxConnections int
	// ConnMaxLifetime is the maximum amount of time a connection may be reused
	ConnMaxLifetime time.Duration
}

// DatabaseSessionService implements SessionService using a SQL database
type DatabaseSessionService struct {
	db     *sql.DB
	config *DatabaseConfig
	// Fallback service used when database is not available
	fallback *InMemorySessionService
}

// NewDatabaseSessionService creates a new session service that uses a SQL database
func NewDatabaseSessionService(config *DatabaseConfig) (*DatabaseSessionService, error) {
	// Open database connection
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	if config.MaxConnections > 0 {
		db.SetMaxOpenConns(config.MaxConnections)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DatabaseSessionService{
		db:       db,
		config:   config,
		fallback: NewInMemorySessionService(),
	}, nil
}

// Init initializes the database tables if they don't exist
func (s *DatabaseSessionService) Init(ctx context.Context) error {
	// Create sessions table
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			app_name TEXT NOT NULL,
			user_id TEXT NOT NULL,
			state JSON,
			create_time TIMESTAMP NOT NULL,
			update_time TIMESTAMP NOT NULL,
			UNIQUE(app_name, user_id, id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create events table
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			event_data JSON NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *DatabaseSessionService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CreateSession creates a new session in the database
func (s *DatabaseSessionService) CreateSession(
	ctx context.Context,
	appName, userID string,
	state map[string]interface{},
	sessionID string,
) (*Session, error) {
	session := NewSession(appName, userID, state, sessionID)

	// Marshal state to JSON
	stateJSON, err := json.Marshal(session.StateMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	// Insert session into database
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, app_name, user_id, state, create_time, update_time)
		VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID, session.AppName, session.UserID, stateJSON,
		session.CreateTime, session.UpdateTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a session by ID from the database
func (s *DatabaseSessionService) GetSession(
	ctx context.Context,
	appName, userID, sessionID string,
	config *GetSessionConfig,
) (*Session, error) {
	// Query session from database
	var (
		id         string
		stateJSON  []byte
		createTime time.Time
		updateTime time.Time
	)

	err := s.db.QueryRowContext(ctx,
		`SELECT id, state, create_time, update_time
		FROM sessions
		WHERE app_name = ? AND user_id = ? AND id = ?`,
		appName, userID, sessionID,
	).Scan(&id, &stateJSON, &createTime, &updateTime)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Unmarshal state JSON
	var stateMap map[string]interface{}
	if err := json.Unmarshal(stateJSON, &stateMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Create session object
	session := &Session{
		ID:         id,
		AppName:    appName,
		UserID:     userID,
		State:      NewState(stateMap, make(map[string]interface{})),
		StateMap:   stateMap,
		CreateTime: createTime,
		UpdateTime: updateTime,
		Events:     []*events.Event{},
	}

	// Query events if needed
	if config == nil || config.NumRecentEvents != 0 {
		// Build query for events
		query := `SELECT event_data FROM events WHERE session_id = ?`
		args := []interface{}{sessionID}

		if config != nil && config.AfterTimestamp > 0 {
			query += " AND timestamp > ?"
			ts := time.Unix(0, int64(config.AfterTimestamp*1e9))
			args = append(args, ts)
		}

		query += " ORDER BY timestamp"

		// Execute query
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to query events: %w", err)
		}
		defer rows.Close()

		// Collect events
		var eventsList []*events.Event
		for rows.Next() {
			var eventJSON []byte
			if err := rows.Scan(&eventJSON); err != nil {
				return nil, fmt.Errorf("failed to scan event: %w", err)
			}

			// Create event and unmarshal
			event := new(events.Event)
			if err := json.Unmarshal(eventJSON, event); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event: %w", err)
			}

			eventsList = append(eventsList, event)
		}

		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating events: %w", err)
		}

		// Apply limit if specified
		if config != nil && config.NumRecentEvents > 0 && len(eventsList) > config.NumRecentEvents {
			start := len(eventsList) - config.NumRecentEvents
			eventsList = eventsList[start:]
		}

		session.Events = eventsList
	}

	return session, nil
}

// ListSessions lists all sessions for a user from the database
func (s *DatabaseSessionService) ListSessions(
	ctx context.Context,
	appName, userID string,
) (*ListSessionsResponse, error) {
	// Query sessions from database (without events for performance)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, state, create_time, update_time
		FROM sessions
		WHERE app_name = ? AND user_id = ?
		ORDER BY update_time DESC`,
		appName, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var (
			id         string
			stateJSON  []byte
			createTime time.Time
			updateTime time.Time
		)

		if err := rows.Scan(&id, &stateJSON, &createTime, &updateTime); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		var stateMap map[string]interface{}
		if err := json.Unmarshal(stateJSON, &stateMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}

		sessions = append(sessions, &Session{
			ID:         id,
			AppName:    appName,
			UserID:     userID,
			StateMap:   stateMap,
			CreateTime: createTime,
			UpdateTime: updateTime,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sessions: %w", err)
	}

	return &ListSessionsResponse{Sessions: sessions}, nil
}

// DeleteSession deletes a session from the database
func (s *DatabaseSessionService) DeleteSession(
	ctx context.Context,
	appName, userID, sessionID string,
) error {
	// Delete session (events will be cascaded due to foreign key)
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions
		WHERE app_name = ? AND user_id = ? AND id = ?`,
		appName, userID, sessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Check if session was found
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// ListEvents lists events in a session from the database
func (s *DatabaseSessionService) ListEvents(
	ctx context.Context,
	appName, userID, sessionID string,
) (*ListEventsResponse, error) {
	// First check if the session exists
	exists := false
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM sessions WHERE app_name = ? AND user_id = ? AND id = ?)`,
		appName, userID, sessionID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check session existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Query events
	rows, err := s.db.QueryContext(ctx,
		`SELECT event_data FROM events WHERE session_id = ? ORDER BY timestamp`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var eventsList []*events.Event
	for rows.Next() {
		var eventJSON []byte
		if err := rows.Scan(&eventJSON); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Create event and unmarshal
		event := new(events.Event)
		if err := json.Unmarshal(eventJSON, event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %w", err)
		}

		eventsList = append(eventsList, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return &ListEventsResponse{Events: eventsList}, nil
}

// CloseSession finalizes a session in the database
func (s *DatabaseSessionService) CloseSession(ctx context.Context, session *Session) error {
	// No special action needed for database sessions
	return nil
}

// AppendEvent adds an event to a session in the database
func (s *DatabaseSessionService) AppendEvent(
	ctx context.Context,
	session *Session,
	event *events.Event,
) (*events.Event, error) {
	if event.Partial {
		return event, nil
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // rollback if not committed

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Insert event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO events (id, session_id, event_data, timestamp)
		VALUES (?, ?, ?, ?)`,
		event.ID, session.ID, eventJSON, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event: %w", err)
	}

	// Update session's state if needed
	if event.Actions != nil && len(event.Actions.StateDelta) > 0 {
		// Filter out temporary state values
		stateDelta := make(map[string]interface{})
		for key, value := range event.Actions.StateDelta {
			if !strings.HasPrefix(key, TempPrefix) {
				stateDelta[key] = value
			}
		}

		if len(stateDelta) > 0 {
			// Update in-memory session
			session.State.Update(stateDelta)
			session.StateMap = session.State.ToMap()
			session.UpdateTime = time.Now()

			// Update database
			stateJSON, err := json.Marshal(session.StateMap)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal updated state: %w", err)
			}

			_, err = tx.ExecContext(ctx,
				`UPDATE sessions SET state = ?, update_time = ? WHERE id = ?`,
				stateJSON, session.UpdateTime, session.ID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update session state: %w", err)
			}
		}
	}

	// Add the event to the session
	session.Events = append(session.Events, event)

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return event, nil
}
