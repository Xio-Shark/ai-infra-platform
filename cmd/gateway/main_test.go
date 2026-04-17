package main

import "testing"

func TestParseBackendsJSON_DefaultWeight(t *testing.T) {
	backends, err := parseBackendsJSON(`[{"id":"vllm","endpoint":"http://127.0.0.1:8000","models":["qwen2.5-1.5b"]}]`)
	if err != nil {
		t.Fatalf("parseBackendsJSON returned error: %v", err)
	}
	if len(backends) != 1 {
		t.Fatalf("expected 1 backend, got %d", len(backends))
	}
	if backends[0].Weight != 1 {
		t.Fatalf("expected default weight 1, got %d", backends[0].Weight)
	}
}

func TestParseBackendsJSON_RejectsMissingEndpoint(t *testing.T) {
	_, err := parseBackendsJSON(`[{"id":"vllm","models":["qwen2.5-1.5b"]}]`)
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestParseRateLimitsJSON(t *testing.T) {
	limits, err := parseRateLimitsJSON(`{"qwen2.5-1.5b":200,"gpt-4o-mini":80}`)
	if err != nil {
		t.Fatalf("parseRateLimitsJSON returned error: %v", err)
	}
	if limits["qwen2.5-1.5b"] != 200 {
		t.Fatalf("expected qwen2.5-1.5b limit 200, got %d", limits["qwen2.5-1.5b"])
	}
	if limits["gpt-4o-mini"] != 80 {
		t.Fatalf("expected gpt-4o-mini limit 80, got %d", limits["gpt-4o-mini"])
	}
}
