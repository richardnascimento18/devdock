package state

// Store binds file access to paths resolved by the application, never HOME.
type Store struct{ Directory string }

func (s Store) Load() (UIState, error)   { return Load(s.Directory) }
func (s Store) Save(value UIState) error { return Save(s.Directory, value) }
