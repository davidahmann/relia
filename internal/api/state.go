package api

type IdemStatus string

type ApprovalStatus string

type NextAction string

// #nosec G101 -- action labels are not credentials.
const (
	IdemPendingApproval IdemStatus = "pending_approval"
	IdemApprovedReady   IdemStatus = "approved_ready"
	IdemIssuing         IdemStatus = "issuing"
	IdemAllowed         IdemStatus = "allowed"
	IdemDenied          IdemStatus = "denied"
	IdemErrored         IdemStatus = "errored"
)

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalDenied   ApprovalStatus = "denied"
)

const (
	ActionReturnFinal      NextAction = "return_final"
	ActionReturnPending    NextAction = "return_pending_approval"
	ActionIssueCredentials NextAction = "issue_credentials" // #nosec G101 -- label only
	ActionEvaluatePolicy   NextAction = "evaluate_policy"
	ActionReturnDenied     NextAction = "return_denied"
	ActionReturnErrored    NextAction = "return_errored"
)

type IdemState struct {
	Status     IdemStatus
	ApprovalID string
}

type ApprovalState struct {
	Status ApprovalStatus
}

// DetermineNextAction maps idempotency + approval state to the next step.
func DetermineNextAction(idem IdemState, approval *ApprovalState) NextAction {
	switch idem.Status {
	case IdemAllowed:
		return ActionReturnFinal
	case IdemDenied:
		return ActionReturnDenied
	case IdemErrored:
		return ActionReturnErrored
	case IdemApprovedReady:
		return ActionIssueCredentials
	case IdemPendingApproval:
		if approval == nil {
			return ActionEvaluatePolicy
		}
		switch approval.Status {
		case ApprovalPending:
			return ActionReturnPending
		case ApprovalDenied:
			return ActionReturnDenied
		case ApprovalApproved:
			return ActionIssueCredentials
		default:
			return ActionEvaluatePolicy
		}
	default:
		return ActionEvaluatePolicy
	}
}
