package api

import "testing"

func TestPlanAuthorizeActionEvaluate(t *testing.T) {
	idem := IdemState{Status: IdemPendingApproval}

	_, err := PlanAuthorizeAction(idem, nil, nil)
	if err == nil {
		t.Fatalf("expected error when verdict missing")
	}

	verdict := VerdictAllow
	out, err := PlanAuthorizeAction(IdemState{Status: ""}, nil, &verdict)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.NextAction != ActionIssueCredentials || out.IdemStatus != IdemIssuing {
		t.Fatalf("unexpected outcome: %v", out)
	}
}

func TestPlanAuthorizeActionTerminal(t *testing.T) {
	out, err := PlanAuthorizeAction(IdemState{Status: IdemAllowed}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.NextAction != ActionReturnFinal || out.IdemStatus != IdemAllowed {
		t.Fatalf("unexpected outcome: %v", out)
	}
}
