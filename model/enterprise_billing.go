package model

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	EnterpriseBillingSourceLocal    = "local"
	EnterpriseBillingSourceS10      = "s10"
	EnterpriseBillingSourceMoligate = "moligate"

	enterpriseBillingS10DSNEnv        = "ENTERPRISE_BILLING_S10_DSN"
	enterpriseBillingMoligateDSNEnv   = "ENTERPRISE_BILLING_MOLIGATE_DSN"
	enterpriseBillingS10TableEnv      = "ENTERPRISE_BILLING_S10_TABLE"
	enterpriseBillingMoligateTableEnv = "ENTERPRISE_BILLING_MOLIGATE_TABLE"
)

// EnterpriseBillingAccount describes one B-end customer's usage source and
// the contract expression used to turn token usage into a USD invoice.
type EnterpriseBillingAccount struct {
	Id          int    `json:"id"`
	Name        string `json:"name" gorm:"type:varchar(128);not null"`
	Code        string `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Source      string `json:"source" gorm:"type:varchar(32);not null;index"`
	Usernames   string `json:"-" gorm:"type:text;not null"`
	PricingRule string `json:"pricing_rule" gorm:"type:text;not null"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (account EnterpriseBillingAccount) UsernameList() []string {
	var usernames []string
	if account.Usernames != "" {
		if err := common.Unmarshal([]byte(account.Usernames), &usernames); err != nil {
			usernames = strings.FieldsFunc(account.Usernames, func(r rune) bool {
				return r == ',' || r == '\n' || r == '\r'
			})
		}
	}
	seen := make(map[string]struct{}, len(usernames))
	result := make([]string, 0, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		result = append(result, username)
	}
	return result
}

func (account *EnterpriseBillingAccount) SetUsernameList(usernames []string) error {
	cleaned := make([]string, 0, len(usernames))
	seen := make(map[string]struct{}, len(usernames))
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		cleaned = append(cleaned, username)
	}
	if len(cleaned) == 0 {
		return errors.New("至少需要配置一个使用账号")
	}
	encoded, err := common.Marshal(cleaned)
	if err != nil {
		return err
	}
	account.Usernames = string(encoded)
	return nil
}

func (account EnterpriseBillingAccount) PublicView() map[string]any {
	return map[string]any{
		"id":                account.Id,
		"name":              account.Name,
		"code":              account.Code,
		"source":            account.Source,
		"usernames":         account.UsernameList(),
		"pricing_rule":      account.PricingRule,
		"enabled":           account.Enabled,
		"source_configured": EnterpriseBillingSourceConfigured(account.Source),
		"created_at":        account.CreatedAt,
		"updated_at":        account.UpdatedAt,
	}
}

var enterpriseBillingTablePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func enterpriseBillingSourceConfig(source string) (string, string, bool) {
	source = strings.ToLower(strings.TrimSpace(source))
	switch source {
	case EnterpriseBillingSourceS10:
		return enterpriseBillingS10DSNEnv, os.Getenv(enterpriseBillingS10TableEnv), true
	case EnterpriseBillingSourceMoligate:
		return enterpriseBillingMoligateDSNEnv, os.Getenv(enterpriseBillingMoligateTableEnv), true
	default:
		return "", "", false
	}
}

func enterpriseBillingTableName(source, configured string) (string, error) {
	table := strings.TrimSpace(configured)
	if table == "" {
		table = "logs"
	}
	if !enterpriseBillingTablePattern.MatchString(table) {
		return "", fmt.Errorf("%s 的账单表名无效", source)
	}
	return table, nil
}

func EnterpriseBillingSourceConfigured(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == EnterpriseBillingSourceLocal {
		return LOG_DB != nil
	}
	envName, _, ok := enterpriseBillingSourceConfig(source)
	return ok && strings.TrimSpace(os.Getenv(envName)) != ""
}

