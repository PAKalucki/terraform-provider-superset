package client

import (
	"encoding/json"
	"testing"
)

func TestSavedQueryUpdateRequestMarshalJSONIncludesNullsForClears(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(SavedQueryUpdateRequest{
		DatabaseID:         11,
		Label:              "Orders",
		SQL:                "select 1",
		IncludeDescription: true,
		IncludeCatalog:     true,
		IncludeSchema:      true,
	})
	if err != nil {
		t.Fatalf("expected update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected update request JSON to decode, got error: %v", err)
	}

	for _, key := range []string{"description", "catalog", "schema"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected %q to be present in update payload", key)
		}

		if body[key] != nil {
			t.Fatalf("expected %q to encode as null, got %#v", key, body[key])
		}
	}
}
