package model

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/samber/hot"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Subscription duration units
const (
	SubscriptionDurationYear   = "year"
	SubscriptionDurationMonth  = "month"
	SubscriptionDurationDay    = "day"
	SubscriptionDurationHour   = "hour"
	SubscriptionDurationCustom = "custom"
)

// Subscription quota reset period
const (
	SubscriptionResetNever   = "never"
	SubscriptionResetDaily   = "daily"
	SubscriptionResetWeekly  = "weekly"
	SubscriptionResetMonthly = "monthly"
	SubscriptionResetCustom  = "custom"

	subscriptionPeriodResetAnchorVersion = 1
)

const SubscriptionCurrencyCNY = "CNY"

var (
	ErrSubscriptionOrderNotFound          = errors.New("subscription order not found")
	ErrSubscriptionOrderStatusInvalid     = errors.New("subscription order status invalid")
	ErrSubscriptionImportedUsageOverLimit = errors.New("imported subscription usage exceeds plan limits")
	ErrSubscriptionImageOverageUnsettled  = errors.New("subscription image overage cannot be settled")
)

// Subscription resource grants are optional entitlements attached to a plan.
// They let an operator sell a model package while gifting quota for another
// model (or a fixed number of image generations) without creating a second
// subscription key.
const (
	SubscriptionResourceTypeQuota      = "quota"
	SubscriptionResourceTypeImageCount = "image_count"
	SubscriptionMaxBonusResources      = 32
	SubscriptionMaxBillingRatio        = 100
)

// SubscriptionBonusResource is the plan-side definition of an additional
// model entitlement. Amount is expressed in quota units for quota resources
// and in generations for image_count resources.
type SubscriptionBonusResource struct {
	ResourceKey  string `json:"resource_key"`
	ResourceType string `json:"resource_type"`
	ModelName    string `json:"model_name"`
	DisplayName  string `json:"display_name"`
	Amount       int64  `json:"amount"`
}

// SubscriptionResourceGrant is the purchased snapshot of a bonus resource.
// Keeping Used in the snapshot means editing a plan never changes an existing
// customer's entitlement.
type SubscriptionResourceGrant struct {
	ResourceKey  string `json:"resource_key"`
	ResourceType string `json:"resource_type"`
	ModelName    string `json:"model_name"`
	DisplayName  string `json:"display_name"`
	Amount       int64  `json:"amount"`
	Used         int64  `json:"used"`
}

// SubscriptionResourceRequest describes the entitlement to consume for a
// request. Image generation requests use image_count so gpt-image-2 packs are
// counted by generations instead of by their normal quota price.
type SubscriptionResourceRequest struct {
	ResourceKey  string
	ResourceType string
	Amount       int64
}

// SubscriptionQuotaPricing lets the locked subscription transaction derive
// the quota amount from the candidate subscription's purchased multiplier.
// This prevents a read-only quote for one package from being reused when a
// different package is selected under the row lock.
type SubscriptionQuotaPricing struct {
	BaseQuotaBeforeGroup float64
	LiveQuota            int64
}

type AdminSubscriptionUsagePreview struct {
	UserId               int      `json:"user_id"`
	PlanId               int      `json:"plan_id"`
	EffectiveStartTime   int64    `json:"effective_start_time"`
	EndTime              int64    `json:"end_time"`
	ApplicableGroups     []string `json:"applicable_groups"`
	AmountUsed           int64    `json:"amount_used"`
	DailyUsed            int64    `json:"daily_used"`
	WeeklyUsed           int64    `json:"weekly_used"`
	MonthlyUsed          int64    `json:"monthly_used"`
	DailyWindowStart     int64    `json:"daily_window_start"`
	WeeklyWindowStart    int64    `json:"weekly_window_start"`
	MonthlyWindowStart   int64    `json:"monthly_window_start"`
	AmountTotal          int64    `json:"amount_total"`
	DailyAmount          int64    `json:"daily_amount"`
	WeeklyAmount         int64    `json:"weekly_amount"`
	MonthlyAmount        int64    `json:"monthly_amount"`
	ExceedsAmountTotal   bool     `json:"exceeds_amount_total"`
	ExceedsDailyAmount   bool     `json:"exceeds_daily_amount"`
	ExceedsWeeklyAmount  bool     `json:"exceeds_weekly_amount"`
	ExceedsMonthlyAmount bool     `json:"exceeds_monthly_amount"`
	ExceedsAnyLimit      bool     `json:"exceeds_any_limit"`
	WalletQuota          int      `json:"wallet_quota"`
	ConsumeLogEnabled    bool     `json:"consume_log_enabled"`
}

type AdminSubscriptionBindOptions struct {
	EffectiveStartTime int64
	ImportUsage        bool
	ClearWallet        bool
	ConfirmOverLimit   bool
}

func refreshSubscriptionUserGroupCache(userId int, operation string) {
	if err := RefreshUserGroupCache(userId); err != nil {
		common.SysError(fmt.Sprintf("failed to refresh user group cache after %s for user %d: %v", operation, userId, err))
	}
}

const (
	subscriptionPlanCacheNamespace     = "new-api:subscription_plan:v1"
	subscriptionPlanInfoCacheNamespace = "new-api:subscription_plan_info:v1"
)

var (
	subscriptionPlanCacheOnce     sync.Once
	subscriptionPlanInfoCacheOnce sync.Once

	subscriptionPlanCache     *cachex.HybridCache[SubscriptionPlan]
	subscriptionPlanInfoCache *cachex.HybridCache[SubscriptionPlanInfo]
)

func subscriptionPlanCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_TTL", 300)
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanInfoCacheTTL() time.Duration {
	ttlSeconds := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_TTL", 120)
	if ttlSeconds <= 0 {
		ttlSeconds = 120
	}
	return time.Duration(ttlSeconds) * time.Second
}

func subscriptionPlanCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_CACHE_CAP", 5000)
	if capacity <= 0 {
		capacity = 5000
	}
	return capacity
}

func subscriptionPlanInfoCacheCapacity() int {
	capacity := common.GetEnvOrDefault("SUBSCRIPTION_PLAN_INFO_CACHE_CAP", 10000)
	if capacity <= 0 {
		capacity = 10000
	}
	return capacity
}

