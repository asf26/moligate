package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// requireTopUpEnabled is shared by every user-initiated payment and wallet
// redemption endpoint. The permission is read from the database on each
// request so an administrator's decision is effective immediately, even when
// an older authentication snapshot is still cached.
func requireTopUpEnabled(c *gin.Context) bool {
	enabled, err := model.IsUserTopUpEnabled(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return false
	}
	if !enabled {
		common.ApiErrorI18n(c, i18n.MsgUserTopUpDisabled)
		return false
	}
	return true
}
