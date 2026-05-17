package flags

// StubProvider is a fake FlagProvider for use in tests.
// It lets each test explicitly control what flags return,
// without any network connection to LaunchDarkly.
type StubProvider struct {
	Bools map[string]bool
	Ints  map[string]int
}

func (s *StubProvider) BoolVariation(key string, defaultVal bool) bool {
	if val, ok := s.Bools[key]; ok {
		return val
	}
	return defaultVal
}

func (s *StubProvider) IntVariation(key string, defaultVal int) int {
	if val, ok := s.Ints[key]; ok {
		return val
	}
	return defaultVal
}

func (s *StubProvider) Close() {} // no-op