func getSubscriptionPlanCache() *cachex.HybridCache[SubscriptionPlan] {
	subscriptionPlanCacheOnce.Do(func() {
		ttl := subscriptionPlanCacheTTL()
		subscriptionPlanCache = cachex.NewHybridCache[SubscriptionPlan](cachex.HybridCacheConfig[SubscriptionPlan]{
			Namespace: cachex.Namespace(subscriptionPlanCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlan]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlan] {
				return hot.NewHotCache[string, SubscriptionPlan](hot.LRU, subscriptionPlanCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanCache
}

func getSubscriptionPlanInfoCache() *cachex.HybridCache[SubscriptionPlanInfo] {
	subscriptionPlanInfoCacheOnce.Do(func() {
		ttl := subscriptionPlanInfoCacheTTL()
		subscriptionPlanInfoCache = cachex.NewHybridCache[SubscriptionPlanInfo](cachex.HybridCacheConfig[SubscriptionPlanInfo]{
			Namespace: cachex.Namespace(subscriptionPlanInfoCacheNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.JSONCodec[SubscriptionPlanInfo]{},
			Memory: func() *hot.HotCache[string, SubscriptionPlanInfo] {
				return hot.NewHotCache[string, SubscriptionPlanInfo](hot.LRU, subscriptionPlanInfoCacheCapacity()).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return subscriptionPlanInfoCache
}

func subscriptionPlanCacheKey(id int) string {
	if id <= 0 {
		return ""
	}
	return strconv.Itoa(id)
}

func InvalidateSubscriptionPlanCache(planId int) {
	if planId <= 0 {
		return
	}
	cache := getSubscriptionPlanCache()
	_, _ = cache.DeleteMany([]string{subscriptionPlanCacheKey(planId)})
	infoCache := getSubscriptionPlanInfoCache()
	_ = infoCache.Purge()
}

// Subscription plan
type SubscriptionPlan struct {
	Id int `json:"id"`

	Title    string `json:"title" gorm:"type:varchar(128);not null"`
	Subtitle string `json:"subtitle" gorm:"type:varchar(255);default:''"`

	// Display money amount (follow existing code style: float64 for money)
	PriceAmount float64 `json:"price_amount" gorm:"type:decimal(10,6);not null;default:0"`
	Currency    string  `json:"currency" gorm:"type:varchar(8);not null;default:'CNY'"`
	// BillingRatio is the fixed consumption multiplier for subscriptions.
	// A zero value uses the product default for known model families. Plans
	// without either an explicit ratio or a known-family default cannot be sold.
	BillingRatio float64 `json:"billing_ratio" gorm:"type:decimal(10,6);not null;default:0"`

	DurationUnit  string `json:"duration_unit" gorm:"type:varchar(16);not null;default:'month'"`
	DurationValue int    `json:"duration_value" gorm:"type:int;not null;default:1"`
	CustomSeconds int64  `json:"custom_seconds" gorm:"type:bigint;not null;default:0"`

	Enabled   bool `json:"enabled" gorm:"default:true"`
	SortOrder int  `json:"sort_order" gorm:"type:int;default:0"`

	AllowBalancePay *bool `json:"allow_balance_pay"`

	// Allow falling back to wallet balance after subscription quota is exhausted (empty = true)
	AllowWalletOverflow *bool `json:"allow_wallet_overflow"`

	StripePriceId         string `json:"stripe_price_id" gorm:"type:varchar(128);default:''"`
	CreemProductId        string `json:"creem_product_id" gorm:"type:varchar(128);default:''"`
	WaffoPancakeProductId string `json:"waffo_pancake_product_id" gorm:"type:varchar(128);default:''"`

	// Max purchases per user (0 = unlimited)
	MaxPurchasePerUser int `json:"max_purchase_per_user" gorm:"type:int;default:0"`

	// Upgrade user group after purchase (empty = no change)
	UpgradeGroup string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`

	// Downgrade user group on expiry (empty = revert to the group held before purchase)
	DowngradeGroup   string   `json:"downgrade_group" gorm:"type:varchar(64);default:''"`
	ApplicableGroups []string `json:"applicable_groups" gorm:"type:text;serializer:json"`

	// Model family and explicit model allow-list used by the subscription catalog.
	// Empty values keep backwards compatibility with plans created before model
	// packages were introduced; the frontend then falls back to a general package.
	ModelFamily    string   `json:"model_family" gorm:"type:varchar(32);default:''"`
	IncludedModels []string `json:"included_models" gorm:"type:text;serializer:json"`

	// Optional public card presentation. Empty benefits keep the generated
	// compatibility copy on the public subscription page.
	BadgeText     string   `json:"badge_text" gorm:"type:varchar(64);default:''"`
	IsRecommended bool     `json:"is_recommended"`
	Benefits      []string `json:"benefits" gorm:"type:text;serializer:json"`

	// Optional model-specific entitlements gifted with this plan. These are
	// copied into UserSubscription.ResourceGrants when the plan is purchased.
	BonusResources []SubscriptionBonusResource `json:"bonus_resources" gorm:"type:text;serializer:json"`

	// Total quota (amount in quota units, 0 = unlimited)
	TotalAmount int64 `json:"total_amount" gorm:"type:bigint;not null;default:0"`

	// Independent calendar-period quotas (0 = unlimited)
	DailyAmount   int64 `json:"daily_amount" gorm:"type:bigint;not null;default:0"`
	WeeklyAmount  int64 `json:"weekly_amount" gorm:"type:bigint;not null;default:0"`
	MonthlyAmount int64 `json:"monthly_amount" gorm:"type:bigint;not null;default:0"`

	// Quota reset period for plan
	QuotaResetPeriod        string `json:"quota_reset_period" gorm:"type:varchar(16);default:'never'"`
	QuotaResetCustomSeconds int64  `json:"quota_reset_custom_seconds" gorm:"type:bigint;default:0"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (p *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *SubscriptionPlan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

func (p *SubscriptionPlan) NormalizeDefaults() {
	if p.AllowBalancePay == nil {
		p.AllowBalancePay = common.GetPointer(true)
	}
	if p.AllowWalletOverflow == nil {
		p.AllowWalletOverflow = common.GetPointer(true)
	}
	p.ApplicableGroups = normalizeSubscriptionGroups(p.ApplicableGroups)
	p.ModelFamily = normalizeSubscriptionModelFamily(p.ModelFamily)
	p.IncludedModels = normalizeSubscriptionModels(p.IncludedModels)
	p.BadgeText = normalizeSubscriptionBadgeText(p.BadgeText)
	p.Benefits = normalizeSubscriptionBenefits(p.Benefits)
	p.BonusResources = normalizeSubscriptionBonusResources(p.BonusResources)
}

// EffectiveBillingRatio returns the fixed multiplier used by subscriptions
// from this plan. Known model families have product defaults so a plan created
// before billing_ratio was added is still protected from live group changes.
func (p *SubscriptionPlan) EffectiveBillingRatio() float64 {
	if p == nil {
		return 0
	}
	if p.BillingRatio > 0 && p.BillingRatio <= SubscriptionMaxBillingRatio && !math.IsNaN(p.BillingRatio) && !math.IsInf(p.BillingRatio, 0) {
		return p.BillingRatio
	}
	switch strings.TrimSpace(strings.ToLower(p.ModelFamily)) {
	case "ccmax", "cc max":
		return 0.9
	case "kiro-claude", "kiro claude", "kiro":
		return 0.17
	case "gpt", "openai":
		return 0.14
	case "gemini", "google":
		return 0.4
	case "chinese", "国产", "cn":
		return 0.35
	case "gpt-image", "gpt image":
		return 0.1
	case "banana", "nano banana", "香蕉生图":
		return 0.15
	default:
		return 0
	}
}

func normalizeSubscriptionModelFamily(family string) string {
	family = strings.TrimSpace(strings.ToLower(family))
	if len(family) > 32 {
		return family[:32]
	}
	return family
}

func normalizeSubscriptionModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, modelName := range models {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" || len(modelName) > 128 {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		result = append(result, modelName)
		if len(result) >= 128 {
			break
		}
	}
	return result
}

func normalizeSubscriptionBadgeText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 64 {
		return text[:64]
	}
	return text
}

func normalizeSubscriptionBenefits(benefits []string) []string {
	seen := make(map[string]struct{}, len(benefits))
	result := make([]string, 0, len(benefits))
	for _, benefit := range benefits {
		benefit = strings.TrimSpace(benefit)
		if benefit == "" || len(benefit) > 240 {
			continue
		}
		if _, ok := seen[benefit]; ok {
			continue
		}
		seen[benefit] = struct{}{}
		result = append(result, benefit)
		if len(result) >= 12 {
			break
		}
	}
	return result
}

func normalizeSubscriptionBonusResources(resources []SubscriptionBonusResource) []SubscriptionBonusResource {
	seen := make(map[string]struct{}, len(resources))
	result := make([]SubscriptionBonusResource, 0, len(resources))
	for _, resource := range resources {
		resource.ResourceKey = strings.TrimSpace(strings.ToLower(resource.ResourceKey))
		resource.ResourceType = strings.TrimSpace(strings.ToLower(resource.ResourceType))
		resource.ModelName = strings.TrimSpace(resource.ModelName)
		resource.DisplayName = strings.TrimSpace(resource.DisplayName)
		if resource.ResourceKey == "" {
			resource.ResourceKey = strings.ToLower(resource.ModelName)
		}
		if resource.ResourceType == "" {
			resource.ResourceType = SubscriptionResourceTypeQuota
		}
		if resource.ResourceKey == "" || resource.ModelName == "" || resource.Amount <= 0 {
			continue
		}
		if resource.ResourceType != SubscriptionResourceTypeQuota && resource.ResourceType != SubscriptionResourceTypeImageCount {
			continue
		}
		if len(resource.ResourceKey) > 128 || len(resource.ModelName) > 128 || len(resource.DisplayName) > 128 {
			continue
		}
		if _, ok := seen[resource.ResourceKey+"\x00"+resource.ResourceType]; ok {
			continue
		}
		seen[resource.ResourceKey+"\x00"+resource.ResourceType] = struct{}{}
		result = append(result, resource)
		if len(result) >= SubscriptionMaxBonusResources {
			break
		}
	}
	return result
}

func normalizeSubscriptionGroups(groups []string) []string {
	seen := make(map[string]struct{}, len(groups))
	result := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" || len(group) > 64 {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		result = append(result, group)
	}
	return result
}

func NormalizeSubscriptionApplicableGroups(groups []string) []string {
	return normalizeSubscriptionGroups(groups)
}

func subscriptionGroupMatches(groups []string, effective string) bool {
	if len(groups) == 0 {
		return true
	}
	effective = strings.TrimSpace(effective)
	for _, group := range groups {
		if group == effective {
			return true
		}
	}
	return false
}

func normalizeSubscriptionResourceType(resourceType string) string {
	resourceType = strings.TrimSpace(strings.ToLower(resourceType))
	if resourceType == SubscriptionResourceTypeImageCount {
		return SubscriptionResourceTypeImageCount
	}
	return SubscriptionResourceTypeQuota
}

func modelNameMatchesSubscriptionValue(modelName, value string) bool {
	modelName = strings.TrimSpace(strings.ToLower(modelName))
	value = strings.TrimSpace(strings.ToLower(value))
	if modelName == "" || value == "" {
		return false
	}
	return modelName == value || strings.HasPrefix(modelName, value+"-") || strings.HasPrefix(modelName, value+"/")
}

func subscriptionResourceMatches(resource SubscriptionResourceGrant, resourceKey, modelName string) bool {
	return modelNameMatchesSubscriptionValue(modelName, resource.ResourceKey) ||
		modelNameMatchesSubscriptionValue(modelName, resource.ModelName) ||
		modelNameMatchesSubscriptionValue(resourceKey, resource.ResourceKey) ||
		modelNameMatchesSubscriptionValue(resourceKey, resource.ModelName)
}

func subscriptionModelMatches(sub UserSubscription, modelName string) bool {
	// Legacy subscriptions have no model snapshot and remain general-purpose.
	if len(sub.IncludedModels) == 0 && strings.TrimSpace(sub.ModelFamily) == "" {
		return true
	}
	// An explicit allow-list is authoritative. It must not silently widen to
	// the family matcher when a model was intentionally omitted.
	if len(sub.IncludedModels) > 0 {
		for _, included := range sub.IncludedModels {
			if modelNameMatchesSubscriptionValue(modelName, included) {
				return true
			}
		}
		return false
	}
	family := strings.TrimSpace(strings.ToLower(sub.ModelFamily))
	if family == "" || family == "all" {
		return family == "all"
	}
	model := strings.ToLower(strings.TrimSpace(modelName))
	switch family {
	case "ccmax", "cc max", "kiro-claude", "kiro claude", "kiro", "claude", "anthropic":
		return strings.Contains(model, "claude") || strings.Contains(model, "anthropic")
	case "gpt", "openai":
		return (strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "openai/")) && !strings.Contains(model, "gpt-image")
	case "gpt-image", "gpt image":
		return strings.HasPrefix(model, "gpt-image-")
	case "gemini", "google":
		return (strings.HasPrefix(model, "gemini") || strings.HasPrefix(model, "google/")) && !isSubscriptionImageModelName(model)
	case "banana", "nano banana", "香蕉生图":
		return isSubscriptionImageModelName(model)
	case "chinese", "国产", "cn":
		for _, prefix := range []string{"deepseek", "glm", "kimi", "minimax"} {
			if strings.HasPrefix(model, prefix) {
				return true
			}
		}
	}
	return false
}

func isSubscriptionImageModelName(model string) bool {
	return strings.HasPrefix(model, "gpt-image-2") ||
		strings.HasPrefix(model, "gemini-3-pro-image") ||
		strings.HasPrefix(model, "gemini-3.1-flash-image") ||
		model == "gemini-2.5-flash-image" ||
		model == "gemini-2.0-flash-exp-image-generation" ||
		model == "gemini-2.0-flash-exp" ||
		model == "nano-banana-pro-preview"
}

func subscriptionCanServeModel(sub UserSubscription, modelName string) bool {
	if subscriptionModelMatches(sub, modelName) {
		return true
	}
	for _, grant := range sub.ResourceGrants {
		if subscriptionResourceMatches(grant, modelName, modelName) && subscriptionUsageHasCapacity(grant.Amount, grant.Used, 1) {
			return true
		}
	}
	return false
}

// Subscription order (payment -> webhook -> create UserSubscription)
type SubscriptionOrder struct {
	Id     int     `json:"id"`
	UserId int     `json:"user_id" gorm:"index"`
	PlanId int     `json:"plan_id" gorm:"index"`
	Money  float64 `json:"money"`

	TradeNo         string `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	Status          string `json:"status"`
	CreateTime      int64  `json:"create_time"`
	CompleteTime    int64  `json:"complete_time"`

	ProviderPayload string `json:"provider_payload" gorm:"type:text"`
}

func (o *SubscriptionOrder) Insert() error {
	if o.CreateTime == 0 {
		o.CreateTime = common.GetTimestamp()
	}
	return DB.Create(o).Error
}

func (o *SubscriptionOrder) Update() error {
	return DB.Save(o).Error
}

func GetSubscriptionOrderByTradeNo(tradeNo string) *SubscriptionOrder {
	if tradeNo == "" {
		return nil
	}
	var order SubscriptionOrder
	if err := DB.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		return nil
	}
	return &order
}

// User subscription instance
type UserSubscription struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"index;index:idx_user_sub_active,priority:1"`
	PlanId int `json:"plan_id" gorm:"index"`

	AmountTotal int64 `json:"amount_total" gorm:"type:bigint;not null;default:0"`
	AmountUsed  int64 `json:"amount_used" gorm:"type:bigint;not null;default:0"`

	DailyAmount    int64 `json:"daily_amount" gorm:"type:bigint;not null;default:0"`
	DailyUsed      int64 `json:"daily_used" gorm:"type:bigint;not null;default:0"`
	DailyResetTime int64 `json:"daily_reset_time" gorm:"type:bigint;default:0"`

	WeeklyAmount    int64 `json:"weekly_amount" gorm:"type:bigint;not null;default:0"`
	WeeklyUsed      int64 `json:"weekly_used" gorm:"type:bigint;not null;default:0"`
	WeeklyResetTime int64 `json:"weekly_reset_time" gorm:"type:bigint;default:0"`

	MonthlyAmount    int64 `json:"monthly_amount" gorm:"type:bigint;not null;default:0"`
	MonthlyUsed      int64 `json:"monthly_used" gorm:"type:bigint;not null;default:0"`
	MonthlyResetTime int64 `json:"monthly_reset_time" gorm:"type:bigint;default:0"`
	// Version 1 anchors recurring quota windows to the subscription start time.
	PeriodResetAnchorVersion int `json:"period_reset_anchor_version" gorm:"column:period_reset_anchor_version"`

	StartTime int64  `json:"start_time" gorm:"bigint"`
	EndTime   int64  `json:"end_time" gorm:"bigint;index;index:idx_user_sub_active,priority:3"`
	Status    string `json:"status" gorm:"type:varchar(32);index;index:idx_user_sub_active,priority:2"` // active/expired/cancelled

	Source string `json:"source" gorm:"type:varchar(32);default:'order'"` // order/admin

	LastResetTime int64 `json:"last_reset_time" gorm:"type:bigint;default:0"`
	NextResetTime int64 `json:"next_reset_time" gorm:"type:bigint;default:0;index"`

	UpgradeGroup  string `json:"upgrade_group" gorm:"type:varchar(64);default:''"`
	PrevUserGroup string `json:"prev_user_group" gorm:"type:varchar(64);default:''"`

	// Downgrade target group on expiry (snapshot from plan; empty = revert to PrevUserGroup)
	DowngradeGroup   string   `json:"downgrade_group" gorm:"type:varchar(64);default:''"`
	ApplicableGroups []string `json:"applicable_groups" gorm:"type:text;serializer:json"`

	// Whether wallet fallback is allowed after this subscription's quota is exhausted (snapshot from plan)
	AllowWalletOverflow bool `json:"allow_wallet_overflow"`

	// Model access and gifted resources are snapshots from the purchased plan.
	ModelFamily    string   `json:"model_family" gorm:"type:varchar(32);default:''"`
	IncludedModels []string `json:"included_models" gorm:"type:text;serializer:json"`
	// BillingRatio is snapshotted at purchase time so later group-ratio edits do
	// not change the consumption rate of an already purchased package.
	BillingRatio   float64                     `json:"billing_ratio" gorm:"type:decimal(10,6);not null;default:0"`
	ResourceGrants []SubscriptionResourceGrant `json:"resource_grants" gorm:"type:text;serializer:json"`

	CreatedAt int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt int64 `json:"updated_at" gorm:"bigint"`
}

func (s *UserSubscription) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	s.CreatedAt = now
	s.UpdatedAt = now
	return nil
}

func (s *UserSubscription) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	return nil
}

type SubscriptionSummary struct {
	Subscription *UserSubscription `json:"subscription"`
}

type SubscriptionResetResult struct {
	PlanId           int    `json:"plan_id"`
	MatchedCount     int    `json:"matched_count"`
	ResetCount       int    `json:"reset_count"`
	UserCount        int    `json:"user_count"`
	AdvanceResetTime bool   `json:"advance_reset_time"`
	PlanTitle        string `json:"-"`
	AffectedUserIds  []int  `json:"-"`
}

func calcPlanEndTime(start time.Time, plan *SubscriptionPlan) (int64, error) {
	if plan == nil {
		return 0, errors.New("plan is nil")
	}
	if plan.DurationValue <= 0 && plan.DurationUnit != SubscriptionDurationCustom {
		return 0, errors.New("duration_value must be > 0")
	}
	switch plan.DurationUnit {
	case SubscriptionDurationYear:
		return start.AddDate(plan.DurationValue, 0, 0).Unix(), nil
	case SubscriptionDurationMonth:
		return start.AddDate(0, plan.DurationValue, 0).Unix(), nil
	case SubscriptionDurationDay:
		return start.Add(time.Duration(plan.DurationValue) * 24 * time.Hour).Unix(), nil
	case SubscriptionDurationHour:
		return start.Add(time.Duration(plan.DurationValue) * time.Hour).Unix(), nil
	case SubscriptionDurationCustom:
		if plan.CustomSeconds <= 0 {
			return 0, errors.New("custom_seconds must be > 0")
		}
		return start.Add(time.Duration(plan.CustomSeconds) * time.Second).Unix(), nil
	default:
		return 0, fmt.Errorf("invalid duration_unit: %s", plan.DurationUnit)
	}
}

func NormalizeResetPeriod(period string) string {
	switch strings.TrimSpace(period) {
	case SubscriptionResetDaily, SubscriptionResetWeekly, SubscriptionResetMonthly, SubscriptionResetCustom:
		return strings.TrimSpace(period)
	default:
		return SubscriptionResetNever
	}
}

func calcNextAnchoredPeriodReset(anchor time.Time, after time.Time, period string, endUnix int64) int64 {
	if after.Before(anchor) {
		after = anchor
	}
	var next time.Time
	switch period {
	case SubscriptionResetDaily:
		anchorDate := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, time.UTC)
		afterDate := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, time.UTC)
		days := int(afterDate.Sub(anchorDate) / (24 * time.Hour))
		next = anchor.AddDate(0, 0, days)
		if !next.After(after) {
			next = next.AddDate(0, 0, 1)
		}
	case SubscriptionResetWeekly:
		anchorDate := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, time.UTC)
		afterDate := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, time.UTC)
		days := int(afterDate.Sub(anchorDate) / (24 * time.Hour))
		weeks := days / 7
		next = anchor.AddDate(0, 0, weeks*7)
		if !next.After(after) {
			next = next.AddDate(0, 0, 7)
		}
	case SubscriptionResetMonthly:
		months := (after.Year()-anchor.Year())*12 + int(after.Month()-anchor.Month())
		if months < 0 {
			months = 0
		}
		for {
			monthStart := time.Date(anchor.Year(), anchor.Month()+time.Month(months), 1, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location())
			lastDay := time.Date(monthStart.Year(), monthStart.Month()+1, 0, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location()).Day()
			day := anchor.Day()
			if day > lastDay {
				day = lastDay
			}
			next = time.Date(monthStart.Year(), monthStart.Month(), day, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location())
			if next.After(after) {
				break
			}
			months++
		}
	default:
		return 0
	}
	if endUnix > 0 && next.Unix() > endUnix {
		return 0
	}
	return next.Unix()
}

func calcNextResetTime(anchor time.Time, after time.Time, plan *SubscriptionPlan, endUnix int64) int64 {
	if plan == nil {
		return 0
	}
	period := NormalizeResetPeriod(plan.QuotaResetPeriod)
	if period == SubscriptionResetNever {
		return 0
	}
	if period == SubscriptionResetCustom {
		if plan.QuotaResetCustomSeconds <= 0 {
			return 0
		}
		next := after.Add(time.Duration(plan.QuotaResetCustomSeconds) * time.Second)
		if endUnix > 0 && next.Unix() > endUnix {
			return 0
		}
		return next.Unix()
	}
	return calcNextAnchoredPeriodReset(anchor, after, period, endUnix)
}

func GetSubscriptionPlanById(id int) (*SubscriptionPlan, error) {
	return getSubscriptionPlanByIdTx(nil, id)
}

func getSubscriptionPlanByIdTx(tx *gorm.DB, id int) (*SubscriptionPlan, error) {
	if id <= 0 {
		return nil, errors.New("invalid plan id")
	}
	key := subscriptionPlanCacheKey(id)
	if key != "" {
		if cached, found, err := getSubscriptionPlanCache().Get(key); err == nil && found {
			cached.NormalizeDefaults()
			return &cached, nil
		}
	}
	var plan SubscriptionPlan
	query := DB
	if tx != nil {
		query = tx
	}
	if err := query.Where("id = ?", id).First(&plan).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	_ = getSubscriptionPlanCache().SetWithTTL(key, plan, subscriptionPlanCacheTTL())
	return &plan, nil
}

func CountUserSubscriptionsByPlan(userId int, planId int) (int64, error) {
	if userId <= 0 || planId <= 0 {
		return 0, errors.New("invalid userId or planId")
	}
	var count int64
	if err := DB.Model(&UserSubscription{}).
		Where("user_id = ? AND plan_id = ?", userId, planId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func getUserGroupByIdTx(tx *gorm.DB, userId int) (string, error) {
	if userId <= 0 {
		return "", errors.New("invalid userId")
	}
	if tx == nil {
		tx = DB
	}
	var group string
	if err := tx.Model(&User{}).Where("id = ?", userId).Select(commonGroupCol).Find(&group).Error; err != nil {
		return "", err
	}
	return group, nil
}

func downgradeUserGroupForSubscriptionTx(tx *gorm.DB, sub *UserSubscription, now int64) (string, error) {
	if tx == nil || sub == nil {
		return "", errors.New("invalid downgrade args")
	}
	downgradeGroup := strings.TrimSpace(sub.DowngradeGroup)
	upgradeGroup := strings.TrimSpace(sub.UpgradeGroup)
	// Nothing to do if neither an explicit downgrade target nor an upgrade snapshot exists.
	if downgradeGroup == "" && upgradeGroup == "" {
		return "", nil
	}
	currentGroup, err := getUserGroupByIdTx(tx, sub.UserId)
	if err != nil {
		return "", err
	}
	// If another active upgraded subscription exists, keep the current group.
	var activeSub UserSubscription
	activeQuery := tx.Where("user_id = ? AND status = ? AND end_time > ? AND id <> ? AND upgrade_group <> ''",
		sub.UserId, "active", now, sub.Id).
		Order("end_time desc, id desc").
		Limit(1).
		Find(&activeSub)
	if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
		return "", nil
	}
	// Determine the downgrade target: an explicit downgrade group takes precedence,
	// otherwise revert to the group held before purchase (legacy behavior).
	target := downgradeGroup
	if target == "" {
		// Legacy behavior: only revert when the subscription actually elevated the user.
		if currentGroup != upgradeGroup {
			return "", nil
		}
		target = strings.TrimSpace(sub.PrevUserGroup)
	}
	if target == "" || target == currentGroup {
		return "", nil
	}
	if err := tx.Model(&User{}).Where("id = ?", sub.UserId).
		Update("group", target).Error; err != nil {
		return "", err
	}
	return target, nil
}

func CreateUserSubscriptionFromPlanTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, source string) (*UserSubscription, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if plan == nil || plan.Id == 0 {
		return nil, errors.New("invalid plan")
	}
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if plan.MaxPurchasePerUser > 0 {
		var count int64
		if err := tx.Model(&UserSubscription{}).
			Where("user_id = ? AND plan_id = ?", userId, plan.Id).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count >= int64(plan.MaxPurchasePerUser) {
			return nil, errors.New("已达到该套餐购买上限")
		}
	}
	nowUnix := getDBTimestampTx(tx)
	now := time.Unix(nowUnix, 0)
	endUnix, err := calcPlanEndTime(now, plan)
	if err != nil {
		return nil, err
	}
	resetBase := now
	nextReset := calcNextResetTime(resetBase, resetBase, plan, endUnix)
	lastReset := int64(0)
	if nextReset > 0 {
		lastReset = now.Unix()
	}
	dailyReset := int64(0)
	if plan.DailyAmount > 0 {
		dailyReset = calcNextAnchoredPeriodReset(now, now, SubscriptionResetDaily, endUnix)
	}
	weeklyReset := int64(0)
	if plan.WeeklyAmount > 0 {
		weeklyReset = calcNextAnchoredPeriodReset(now, now, SubscriptionResetWeekly, endUnix)
	}
	monthlyReset := int64(0)
	if plan.MonthlyAmount > 0 {
		monthlyReset = calcNextAnchoredPeriodReset(now, now, SubscriptionResetMonthly, endUnix)
	}
	upgradeGroup := strings.TrimSpace(plan.UpgradeGroup)
	prevGroup := ""
	if upgradeGroup != "" {
		currentGroup, err := getUserGroupByIdTx(tx, userId)
		if err != nil {
			return nil, err
		}
		if currentGroup != upgradeGroup {
			prevGroup = currentGroup
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("group", upgradeGroup).Error; err != nil {
				return nil, err
			}
		}
	}
	allowWalletOverflow := true
	if plan.AllowWalletOverflow != nil {
		allowWalletOverflow = *plan.AllowWalletOverflow
	}
	bonusResources := normalizeSubscriptionBonusResources(plan.BonusResources)
	resourceGrants := make([]SubscriptionResourceGrant, 0, len(bonusResources))
	for _, resource := range bonusResources {
		resourceGrants = append(resourceGrants, SubscriptionResourceGrant{
			ResourceKey:  resource.ResourceKey,
			ResourceType: resource.ResourceType,
			ModelName:    resource.ModelName,
			DisplayName:  resource.DisplayName,
			Amount:       resource.Amount,
		})
	}
	billingRatio := plan.EffectiveBillingRatio()
	if billingRatio <= 0 || billingRatio > SubscriptionMaxBillingRatio || math.IsNaN(billingRatio) || math.IsInf(billingRatio, 0) {
		return nil, errors.New("subscription plan has no valid fixed billing ratio")
	}
	sub := &UserSubscription{
		UserId:                   userId,
		PlanId:                   plan.Id,
		AmountTotal:              plan.TotalAmount,
		AmountUsed:               0,
		DailyAmount:              plan.DailyAmount,
		DailyUsed:                0,
		DailyResetTime:           dailyReset,
		WeeklyAmount:             plan.WeeklyAmount,
		WeeklyUsed:               0,
		WeeklyResetTime:          weeklyReset,
		MonthlyAmount:            plan.MonthlyAmount,
		MonthlyUsed:              0,
		MonthlyResetTime:         monthlyReset,
		PeriodResetAnchorVersion: subscriptionPeriodResetAnchorVersion,
		StartTime:                now.Unix(),
		EndTime:                  endUnix,
		Status:                   "active",
		Source:                   source,
		LastResetTime:            lastReset,
		NextResetTime:            nextReset,
		UpgradeGroup:             upgradeGroup,
		PrevUserGroup:            prevGroup,
		DowngradeGroup:           strings.TrimSpace(plan.DowngradeGroup),
		ApplicableGroups:         normalizeSubscriptionGroups(plan.ApplicableGroups),
		AllowWalletOverflow:      allowWalletOverflow,
		ModelFamily:              normalizeSubscriptionModelFamily(plan.ModelFamily),
		IncludedModels:           normalizeSubscriptionModels(plan.IncludedModels),
		BillingRatio:             billingRatio,
		ResourceGrants:           resourceGrants,
		CreatedAt:                common.GetTimestamp(),
		UpdatedAt:                common.GetTimestamp(),
	}
	if err := tx.Create(sub).Error; err != nil {
		return nil, err
	}
	return sub, nil
}

// migrateActiveSubscriptionBillingRatios backfills the fixed-rate snapshot for
// active subscriptions created before BillingRatio was introduced. New
// purchases are populated by CreateUserSubscriptionFromPlanTx; this migration
// keeps already purchased, still-valid packages from silently switching to a
// live group ratio after an administrator edits pricing.
func migrateActiveSubscriptionBillingRatios() error {
	now := GetDBTimestamp()
	var candidates []UserSubscription
	if err := DB.Where(
		"status = ? AND end_time > ? AND COALESCE(billing_ratio, 0) <= ?",
		"active", now, 0,
	).Order("id asc").Find(&candidates).Error; err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := DB.Transaction(func(tx *gorm.DB) error {
			var sub UserSubscription
			if err := lockForUpdate(tx).
				Where("id = ? AND status = ? AND end_time > ? AND COALESCE(billing_ratio, 0) <= ?",
					candidate.Id, "active", now, 0).
				First(&sub).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}

			ratio := 0.0
			if sub.PlanId > 0 {
				plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				if plan != nil {
					ratio = plan.EffectiveBillingRatio()
				}
			}
			if ratio <= 0 {
				ratio = (&SubscriptionPlan{ModelFamily: sub.ModelFamily}).EffectiveBillingRatio()
			}
			if ratio <= 0 || ratio > SubscriptionMaxBillingRatio || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
				return nil
			}
			return tx.Model(&UserSubscription{}).
				Where("id = ? AND COALESCE(billing_ratio, 0) <= ?", sub.Id, 0).
				Update("billing_ratio", ratio).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

// Complete a subscription order (idempotent). Creates a UserSubscription snapshot from the plan.
// expectedPaymentProvider guards against cross-gateway callback attacks (empty skips the check).
// actualPaymentMethod updates the order's PaymentMethod to reflect the real payment type used (empty skips update).
func CompleteSubscriptionOrder(tradeNo string, providerPayload string, expectedPaymentProvider string, actualPaymentMethod string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	var logUserId int
	var logPlanTitle string
	var logMoney float64
	var logPaymentMethod string
	var upgradeGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status == common.TopUpStatusSuccess {
			return nil
		}
		if order.Status != common.TopUpStatusPending {
			return ErrSubscriptionOrderStatusInvalid
		}
		plan, err := GetSubscriptionPlanById(order.PlanId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			// still allow completion for already purchased orders
		}
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		_, err = CreateUserSubscriptionFromPlanTx(tx, order.UserId, plan, "order")
		if err != nil {
			return err
		}
		if err := upsertSubscriptionTopUpTx(tx, &order); err != nil {
			return err
		}
		order.Status = common.TopUpStatusSuccess
		order.CompleteTime = common.GetTimestamp()
		if providerPayload != "" {
			order.ProviderPayload = providerPayload
		}
		if actualPaymentMethod != "" && order.PaymentMethod != actualPaymentMethod {
			order.PaymentMethod = actualPaymentMethod
		}
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		logUserId = order.UserId
		logPlanTitle = plan.Title
		logMoney = order.Money
		logPaymentMethod = order.PaymentMethod
		return nil
	})
	if err != nil {
		return err
	}
	if upgradeGroup != "" && logUserId > 0 {
		_ = UpdateUserGroupCache(logUserId, upgradeGroup)
	}
	if logUserId > 0 {
		msg := fmt.Sprintf("订阅购买成功，套餐: %s，支付金额: %.2f，支付方式: %s", logPlanTitle, logMoney, logPaymentMethod)
		RecordLog(logUserId, LogTypeTopup, msg)
	}
	return nil
}

func upsertSubscriptionTopUpTx(tx *gorm.DB, order *SubscriptionOrder) error {
	if tx == nil || order == nil {
		return errors.New("invalid subscription order")
	}
	now := common.GetTimestamp()
	var topup TopUp
	if err := tx.Where("trade_no = ?", order.TradeNo).First(&topup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			topup = TopUp{
				UserId:        order.UserId,
				Amount:        0,
				Money:         order.Money,
				TradeNo:       order.TradeNo,
				PaymentMethod: order.PaymentMethod,
				CreateTime:    order.CreateTime,
				CompleteTime:  now,
				Status:        common.TopUpStatusSuccess,
			}
			return tx.Create(&topup).Error
		}
		return err
	}
	topup.Money = order.Money
	if topup.PaymentMethod == "" {
		topup.PaymentMethod = order.PaymentMethod
	} else if topup.PaymentMethod != order.PaymentMethod {
		return ErrPaymentMethodMismatch
	}
	if topup.CreateTime == 0 {
		topup.CreateTime = order.CreateTime
	}
	topup.CompleteTime = now
	topup.Status = common.TopUpStatusSuccess
	return tx.Save(&topup).Error
}

func ExpireSubscriptionOrder(tradeNo string, expectedPaymentProvider string) error {
	if tradeNo == "" {
		return errors.New("tradeNo is empty")
	}
	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order SubscriptionOrder
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&order).Error; err != nil {
			return ErrSubscriptionOrderNotFound
		}
		if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if order.Status != common.TopUpStatusPending {
			return nil
		}
		order.Status = common.TopUpStatusExpired
		order.CompleteTime = common.GetTimestamp()
		return tx.Save(&order).Error
	})
}

