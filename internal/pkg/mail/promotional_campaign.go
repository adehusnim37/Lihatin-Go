package mail

import "fmt"

type PromotionalCampaignEmailData struct {
	ToEmail                string
	UserName               string
	Subject                string
	Preheader              string
	Body                   string
	ImageURL               string
	ImageAlt               string
	CTALabel               string
	CTAURL                 string
	BaseURL                string
	PreferencesURL         string
	UnsubscribeURL         string
	OneClickUnsubscribeURL string
}

func (es *EmailService) SendPromotionalCampaignEmail(data PromotionalCampaignEmailData) error {
	actions := make([]emailAction, 0, 1)
	if data.CTALabel != "" && data.CTAURL != "" {
		actions = append(actions, emailAction{
			Label:   data.CTALabel,
			URL:     data.CTAURL,
			Variant: "primary",
		})
	}

	subtitle := data.Preheader
	if subtitle == "" {
		subtitle = "News and offers from Lihatin."
	}

	htmlBody := renderEmailTemplate(emailTemplate{
		Badge:                "Lihatin",
		Title:                data.Subject,
		Subtitle:             subtitle,
		Greeting:             fmt.Sprintf("Hi %s,", data.UserName),
		Intro:                data.Body,
		HeroImageURL:         data.ImageURL,
		HeroImageAlt:         data.ImageAlt,
		Actions:              actions,
		FooterBaseURL:        data.BaseURL,
		FooterPreferencesURL: data.PreferencesURL,
		FooterUnsubscribeURL: data.UnsubscribeURL,
	})

	textBody := fmt.Sprintf(`LIHATIN

Hi %s,

%s
`, data.UserName, data.Body)
	if data.CTAURL != "" {
		textBody += fmt.Sprintf("\n%s: %s\n", data.CTALabel, data.CTAURL)
	}
	textBody += fmt.Sprintf(`
Email preferences: %s
Unsubscribe from promotions: %s
`, data.PreferencesURL, data.UnsubscribeURL)

	headers := map[string]string{
		"List-Unsubscribe":      fmt.Sprintf("<%s>", data.OneClickUnsubscribeURL),
		"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
	}
	return es.sendEmailWithHeaders(data.ToEmail, data.Subject, textBody, htmlBody, headers)
}
