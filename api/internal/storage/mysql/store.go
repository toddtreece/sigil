package mysql

type Store struct {
	dsn string
}

func NewStore(dsn string) *Store {
	return &Store{dsn: dsn}
}

func (s *Store) DSN() string {
	return s.dsn
}
