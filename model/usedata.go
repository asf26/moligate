package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index;index:idx_qdt_user_token_model_time,priority:1"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	TokenID   int    `json:"token_id" gorm:"default:0;index;index:idx_qdt_user_token_model_time,priority:2"`
	TokenName string `json:"token_name" gorm:"size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;index:idx_qdt_user_token_model_time,priority:3;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2;index:idx_qdt_user_token_model_time,priority:4"`
	UseGroup  string `json:"use_group" gorm:"index;size:64;default:''"`
	ChannelID int    `json:"channel_id" gorm:"index;default:0"`
	NodeName  string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`
}

type QuotaDataLogParams struct {
	UserID    int
	Username  string
	TokenID   int
	TokenName string
	ModelName string
	Quota     int
	CreatedAt int64
	TokenUsed int
	UseGroup  string
	ChannelID int
	NodeName  string
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%d\x00%s\x00%d\x00%s\x00%d\x00%s",
		quotaData.UserID,
		quotaData.TokenID,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	count := quotaData.Count
	quota := quotaData.Quota
	tokenUsed := quotaData.TokenUsed
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		cachedQuotaData.Count += count
		cachedQuotaData.Quota += quota
		cachedQuotaData.TokenUsed += tokenUsed
		cachedQuotaData.Username = quotaData.Username
		cachedQuotaData.TokenName = quotaData.TokenName
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	quotaData := &QuotaData{
		UserID:    params.UserID,
		Username:  params.Username,
		TokenID:   params.TokenID,
		TokenName: params.TokenName,
		ModelName: params.ModelName,
		CreatedAt: createdAt,
		UseGroup:  params.UseGroup,
		ChannelID: params.ChannelID,
		NodeName:  params.NodeName,
		Count:     1,
		Quota:     params.Quota,
		TokenUsed: params.TokenUsed,
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and token_id = ? and model_name = ? and created_at = ? and use_group = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.TokenID, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and token_id = ? and model_name = ? and created_at = ? and use_group = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.TokenID, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":      gorm.Expr("count + ?", quotaData.Count),
			"quota":      gorm.Expr("quota + ?", quotaData.Quota),
			"token_used": gorm.Expr("token_used + ?", quotaData.TokenUsed),
			"username":   quotaData.Username,
			"token_name": quotaData.TokenName,
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func aggregateQuotaDataByModel(query *gorm.DB, tokenId int, includeUserFields bool) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	userFields := ""
	if includeUserFields {
		userFields = "min(id) as id, max(user_id) as user_id, max(username) as username, "
	}
	if tokenId > 0 {
		query = query.Select(userFields + "model_name, created_at, max(token_id) as token_id, max(token_name) as token_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used")
	} else {
		query = query.Select(userFields + "model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used")
	}
	err = query.Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64, tokenId int) (quotaData []*QuotaData, err error) {
	query := DB.Table("quota_data").Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime)
	if tokenId > 0 {
		query = query.Where("token_id = ?", tokenId)
	}
	return aggregateQuotaDataByModel(query, tokenId, true)
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64, tokenId int) (quotaData []*QuotaData, err error) {
	query := DB.Table("quota_data").Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime)
	if tokenId > 0 {
		query = query.Where("token_id = ?", tokenId)
	}
	return aggregateQuotaDataByModel(query, tokenId, true)
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string, tokenId int) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime, tokenId)
	}
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	query := DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime)
	if tokenId > 0 {
		query = query.Where("token_id = ?", tokenId)
	}
	return aggregateQuotaDataByModel(query, tokenId, false)
}