// OpenEnterpriseBillingSource opens a configured remote source. Credentials
// are read only from server environment variables and are never returned to
// the API client. The returned close function is safe to call for every source.
func OpenEnterpriseBillingSource(source string) (*gorm.DB, func(), common.DatabaseType, error) {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == EnterpriseBillingSourceLocal {
		if LOG_DB == nil {
			return nil, func() {}, common.DatabaseTypeSQLite, errors.New("日志数据库未初始化")
		}
		return LOG_DB, func() {}, common.LogDatabaseType(), nil
	}
	envName, _, ok := enterpriseBillingSourceConfig(source)
	if !ok {
		return nil, func() {}, "", fmt.Errorf("不支持的账单数据源: %s", source)
	}
	if strings.TrimSpace(os.Getenv(envName)) == "" {
		return nil, func() {}, "", fmt.Errorf("数据源 %s 尚未配置", source)
	}
	db, dbType, err := chooseDB(envName, true)
	if err != nil {
		return nil, func() {}, dbType, fmt.Errorf("连接 %s 数据源失败: %w", source, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, func() {}, dbType, err
	}
	return db, func() { _ = sqlDB.Close() }, dbType, nil
}

func EnterpriseBillingSourceTable(source string) (string, error) {
	if source == EnterpriseBillingSourceLocal {
		return "logs", nil
	}
	_, configured, ok := enterpriseBillingSourceConfig(source)
	if !ok {
		return "", fmt.Errorf("不支持的账单数据源: %s", source)
	}
	return enterpriseBillingTableName(source, configured)
}

func queryEnterpriseBillingLogs(db *gorm.DB, dbType common.DatabaseType, table string, start, end int64, usernames []string) ([]*Log, error) {
	if len(usernames) == 0 {
		return []*Log{}, nil
	}
	groupColumn := "`group`"
	if dbType == common.DatabaseTypePostgreSQL {
		groupColumn = `"group"`
	}
	columns := "id,user_id,created_at,type,username,token_name,model_name,quota,prompt_tokens,completion_tokens,use_time,channel_id," + groupColumn + ",request_id"
	var logs []*Log
	err := db.Table(table).
		Select(columns).
		Where("type = ?", LogTypeConsume).
		Where("created_at >= ? AND created_at < ?", start, end).
		Where("username IN ?", usernames).
		Order("created_at asc, id asc").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	if err := hydrateEnterpriseBillingChannelNames(db, logs); err != nil {
		// A remote usage database may intentionally expose only its logs table.
		// Keep the usage rows usable even when channel metadata is unavailable.
		for _, log := range logs {
			if log.ChannelName == "" && log.ChannelId > 0 {
				log.ChannelName = fmt.Sprintf("channel-%d", log.ChannelId)
			}
		}
	}
	return logs, nil
}

func hydrateEnterpriseBillingChannelNames(db *gorm.DB, logs []*Log) error {
	channelIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, log := range logs {
		if log.ChannelId == 0 {
			continue
		}
		if _, ok := seen[log.ChannelId]; ok {
			continue
		}
		seen[log.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, log.ChannelId)
	}
	if len(channelIDs) == 0 {
		return nil
	}
	var channels []struct {
		Id   int    `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	queryChannels := func(source *gorm.DB) error {
		return source.Table("channels").Select("id, name").Where("id IN ?", channelIDs).Find(&channels).Error
	}
	err := queryChannels(db)
	if (err != nil || len(channels) == 0) && DB != nil && DB != db {
		// A remote log store such as ClickHouse may not carry the gateway's
		// channels table. Use the primary database as the authoritative fallback.
		channels = nil
		if fallbackErr := queryChannels(DB); fallbackErr != nil {
			return fallbackErr
		}
	} else if err != nil {
		return err
	}
	channelMap := make(map[int]string, len(channels))
	for _, channel := range channels {
		channelMap[channel.Id] = channel.Name
	}
	for _, log := range logs {
		if channelName := channelMap[log.ChannelId]; channelName != "" {
			log.ChannelName = channelName
		} else if log.ChannelName == "" && log.ChannelId > 0 {
			log.ChannelName = fmt.Sprintf("channel-%d", log.ChannelId)
		}
	}
	return nil
}

func QueryEnterpriseBillingLogs(source string, start, end int64, usernames []string) ([]*Log, error) {
	db, closeDB, dbType, err := OpenEnterpriseBillingSource(source)
	if err != nil {
		return nil, err
	}
	defer closeDB()
	table, err := EnterpriseBillingSourceTable(source)
	if err != nil {
		return nil, err
	}
	return queryEnterpriseBillingLogs(db, dbType, table, start, end, usernames)
}

func ListEnterpriseBillingAccounts() ([]*EnterpriseBillingAccount, error) {
	if DB == nil {
		return nil, errors.New("主数据库未初始化")
	}
	var accounts []*EnterpriseBillingAccount
	if err := DB.Order("name asc, id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func GetEnterpriseBillingAccount(id int) (*EnterpriseBillingAccount, error) {
	if DB == nil {
		return nil, errors.New("主数据库未初始化")
	}
	var account EnterpriseBillingAccount
	if err := DB.First(&account, id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func CreateEnterpriseBillingAccount(account *EnterpriseBillingAccount) error {
	if DB == nil {
		return errors.New("主数据库未初始化")
	}
	return DB.Create(account).Error
}

func SaveEnterpriseBillingAccount(account *EnterpriseBillingAccount) error {
	if DB == nil {
		return errors.New("主数据库未初始化")
	}
	return DB.Save(account).Error
}

func DeleteEnterpriseBillingAccount(id int) error {
	if DB == nil {
		return errors.New("主数据库未初始化")
	}
	return DB.Delete(&EnterpriseBillingAccount{}, id).Error
}
