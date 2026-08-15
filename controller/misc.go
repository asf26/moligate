package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/console_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func TestStatus(c *gin.Context) {
	err := model.PingDB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "数据库连接失败",
		})
		return
	}
	// 获取HTTP统计信息
	httpStats := middleware.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Server is running",
		"http_stats": httpStats,
	})
	return
}

func GetStatus(c *gin.Context) {

	cs := console_setting.GetConsoleSetting()
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	passkeySetting := system_setting.GetPasskeySettings()
	legalSetting := system_setting.GetLegalSettings()
	monitorSetting := operation_setting.GetMonitorSetting()

	data := gin.H{
		"version":                                  common.Version,
		"start_time":                               common.StartTime,
		"email_verification":                       common.EmailVerificationEnabled,
		"github_oauth":                             common.GitHubOAuthEnabled,
		"github_client_id":                         common.GitHubClientId,
		"discord_oauth":                            system_setting.GetDiscordSettings().Enabled,
		"discord_client_id":                        system_setting.GetDiscordSettings().ClientId,
		"linuxdo_oauth":                            common.LinuxDOOAuthEnabled,
		"linuxdo_client_id":                        common.LinuxDOClientId,
		"linuxdo_minimum_trust_level":              common.LinuxDOMinimumTrustLevel,
		"telegram_oauth":                           common.TelegramOAuthEnabled,
		"telegram_bot_name":                        common.TelegramBotName,
		"theme":                                    "default",
		"system_name":                              common.SystemName,
		"logo":                                     common.Logo,
		"footer_html":                              common.Footer,
		"wechat_qrcode":                            common.WeChatAccountQRCodeImageURL,
		"wechat_login":                             common.WeChatAuthEnabled,
		"qq_group_enabled":                         common.QQGroupEnabled,
		"qq_group_number":                          common.QQGroupNumber,
		"qq_group_qrcode_url_light":                common.QQGroupQRCodeURLLight,
		"qq_group_qrcode_url_dark":                 common.QQGroupQRCodeURLDark,
		"wechat_group_enabled":                     common.WeChatGroupEnabled,
		"wechat_group_qrcode_url_light":            common.WeChatGroupQRCodeURLLight,
		"wechat_group_qrcode_url_dark":             common.WeChatGroupQRCodeURLDark,
		"assistant_version":                        common.AssistantVersion,
		"assistant_force_update":                   common.AssistantForceUpdate,
		"assistant_release_notes":                  common.AssistantReleaseNotes,
		"assistant_mac_download_url":               common.AssistantMacDownloadURL,
		"assistant_mac_signature":                  common.AssistantMacSignature,
		"assistant_win_download_url":               common.AssistantWinDownloadURL,
		"assistant_win_signature":                  common.AssistantWinSignature,
		"assistant_published_at":                   common.AssistantPublishedAt,
		"server_address":                           system_setting.ServerAddress,
		"turnstile_check":                          common.TurnstileCheckEnabled,
		"turnstile_site_key":                       common.TurnstileSiteKey,
		"docs_link":                                operation_setting.GetGeneralSetting().DocsLink,
		"quota_per_unit":                           common.QuotaPerUnit,
		"channel_monitor_default_interval_seconds": monitorSetting.ChannelMonitorDefaultIntervalSeconds,
		// 兼容旧前端：保留 display_in_currency，同时提供新的 quota_display_type
		"display_in_currency":           operation_setting.IsCurrencyDisplay(),
		"quota_display_type":            operation_setting.GetQuotaDisplayType(),
		"custom_currency_symbol":        operation_setting.GetGeneralSetting().CustomCurrencySymbol,
		"custom_currency_exchange_rate": operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate,
		"enable_batch_update":           common.BatchUpdateEnabled,
		"enable_drawing":                common.DrawingEnabled,
		"enable_task":                   common.TaskEnabled,
		"enable_data_export":            common.DataExportEnabled,
		"data_export_default_time":      common.DataExportDefaultTime,
		"default_collapse_sidebar":      common.DefaultCollapseSidebar,
		"mj_notify_enabled":             setting.MjNotifyEnabled,
		"chats":                         setting.Chats,
		"demo_site_enabled":             operation_setting.DemoSiteEnabled,
		"self_use_mode_enabled":         operation_setting.SelfUseModeEnabled,
		"register_enabled":              common.RegisterEnabled,
		"password_login_enabled":        common.PasswordLoginEnabled,
		"password_register_enabled":     common.PasswordRegisterEnabled,
		"default_use_auto_group":        setting.DefaultUseAutoGroup,
		"notice_force_popup":            common.OptionMap["NoticeForcePopup"] == "true",

		"usd_exchange_rate": operation_setting.USDExchangeRate,
		"price":             operation_setting.Price,
		"stripe_unit_price": setting.StripeUnitPrice,

		// 面板启用开关
		"api_info_enabled":      cs.ApiInfoEnabled,
		"uptime_kuma_enabled":   cs.UptimeKumaEnabled,
		"announcements_enabled": cs.AnnouncementsEnabled,
		"faq_enabled":           cs.FAQEnabled,

		// 模块管理配置
		"HeaderNavModules":    common.OptionMap["HeaderNavModules"],
		"SidebarModulesAdmin": common.OptionMap["SidebarModulesAdmin"],

		"oidc_enabled":                system_setting.GetOIDCSettings().Enabled,
		"oidc_client_id":              system_setting.GetOIDCSettings().ClientId,
		"oidc_authorization_endpoint": system_setting.GetOIDCSettings().AuthorizationEndpoint,
		"oidc_display_name":           system_setting.GetOIDCSettings().GetEffectiveDisplayName(),
		"passkey_login":               passkeySetting.Enabled,
		"passkey_display_name":        passkeySetting.RPDisplayName,
		"passkey_rp_id":               passkeySetting.RPID,
		"passkey_origins":             passkeySetting.Origins,
		"passkey_allow_insecure":      passkeySetting.AllowInsecureOrigin,
		"passkey_user_verification":   passkeySetting.UserVerification,
		"passkey_attachment":          passkeySetting.AttachmentPreference,
		"setup":                       constant.Setup,
		"user_agreement_enabled":      legalSetting.UserAgreement != "",
		"privacy_policy_enabled":      legalSetting.PrivacyPolicy != "",
		"checkin_enabled":             operation_setting.GetCheckinSetting().Enabled,
	}

	// 根据启用状态注入可选内容
	if cs.ApiInfoEnabled {
		data["api_info"] = console_setting.GetApiInfo()
	}
	if cs.AnnouncementsEnabled {
		data["announcements"] = console_setting.GetAnnouncements()
	}
	if cs.FAQEnabled {
		data["faq"] = console_setting.GetFAQ()
	}

	// Add enabled custom OAuth providers
	customProviders := oauth.GetEnabledCustomProviders()
	if len(customProviders) > 0 {
		type CustomOAuthInfo struct {
			Id                    int    `json:"id"`
			Name                  string `json:"name"`
			Slug                  string `json:"slug"`
			Icon                  string `json:"icon"`
			ClientId              string `json:"client_id"`
			AuthorizationEndpoint string `json:"authorization_endpoint"`
			Scopes                string `json:"scopes"`
		}
		providersInfo := make([]CustomOAuthInfo, 0, len(customProviders))
		for _, p := range customProviders {
			config := p.GetConfig()
			providersInfo = append(providersInfo, CustomOAuthInfo{
				Id:                    config.Id,
				Name:                  config.Name,
				Slug:                  config.Slug,
				Icon:                  config.Icon,
				ClientId:              config.ClientId,
				AuthorizationEndpoint: config.AuthorizationEndpoint,
				Scopes:                config.Scopes,
			})
		}
		data["custom_oauth_providers"] = providersInfo
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
	return
}

func GetAssistantVersion(c *gin.Context) {
	target := strings.ToLower(strings.TrimSpace(c.Query("target")))
	arch := strings.ToLower(strings.TrimSpace(c.Query("arch")))
	currentVersion := normalizeAssistantVersion(c.Query("current_version"))
	if target != "" || arch != "" || currentVersion != "" {
		writeAssistantUpdateManifest(c, target, arch, currentVersion)
		return
	}

	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"version":          common.AssistantVersion,
		"force_update":     common.AssistantForceUpdate,
		"release_notes":    common.AssistantReleaseNotes,
		"mac_download_url": common.AssistantMacDownloadURL,
		"mac_signature":    common.AssistantMacSignature,
		"win_download_url": common.AssistantWinDownloadURL,
		"win_signature":    common.AssistantWinSignature,
		"published_at":     common.AssistantPublishedAt,
	})
}

