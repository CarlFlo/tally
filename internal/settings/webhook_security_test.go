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