func anchoredPeriodWindowStart(anchor, now time.Time, period string) int64 {
	next := calcNextAnchoredPeriodReset(anchor, now, period, 0)
	if next == 0 {
		return anchor.Unix()
	}
	nextTime := time.Unix(next, 0).In(anchor.Location())
	switch period {
	case SubscriptionResetDaily:
		return nextTime.AddDate(0, 0, -1).Unix()
	case SubscriptionResetWeekly:
		return nextTime.AddDate(0, 0, -7).Unix()
	case SubscriptionResetMonthly:
		months := (nextTime.Year()-anchor.Year())*12 + int(nextTime.Month()-anchor.Month()) - 1
		monthStart := time.Date(anchor.Year(), anchor.Month()+time.Month(months), 1, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location())
		lastDay := time.Date(monthStart.Year(), monthStart.Month()+1, 0, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location()).Day()
		day := anchor.Day()
		if day > lastDay {
			day = lastDay
		}
		return time.Date(monthStart.Year(), monthStart.Month(), day, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location()).Unix()
	default:
		return anchor.Unix()
	}
}

func addImportedSubscriptionUsage(current, amount int64) int64 {
	if amount <= 0 {
		return current
	}
	if current > math.MaxInt64-amount {
		return math.MaxInt64
	}
	return current + amount
}

