package api

import (
	"sync"

	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/pkg/types"
)

type ArtifactStore struct {
	mu        sync.Mutex
	contexts  map[string]types.ContextRecord
	decisions map[string]types.DecisionRecord
	receipts  map[string]ledger.StoredReceipt
	policies  map[string][]byte
}

func NewArtifactStore() *ArtifactStore {
	return &ArtifactStore{
		contexts:  make(map[string]types.ContextRecord),
		decisions: make(map[string]types.DecisionRecord),
		receipts:  make(map[string]ledger.StoredReceipt),
		policies:  make(map[string][]byte),
	}
}

func (s *ArtifactStore) PutContext(record types.ContextRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contexts[record.ContextID] = record
}

func (s *ArtifactStore) GetContext(contextID string) (types.ContextRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.contexts[contextID]
	return record, ok
}

func (s *ArtifactStore) PutDecision(record types.DecisionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decisions[record.DecisionID] = record
}

func (s *ArtifactStore) GetDecision(decisionID string) (types.DecisionRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.decisions[decisionID]
	return record, ok
}

func (s *ArtifactStore) PutReceipt(receipt ledger.StoredReceipt) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts[receipt.ReceiptID] = receipt
}

func (s *ArtifactStore) GetReceipt(receiptID string) (ledger.StoredReceipt, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.receipts[receiptID]
	return record, ok
}

func (s *ArtifactStore) PutPolicy(hash string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[hash] = data
}

func (s *ArtifactStore) GetPolicy(hash string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.policies[hash]
	return data, ok
}
