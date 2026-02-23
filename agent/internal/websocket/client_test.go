package websocket

import (
	"encoding/json"
	"testing"
)

type testMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func TestClientSendRefactor(t *testing.T) {
	msg := testMessage{Type: "test", Data: "hello"}

	dataBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	if len(dataBytes) == 0 {
		t.Fatal("Expected non-empty JSON")
	}

	var parsed testMessage
	err = json.Unmarshal(dataBytes, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.Type != "test" {
		t.Errorf("Expected Type='test', got '%s'", parsed.Type)
	}
	if parsed.Data != "hello" {
		t.Errorf("Expected Data='hello', got '%s'", parsed.Data)
	}
}
