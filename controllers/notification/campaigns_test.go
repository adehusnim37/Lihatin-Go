package notification

import "testing"

func TestValidateCampaignImageURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty is optional", value: "", wantErr: false},
		{name: "https URL", value: "https://cdn.example.com/campaign.jpg", wantErr: false},
		{name: "http URL", value: "http://localhost:9000/campaign.png", wantErr: false},
		{name: "relative URL", value: "/campaign.png", wantErr: true},
		{name: "unsupported scheme", value: "javascript:alert(1)", wantErr: true},
		{name: "missing host", value: "https:///campaign.png", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			errMessage := validateCampaignImageURL(test.value)
			if test.wantErr && errMessage == "" {
				t.Fatalf("expected validation error for %q", test.value)
			}
			if !test.wantErr && errMessage != "" {
				t.Fatalf("unexpected validation error for %q: %s", test.value, errMessage)
			}
		})
	}
}
