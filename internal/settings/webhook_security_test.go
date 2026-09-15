package settings

import "testing"

func TestWebhookRejectsUnsafeURLsAndInvalidTemplates(t *testing.T) {
	for _, endpoint := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://user:secret@example.com", "https://example.com/#fragment"} {
		if ValidateWebhook(Webhook{Enabled: true, URL: endpoint}) == nil {
			t.Errorf("accepted unsafe URL %q", endpoint)
		}
	}
	for _, body := range []string{`{"{{show}}":"value"}`, `{"message":"\u007b\u007bunknown}}"}`, `{"nested":["{{unknown}}"]}`, `{"message":{{message}}}`} {
		if ValidateWebhook(Webhook{URL: "http://localhost/hook", Body: body}) == nil {
			t.Errorf("accepted invalid template %s", body)
		}
	}
	if err := ValidateWebhook(Webhook{URL: "http://localhost/hook", Body: `{"nested":["{{show}}",{"message":"{{message}}"}]}`}); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookDefaultsUseServerTimezoneAsAuthority(t *testing.T) {
	if got := (Webhook{}).Defaults("Europe/Stockholm").Timezone; got != "Europe/Stockholm" {
		t.Fatalf("default timezone = %q", got)
	}
	if got := (Webhook{Timezone: "America/New_York"}).Defaults("Europe/Stockholm").Timezone; got != "Europe/Stockholm" {
		t.Fatalf("legacy stored timezone overrode server timezone: %q", got)
	}
	if got := (Webhook{}).Defaults().Timezone; got != "UTC" {
		t.Fatalf("fallback timezone = %q", got)
	}
}
