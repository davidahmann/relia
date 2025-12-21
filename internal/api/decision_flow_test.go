package api

import "testing"

func TestTransitionFromDecision(t *testing.T) {
	cases := []struct {
		verdict DecisionVerdict
		status  IdemStatus
		action  NextAction
	}{
		{VerdictAllow, IdemIssuing, ActionIssueCredentials},
		{VerdictDeny, IdemDenied, ActionReturnDenied},
		{VerdictRequireApproval, IdemPendingApproval, ActionReturnPending},
		{"unknown", IdemErrored, ActionReturnErrored},
	}

	for _, tc := range cases {
		status, action := TransitionFromDecision(tc.verdict)
		if status != tc.status || action != tc.action {
			t.Fatalf("verdict %s expected %s/%s got %s/%s", tc.verdict, tc.status, tc.action, status, action)
		}
	}
}
