package api

import "sync"

type IdemRecord struct {
	IdemKey    string
	Status     IdemStatus
	ApprovalID string
	ReceiptID  string
	ContextID  string
	DecisionID string
}

type InMemoryIdemStore struct {
	mu    sync.Mutex
	items map[string]IdemRecord
}

func NewInMemoryIdemStore() *InMemoryIdemStore {
	return &InMemoryIdemStore{items: make(map[string]IdemRecord)}
}

func (s *InMemoryIdemStore) Get(idemKey string) (IdemRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.items[idemKey]
	return rec, ok
}

func (s *InMemoryIdemStore) Put(record IdemRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[record.IdemKey] = record
}
