package notification

import (
	"github.com/adehusnim37/lihatin-go/controllers"
	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/adehusnim37/lihatin-go/internal/pkg/storage"
	"github.com/adehusnim37/lihatin-go/repositories/userrepo"
)

type Controller struct {
	*controllers.BaseController
	preferences        *userrepo.NotificationPreferenceRepository
	campaignImageStore *storage.S3CampaignImageStorage
}

func NewController(base *controllers.BaseController) *Controller {
	if base == nil || base.GormDB == nil {
		panic("GormDB is required for NotificationController")
	}

	campaignImageStore, campaignImageStoreErr := storage.NewS3CampaignImageStorageFromEnv()
	if campaignImageStoreErr != nil {
		logger.Logger.Warn("Campaign image storage is not configured", "error", campaignImageStoreErr.Error())
	}

	return &Controller{
		BaseController:     base,
		preferences:        userrepo.NewNotificationPreferenceRepository(base.GormDB),
		campaignImageStore: campaignImageStore,
	}
}
