package fault

import (
	"errors"
	"testing"
)

func TestTypedErrorWrapsCause(t *testing.T) {
	cause := errors.New("dial refused")
	err := New("tcp", "dial", KindUnavailable, true, cause)
	if !errors.Is(err, cause) || !IsKind(err, KindUnavailable) || IsKind(err, KindProtocol) {
		t.Fatalf("unexpected typed error behavior: %v", err)
	}
	var typed *Error
	if !errors.As(err, &typed) || !typed.Retryable || typed.Component != "tcp" {
		t.Fatalf("unexpected fields: %#v", typed)
	}
}
