package collect

import "testing"

func TestQuotaFromOpenusageIncludesManualResetExpiry(t *testing.T) {
	const expiresAt = "2026-09-20T22:56:54Z"
	report := map[string]any{
		"plan": "Plus",
		"lines": []any{
			map[string]any{
				"type": "progress", "label": "Weekly", "used": 6.0, "limit": 100.0,
				"format":   map[string]any{"kind": "percent"},
				"resetsAt": "2026-08-27T07:38:07Z",
			},
			map[string]any{
				"type": "text", "label": "Manual reset", "value": "1 available",
				"subtitle": expiresAt,
			},
		},
	}

	quota := QuotaFromOpenusage(report, "codex", "Codex", 1)
	if len(quota.Meta) != 1 {
		t.Fatalf("meta rows = %d, want 1: %+v", len(quota.Meta), quota.Meta)
	}
	expires, parsed := ParseISOEpoch(expiresAt)
	if !parsed {
		t.Fatal("test expiration did not parse")
	}
	want := "1 available · expires " + FormatClockTime(expires)
	if quota.Meta[0].Label != "Manual reset" || quota.Meta[0].Value != want {
		t.Fatalf("manual reset = %+v, want value %q", quota.Meta[0], want)
	}
}

func TestOpenusageTextMetaIgnoresUsageHistory(t *testing.T) {
	line := map[string]any{
		"type": "text", "label": "Today", "value": "$5.09 · 7.4M tokens",
	}
	if meta, ok := OpenusageTextMeta(line); ok {
		t.Fatalf("usage history became quota metadata: %+v", meta)
	}
}
