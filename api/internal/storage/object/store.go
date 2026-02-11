package object

type Store struct {
	endpoint string
	bucket   string
}

func NewStore(endpoint string, bucket string) *Store {
	return &Store{endpoint: endpoint, bucket: bucket}
}

func (s *Store) Endpoint() string {
	return s.endpoint
}

func (s *Store) Bucket() string {
	return s.bucket
}
