package notification

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/adehusnim37/lihatin-go/models/user"
)

func TestListCampaignsFiltersAndKeepsGlobalStatusCounts(t *testing.T) {
	controller, db := newAnnouncementTestController(t, "campaign-list-status-counts")
	if err := db.AutoMigrate(&user.PromotionalCampaign{}); err != nil {
		t.Fatal(err)
	}
	for i, status := range []user.PromotionalCampaignStatus{
		user.PromotionalCampaignDraft, user.PromotionalCampaignCompleted,
		user.PromotionalCampaignDraft, user.PromotionalCampaignFailed,
	} {
		campaign := user.PromotionalCampaign{
			ID: string(rune('a' + i)), Name: "Campaign", Subject: "Subject", Body: "Body",
			CreatedBy: "admin", Status: status, CreatedAt: time.Unix(int64(i+1), 0),
		}
		if err := db.Create(&campaign).Error; err != nil {
			t.Fatal(err)
		}
	}
	recorder := serveAnnouncement(controller, http.MethodGet,
		"/v1/auth/admin/promotional-campaigns?status=draft&page=2&limit=1", "", controller.ListCampaigns)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			Campaigns    []user.PromotionalCampaign `json:"campaigns"`
			Total        int64                      `json:"total"`
			TotalPages   int                        `json:"total_pages"`
			StatusCounts map[string]int64           `json:"status_counts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	data := response.Data
	if data.Total != 2 || data.TotalPages != 2 || len(data.Campaigns) != 1 || data.Campaigns[0].ID != "a" {
		t.Fatalf("unexpected filtered page: %+v", data)
	}
	if data.StatusCounts["draft"] != 2 || data.StatusCounts["completed"] != 1 || data.StatusCounts["failed"] != 1 {
		t.Fatalf("counts must include campaigns outside the filtered page: %+v", data.StatusCounts)
	}
	invalid := serveAnnouncement(controller, http.MethodGet,
		"/v1/auth/admin/promotional-campaigns?status=unknown", "", controller.ListCampaigns)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalid.Code)
	}
}
