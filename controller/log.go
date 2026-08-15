package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const logExportLimit = 50000

type logExportField struct {
	Key       string
	Label     string
	AdminOnly bool
	Value     func(*model.Log) string
}

func parseLogQuery(c *gin.Context) model.LogQuery {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channel, _ := strconv.Atoi(c.Query("channel"))
	return model.LogQuery{
		UserId:            c.GetInt("id"),
		LogType:           logType,
		StartTimestamp:    startTimestamp,
		EndTimestamp:      endTimestamp,
		ModelName:         c.Query("model_name"),
		Username:          c.Query("username"),
		TokenName:         c.Query("token_name"),
		Channel:           channel,
		Group:             c.Query("group"),
		RequestId:         c.Query("request_id"),
		UpstreamRequestId: c.Query("upstream_request_id"),
	}
}

func logExportFields() []logExportField {
	return []logExportField{
		{Key: "id", Label: "ID", Value: func(log *model.Log) string { return strconv.Itoa(log.Id) }},
		{Key: "created_at", Label: "Time", Value: func(log *model.Log) string { return strconv.FormatInt(log.CreatedAt, 10) }},
		{Key: "type", Label: "Type", Value: func(log *model.Log) string { return strconv.Itoa(log.Type) }},
		{Key: "content", Label: "Content", Value: func(log *model.Log) string { return log.Content }},
		{Key: "user_id", Label: "User ID", AdminOnly: true, Value: func(log *model.Log) string { return strconv.Itoa(log.UserId) }},
		{Key: "username", Label: "Username", AdminOnly: true, Value: func(log *model.Log) string { return log.Username }},
		{Key: "token_id", Label: "Token ID", Value: func(log *model.Log) string { return strconv.Itoa(log.TokenId) }},
		{Key: "token_name", Label: "Token", Value: func(log *model.Log) string { return log.TokenName }},
		{Key: "model_name", Label: "Model", Value: func(log *model.Log) string { return log.ModelName }},
		{Key: "quota", Label: "Quota", Value: func(log *model.Log) string { return strconv.Itoa(log.Quota) }},
		{Key: "prompt_tokens", Label: "Prompt Tokens", Value: func(log *model.Log) string { return strconv.Itoa(log.PromptTokens) }},
		{Key: "completion_tokens", Label: "Completion Tokens", Value: func(log *model.Log) string { return strconv.Itoa(log.CompletionTokens) }},
		{Key: "use_time", Label: "Use Time", Value: func(log *model.Log) string { return strconv.Itoa(log.UseTime) }},
		{Key: "is_stream", Label: "Stream", Value: func(log *model.Log) string { return strconv.FormatBool(log.IsStream) }},
		{Key: "channel", Label: "Channel ID", AdminOnly: true, Value: func(log *model.Log) string { return strconv.Itoa(log.ChannelId) }},
		{Key: "channel_name", Label: "Channel Name", AdminOnly: true, Value: func(log *model.Log) string { return log.ChannelName }},
		{Key: "group", Label: "Group", Value: func(log *model.Log) string { return log.Group }},
		{Key: "ip", Label: "IP", AdminOnly: true, Value: func(log *model.Log) string { return log.Ip }},
		{Key: "request_id", Label: "Request ID", Value: func(log *model.Log) string { return log.RequestId }},
		{Key: "upstream_request_id", Label: "Upstream Request ID", Value: func(log *model.Log) string { return log.UpstreamRequestId }},
		{Key: "other", Label: "Other", Value: func(log *model.Log) string { return log.Other }},
	}
}

func selectedLogExportFields(rawFields string, adminView bool) []logExportField {
	fields := logExportFields()
	fieldByKey := make(map[string]logExportField, len(fields))
	for _, field := range fields {
		fieldByKey[field.Key] = field
	}
	defaultKeys := []string{"id", "created_at", "type", "username", "token_name", "model_name", "quota", "prompt_tokens", "completion_tokens", "use_time", "channel", "channel_name", "group", "request_id", "upstream_request_id"}
	if !adminView {
		defaultKeys = []string{"id", "created_at", "type", "content", "token_name", "model_name", "quota", "prompt_tokens", "completion_tokens", "use_time", "group", "request_id", "upstream_request_id"}
	}

	rawItems := strings.Split(rawFields, ",")
	keys := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		key := strings.TrimSpace(raw)
		if key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		keys = defaultKeys
	}

	selected := make([]logExportField, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		field, ok := fieldByKey[key]
		if !ok || field.AdminOnly && !adminView {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		selected = append(selected, field)
	}
	if len(selected) == 0 {
		return selectedLogExportFields("", adminView)
	}
	return selected
}

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := parseLogQuery(c)
	logs, total, err := model.GetAllLogs(query.LogType, query.StartTimestamp, query.EndTimestamp, query.ModelName, query.Username, query.TokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), query.Channel, query.Group, query.RequestId, query.UpstreamRequestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := parseLogQuery(c)
	logs, total, err := model.GetUserLogs(query.UserId, query.LogType, query.StartTimestamp, query.EndTimestamp, query.ModelName, query.TokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), query.Group, query.RequestId, query.UpstreamRequestId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func ExportAllLogs(c *gin.Context) {
	query := parseLogQuery(c)
	logs, err := model.ExportLogs(query, true, logExportLimit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fields := selectedLogExportFields(c.Query("fields"), true)

	filename := fmt.Sprintf("usage-logs-%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Status(http.StatusOK)

	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = field.Label
	}
	_ = writer.Write(headers)
	for _, log := range logs {
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = field.Value(log)
		}
		_ = writer.Write(row)
	}
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	if tokenId == 0 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "无效的令牌",
		})
		return
	}
	logs, err := model.GetLogByTokenId(tokenId)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data":    logs,
	})
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": stat.Quota,
			"rpm":   stat.Rpm,
			"tpm":   stat.Tpm,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota": quotaNum.Quota,
			"rpm":   quotaNum.Rpm,
			"tpm":   quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}