func aggregateSubscriptionWalletUsage(userId int, startTime, dailyStart, weeklyStart, monthlyStart int64, groups []string) (int64, int64, int64, int64, error) {
	query := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at <= ?", userId, LogTypeConsume, startTime, GetDBTimestamp())
	if len(groups) > 0 {
		query = query.Where(map[string]interface{}{"group": groups})
	}
	rows, err := query.Select("created_at, quota, other").Rows()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer rows.Close()
	var amountUsed, dailyUsed, weeklyUsed, monthlyUsed int64
	for rows.Next() {
		var createdAt, quota int64
		var other string
		if err := rows.Scan(&createdAt, &quota, &other); err != nil {
			return 0, 0, 0, 0, err
		}
		if quota <= 0 {
			continue
		}
		if other != "" {
			var metadata struct {
				BillingSource string `json:"billing_source"`
			}
			if err := common.UnmarshalJsonStr(other, &metadata); err == nil && metadata.BillingSource == "subscription" {
				continue
			}
		}
		amountUsed = addImportedSubscriptionUsage(amountUsed, quota)
		if createdAt >= dailyStart {
			dailyUsed = addImportedSubscriptionUsage(dailyUsed, quota)
		}
		if createdAt >= weeklyStart {
			weeklyUsed = addImportedSubscriptionUsage(weeklyUsed, quota)
		}
		if createdAt >= monthlyStart {
			monthlyUsed = addImportedSubscriptionUsage(monthlyUsed, quota)
		}
	}
	return amountUsed, dailyUsed, weeklyUsed, monthlyUsed, rows.Err()
}

func PreviewAdminSubscriptionUsage(userId, planId int, effectiveStartTime int64) (*AdminSubscriptionUsagePreview, error) {
	return previewAdminSubscriptionUsage(userId, planId, effectiveStartTime, true)
}

func previewAdminSubscriptionUsage(userId, planId int, effectiveStartTime int64, includeUsage bool) (*AdminSubscriptionUsagePreview, error) {
	if userId <= 0 || planId <= 0 {
		return nil, errors.New("invalid userId or planId")
	}
	nowUnix := GetDBTimestamp()
	if effectiveStartTime == 0 {
		effectiveStartTime = nowUnix
	}
	if effectiveStartTime > nowUnix {
		return nil, errors.New("effective start time cannot be in the future")
	}
	plan, err := GetSubscriptionPlanById(planId)
	if err != nil {
		return nil, err
	}
	start := time.Unix(effectiveStartTime, 0)
	endTime, err := calcPlanEndTime(start, plan)
	if err != nil {
		return nil, err
	}
	if endTime <= nowUnix {
		return nil, errors.New("subscription would already be expired at the selected start time")
	}
	groups := normalizeSubscriptionGroups(plan.ApplicableGroups)
	dailyStart := anchoredPeriodWindowStart(start, time.Unix(nowUnix, 0), SubscriptionResetDaily)
	weeklyStart := anchoredPeriodWindowStart(start, time.Unix(nowUnix, 0), SubscriptionResetWeekly)
	monthlyStart := anchoredPeriodWindowStart(start, time.Unix(nowUnix, 0), SubscriptionResetMonthly)
	var amountUsed, dailyUsed, weeklyUsed, monthlyUsed int64
	if includeUsage {
		amountUsed, dailyUsed, weeklyUsed, monthlyUsed, err = aggregateSubscriptionWalletUsage(userId, effectiveStartTime, dailyStart, weeklyStart, monthlyStart, groups)
		if err != nil {
			return nil, err
		}
	}
	walletQuota, err := GetUserQuota(userId, false)
	if err != nil {
		return nil, err
	}
	preview := &AdminSubscriptionUsagePreview{
		UserId: userId, PlanId: planId, EffectiveStartTime: effectiveStartTime, EndTime: endTime,
		ApplicableGroups: groups, AmountUsed: amountUsed, DailyUsed: dailyUsed, WeeklyUsed: weeklyUsed, MonthlyUsed: monthlyUsed,
		DailyWindowStart: dailyStart, WeeklyWindowStart: weeklyStart, MonthlyWindowStart: monthlyStart,
		AmountTotal: plan.TotalAmount, DailyAmount: plan.DailyAmount, WeeklyAmount: plan.WeeklyAmount, MonthlyAmount: plan.MonthlyAmount,
		WalletQuota: walletQuota, ConsumeLogEnabled: common.LogConsumeEnabled,
	}
	preview.ExceedsAmountTotal = preview.AmountTotal > 0 && preview.AmountUsed > preview.AmountTotal
	preview.ExceedsDailyAmount = preview.DailyAmount > 0 && preview.DailyUsed > preview.DailyAmount
	preview.ExceedsWeeklyAmount = preview.WeeklyAmount > 0 && preview.WeeklyUsed > preview.WeeklyAmount
	preview.ExceedsMonthlyAmount = preview.MonthlyAmount > 0 && preview.MonthlyUsed > preview.MonthlyAmount
	preview.ExceedsAnyLimit = preview.ExceedsAmountTotal || preview.ExceedsDailyAmount || preview.ExceedsWeeklyAmount || preview.ExceedsMonthlyAmount
	return preview, nil
}

