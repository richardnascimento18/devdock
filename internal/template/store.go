package template

// Store binds file access to paths resolved by the application, never HOME.
type Store struct{ Directory string }

func (s Store) Load() ([]Template, error)   { return Load(s.Directory) }
func (s Store) Save(value []Template) error { return Save(s.Directory, value) }
