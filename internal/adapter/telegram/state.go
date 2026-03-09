package telegram

import "sync"

// BookingState — состояние записи на визит (выбранные услуги).
type BookingState struct {
	ServiceIDs []int64
}

type stateStore struct {
	mu   sync.Mutex
	data map[int64]*BookingState
}

func newStateStore() *stateStore {
	return &stateStore{data: make(map[int64]*BookingState)}
}

func (s *stateStore) Get(chatID int64) *BookingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[chatID]
}

func (s *stateStore) Set(chatID int64, state *BookingState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state == nil {
		delete(s.data, chatID)
		return
	}
	s.data[chatID] = state
}

func (s *stateStore) AddService(chatID int64, serviceID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[chatID] == nil {
		s.data[chatID] = &BookingState{ServiceIDs: []int64{}}
	}
	for _, id := range s.data[chatID].ServiceIDs {
		if id == serviceID {
			return
		}
	}
	s.data[chatID].ServiceIDs = append(s.data[chatID].ServiceIDs, serviceID)
}

// ToggleService добавляет услугу, если её ещё нет, иначе убирает. Возвращает true, если услуга теперь выбрана.
func (s *stateStore) ToggleService(chatID int64, serviceID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data[chatID] == nil {
		s.data[chatID] = &BookingState{ServiceIDs: []int64{serviceID}}
		return true
	}
	ids := s.data[chatID].ServiceIDs
	for i, id := range ids {
		if id == serviceID {
			s.data[chatID].ServiceIDs = append(ids[:i], ids[i+1:]...)
			return false
		}
	}
	s.data[chatID].ServiceIDs = append(ids, serviceID)
	return true
}