// Admin bind (no payment). Creates a UserSubscription from a plan.
func AdminBindSubscriptionWithOptions(userId int, planId int, options AdminSubscriptionBindOptions) (*UserSubscription, *AdminSubscriptionUsagePreview, string, error) {
	if userId <= 0 || planId <= 0 {
		return nil, nil, "", errors.New("invalid userId or planId")
	}
	plan, err := GetSubscriptionPlanById(planId)
	if err != nil {
		return nil, nil, "", err
	}
	preview, err := previewAdminSubscriptionUsage(userId, planId, options.EffectiveStartTime, options.ImportUsage)
	if err != nil {
		return nil, nil, "", err
	}
	if options.ImportUsage && preview.ExceedsAnyLimit && !options.ConfirmOverLimit {
		return nil, preview, "", ErrSubscriptionImportedUsageOverLimit
	}
	var subscription *UserSubscription
	bindNow := time.Unix(GetDBTimestamp(), 0)
	pendingWalletDelta := 0
	hadPendingWalletDelta := false
	if options.ClearWallet && common.BatchUpdateEnabled {
		batchUpdateLocks[BatchUpdateTypeUserQuota].Lock()
		pendingWalletDelta, hadPendingWalletDelta = batchUpdateStores[BatchUpdateTypeUserQuota][userId]
		delete(batchUpdateStores[BatchUpdateTypeUserQuota], userId)
		defer batchUpdateLocks[BatchUpdateTypeUserQuota].Unlock()
	}
	groupChanged := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		var userRow User
		if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&userRow).Error; err != nil {
			return err
		}
		subscription, err = CreateUserSubscriptionFromPlanTx(tx, userId, plan, "admin")
		if err != nil {
			return err
		}
		groupChanged = subscription.PrevUserGroup != ""
		start := time.Unix(preview.EffectiveStartTime, 0)
		subscription.StartTime = preview.EffectiveStartTime
		subscription.EndTime = preview.EndTime
		subscription.DailyResetTime = 0
		if subscription.DailyAmount > 0 {
			subscription.DailyResetTime = calcNextAnchoredPeriodReset(start, bindNow, SubscriptionResetDaily, preview.EndTime)
		}
		subscription.WeeklyResetTime = 0
		if subscription.WeeklyAmount > 0 {
			subscription.WeeklyResetTime = calcNextAnchoredPeriodReset(start, bindNow, SubscriptionResetWeekly, preview.EndTime)
		}
		subscription.MonthlyResetTime = 0
		if subscription.MonthlyAmount > 0 {
			subscription.MonthlyResetTime = calcNextAnchoredPeriodReset(start, bindNow, SubscriptionResetMonthly, preview.EndTime)
		}
		subscription.NextResetTime = calcNextResetTime(start, bindNow, plan, preview.EndTime)
		if options.ImportUsage {
			subscription.AmountUsed = preview.AmountUsed
			subscription.DailyUsed = preview.DailyUsed
			subscription.WeeklyUsed = preview.WeeklyUsed
			subscription.MonthlyUsed = preview.MonthlyUsed
		}
		if err := tx.Save(subscription).Error; err != nil {
			return err
		}
		if options.ClearWallet {
			return tx.Model(&User{}).Where("id = ?", userId).Update("quota", 0).Error
		}
		return nil
	})
	if err != nil {
		if hadPendingWalletDelta {
			batchUpdateStores[BatchUpdateTypeUserQuota][userId] += pendingWalletDelta
		}
		return nil, preview, "", err
	}
	if options.ClearWallet {
		_ = InvalidateUserCache(userId)
	}
	if groupChanged {
		refreshSubscriptionUserGroupCache(userId, "admin subscription creation")
	}
	if strings.TrimSpace(plan.UpgradeGroup) != "" {
		_ = UpdateUserGroupCache(userId, plan.UpgradeGroup)
		return subscription, preview, fmt.Sprintf("用户分组将升级到 %s", plan.UpgradeGroup), nil
	}
	return subscription, preview, "", nil
}

func AdminBindSubscription(userId int, planId int, sourceNote string) (string, error) {
	_, _, message, err := AdminBindSubscriptionWithOptions(userId, planId, AdminSubscriptionBindOptions{})
	return message, err
}

func calcSubscriptionBalanceQuota(priceAmount float64) (int, error) {
	if priceAmount <= 0 {
		return 0, nil
	}
	if common.QuotaPerUnit <= 0 {
		return 0, errors.New("额度单位配置错误")
	}
	quota := decimal.NewFromFloat(priceAmount).
		Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
		Ceil().
		IntPart()
	return int(quota), nil
}

// PurchaseSubscriptionWithBalance creates a subscription by deducting the user's wallet quota.
func PurchaseSubscriptionWithBalance(userId int, planId int) error {
	if userId <= 0 || planId <= 0 {
		return errors.New("invalid userId or planId")
	}

	var logPlanTitle string
	var logMoney float64
	var chargedQuota int
	var upgradeGroup string
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		if !plan.Enabled {
			return errors.New("套餐未启用")
		}
		if plan.PriceAmount < 0 {
			return errors.New("套餐价格不能为负数")
		}
		if plan.AllowBalancePay != nil && !*plan.AllowBalancePay {
			return errors.New("该套餐不允许使用余额兑换")
		}

		requiredQuota, err := calcSubscriptionBalanceQuota(plan.PriceAmount)
		if err != nil {
			return err
		}

		var user User
		if err := lockForUpdate(tx).Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if requiredQuota > 0 && user.Quota < requiredQuota {
			return errors.New("余额不足")
		}
		if requiredQuota > 0 {
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("quota", gorm.Expr("quota - ?", requiredQuota)).Error; err != nil {
				return err
			}
		}

		if _, err := CreateUserSubscriptionFromPlanTx(tx, userId, plan, PaymentMethodBalance); err != nil {
			return err
		}

		now := common.GetTimestamp()
		tradeNo := fmt.Sprintf("SUBBALUSR%dNO%s%d", userId, common.GetRandomString(6), time.Now().UnixNano())
		order := &SubscriptionOrder{
			UserId:          userId,
			PlanId:          plan.Id,
			Money:           plan.PriceAmount,
			TradeNo:         tradeNo,
			PaymentMethod:   PaymentMethodBalance,
			PaymentProvider: PaymentProviderBalance,
			Status:          common.TopUpStatusSuccess,
			CreateTime:      now,
			CompleteTime:    now,
			ProviderPayload: fmt.Sprintf("charged_quota=%d", requiredQuota),
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		logPlanTitle = plan.Title
		logMoney = plan.PriceAmount
		chargedQuota = requiredQuota
		upgradeGroup = strings.TrimSpace(plan.UpgradeGroup)
		return nil
	})
	if err != nil {
		return err
	}

	if chargedQuota > 0 {
		if err := cacheDecrUserQuota(userId, int64(chargedQuota)); err != nil {
			common.SysLog("failed to decrease user quota cache after subscription balance purchase: " + err.Error())
		}
	}
	if upgradeGroup != "" {
		_ = UpdateUserGroupCache(userId, upgradeGroup)
	}
	msg := fmt.Sprintf("使用余额购买订阅成功，套餐: %s，支付金额: %.2f，扣除额度: %d", logPlanTitle, logMoney, chargedQuota)
	RecordLog(userId, LogTypeTopup, msg)
	return nil
}

// GetAllActiveUserSubscriptions returns all active subscriptions for a user.
func GetAllActiveUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	err := DB.Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

func AdminUpdateUserSubscriptionApplicableGroups(subscriptionId int, groups []string) error {
	if subscriptionId <= 0 {
		return errors.New("invalid subscription id")
	}
	var subscription UserSubscription
	if err := DB.First(&subscription, subscriptionId).Error; err != nil {
		return err
	}
	subscription.ApplicableGroups = normalizeSubscriptionGroups(groups)
	return DB.Save(&subscription).Error
}

// HasActiveUserSubscription returns whether the user has any active subscription.
// This is a lightweight existence check to avoid heavy pre-consume transactions.
func HasActiveUserSubscription(userId int, effectiveGroups ...string) (bool, error) {
	return HasActiveUserSubscriptionForModel(userId, "", effectiveGroups...)
}