func writeAssistantUpdateManifest(c *gin.Context, target string, arch string, currentVersion string) {
	common.OptionMapRWMutex.RLock()
	version := normalizeAssistantVersion(common.AssistantVersion)
	notes := common.AssistantReleaseNotes
	publishedAt := common.AssistantPublishedAt
	url, signature := assistantUpdateArtifact(target, arch)
	common.OptionMapRWMutex.RUnlock()

	if version == "" || url == "" || signature == "" {
		c.Status(http.StatusNoContent)
		return
	}
	if version == currentVersion {
		c.Status(http.StatusNoContent)
		return
	}
	if compareAssistantVersion(version, currentVersion) <= 0 {
		c.Status(http.StatusNoContent)
		return
	}

	response := gin.H{
		"version":   version,
		"url":       url,
		"signature": signature,
	}
	if strings.TrimSpace(notes) != "" {
		response["notes"] = notes
	}
	if publishedAt > 0 {
		response["pub_date"] = time.Unix(publishedAt, 0).UTC().Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, response)
}

func assistantUpdateArtifact(target string, arch string) (string, string) {
	switch target {
	case "darwin":
		if arch == "x86_64" || arch == "aarch64" {
			return common.AssistantMacDownloadURL, common.AssistantMacSignature
		}
	case "windows":
		if arch == "x86_64" {
			return common.AssistantWinDownloadURL, common.AssistantWinSignature
		}
	}
	return "", ""
}

