package services

type Service struct{}

func (s *Service) Ping() string {
	return "Test"
}