// HasActiveUserSubscriptionForModel scopes the existence check to subscriptions
// that can serve modelName. An empty modelName preserves the legacy group-only
// behavior for callers that are not preparing a relay request.
func HasActiveUserSubscriptionForModel(userId int, modelName string, effectiveGroups ...string) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	if err := DB.
		Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
		Find(&subs).Error; err != nil {
		return false, err
	}
	effectiveGroup := ""
	if len(effectiveGroups) > 0 {
		effectiveGroup = effectiveGroups[0]
	}
	for _, sub := range subs {
		if subscriptionGroupMatches(sub.ApplicableGroups, effectiveGroup) {
			if strings.TrimSpace(modelName) == "" || subscriptionCanServeModel(sub, modelName) {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetActiveSubscriptionBillingRatio returns the fixed consumption multiplier
// for the first active subscription that can serve modelName in effectiveGroup.
// A false result preserves legacy wallet/live-group billing behavior.
func GetActiveSubscriptionBillingRatio(userId int, modelName string, effectiveGroups ...string) (float64, bool, error) {
	if userId <= 0 || strings.TrimSpace(modelName) == "" {
		return 0, false, nil
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	if err := DB.Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
		Order("end_time asc, id asc").Find(&subs).Error; err != nil {
		return 0, false, err
	}
	effectiveGroup := ""
	if len(effectiveGroups) > 0 {
		effectiveGroup = effectiveGroups[0]
	}
	for _, sub := range subs {
		if !subscriptionGroupMatches(sub.ApplicableGroups, effectiveGroup) || !subscriptionCanServeModel(sub, modelName) {
			continue
		}
		ratio := sub.BillingRatio
		if ratio <= 0 || ratio > SubscriptionMaxBillingRatio || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			continue
		}
		return ratio, true, nil
	}
	return 0, false, nil
}

// UserActiveSubscriptionsAllowWalletOverflow returns whether wallet balance may be used
// after the user's subscription quota is exhausted. A single active subscription that
// disallows wallet overflow (allow_wallet_overflow = false) blocks the fallback.
func UserActiveSubscriptionsAllowWalletOverflow(userId int, effectiveGroups ...string) (bool, error) {
	return UserActiveSubscriptionsAllowWalletOverflowForModel(userId, "", effectiveGroups...)
}

// UserActiveSubscriptionsAllowWalletOverflowForModel only considers packages
// that can serve modelName. An unrelated package must not block wallet fallback.
func UserActiveSubscriptionsAllowWalletOverflowForModel(userId int, modelName string, effectiveGroups ...string) (bool, error) {
	if userId <= 0 {
		return false, errors.New("invalid userId")
	}
	now := common.GetTimestamp()
	var subs []UserSubscription
	if err := DB.Where("user_id = ? AND status = ? AND end_time > ? AND allow_wallet_overflow = ?",
		userId, "active", now, false).Find(&subs).Error; err != nil {
		return false, err
	}
	effectiveGroup := ""
	if len(effectiveGroups) > 0 {
		effectiveGroup = effectiveGroups[0]
	}
	for _, sub := range subs {
		if subscriptionGroupMatches(sub.ApplicableGroups, effectiveGroup) &&
			(strings.TrimSpace(modelName) == "" || subscriptionCanServeModel(sub, modelName)) {
			return false, nil
		}
	}
	return true, nil
}

// GetAllUserSubscriptions returns all subscriptions (active and expired) for a user.
func GetAllUserSubscriptions(userId int) ([]SubscriptionSummary, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	var subs []UserSubscription
	err := DB.Where("user_id = ?", userId).
		Order("end_time desc, id desc").
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return buildSubscriptionSummaries(subs), nil
}

func buildSubscriptionSummaries(subs []UserSubscription) []SubscriptionSummary {
	if len(subs) == 0 {
		return []SubscriptionSummary{}
	}
	result := make([]SubscriptionSummary, 0, len(subs))
	for _, sub := range subs {
		subCopy := sub
		result = append(result, SubscriptionSummary{
			Subscription: &subCopy,
		})
	}
	return result
}

// AdminInvalidateUserSubscription marks a user subscription as cancelled and ends it immediately.
func AdminInvalidateUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		if err := tx.Model(&sub).Updates(map[string]interface{}{
			"status":     "cancelled",
			"end_time":   now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

// AdminDeleteUserSubscription hard-deletes a user subscription.
func AdminDeleteUserSubscription(userSubscriptionId int) (string, error) {
	if userSubscriptionId <= 0 {
		return "", errors.New("invalid userSubscriptionId")
	}
	now := common.GetTimestamp()
	cacheGroup := ""
	downgradeGroup := ""
	var userId int
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).
			Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		userId = sub.UserId
		target, err := downgradeUserGroupForSubscriptionTx(tx, &sub, now)
		if err != nil {
			return err
		}
		if target != "" {
			cacheGroup = target
			downgradeGroup = target
		}
		if err := tx.Where("id = ?", userSubscriptionId).Delete(&UserSubscription{}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if cacheGroup != "" && userId > 0 {
		_ = UpdateUserGroupCache(userId, cacheGroup)
	}
	if downgradeGroup != "" {
		return fmt.Sprintf("用户分组将回退到 %s", downgradeGroup), nil
	}
	return "", nil
}

func resetUserSubscriptionTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64, advanceResetTime bool) error {
	if tx == nil || sub == nil || plan == nil {
		return errors.New("invalid reset args")
	}
	sub.AmountUsed = 0
	sub.DailyUsed = 0
	sub.WeeklyUsed = 0
	sub.MonthlyUsed = 0
	for i := range sub.ResourceGrants {
		sub.ResourceGrants[i].Used = 0
	}
	if advanceResetTime {
		resetAnchor := time.Unix(now, 0)
		nextReset := calcNextResetTime(resetAnchor, resetAnchor, plan, sub.EndTime)
		sub.NextResetTime = nextReset
		if nextReset > 0 {
			sub.LastResetTime = now
		} else {
			sub.LastResetTime = 0
		}
		sub.DailyResetTime = 0
		if sub.DailyAmount > 0 {
			sub.DailyResetTime = calcNextAnchoredPeriodReset(resetAnchor, resetAnchor, SubscriptionResetDaily, sub.EndTime)
		}
		sub.WeeklyResetTime = 0
		if sub.WeeklyAmount > 0 {
			sub.WeeklyResetTime = calcNextAnchoredPeriodReset(resetAnchor, resetAnchor, SubscriptionResetWeekly, sub.EndTime)
		}
		sub.MonthlyResetTime = 0
		if sub.MonthlyAmount > 0 {
			sub.MonthlyResetTime = calcNextAnchoredPeriodReset(resetAnchor, resetAnchor, SubscriptionResetMonthly, sub.EndTime)
		}
		sub.PeriodResetAnchorVersion = subscriptionPeriodResetAnchorVersion
	}
	return tx.Save(sub).Error
}

func buildSubscriptionResetResult(plan *SubscriptionPlan, subs []UserSubscription, advanceResetTime bool) *SubscriptionResetResult {
	userIds := make([]int, 0, len(subs))
	seenUsers := make(map[int]struct{}, len(subs))
	for _, sub := range subs {
		if _, ok := seenUsers[sub.UserId]; ok {
			continue
		}
		seenUsers[sub.UserId] = struct{}{}
		userIds = append(userIds, sub.UserId)
	}
	return &SubscriptionResetResult{
		PlanId:           plan.Id,
		MatchedCount:     len(subs),
		ResetCount:       len(subs),
		UserCount:        len(userIds),
		AdvanceResetTime: advanceResetTime,
		PlanTitle:        plan.Title,
		AffectedUserIds:  userIds,
	}
}

func adminResetUserSubscriptionsByPlanTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, now int64, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if tx == nil || plan == nil {
		return nil, errors.New("invalid reset args")
	}
	var subs []UserSubscription
	if err := lockForUpdate(tx).
		Where("user_id = ? AND plan_id = ? AND status = ? AND end_time > ?", userId, plan.Id, "active", now).
		Order("end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, errors.New("该用户没有有效的此套餐订阅")
	}
	for i := range subs {
		if err := resetUserSubscriptionTx(tx, &subs[i], plan, now, advanceResetTime); err != nil {
			return nil, err
		}
	}
	return buildSubscriptionResetResult(plan, subs, advanceResetTime), nil
}

func adminResetPlanSubscriptionsTx(tx *gorm.DB, plan *SubscriptionPlan, now int64, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if tx == nil || plan == nil {
		return nil, errors.New("invalid reset args")
	}
	var subs []UserSubscription
	if err := lockForUpdate(tx).
		Where("plan_id = ? AND status = ? AND end_time > ?", plan.Id, "active", now).
		Order("user_id asc, end_time asc, id asc").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	for i := range subs {
		if err := resetUserSubscriptionTx(tx, &subs[i], plan, now, advanceResetTime); err != nil {
			return nil, err
		}
	}
	return buildSubscriptionResetResult(plan, subs, advanceResetTime), nil
}

func AdminResetUserSubscriptionsByPlan(userId int, planId int, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if userId <= 0 || planId <= 0 {
		return nil, errors.New("invalid userId or planId")
	}
	var result *SubscriptionResetResult
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		result, err = adminResetUserSubscriptionsByPlanTx(tx, userId, plan, now, advanceResetTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func AdminResetPlanSubscriptions(planId int, advanceResetTime bool) (*SubscriptionResetResult, error) {
	if planId <= 0 {
		return nil, errors.New("invalid planId")
	}
	var result *SubscriptionResetResult
	now := GetDBTimestamp()
	err := DB.Transaction(func(tx *gorm.DB) error {
		plan, err := getSubscriptionPlanByIdTx(tx, planId)
		if err != nil {
			return err
		}
		result, err = adminResetPlanSubscriptionsTx(tx, plan, now, advanceResetTime)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SubscriptionPreConsumeResult struct {
	UserSubscriptionId int
	// BillingRatio is the multiplier captured on the subscription that was
	// actually selected by the pre-consume transaction.  Callers must use this
	// value instead of the first matching subscription returned by a read-only
	// quote, because an earlier subscription may be exhausted or otherwise
	// ineligible when the row lock is acquired.
	BillingRatio        float64
	PreConsumed         int64
	AmountTotal         int64
	AmountUsedBefore    int64
	AmountUsedAfter     int64
	ResourceKey         string
	ResourceType        string
	ResourcePreConsumed int64
	ResourceUsedAfter   int64
}

// ExpireDueSubscriptions marks expired subscriptions and handles group downgrade.
func ExpireDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where("status = ? AND end_time > 0 AND end_time <= ?", "active", now).
		Order("end_time asc, id asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	expiredCount := 0
	userIds := make(map[int]struct{}, len(subs))
	for _, sub := range subs {
		if sub.UserId > 0 {
			userIds[sub.UserId] = struct{}{}
		}
	}
	for userId := range userIds {
		cacheGroup := ""
		err := DB.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&UserSubscription{}).
				Where("user_id = ? AND status = ? AND end_time > 0 AND end_time <= ?", userId, "active", now).
				Updates(map[string]interface{}{
					"status":     "expired",
					"updated_at": common.GetTimestamp(),
				})
			if res.Error != nil {
				return res.Error
			}
			expiredCount += int(res.RowsAffected)

			// If there's an active upgraded subscription, keep current group.
			var activeSub UserSubscription
			activeQuery := tx.Where("user_id = ? AND status = ? AND end_time > ? AND upgrade_group <> ''",
				userId, "active", now).
				Order("end_time desc, id desc").
				Limit(1).
				Find(&activeSub)
			if activeQuery.Error == nil && activeQuery.RowsAffected > 0 {
				return nil
			}

			// Find the most recently expired subscription that defines a group transition
			// (an explicit downgrade target or an upgrade snapshot to revert).
			var lastExpired UserSubscription
			expiredQuery := tx.Where("user_id = ? AND status = ? AND (downgrade_group <> '' OR upgrade_group <> '')",
				userId, "expired").
				Order("end_time desc, id desc").
				Limit(1).
				Find(&lastExpired)
			if expiredQuery.Error != nil || expiredQuery.RowsAffected == 0 {
				return nil
			}
			currentGroup, err := getUserGroupByIdTx(tx, userId)
			if err != nil {
				return err
			}
			// An explicit downgrade group takes precedence; otherwise revert to the
			// group held before purchase (legacy behavior, only when the subscription
			// actually elevated the user).
			target := strings.TrimSpace(lastExpired.DowngradeGroup)
			if target == "" {
				upgradeGroup := strings.TrimSpace(lastExpired.UpgradeGroup)
				prevGroup := strings.TrimSpace(lastExpired.PrevUserGroup)
				if upgradeGroup == "" || prevGroup == "" {
					return nil
				}
				if currentGroup != upgradeGroup {
					return nil
				}
				target = prevGroup
			}
			if target == "" || target == currentGroup {
				return nil
			}
			if err := tx.Model(&User{}).Where("id = ?", userId).
				Update("group", target).Error; err != nil {
				return err
			}
			cacheGroup = target
			return nil
		})
		if err != nil {
			return expiredCount, err
		}
		if cacheGroup != "" {
			_ = UpdateUserGroupCache(userId, cacheGroup)
		}
	}
	return expiredCount, nil
}

// SubscriptionPreConsumeRecord stores idempotent pre-consume operations per request.
type SubscriptionPreConsumeRecord struct {
	Id                 int    `json:"id"`
	RequestId          string `json:"request_id" gorm:"type:varchar(64);uniqueIndex"`
	UserId             int    `json:"user_id" gorm:"index"`
	UserSubscriptionId int    `json:"user_subscription_id" gorm:"index"`
	PreConsumed        int64  `json:"pre_consumed" gorm:"type:bigint;not null;default:0"`
	ResourceKey        string `json:"resource_key" gorm:"type:varchar(128);index"`
	ResourceType       string `json:"resource_type" gorm:"type:varchar(32);index"`
	ResourceAmount     int64  `json:"resource_amount" gorm:"type:bigint;not null;default:0"`
	Status             string `json:"status" gorm:"type:varchar(32);index"` // consumed/refunded
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;index"`
}

func (r *SubscriptionPreConsumeRecord) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *SubscriptionPreConsumeRecord) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func subscriptionUsageHasCapacity(limit int64, used int64, amount int64) bool {
	if amount <= 0 || used < 0 || used > math.MaxInt64-amount {
		return false
	}
	if limit <= 0 {
		return true
	}
	return used <= limit && amount <= limit-used
}

func applySubscriptionUsageDelta(current int64, delta int64) (int64, error) {
	if current < 0 {
		return 0, errors.New("subscription usage is negative")
	}
	if delta > 0 {
		if current > math.MaxInt64-delta {
			return 0, errors.New("subscription usage overflow")
		}
		return current + delta, nil
	}
	if delta == math.MinInt64 || -delta >= current {
		return 0, nil
	}
	return current + delta, nil
}

func migrateUserSubscriptionResetAnchorTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64) error {
	if tx == nil || sub == nil {
		return errors.New("invalid reset anchor migration args")
	}
	if sub.PeriodResetAnchorVersion >= subscriptionPeriodResetAnchorVersion {
		return nil
	}
	anchor := time.Unix(sub.StartTime, 0)
	after := time.Unix(now, 0)
	if plan != nil && NormalizeResetPeriod(plan.QuotaResetPeriod) != SubscriptionResetNever {
		sub.NextResetTime = calcNextResetTime(anchor, after, plan, sub.EndTime)
	}
	if sub.DailyAmount > 0 {
		sub.DailyResetTime = calcNextAnchoredPeriodReset(anchor, after, SubscriptionResetDaily, sub.EndTime)
	}
	if sub.WeeklyAmount > 0 {
		sub.WeeklyResetTime = calcNextAnchoredPeriodReset(anchor, after, SubscriptionResetWeekly, sub.EndTime)
	}
	if sub.MonthlyAmount > 0 {
		sub.MonthlyResetTime = calcNextAnchoredPeriodReset(anchor, after, SubscriptionResetMonthly, sub.EndTime)
	}
	sub.PeriodResetAnchorVersion = subscriptionPeriodResetAnchorVersion
	return tx.Save(sub).Error
}

// MigrateActiveSubscriptionPeriodResetAnchors moves legacy calendar-bound reset
// timestamps to subscription-start anchors without clearing any usage counters.
func MigrateActiveSubscriptionPeriodResetAnchors(limit int) (int, error) {
	if limit <= 0 {
		limit = 300
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where(
		"status = ? AND end_time > ? AND COALESCE(period_reset_anchor_version, 0) < ?",
		"active", now, subscriptionPeriodResetAnchorVersion,
	).
		Order("id asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	migrated := 0
	for _, candidate := range subs {
		err := DB.Transaction(func(tx *gorm.DB) error {
			var sub UserSubscription
			if err := lockForUpdate(tx).
				Where("id = ? AND COALESCE(period_reset_anchor_version, 0) < ?", candidate.Id, subscriptionPeriodResetAnchorVersion).
				First(&sub).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			var plan *SubscriptionPlan
			if sub.PlanId > 0 {
				loaded, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				plan = loaded
			}
			if err := migrateUserSubscriptionResetAnchorTx(tx, &sub, plan, now); err != nil {
				return err
			}
			migrated++
			return nil
		})
		if err != nil {
			return migrated, err
		}
	}
	return migrated, nil
}

func maybeResetUserSubscriptionWithPlanTx(tx *gorm.DB, sub *UserSubscription, plan *SubscriptionPlan, now int64) error {
	if tx == nil || sub == nil || plan == nil {
		return errors.New("invalid reset args")
	}
	if err := migrateUserSubscriptionResetAnchorTx(tx, sub, plan, now); err != nil {
		return err
	}
	changed := false
	if NormalizeResetPeriod(plan.QuotaResetPeriod) != SubscriptionResetNever &&
		(sub.NextResetTime == 0 || sub.NextResetTime <= now) {
		baseUnix := sub.LastResetTime
		if baseUnix <= 0 {
			baseUnix = sub.StartTime
		}
		anchor := time.Unix(sub.StartTime, 0)
		base := time.Unix(baseUnix, 0)
		next := calcNextResetTime(anchor, base, plan, sub.EndTime)
		advanced := false
		for next > 0 && next <= now {
			advanced = true
			base = time.Unix(next, 0)
			next = calcNextResetTime(anchor, base, plan, sub.EndTime)
		}
		if advanced {
			sub.AmountUsed = 0
			for i := range sub.ResourceGrants {
				sub.ResourceGrants[i].Used = 0
			}
			sub.LastResetTime = base.Unix()
			sub.NextResetTime = next
			changed = true
		} else if sub.NextResetTime == 0 && next > 0 {
			sub.NextResetTime = next
			sub.LastResetTime = base.Unix()
			changed = true
		}
	}

	periods := []struct {
		amount    int64
		used      *int64
		resetTime *int64
		period    string
	}{
		{amount: sub.DailyAmount, used: &sub.DailyUsed, resetTime: &sub.DailyResetTime, period: SubscriptionResetDaily},
		{amount: sub.WeeklyAmount, used: &sub.WeeklyUsed, resetTime: &sub.WeeklyResetTime, period: SubscriptionResetWeekly},
		{amount: sub.MonthlyAmount, used: &sub.MonthlyUsed, resetTime: &sub.MonthlyResetTime, period: SubscriptionResetMonthly},
	}
	for _, period := range periods {
		if period.amount <= 0 {
			continue
		}
		anchor := time.Unix(sub.StartTime, 0)
		if *period.resetTime == 0 {
			*period.resetTime = calcNextAnchoredPeriodReset(anchor, anchor, period.period, sub.EndTime)
			changed = *period.resetTime > 0 || changed
		}
		for *period.resetTime > 0 && *period.resetTime <= now {
			*period.used = 0
			*period.resetTime = calcNextAnchoredPeriodReset(anchor, time.Unix(*period.resetTime, 0), period.period, sub.EndTime)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return tx.Save(sub).Error
}

// PreConsumeUserSubscription preserves the legacy quota-only API. New callers
// should use PreConsumeUserSubscriptionWithResource when a model-specific
// bonus grant may satisfy the request.
func PreConsumeUserSubscription(requestId string, userId int, modelName string, quotaType int, amount int64, effectiveGroups ...string) (*SubscriptionPreConsumeResult, error) {
	return PreConsumeUserSubscriptionWithResource(requestId, userId, modelName, quotaType, amount, nil, effectiveGroups...)
}

// PreConsumeUserSubscriptionWithResource pre-consumes either the primary
// package quota or a matching model-specific resource grant. A matching grant
// is used first; when it is exhausted the subscription is not silently charged
// against an unrelated model package.
func PreConsumeUserSubscriptionWithResource(requestId string, userId int, modelName string, quotaType int, amount int64, resource *SubscriptionResourceRequest, effectiveGroups ...string) (*SubscriptionPreConsumeResult, error) {
	return PreConsumeUserSubscriptionWithResourceAndPricing(requestId, userId, modelName, quotaType, amount, resource, nil, effectiveGroups...)
}

// PreConsumeUserSubscriptionWithResourceAndPricing is the fixed-package entry
// point used by BillingSession. It calculates each candidate's amount inside
// the same transaction that selects and reserves that subscription.
func PreConsumeUserSubscriptionWithResourceAndPricing(requestId string, userId int, modelName string, quotaType int, amount int64, resource *SubscriptionResourceRequest, pricing *SubscriptionQuotaPricing, effectiveGroups ...string) (*SubscriptionPreConsumeResult, error) {
	if userId <= 0 {
		return nil, errors.New("invalid userId")
	}
	if strings.TrimSpace(requestId) == "" {
		return nil, errors.New("requestId is empty")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be > 0")
	}
	if amount > int64(common.MaxQuota) {
		return nil, errors.New("amount exceeds subscription quota limit")
	}
	if resource != nil {
		resource.ResourceKey = strings.TrimSpace(resource.ResourceKey)
		resource.ResourceType = normalizeSubscriptionResourceType(resource.ResourceType)
		if resource.ResourceKey == "" || resource.Amount <= 0 || resource.Amount > int64(common.MaxQuota) {
			return nil, errors.New("invalid subscription resource request")
		}
	}
	now := GetDBTimestamp()
	effectiveGroup := ""
	if len(effectiveGroups) > 0 {
		effectiveGroup = effectiveGroups[0]
	}

	returnValue := &SubscriptionPreConsumeResult{}

	err := DB.Transaction(func(tx *gorm.DB) error {
		var existing SubscriptionPreConsumeRecord
		query := tx.Where("request_id = ?", requestId).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected > 0 {
			if existing.Status == "refunded" {
				return errors.New("subscription pre-consume already refunded")
			}
			var sub UserSubscription
			if err := tx.Where("id = ?", existing.UserSubscriptionId).First(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.BillingRatio = sub.BillingRatio
			returnValue.PreConsumed = existing.PreConsumed
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = sub.AmountUsed
			returnValue.AmountUsedAfter = sub.AmountUsed
			returnValue.ResourceKey = existing.ResourceKey
			returnValue.ResourceType = existing.ResourceType
			returnValue.ResourcePreConsumed = existing.ResourceAmount
			if existing.ResourceKey != "" {
				for _, grant := range sub.ResourceGrants {
					if grant.ResourceKey == existing.ResourceKey && grant.ResourceType == existing.ResourceType {
						returnValue.ResourceUsedAfter = grant.Used
						break
					}
				}
			}
			return nil
		}

		var subs []UserSubscription
		if err := lockForUpdate(tx).
			Where("user_id = ? AND status = ? AND end_time > ?", userId, "active", now).
			Order("end_time asc, id asc").
			Find(&subs).Error; err != nil {
			return errors.New("no active subscription")
		}
		if len(subs) == 0 {
			return errors.New("no active subscription")
		}
		for _, candidate := range subs {
			sub := candidate
			if !subscriptionGroupMatches(sub.ApplicableGroups, effectiveGroup) {
				continue
			}
			if sub.BillingRatio <= 0 || sub.BillingRatio > SubscriptionMaxBillingRatio || math.IsNaN(sub.BillingRatio) || math.IsInf(sub.BillingRatio, 0) {
				continue
			}
			plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
			if err != nil {
				return err
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &sub, plan, now); err != nil {
				return err
			}
			candidateAmount := amount
			if pricing != nil && pricing.BaseQuotaBeforeGroup > 0 && sub.BillingRatio > 0 && !math.IsNaN(sub.BillingRatio) && !math.IsInf(sub.BillingRatio, 0) {
				pricedQuota, err := common.QuotaFromFloatStrict(pricing.BaseQuotaBeforeGroup * sub.BillingRatio)
				if err != nil {
					return err
				}
				candidateAmount = int64(pricedQuota)
				if candidateAmount <= 0 {
					candidateAmount = 1
				}
			} else if pricing != nil && pricing.LiveQuota > 0 {
				candidateAmount = pricing.LiveQuota
			}

			resourceIndex := -1
			if resource != nil {
				resourceMatched := false
				for i := range sub.ResourceGrants {
					grant := sub.ResourceGrants[i]
					if grant.ResourceType != resource.ResourceType || !subscriptionResourceMatches(grant, resource.ResourceKey, modelName) {
						continue
					}
					resourceMatched = true
					resourceAmount := resource.Amount
					if resource.ResourceType == SubscriptionResourceTypeQuota {
						resourceAmount = candidateAmount
					}
					if subscriptionUsageHasCapacity(grant.Amount, grant.Used, resourceAmount) {
						resourceIndex = i
						break
					}
				}
				// A configured but exhausted grant must not silently charge a
				// different entitlement for the same model.
				if resourceMatched && resourceIndex < 0 {
					continue
				}
			}
			if resourceIndex < 0 && !subscriptionModelMatches(sub, modelName) {
				continue
			}
			usedBefore := sub.AmountUsed
			if resourceIndex < 0 {
				if !subscriptionUsageHasCapacity(sub.AmountTotal, sub.AmountUsed, candidateAmount) {
					continue
				}
				if sub.DailyAmount > 0 && !subscriptionUsageHasCapacity(sub.DailyAmount, sub.DailyUsed, candidateAmount) {
					continue
				}
				if sub.WeeklyAmount > 0 && !subscriptionUsageHasCapacity(sub.WeeklyAmount, sub.WeeklyUsed, candidateAmount) {
					continue
				}
				if sub.MonthlyAmount > 0 && !subscriptionUsageHasCapacity(sub.MonthlyAmount, sub.MonthlyUsed, candidateAmount) {
					continue
				}
			}
			record := &SubscriptionPreConsumeRecord{
				RequestId:          requestId,
				UserId:             userId,
				UserSubscriptionId: sub.Id,
				PreConsumed:        candidateAmount,
				Status:             "consumed",
			}
			if resourceIndex >= 0 {
				grant := &sub.ResourceGrants[resourceIndex]
				resourceAmount := resource.Amount
				if resource.ResourceType == SubscriptionResourceTypeQuota {
					resourceAmount = candidateAmount
				}
				grant.Used, err = applySubscriptionUsageDelta(grant.Used, resourceAmount)
				if err != nil {
					return err
				}
				record.ResourceKey = grant.ResourceKey
				record.ResourceType = grant.ResourceType
				record.ResourceAmount = resourceAmount
			}
			if err := tx.Create(record).Error; err != nil {
				var dup SubscriptionPreConsumeRecord
				if err2 := tx.Where("request_id = ?", requestId).First(&dup).Error; err2 == nil {
					if dup.Status == "refunded" {
						return errors.New("subscription pre-consume already refunded")
					}
					var duplicateSub UserSubscription
					if err2 := tx.Where("id = ?", dup.UserSubscriptionId).First(&duplicateSub).Error; err2 != nil {
						return err2
					}
					returnValue.UserSubscriptionId = duplicateSub.Id
					returnValue.BillingRatio = duplicateSub.BillingRatio
					returnValue.PreConsumed = dup.PreConsumed
					returnValue.AmountTotal = duplicateSub.AmountTotal
					returnValue.AmountUsedBefore = duplicateSub.AmountUsed
					returnValue.AmountUsedAfter = duplicateSub.AmountUsed
					returnValue.ResourceKey = dup.ResourceKey
					returnValue.ResourceType = dup.ResourceType
					returnValue.ResourcePreConsumed = dup.ResourceAmount
					if dup.ResourceKey != "" {
						for _, grant := range duplicateSub.ResourceGrants {
							if grant.ResourceKey == dup.ResourceKey && grant.ResourceType == dup.ResourceType {
								returnValue.ResourceUsedAfter = grant.Used
								break
							}
						}
					}
					return nil
				}
				return err
			}
			if resourceIndex < 0 {
				sub.AmountUsed, err = applySubscriptionUsageDelta(sub.AmountUsed, candidateAmount)
				if err != nil {
					return err
				}
				if sub.DailyAmount > 0 {
					sub.DailyUsed, err = applySubscriptionUsageDelta(sub.DailyUsed, candidateAmount)
					if err != nil {
						return err
					}
				}
				if sub.WeeklyAmount > 0 {
					sub.WeeklyUsed, err = applySubscriptionUsageDelta(sub.WeeklyUsed, candidateAmount)
					if err != nil {
						return err
					}
				}
				if sub.MonthlyAmount > 0 {
					sub.MonthlyUsed, err = applySubscriptionUsageDelta(sub.MonthlyUsed, candidateAmount)
					if err != nil {
						return err
					}
				}
			}
			if err := tx.Save(&sub).Error; err != nil {
				return err
			}
			returnValue.UserSubscriptionId = sub.Id
			returnValue.BillingRatio = sub.BillingRatio
			returnValue.PreConsumed = candidateAmount
			returnValue.AmountTotal = sub.AmountTotal
			returnValue.AmountUsedBefore = usedBefore
			returnValue.AmountUsedAfter = sub.AmountUsed
			if resourceIndex >= 0 {
				grant := sub.ResourceGrants[resourceIndex]
				returnValue.ResourceKey = grant.ResourceKey
				returnValue.ResourceType = grant.ResourceType
				returnValue.ResourcePreConsumed = record.ResourceAmount
				returnValue.ResourceUsedAfter = grant.Used
			}
			return nil
		}
		return fmt.Errorf("subscription quota insufficient, need=%d", amount)
	})
	if err != nil {
		return nil, err
	}
	return returnValue, nil
}

// RefundSubscriptionPreConsume is idempotent and refunds pre-consumed subscription quota by requestId.
func RefundSubscriptionPreConsume(requestId string) error {
	if strings.TrimSpace(requestId) == "" {
		return errors.New("requestId is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var record SubscriptionPreConsumeRecord
		if err := lockForUpdate(tx).
			Where("request_id = ?", requestId).First(&record).Error; err != nil {
			return err
		}
		if record.Status == "refunded" {
			return nil
		}
		if record.PreConsumed <= 0 {
			if record.ResourceAmount <= 0 {
				record.Status = "refunded"
				return tx.Save(&record).Error
			}
			if err := postConsumeUserSubscriptionResourceDeltaTx(tx, record.UserSubscriptionId, record.ResourceKey, record.ResourceType, -record.ResourceAmount); err != nil {
				return err
			}
			record.Status = "refunded"
			return tx.Save(&record).Error
		}
		if record.ResourceKey != "" && record.ResourceAmount > 0 {
			if err := postConsumeUserSubscriptionResourceDeltaTx(tx, record.UserSubscriptionId, record.ResourceKey, record.ResourceType, -record.ResourceAmount); err != nil {
				return err
			}
			record.Status = "refunded"
			return tx.Save(&record).Error
		}
		if err := postConsumeUserSubscriptionDeltaTx(tx, record.UserSubscriptionId, -record.PreConsumed); err != nil {
			return err
		}
		record.Status = "refunded"
		return tx.Save(&record).Error
	})
}

// ResetDueSubscriptions resets subscriptions with any due quota period.
func ResetDueSubscriptions(limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	now := GetDBTimestamp()
	var subs []UserSubscription
	if err := DB.Where(
		"status = ? AND ((next_reset_time > 0 AND next_reset_time <= ?) OR (daily_reset_time > 0 AND daily_reset_time <= ?) OR (weekly_reset_time > 0 AND weekly_reset_time <= ?) OR (monthly_reset_time > 0 AND monthly_reset_time <= ?))",
		"active", now, now, now, now,
	).
		Order("id asc").
		Limit(limit).
		Find(&subs).Error; err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, nil
	}
	resetCount := 0
	for _, sub := range subs {
		subCopy := sub
		plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
		if err != nil || plan == nil {
			continue
		}
		err = DB.Transaction(func(tx *gorm.DB) error {
			var locked UserSubscription
			if err := lockForUpdate(tx).
				Where(
					"id = ? AND status = ? AND ((next_reset_time > 0 AND next_reset_time <= ?) OR (daily_reset_time > 0 AND daily_reset_time <= ?) OR (weekly_reset_time > 0 AND weekly_reset_time <= ?) OR (monthly_reset_time > 0 AND monthly_reset_time <= ?))",
					subCopy.Id, "active", now, now, now, now,
				).
				First(&locked).Error; err != nil {
				return nil
			}
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &locked, plan, now); err != nil {
				return err
			}
			resetCount++
			return nil
		})
		if err != nil {
			return resetCount, err
		}
	}
	return resetCount, nil
}

// CleanupSubscriptionPreConsumeRecords removes old idempotency records to keep table small.
func CleanupSubscriptionPreConsumeRecords(olderThanSeconds int64) (int64, error) {
	if olderThanSeconds <= 0 {
		olderThanSeconds = 7 * 24 * 3600
	}
	cutoff := GetDBTimestamp() - olderThanSeconds
	res := DB.Where("updated_at < ?", cutoff).Delete(&SubscriptionPreConsumeRecord{})
	return res.RowsAffected, res.Error
}

type SubscriptionPlanInfo struct {
	PlanId    int
	PlanTitle string
}

func GetSubscriptionPlanInfoByUserSubscriptionId(userSubscriptionId int) (*SubscriptionPlanInfo, error) {
	if userSubscriptionId <= 0 {
		return nil, errors.New("invalid userSubscriptionId")
	}
	cacheKey := fmt.Sprintf("sub:%d", userSubscriptionId)
	if cached, found, err := getSubscriptionPlanInfoCache().Get(cacheKey); err == nil && found {
		return &cached, nil
	}
	var sub UserSubscription
	if err := DB.Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
		return nil, err
	}
	plan, err := getSubscriptionPlanByIdTx(nil, sub.PlanId)
	if err != nil {
		return nil, err
	}
	info := &SubscriptionPlanInfo{
		PlanId:    sub.PlanId,
		PlanTitle: plan.Title,
	}
	_ = getSubscriptionPlanInfoCache().SetWithTTL(cacheKey, *info, subscriptionPlanInfoCacheTTL())
	return info, nil
}

// PostConsumeUserSubscriptionDelta updates every configured usage counter by delta.
func PostConsumeUserSubscriptionDelta(userSubscriptionId int, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if delta == 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return postConsumeUserSubscriptionDeltaTx(tx, userSubscriptionId, delta)
	})
}

// PostConsumeUserSubscriptionResourceDelta settles a model-specific quota
// grant. Image-count grants are consumed at pre-consume time; only a negative
// delta is accepted here to refund a failed request.
func PostConsumeUserSubscriptionResourceDelta(userSubscriptionId int, resourceKey, resourceType string, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if strings.TrimSpace(resourceKey) == "" {
		return errors.New("resource key is empty")
	}
	if delta == 0 {
		return nil
	}
	if normalizeSubscriptionResourceType(resourceType) == SubscriptionResourceTypeImageCount && delta > 0 {
		return errors.New("image-count resources are consumed at pre-consume time")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return postConsumeUserSubscriptionResourceDeltaTx(tx, userSubscriptionId, resourceKey, resourceType, delta)
	})
}

// SettleUserSubscriptionImageResourceDelta applies the validated difference
// between requested and actually returned image generations. Unlike the public
// refund helper above, this path may consume additional generations, and the
// locked update still enforces the purchased grant's capacity.
func SettleUserSubscriptionImageResourceDelta(userSubscriptionId int, resourceKey string, delta int64) error {
	if userSubscriptionId <= 0 {
		return errors.New("invalid userSubscriptionId")
	}
	if strings.TrimSpace(resourceKey) == "" {
		return errors.New("resource key is empty")
	}
	if delta == 0 {
		return nil
	}
	if delta == math.MinInt64 || delta > int64(common.MaxQuota) || delta < -int64(common.MaxQuota) {
		return errors.New("image resource delta exceeds limit")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return postConsumeUserSubscriptionResourceDeltaTx(tx, userSubscriptionId, resourceKey, SubscriptionResourceTypeImageCount, delta)
	})
}

// SettleUserSubscriptionImageResourceDeltaWithQuota settles an image grant and
// charges a fixed-ratio fallback against the subscription's primary quota when
// the upstream returns more images than the grant can cover. quotaPerImage is
// captured from the purchased subscription snapshot. Both updates are
// performed under one row lock so a failed fallback rolls back the grant
// update as well. The returned value is the primary-quota amount applied.
func SettleUserSubscriptionImageResourceDeltaWithQuota(userSubscriptionId int, resourceKey string, resourceDelta, quotaPerImage int64) (int64, error) {
	if userSubscriptionId <= 0 {
		return 0, errors.New("invalid userSubscriptionId")
	}
	if strings.TrimSpace(resourceKey) == "" {
		return 0, errors.New("resource key is empty")
	}
	if resourceDelta == math.MinInt64 || resourceDelta > int64(common.MaxQuota) || resourceDelta < -int64(common.MaxQuota) {
		return 0, errors.New("image resource delta exceeds limit")
	}
	if quotaPerImage < 0 || quotaPerImage > int64(common.MaxQuota) {
		return 0, errors.New("image unit quota exceeds limit")
	}
	if resourceDelta == 0 {
		return 0, nil
	}

	var appliedQuota int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sub UserSubscription
		if err := lockForUpdate(tx).Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
			return err
		}
		if sub.PlanId > 0 {
			plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if plan != nil {
				if err := maybeResetUserSubscriptionWithPlanTx(tx, &sub, plan, getDBTimestampTx(tx)); err != nil {
					return err
				}
			}
		}

		normalizedKey := strings.TrimSpace(strings.ToLower(resourceKey))
		for i := range sub.ResourceGrants {
			grant := &sub.ResourceGrants[i]
			if grant.ResourceType != SubscriptionResourceTypeImageCount || grant.ResourceKey != normalizedKey {
				continue
			}

			grantDelta := resourceDelta
			overage := int64(0)
			if resourceDelta > 0 && grant.Amount > 0 {
				available := int64(0)
				if grant.Used < grant.Amount {
					available = grant.Amount - grant.Used
				}
				if grantDelta > available {
					grantDelta = available
					overage = resourceDelta - available
				}
			}
			newUsed, err := applySubscriptionUsageDelta(grant.Used, grantDelta)
			if err != nil {
				return fmt.Errorf("subscription image resource usage update failed: %w", err)
			}
			grant.Used = newUsed
			if overage > 0 {
				if quotaPerImage <= 0 || overage > int64(common.MaxQuota)/quotaPerImage {
					return fmt.Errorf("%w: image_count=%d", ErrSubscriptionImageOverageUnsettled, overage)
				}
				quotaDelta := quotaPerImage * overage
				if err := applySubscriptionQuotaDeltaTx(&sub, quotaDelta); err != nil {
					return fmt.Errorf("%w: %v", ErrSubscriptionImageOverageUnsettled, err)
				}
				appliedQuota = quotaDelta
			}
			return tx.Save(&sub).Error
		}
		return errors.New("subscription image resource grant not found")
	})
	if err != nil {
		return 0, err
	}
	return appliedQuota, nil
}

