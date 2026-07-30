package bench

import "testing"

func TestValidateResponse_StatusMatch(t *testing.T) {
	cfg := Config{ExpectStatus: 200}
	if msg := validateResponse(cfg, 200, nil); msg != "" {
		t.Fatalf("expected no error, got %q", msg)
	}
}

func TestValidateResponse_StatusMismatch(t *testing.T) {
	cfg := Config{ExpectStatus: 200}
	msg := validateResponse(cfg, 404, nil)
	if msg == "" {
		t.Fatal("expected error for status mismatch")
	}
	if msg != "status 404, expected 200" {
		t.Fatalf("unexpected msg: %q", msg)
	}
}

func TestValidateResponse_BodyContains(t *testing.T) {
	cfg := Config{ExpectBody: "ok"}
	if msg := validateResponse(cfg, 200, []byte(`{"status":"ok"}`)); msg != "" {
		t.Fatalf("expected no error, got %q", msg)
	}
}

func TestValidateResponse_BodyMissing(t *testing.T) {
	cfg := Config{ExpectBody: "secret"}
	msg := validateResponse(cfg, 200, []byte(`hello world`))
	if msg == "" {
		t.Fatal("expected error for missing body substring")
	}
}

func TestValidateResponse_NoValidation(t *testing.T) {
	cfg := Config{}
	if msg := validateResponse(cfg, 500, []byte("error")); msg != "" {
		t.Fatalf("expected no error when no validation configured, got %q", msg)
	}
}

func TestValidateResponse_BothChecks(t *testing.T) {
	cfg := Config{ExpectStatus: 200, ExpectBody: "ok"}
	// Status fails first
	msg := validateResponse(cfg, 500, []byte("ok"))
	if msg == "" {
		t.Fatal("expected status mismatch error")
	}
	// Status ok but body fails
	msg = validateResponse(cfg, 200, []byte("fail"))
	if msg == "" {
		t.Fatal("expected body mismatch error")
	}
	// Both pass
	msg = validateResponse(cfg, 200, []byte("ok"))
	if msg != "" {
		t.Fatalf("expected no error, got %q", msg)
	}
}
