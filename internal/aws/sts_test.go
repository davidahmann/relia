package aws

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

func TestDevBroker(t *testing.T) {
	creds, err := (DevBroker{}).AssumeRoleWithWebIdentity(AssumeRoleInput{TTLSeconds: 60})
	if err != nil {
		t.Fatalf("assume: %v", err)
	}
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" || creds.SessionToken == "" {
		t.Fatalf("expected placeholder credentials")
	}
	if creds.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expected expiry in the future")
	}
}

func TestSTSBrokerValidationAndHelpers(t *testing.T) {
	b := &STSBroker{}
	if _, err := b.AssumeRoleWithWebIdentity(AssumeRoleInput{}); err == nil {
		t.Fatalf("expected error for missing role arn")
	}
	if _, err := b.AssumeRoleWithWebIdentity(AssumeRoleInput{RoleARN: "arn"}); err == nil {
		t.Fatalf("expected error for missing token")
	}
	if _, err := b.AssumeRoleWithWebIdentity(AssumeRoleInput{RoleARN: "arn", WebIdentityToken: "t", TTLSeconds: 0}); err == nil {
		t.Fatalf("expected error for ttl")
	}

	if v := int32Ptr(3); v == nil || *v != 3 {
		t.Fatalf("int32Ptr mismatch")
	}
	if got := strOrEmpty(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	s := "x"
	if got := strOrEmpty(&s); got != "x" {
		t.Fatalf("expected x, got %q", got)
	}
}

func TestNewSTSBrokerBestEffort(t *testing.T) {
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	// Some environments may not have AWS config available; execute the codepath but don't hard-fail.
	b, err := NewSTSBroker("us-east-1")
	if err != nil {
		t.Skipf("aws config not available: %v", err)
	}
	_ = b

	// Ensure no accidental shared env leakage.
	if os.Getenv("AWS_EC2_METADATA_DISABLED") == "" {
		t.Fatalf("expected env to be set")
	}
}

func TestNewSTSBrokerMissingRegion(t *testing.T) {
	if _, err := NewSTSBroker(" "); err == nil {
		t.Fatalf("expected error")
	}
}

type fakeSTSClient struct {
	out *sts.AssumeRoleWithWebIdentityOutput
	err error
}

func (f fakeSTSClient) AssumeRoleWithWebIdentity(_ context.Context, _ *sts.AssumeRoleWithWebIdentityInput, _ ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	return f.out, f.err
}

func TestSTSBrokerAssumeRoleSuccess(t *testing.T) {
	exp := time.Now().UTC().Add(10 * time.Minute)
	ak := "AKIA"
	sk := "SECRET"
	st := "TOKEN"
	b := &STSBroker{
		client: fakeSTSClient{
			out: &sts.AssumeRoleWithWebIdentityOutput{
				Credentials: &types.Credentials{
					AccessKeyId:     &ak,
					SecretAccessKey: &sk,
					SessionToken:    &st,
					Expiration:      &exp,
				},
			},
		},
	}

	creds, err := b.AssumeRoleWithWebIdentity(AssumeRoleInput{RoleARN: "arn", WebIdentityToken: "jwt", TTLSeconds: 900})
	if err != nil {
		t.Fatalf("assume: %v", err)
	}
	if creds.AccessKeyID != ak || creds.SecretAccessKey != sk || creds.SessionToken != st {
		t.Fatalf("unexpected creds: %+v", creds)
	}
	if !creds.ExpiresAt.Equal(exp) {
		t.Fatalf("expected expiry %v got %v", exp, creds.ExpiresAt)
	}
}

func TestSTSBrokerAssumeRoleMissingCredentials(t *testing.T) {
	b := &STSBroker{client: fakeSTSClient{out: &sts.AssumeRoleWithWebIdentityOutput{Credentials: nil}}}
	if _, err := b.AssumeRoleWithWebIdentity(AssumeRoleInput{RoleARN: "arn", WebIdentityToken: "jwt", TTLSeconds: 1}); err == nil {
		t.Fatalf("expected error")
	}
}