func postConsumeUserSubscriptionResourceDeltaTx(tx *gorm.DB, userSubscriptionId int, resourceKey, resourceType string, delta int64) error {
	normalizedType := normalizeSubscriptionResourceType(resourceType)
	var sub UserSubscription
	if err := lockForUpdate(tx).Where("id = ?", userSubscriptionId).First(&sub).Error; err != nil {
		return err
	}
	for i := range sub.ResourceGrants {
		grant := &sub.ResourceGrants[i]
		if grant.ResourceType != normalizedType || grant.ResourceKey != strings.TrimSpace(strings.ToLower(resourceKey)) {
			continue
		}
		newUsed, err := applySubscriptionUsageDelta(grant.Used, delta)
		if err != nil {
			return fmt.Errorf("subscription resource usage update failed: %w", err)
		}
		if delta > 0 && grant.Amount > 0 && newUsed > grant.Amount {
			return fmt.Errorf("subscription resource used exceeds grant, used=%d total=%d", newUsed, grant.Amount)
		}
		grant.Used = newUsed
		return tx.Save(&sub).Error
	}
	return errors.New("subscription resource grant not found")
}

func postConsumeUserSubscriptionDeltaTx(tx *gorm.DB, userSubscriptionId int, delta int64) error {
	var sub UserSubscription
	if err := lockForUpdate(tx).
		Where("id = ?", userSubscriptionId).
		First(&sub).Error; err != nil {
		return err
	}
	if sub.PlanId > 0 {
		plan, err := getSubscriptionPlanByIdTx(tx, sub.PlanId)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if plan != nil {
			if err := maybeResetUserSubscriptionWithPlanTx(tx, &sub, plan, getDBTimestampTx(tx)); err != nil {
				return err
			}
		}
	}
	if err := applySubscriptionQuotaDeltaTx(&sub, delta); err != nil {
		return err
	}
	return tx.Save(&sub).Error
}

