package mail

import (
	"strings"
	"testing"
)

func TestRenderEmailTemplateHeroImage(t *testing.T) {
	t.Parallel()

	rendered := renderEmailTemplate(emailTemplate{
		Title:        "Campaign",
		HeroImageURL: `https://cdn.example.com/image.jpg?x=1&y="quoted"`,
		HeroImageAlt: `Dashboard "preview"`,
	})

	if !strings.Contains(rendered, `src="https://cdn.example.com/image.jpg?x=1&amp;y=&#34;quoted&#34;"`) {
		t.Fatal("expected escaped hero image URL")
	}
	if !strings.Contains(rendered, `alt="Dashboard &#34;preview&#34;"`) {
		t.Fatal("expected escaped hero image alternative text")
	}
}
