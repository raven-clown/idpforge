package apiclient

import (
	"reflect"
	"testing"
)

func TestFilterFields(t *testing.T) {
	full := map[string]interface{}{
		"id":            "u1",
		"username":      "alice",
		"email":         "alice@example.com",
		"password_hash": "should-never-leak",
		"mfa_secret":    "should-never-leak",
	}

	got := FilterFields(full, []string{"id", "username", "email"})
	want := map[string]interface{}{
		"id":       "u1",
		"username": "alice",
		"email":    "alice@example.com",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("FilterFields() = %v, want %v", got, want)
	}

	if _, leaked := got["password_hash"]; leaked {
		t.Error("password_hash leaked through FilterFields despite not being in allowed_fields")
	}
	if _, leaked := got["mfa_secret"]; leaked {
		t.Error("mfa_secret leaked through FilterFields despite not being in allowed_fields")
	}
}

func TestFilterFieldsMissingField(t *testing.T) {
	full := map[string]interface{}{"id": "u1"}
	got := FilterFields(full, []string{"id", "does_not_exist"})
	if len(got) != 1 {
		t.Errorf("expected only present fields to be included, got %v", got)
	}
}

func TestHashKeyDeterministicAndDistinct(t *testing.T) {
	a := hashKey("secret-a")
	b := hashKey("secret-a")
	c := hashKey("secret-b")

	if a != b {
		t.Error("hashKey should be deterministic for the same input")
	}
	if a == c {
		t.Error("hashKey should differ for different inputs")
	}
}