func applySubscriptionQuotaDeltaTx(sub *UserSubscription, delta int64) error {
	if sub == nil {
		return errors.New("subscription is nil")
	}
	newUsed, err := applySubscriptionUsageDelta(sub.AmountUsed, delta)
	if err != nil {
		return fmt.Errorf("subscription total usage update failed: %w", err)
	}
	newDailyUsed := sub.DailyUsed
	newWeeklyUsed := sub.WeeklyUsed
	newMonthlyUsed := sub.MonthlyUsed
	if sub.DailyAmount > 0 {
		newDailyUsed, err = applySubscriptionUsageDelta(sub.DailyUsed, delta)
		if err != nil {
			return fmt.Errorf("subscription daily usage update failed: %w", err)
		}
	}
	if sub.WeeklyAmount > 0 {
		newWeeklyUsed, err = applySubscriptionUsageDelta(sub.WeeklyUsed, delta)
		if err != nil {
			return fmt.Errorf("subscription weekly usage update failed: %w", err)
		}
	}
	if sub.MonthlyAmount > 0 {
		newMonthlyUsed, err = applySubscriptionUsageDelta(sub.MonthlyUsed, delta)
		if err != nil {
			return fmt.Errorf("subscription monthly usage update failed: %w", err)
		}
	}
	if delta > 0 {
		if sub.AmountTotal > 0 && newUsed > sub.AmountTotal {
			return fmt.Errorf("subscription used exceeds total, used=%d total=%d", newUsed, sub.AmountTotal)
		}
		if sub.DailyAmount > 0 && newDailyUsed > sub.DailyAmount {
			return fmt.Errorf("subscription daily usage exceeds limit, used=%d total=%d", newDailyUsed, sub.DailyAmount)
		}
		if sub.WeeklyAmount > 0 && newWeeklyUsed > sub.WeeklyAmount {
			return fmt.Errorf("subscription weekly usage exceeds limit, used=%d total=%d", newWeeklyUsed, sub.WeeklyAmount)
		}
		if sub.MonthlyAmount > 0 && newMonthlyUsed > sub.MonthlyAmount {
			return fmt.Errorf("subscription monthly usage exceeds limit, used=%d total=%d", newMonthlyUsed, sub.MonthlyAmount)
		}
	}
	sub.AmountUsed = newUsed
	sub.DailyUsed = newDailyUsed
	sub.WeeklyUsed = newWeeklyUsed
	sub.MonthlyUsed = newMonthlyUsed
	return nil
}
