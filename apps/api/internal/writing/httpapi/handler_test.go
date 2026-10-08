package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWithoutModelMetadataRemovesNestedProviderDetails(t *testing.T) {
	value := map[string]any{
		"model":  "private-model",
		"report": json.RawMessage(`{"model_version":"private-version","summary":"ok"}`),
	}
	b, err := json.Marshal(withoutModelMetadata(value))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Contains(got, "private-model") || strings.Contains(got, "private-version") || !strings.Contains(got, `"summary":"ok"`) {
		t.Fatalf("unexpected public response: %s", got)
	}
}
