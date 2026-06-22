package db

// GetSettings returns the singleton settings row.
func (s *Store) GetSettings() (Settings, error) {
	var settings Settings
	err := s.db.Get(&settings, "SELECT * FROM settings WHERE id = 1")
	return settings, err
}

// SetCurrentWorkspace persists the given workspace name as the current workspace.
func (s *Store) SetCurrentWorkspace(name string) error {
	_, err := s.db.Exec("UPDATE settings SET last_active_workspace = ? WHERE id = 1", name)
	return err
}

// CurrentWorkspace returns the name of the current workspace, or empty string if none.
func (s *Store) CurrentWorkspace() (string, error) {
	var name string
	err := s.db.Get(&name, "SELECT last_active_workspace FROM settings WHERE id = 1")
	if err != nil {
		return "", err
	}
	return name, nil
}
