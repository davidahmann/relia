package api

import "testing"

func TestDetermineNextActionTerminalStates(t *testing.T) {
	cases := []struct {
		status IdemStatus
		action NextAction
	}{
		{IdemAllowed, ActionReturnFinal},
		{IdemDenied, ActionReturnDenied},
		{IdemErrored, ActionReturnErrored},
	}

	for _, tc := range cases {
		got := DetermineNextAction(IdemState{Status: tc.status}, nil)
		if got != tc.action {
			t.Fatalf("status %s expected %s got %s", tc.status, tc.action, got)
		}
	}
}

func TestDetermineNextActionPendingApproval(t *testing.T) {
	idem := IdemState{Status: IdemPendingApproval}

	if got := DetermineNextAction(idem, nil); got != ActionEvaluatePolicy {
		t.Fatalf("expected evaluate_policy, got %s", got)
	}

	if got := DetermineNextAction(idem, &ApprovalState{Status: ApprovalPending}); got != ActionReturnPending {
		t.Fatalf("expected return_pending_approval, got %s", got)
	}

	if got := DetermineNextAction(idem, &ApprovalState{Status: ApprovalDenied}); got != ActionReturnDenied {
		t.Fatalf("expected return_denied, got %s", got)
	}

	if got := DetermineNextAction(idem, &ApprovalState{Status: ApprovalApproved}); got != ActionIssueCredentials {
		t.Fatalf("expected issue_credentials, got %s", got)
	}
}

func TestDetermineNextActionApprovedReady(t *testing.T) {
	got := DetermineNextAction(IdemState{Status: IdemApprovedReady}, nil)
	if got != ActionIssueCredentials {
		t.Fatalf("expected issue_credentials, got %s", got)
	}
}
