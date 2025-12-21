package ledger

import "sync"

type InMemoryStore struct {
	mu sync.Mutex

	keys      map[string]KeyRecord
	outbox    map[string]SlackOutboxRecord
	policies  map[string]PolicyVersionRecord
	contexts  map[string]ContextRecord
	decisions map[string]DecisionRecord
	receipts  map[string]ReceiptRecord
	approvals map[string]ApprovalRecord
	idemKeys  map[string]IdempotencyKey
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		keys:      make(map[string]KeyRecord),
		outbox:    make(map[string]SlackOutboxRecord),
		policies:  make(map[string]PolicyVersionRecord),
		contexts:  make(map[string]ContextRecord),
		decisions: make(map[string]DecisionRecord),
		receipts:  make(map[string]ReceiptRecord),
		approvals: make(map[string]ApprovalRecord),
		idemKeys:  make(map[string]IdempotencyKey),
	}
}

func (s *InMemoryStore) WithTx(fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn((*memTx)(s))
}

type memTx InMemoryStore

func (s *InMemoryStore) PutKey(key KeyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key.KeyID] = key
	return nil
}

func (s *InMemoryStore) GetKey(keyID string) (KeyRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[keyID]
	return key, ok
}

func (s *InMemoryStore) PutSlackOutbox(rec SlackOutboxRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbox[rec.NotificationID] = rec
	return nil
}

func (s *InMemoryStore) GetSlackOutbox(notificationID string) (SlackOutboxRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.outbox[notificationID]
	return rec, ok
}

func (s *InMemoryStore) ListSlackOutboxDue(now string, limit int) ([]SlackOutboxRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []SlackOutboxRecord{}
	for _, rec := range s.outbox {
		if rec.Status != "pending" {
			continue
		}
		if rec.NextAttemptAt > now {
			continue
		}
		out = append(out, rec)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *InMemoryStore) PutPolicyVersion(policy PolicyVersionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.PolicyHash] = policy
	return nil
}

func (s *InMemoryStore) GetPolicyVersion(policyHash string) (PolicyVersionRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	policy, ok := s.policies[policyHash]
	return policy, ok
}

func (s *InMemoryStore) PutContext(ctx ContextRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contexts[ctx.ContextID] = ctx
	return nil
}

func (s *InMemoryStore) GetContext(contextID string) (ContextRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx, ok := s.contexts[contextID]
	return ctx, ok
}

func (s *InMemoryStore) PutDecision(decision DecisionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.decisions[decision.DecisionID] = decision
	return nil
}

func (s *InMemoryStore) GetDecision(decisionID string) (DecisionRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	decision, ok := s.decisions[decisionID]
	return decision, ok
}

func (s *InMemoryStore) PutReceipt(receipt ReceiptRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts[receipt.ReceiptID] = receipt
	return nil
}

func (s *InMemoryStore) GetReceipt(receiptID string) (ReceiptRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	receipt, ok := s.receipts[receiptID]
	return receipt, ok
}

func (s *InMemoryStore) PutApproval(approval ApprovalRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[approval.ApprovalID] = approval
	return nil
}

func (s *InMemoryStore) GetApproval(approvalID string) (ApprovalRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, ok := s.approvals[approvalID]
	return approval, ok
}

func (s *InMemoryStore) GetApprovalByIdemKey(idemKey string) (ApprovalRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, approval := range s.approvals {
		if approval.IdemKey == idemKey {
			return approval, true
		}
	}
	return ApprovalRecord{}, false
}

func (s *InMemoryStore) PutIdempotencyKey(key IdempotencyKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idemKeys[key.IdemKey] = key
	return nil
}

func (s *InMemoryStore) GetIdempotencyKey(idemKey string) (IdempotencyKey, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.idemKeys[idemKey]
	return key, ok
}

func (t *memTx) PutPolicyVersion(policy PolicyVersionRecord) error {
	(*InMemoryStore)(t).policies[policy.PolicyHash] = policy
	return nil
}

func (t *memTx) PutKey(key KeyRecord) error {
	(*InMemoryStore)(t).keys[key.KeyID] = key
	return nil
}

func (t *memTx) GetKey(keyID string) (KeyRecord, bool) {
	key, ok := (*InMemoryStore)(t).keys[keyID]
	return key, ok
}

func (t *memTx) PutSlackOutbox(rec SlackOutboxRecord) error {
	(*InMemoryStore)(t).outbox[rec.NotificationID] = rec
	return nil
}

func (t *memTx) GetSlackOutbox(notificationID string) (SlackOutboxRecord, bool) {
	rec, ok := (*InMemoryStore)(t).outbox[notificationID]
	return rec, ok
}

func (t *memTx) GetPolicyVersion(policyHash string) (PolicyVersionRecord, bool) {
	policy, ok := (*InMemoryStore)(t).policies[policyHash]
	return policy, ok
}

func (t *memTx) PutContext(ctx ContextRecord) error {
	(*InMemoryStore)(t).contexts[ctx.ContextID] = ctx
	return nil
}

func (t *memTx) GetContext(contextID string) (ContextRecord, bool) {
	ctx, ok := (*InMemoryStore)(t).contexts[contextID]
	return ctx, ok
}

func (t *memTx) PutDecision(decision DecisionRecord) error {
	(*InMemoryStore)(t).decisions[decision.DecisionID] = decision
	return nil
}

func (t *memTx) GetDecision(decisionID string) (DecisionRecord, bool) {
	decision, ok := (*InMemoryStore)(t).decisions[decisionID]
	return decision, ok
}

func (t *memTx) PutReceipt(receipt ReceiptRecord) error {
	(*InMemoryStore)(t).receipts[receipt.ReceiptID] = receipt
	return nil
}

func (t *memTx) GetReceipt(receiptID string) (ReceiptRecord, bool) {
	receipt, ok := (*InMemoryStore)(t).receipts[receiptID]
	return receipt, ok
}

func (t *memTx) PutApproval(approval ApprovalRecord) error {
	(*InMemoryStore)(t).approvals[approval.ApprovalID] = approval
	return nil
}

func (t *memTx) GetApproval(approvalID string) (ApprovalRecord, bool) {
	approval, ok := (*InMemoryStore)(t).approvals[approvalID]
	return approval, ok
}

func (t *memTx) GetApprovalByIdemKey(idemKey string) (ApprovalRecord, bool) {
	for _, approval := range (*InMemoryStore)(t).approvals {
		if approval.IdemKey == idemKey {
			return approval, true
		}
	}
	return ApprovalRecord{}, false
}

func (t *memTx) PutIdempotencyKey(key IdempotencyKey) error {
	(*InMemoryStore)(t).idemKeys[key.IdemKey] = key
	return nil
}

func (t *memTx) GetIdempotencyKey(idemKey string) (IdempotencyKey, bool) {
	key, ok := (*InMemoryStore)(t).idemKeys[idemKey]
	return key, ok
}
