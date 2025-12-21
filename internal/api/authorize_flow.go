package api

import "fmt"

type AuthorizeOutcome struct {
	NextAction NextAction
	IdemStatus IdemStatus
}

// PlanAuthorizeAction determines the next action and target idempotency status.
// If evaluation is required, verdict must be provided.
func PlanAuthorizeAction(idem IdemState, approval *ApprovalState, verdict *DecisionVerdict) (AuthorizeOutcome, error) {
	action := DetermineNextAction(idem, approval)

	switch action {
	case ActionEvaluatePolicy:
		if verdict == nil {
			return AuthorizeOutcome{}, fmt.Errorf("missing decision verdict")
		}
		status, next := TransitionFromDecision(*verdict)
		return AuthorizeOutcome{NextAction: next, IdemStatus: status}, nil
	case ActionReturnFinal:
		return AuthorizeOutcome{NextAction: ActionReturnFinal, IdemStatus: IdemAllowed}, nil
	case ActionReturnDenied:
		return AuthorizeOutcome{NextAction: ActionReturnDenied, IdemStatus: IdemDenied}, nil
	case ActionReturnErrored:
		return AuthorizeOutcome{NextAction: ActionReturnErrored, IdemStatus: IdemErrored}, nil
	case ActionIssueCredentials:
		return AuthorizeOutcome{NextAction: ActionIssueCredentials, IdemStatus: IdemIssuing}, nil
	case ActionReturnPending:
		return AuthorizeOutcome{NextAction: ActionReturnPending, IdemStatus: IdemPendingApproval}, nil
	default:
		return AuthorizeOutcome{}, fmt.Errorf("unsupported next action")
	}
}
