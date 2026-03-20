package client

import (
	"encoding/json"
	"testing"
)

func TestUserUpdateRequestMarshalJSONOmitsPasswordWhenUnset(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(UserUpdateRequest{
		Username:  "analyst",
		FirstName: "Analytics",
		LastName:  "User",
		Email:     "analyst@example.com",
		Active:    true,
		Roles:     []int64{3},
		Groups:    []int64{},
	})
	if err != nil {
		t.Fatalf("expected update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected update request JSON to decode, got error: %v", err)
	}

	if _, ok := body["password"]; ok {
		t.Fatal("expected password to be omitted from update payload when unset")
	}
}
