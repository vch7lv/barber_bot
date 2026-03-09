package telegram

import "sync"

type barberState struct {
	Step string
	Data string
}

type barberStateStore struct {
	mu   sync.Mutex
	data map[int64]*barberState
}

func newBarberStateStore() *barberStateStore {
	return &barberStateStore{data: make(map[int64]*barberState)}
}

func (s *barberStateStore) Get(chatID int64) *barberState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[chatID]
}

func (s *barberStateStore) Set(chatID int64, step, data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if step == "" {
		delete(s.data, chatID)
		return
	}
	s.data[chatID] = &barberState{Step: step, Data: data}
}

func (s *barberStateStore) Clear(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, chatID)
}

// barberClientModeStore — режим «как клиент» для барбера (тестирование одним аккаунтом).
type barberClientModeStore struct {
	mu   sync.Mutex
	data map[int64]bool
}

func newBarberClientModeStore() barberClientModeStore {
	return barberClientModeStore{data: make(map[int64]bool)}
}

func (s *barberClientModeStore) Get(chatID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[chatID]
}

func (s *barberClientModeStore) Set(chatID int64, on bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if on {
		s.data[chatID] = true
	} else {
		delete(s.data, chatID)
	}
}
