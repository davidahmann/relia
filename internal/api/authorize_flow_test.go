package api

import "testing"

func TestPlanAuthorizeAction(t *testing.T) {
	_, err := PlanAuthorizeAction(IdemState{Status: IdemPendingApproval}, nil, nil)
	if err == nil {
		t.Fatalf("expected error for missing verdict")
	}

	v := VerdictAllow
	out, err := PlanAuthorizeAction(IdemState{Status: IdemPendingApproval}, nil, &v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.NextAction != ActionIssueCredentials || out.IdemStatus != IdemIssuing {
		t.Fatalf("unexpected outcome: %+v", out)
	}

	out, err = PlanAuthorizeAction(IdemState{Status: IdemAllowed}, nil, nil)
	if err != nil || out.NextAction != ActionReturnFinal {
		t.Fatalf("expected return_final, got %+v err=%v", out, err)
	}

	out, err = PlanAuthorizeAction(IdemState{Status: IdemDenied}, nil, nil)
	if err != nil || out.NextAction != ActionReturnDenied {
		t.Fatalf("expected return_denied, got %+v err=%v", out, err)
	}

	out, err = PlanAuthorizeAction(IdemState{Status: IdemErrored}, nil, nil)
	if err != nil || out.NextAction != ActionReturnErrored {
		t.Fatalf("expected return_errored, got %+v err=%v", out, err)
	}

	out, err = PlanAuthorizeAction(IdemState{Status: IdemApprovedReady}, nil, nil)
	if err != nil || out.NextAction != ActionIssueCredentials {
		t.Fatalf("expected issue_credentials, got %+v err=%v", out, err)
	}

	out, err = PlanAuthorizeAction(IdemState{Status: IdemPendingApproval}, &ApprovalState{Status: ApprovalPending}, nil)
	if err != nil || out.NextAction != ActionReturnPending {
		t.Fatalf("expected return_pending, got %+v err=%v", out, err)
	}
}
