package api

import (
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
)

type AdminHandler struct {
	Service     *adminservice.Service
	TaskMonitor *serviceutil.BackgroundTaskMonitor
	Reauth      *servicereauth.Service
	Backup      *servicebackup.Service
}

func NewAdminHandler(s *adminservice.Service, taskMonitor *serviceutil.BackgroundTaskMonitor, reauth *servicereauth.Service, backup *servicebackup.Service) *AdminHandler {
	return &AdminHandler{Service: s, TaskMonitor: taskMonitor, Reauth: reauth, Backup: backup}
}