func normalizeAssistantVersion(value string) string {
	return strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(value), "v"), "V")
}

func compareAssistantVersion(left string, right string) int {
	leftParts := assistantVersionParts(left)
	rightParts := assistantVersionParts(right)
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for i := 0; i < maxLen; i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func assistantVersionParts(version string) []int {
	parts := strings.FieldsFunc(version, func(r rune) bool {
		return r < '0' || r > '9'
	})
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(part)
		if err == nil {
			values = append(values, value)
		}
	}
	return values
}

func GetNotice(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Notice"],
	})
	return
}

func GetAbout(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["About"],
	})
	return
}

func GetUserAgreement(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().UserAgreement,
	})
	return
}

func GetPrivacyPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    system_setting.GetLegalSettings().PrivacyPolicy,
	})
	return
}

func GetMidjourney(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["Midjourney"],
	})
	return
}

func GetHomePageContent(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    common.OptionMap["HomePageContent"],
	})
	return
}

func SendEmailVerification(c *gin.Context) {
	email := model.NormalizeEmail(c.Query("email"))
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的邮箱地址",
		})
		return
	}
	localPart := parts[0]
	domainPart := parts[1]
	if common.EmailDomainRestrictionEnabled {
		allowed := false
		for _, domain := range common.EmailDomainWhitelist {
			if domainPart == domain {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "The administrator has enabled the email domain name whitelist, and your email address is not allowed due to special symbols or it's not in the whitelist.",
			})
			return
		}
	}
	if common.EmailAliasRestrictionEnabled {
		containsSpecialSymbols := strings.Contains(localPart, "+") || strings.Contains(localPart, ".")
		if containsSpecialSymbols {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员已启用邮箱地址别名限制，您的邮箱地址由于包含特殊符号而被拒绝。",
			})
			return
		}
	}

	if model.IsEmailAlreadyTaken(email) {
		common.ApiErrorI18n(c, i18n.MsgUserEmailAlreadyTaken)
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(email, code, common.EmailVerificationPurpose)
	subject := fmt.Sprintf("%s邮箱验证邮件", common.SystemName)
	content := fmt.Sprintf("<p>您好，你正在进行%s邮箱验证。</p>"+
		"<p>您的验证码为: <strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, code, common.VerificationValidMinutes)
	err := common.SendEmail(subject, email, content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func SendPasswordResetEmail(c *gin.Context) {
	email := model.NormalizeEmail(c.Query("email"))
	if err := common.Validate.Var(email, "required,email"); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !model.IsEmailAlreadyTaken(email) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该邮箱未绑定任何账号",
		})
		return
	}
	code := common.GenerateVerificationCode(0)
	common.RegisterVerificationCodeWithKey(email, code, common.PasswordResetPurpose)
	link := fmt.Sprintf("%s/user/reset?email=%s&token=%s", system_setting.ServerAddress, email, code)
	subject := fmt.Sprintf("%s密码重置", common.SystemName)
	content := fmt.Sprintf("<p>您好，你正在进行%s密码重置。</p>"+
		"<p>点击 <a href='%s'>此处</a> 进行密码重置。</p>"+
		"<p>如果链接无法点击，请尝试点击下面的链接或将其复制到浏览器中打开：<br> %s </p>"+
		"<p>重置链接 %d 分钟内有效，如果不是本人操作，请忽略。</p>", common.SystemName, link, link, common.VerificationValidMinutes)
	if err := common.SendEmail(subject, email, content); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to send password reset email to %s: %s", email, err.Error()))
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "重置邮件发送失败，请检查 SMTP 配置",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

type PasswordResetRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

func ResetPassword(c *gin.Context) {
	var req PasswordResetRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	req.Email = model.NormalizeEmail(req.Email)
	if req.Email == "" || req.Token == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if !common.VerifyCodeWithKey(req.Email, req.Token, common.PasswordResetPurpose) {
		common.ApiErrorI18n(c, i18n.MsgUserPasswordResetLinkInvalid)
		return
	}
	password := common.GenerateVerificationCode(12)
	err = model.ResetUserPasswordByEmail(req.Email, password)
	if err != nil {
		if errors.Is(err, model.ErrEmailNotFound) || errors.Is(err, model.ErrEmailAmbiguous) {
			common.ApiErrorI18n(c, i18n.MsgUserPasswordResetLinkInvalid)
			return
		}
		common.ApiError(c, err)
		return
	}
	common.DeleteKey(req.Email, common.PasswordResetPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    password,
	})
	return
}
