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

// AutoSubmitChoice returns whether choice questions auto-submit on selection.
func (s *Store) AutoSubmitChoice() (bool, error) {
	var v bool
	err := s.db.Get(&v, "SELECT auto_submit_choice FROM settings WHERE id = 1")
	return v, err
}

// SetAutoSubmitChoice toggles whether choice questions auto-submit on selection.
func (s *Store) SetAutoSubmitChoice(v bool) error {
	_, err := s.db.Exec("UPDATE settings SET auto_submit_choice = ? WHERE id = 1", v)
	return err
}
