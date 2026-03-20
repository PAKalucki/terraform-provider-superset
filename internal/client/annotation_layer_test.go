package client

import (
	"encoding/json"
	"testing"
)

func TestAnnotationLayerUpdateRequestMarshalJSONIncludesEmptyDescriptionForClears(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(AnnotationLayerUpdateRequest{
		Name:               "Deployments",
		IncludeDescription: true,
	})
	if err != nil {
		t.Fatalf("expected update request to marshal, got error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("expected update request JSON to decode, got error: %v", err)
	}

	if _, ok := body["descr"]; !ok {
		t.Fatal("expected descr to be present in update payload")
	}

	descr, ok := body["descr"].(string)
	if !ok {
		t.Fatalf("expected descr to encode as string, got %#v", body["descr"])
	}

	if descr != "" {
		t.Fatalf("expected descr to encode as empty string, got %#v", body["descr"])
	}
}
