package migrations

import (
	"gorm.io/gorm"
)

// AddIndexes adds indexes to the database to improve query performance
func AddIndexes(db *gorm.DB) error {
	// Add indexes to the users table
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_is_identified ON users (\"isIdentified\")").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_is_client ON users (\"isClient\")").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_combined_types ON users (\"isIdentified\", \"isClient\")").Error; err != nil {
		return err
	}

	// Add indexes to the sessions table
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_session_start ON sessions (\"sessionStart\")").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_profession_id ON sessions (profession_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_product_id ON sessions (product_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_funnel_id ON sessions (funnel_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_sessions_is_active ON sessions (\"isActive\")").Error; err != nil {
		return err
	}

	// Add indexes to the events table
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_event_time ON events (event_time)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_session_id ON events (session_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_profession_id ON events (profession_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_events_funnel_id ON events (funnel_id)").Error; err != nil {
		return err
	}

	return nil
}
