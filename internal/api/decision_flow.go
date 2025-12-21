package api

type DecisionVerdict string

const (
	VerdictAllow           DecisionVerdict = "allow"
	VerdictDeny            DecisionVerdict = "deny"
	VerdictRequireApproval DecisionVerdict = "require_approval"
)

// TransitionFromDecision maps a policy verdict to the next idempotency status and action.
func TransitionFromDecision(verdict DecisionVerdict) (IdemStatus, NextAction) {
	switch verdict {
	case VerdictDeny:
		return IdemDenied, ActionReturnDenied
	case VerdictRequireApproval:
		return IdemPendingApproval, ActionReturnPending
	case VerdictAllow:
		return IdemIssuing, ActionIssueCredentials
	default:
		return IdemErrored, ActionReturnErrored
	}
}
