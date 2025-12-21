package ledger

type Store interface {
	WithTx(fn func(Tx) error) error

	PutKey(key KeyRecord) error
	GetKey(keyID string) (KeyRecord, bool)

	PutSlackOutbox(rec SlackOutboxRecord) error
	GetSlackOutbox(notificationID string) (SlackOutboxRecord, bool)
	ListSlackOutboxDue(now string, limit int) ([]SlackOutboxRecord, error)

	PutPolicyVersion(policy PolicyVersionRecord) error
	GetPolicyVersion(policyHash string) (PolicyVersionRecord, bool)

	PutContext(ctx ContextRecord) error
	GetContext(contextID string) (ContextRecord, bool)

	PutDecision(decision DecisionRecord) error
	GetDecision(decisionID string) (DecisionRecord, bool)

	PutReceipt(receipt ReceiptRecord) error
	GetReceipt(receiptID string) (ReceiptRecord, bool)

	PutApproval(approval ApprovalRecord) error
	GetApproval(approvalID string) (ApprovalRecord, bool)
	GetApprovalByIdemKey(idemKey string) (ApprovalRecord, bool)

	PutIdempotencyKey(key IdempotencyKey) error
	GetIdempotencyKey(idemKey string) (IdempotencyKey, bool)
}

type Tx interface {
	PutKey(key KeyRecord) error
	GetKey(keyID string) (KeyRecord, bool)

	PutSlackOutbox(rec SlackOutboxRecord) error
	GetSlackOutbox(notificationID string) (SlackOutboxRecord, bool)

	PutPolicyVersion(policy PolicyVersionRecord) error
	GetPolicyVersion(policyHash string) (PolicyVersionRecord, bool)

	PutContext(ctx ContextRecord) error
	GetContext(contextID string) (ContextRecord, bool)

	PutDecision(decision DecisionRecord) error
	GetDecision(decisionID string) (DecisionRecord, bool)

	PutReceipt(receipt ReceiptRecord) error
	GetReceipt(receiptID string) (ReceiptRecord, bool)

	PutApproval(approval ApprovalRecord) error
	GetApproval(approvalID string) (ApprovalRecord, bool)
	GetApprovalByIdemKey(idemKey string) (ApprovalRecord, bool)

	PutIdempotencyKey(key IdempotencyKey) error
	GetIdempotencyKey(idemKey string) (IdempotencyKey, bool)
}

type PolicyVersionRecord struct {
	PolicyHash    string
	PolicyID      string
	PolicyVersion string
	PolicyYAML    string
	CreatedAt     string
}

type KeyRecord struct {
	KeyID     string
	PublicKey []byte
	CreatedAt string
	RotatedAt *string
}

type SlackOutboxRecord struct {
	NotificationID string
	ApprovalID     string
	Channel        string
	MessageJSON    []byte
	Status         string // pending | sent
	AttemptCount   int
	NextAttemptAt  string
	LastError      *string
	SentAt         *string
	CreatedAt      string
	UpdatedAt      string
}

type ContextRecord struct {
	ContextID string
	BodyJSON  []byte
	CreatedAt string
}

type DecisionRecord struct {
	DecisionID string
	ContextID  string
	PolicyHash string
	Verdict    string
	BodyJSON   []byte
	CreatedAt  string
}

type ReceiptRecord struct {
	ReceiptID           string
	IdemKey             string
	CreatedAt           string
	SupersedesReceiptID *string
	ContextID           string
	DecisionID          string
	PolicyHash          string
	ApprovalID          *string
	OutcomeStatus       string
	Final               bool
	ExpiresAt           *string
	BodyJSON            []byte
	BodyDigest          string
	KeyID               string
	Sig                 []byte
}

type ApprovalRecord struct {
	ApprovalID   string
	IdemKey      string
	Status       string
	SlackChannel *string
	SlackMsgTS   *string
	ApprovedBy   *string
	ApprovedAt   *string
	CreatedAt    string
	UpdatedAt    string
}

type IdempotencyKey struct {
	IdemKey         string
	Status          string
	ApprovalID      *string
	LatestReceiptID *string
	FinalReceiptID  *string
	CreatedAt       string
	UpdatedAt       string
	TTLExpiresAt    *string
}
