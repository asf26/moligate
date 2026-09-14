package service

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// BillingSession — 统一计费会话
// ---------------------------------------------------------------------------

// BillingSession 封装单次请求的预扣费/结算/退款生命周期。
// 实现 relaycommon.BillingSettler 接口。
type BillingSession struct {
	relayInfo        *relaycommon.RelayInfo
	funding          FundingSource
	preConsumedQuota int  // 实际预扣额度（信任用户可能为 0）
	tokenConsumed    int  // 令牌额度实际扣减量
	extraReserved    int  // 发送前补充预扣的额度（订阅退款时需要单独回滚）
	trusted          bool // 是否命中信任额度旁路
	fundingSettled   bool // funding.Settle 已成功，资金来源已提交
	settled          bool // Settle 全部完成（资金 + 令牌）
	refunded         bool // Refund 已调用
	mu               sync.Mutex
}

// Settle 根据实际消耗额度进行结算。
// 资金来源和令牌额度分两步提交：若资金来源已提交但令牌调整失败，
// 会标记 fundingSettled 防止 Refund 对已提交的资金来源执行退款。
func (s *BillingSession) Settle(actualQuota int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled {
		return nil
	}
	delta := actualQuota - s.preConsumedQuota
	// 1) 调整资金来源（仅在尚未提交时执行，防止重复调用）
	if !s.fundingSettled {
		if subscriptionFunding, ok := s.funding.(*SubscriptionFunding); ok &&
			subscriptionFunding.resource != nil &&
			subscriptionFunding.resource.ResourceType == model.SubscriptionResourceTypeImageCount {
			actualCount := subscriptionFunding.resourcePreConsumed
			if s.relayInfo.ActualImageCountSet {
				actualCount = s.relayInfo.ActualImageCount
			} else if count, ok := s.relayInfo.PriceData.OtherRatios()["n"]; ok {
				if count < 0 || count > float64(dto.MaxImageN) || math.IsNaN(count) || math.IsInf(count, 0) || math.Trunc(count) != count {
					return errors.New("invalid settled image generation count")
				}
				actualCount = int64(count)
			}
			if err := subscriptionFunding.settleImageCount(actualCount); err != nil {
				return err
			}
			s.relayInfo.SubscriptionResourceAmount = actualCount
		} else if delta != 0 {
			if err := s.funding.Settle(delta); err != nil {
				return err
			}
		}
		s.fundingSettled = true
	}
	if delta == 0 {
		s.settled = true
		return nil
	}
	// 2) 调整令牌额度
	var tokenErr error
	if !s.relayInfo.IsPlayground {
		if delta > 0 {
			tokenErr = model.DecreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, delta)
		} else {
			tokenErr = model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, -delta)
		}
		if tokenErr != nil {
			// 资金来源已提交，令牌调整失败只能记录日志；标记 settled 防止 Refund 误退资金
			common.SysLog(fmt.Sprintf("error adjusting token quota after funding settled (userId=%d, tokenId=%d, delta=%d): %s",
				s.relayInfo.UserId, s.relayInfo.TokenId, delta, tokenErr.Error()))
		}
	}
	// 3) 更新 relayInfo 上的订阅 PostDelta（用于日志）
	if s.funding.Source() == BillingSourceSubscription {
		s.relayInfo.SubscriptionPostDelta += int64(delta)
	}
	s.settled = true
	return tokenErr
}

// Refund 退还所有预扣费，幂等安全，异步执行。
func (s *BillingSession) Refund(c *gin.Context) {
	s.mu.Lock()
	if s.settled || s.refunded || !s.needsRefundLocked() {
		s.mu.Unlock()
		return
	}
	s.refunded = true
	s.mu.Unlock()

	logger.LogInfo(c, fmt.Sprintf("用户 %d 请求失败, 返还预扣费（token_quota=%s, funding=%s）",
		s.relayInfo.UserId,
		logger.FormatQuota(s.tokenConsumed),
		s.funding.Source(),
	))

	// 复制需要的值到闭包中
	tokenId := s.relayInfo.TokenId
	tokenKey := s.relayInfo.TokenKey
	isPlayground := s.relayInfo.IsPlayground
	tokenConsumed := s.tokenConsumed
	extraReserved := s.extraReserved
	subscriptionId := s.relayInfo.SubscriptionId
	funding := s.funding

	gopool.Go(func() {
		// 1) 退还资金来源
		if err := funding.Refund(); err != nil {
			common.SysLog("error refunding billing source: " + err.Error())
		}
		if extraReserved > 0 && funding.Source() == BillingSourceSubscription && subscriptionId > 0 {
			var err error
			if subscriptionFunding, ok := funding.(*SubscriptionFunding); ok {
				err = subscriptionFunding.ReserveDelta(-int64(extraReserved))
			} else {
				err = model.PostConsumeUserSubscriptionDelta(subscriptionId, -int64(extraReserved))
			}
			if err != nil {
				common.SysLog("error refunding subscription extra reserved quota: " + err.Error())
			}
		}
		// 2) 退还令牌额度
		if tokenConsumed > 0 && !isPlayground {
			if err := model.IncreaseTokenQuota(tokenId, tokenKey, tokenConsumed); err != nil {
				common.SysLog("error refunding token quota: " + err.Error())
			}
		}
	})
}

// NeedsRefund 返回是否存在需要退还的预扣状态。
func (s *BillingSession) NeedsRefund() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.needsRefundLocked()
}

func (s *BillingSession) needsRefundLocked() bool {
	if s.settled || s.refunded || s.fundingSettled {
		// fundingSettled 时资金来源已提交结算，不能再退预扣费
		return false
	}
	if s.tokenConsumed > 0 {
		return true
	}
	// 订阅可能在 tokenConsumed=0 时仍预扣了额度
	if sub, ok := s.funding.(*SubscriptionFunding); ok && sub.preConsumed > 0 {
		return true
	}
	return false
}

// GetPreConsumedQuota 返回实际预扣的额度。
func (s *BillingSession) GetPreConsumedQuota() int {
	return s.preConsumedQuota
}

func (s *BillingSession) Reserve(targetQuota int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.settled || s.refunded || s.trusted || targetQuota <= s.preConsumedQuota {
		return nil
	}

	delta := targetQuota - s.preConsumedQuota
	if delta <= 0 {
		return nil
	}

	if err := s.reserveFunding(delta); err != nil {
		return err
	}
	if err := s.reserveToken(delta); err != nil {
		s.rollbackFundingReserve(delta)
		return err
	}

	s.preConsumedQuota += delta
	s.tokenConsumed += delta
	s.extraReserved += delta
	s.syncRelayInfo()
	return nil
}

// ---------------------------------------------------------------------------
// PreConsume — 统一预扣费入口（含信任额度旁路）
// ---------------------------------------------------------------------------

// preConsume 执行预扣费：信任检查 -> 令牌预扣 -> 资金来源预扣。
// 任一步骤失败时原子回滚已完成的步骤。
func (s *BillingSession) preConsume(c *gin.Context, quota int) *types.NewAPIError {
	effectiveQuota := quota

	// ---- 信任额度旁路 ----
	if s.shouldTrust(c) {
		s.trusted = true
		effectiveQuota = 0
		logger.LogInfo(c, fmt.Sprintf("用户 %d 额度充足, 信任且不需要预扣费 (funding=%s)", s.relayInfo.UserId, s.funding.Source()))
	} else if effectiveQuota > 0 {
		logger.LogInfo(c, fmt.Sprintf("用户 %d 需要预扣费 %s (funding=%s)", s.relayInfo.UserId, logger.FormatQuota(effectiveQuota), s.funding.Source()))
	}

	// ---- 1) 预扣令牌额度 ----
	if effectiveQuota > 0 {
		if err := PreConsumeTokenQuota(s.relayInfo, effectiveQuota); err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		s.tokenConsumed = effectiveQuota
	}

	// ---- 2) 预扣资金来源 ----
	if err := s.funding.PreConsume(effectiveQuota); err != nil {
		// 预扣费失败，回滚令牌额度
		if s.tokenConsumed > 0 && !s.relayInfo.IsPlayground {
			if rollbackErr := model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, s.tokenConsumed); rollbackErr != nil {
				common.SysLog(fmt.Sprintf("error rolling back token quota (userId=%d, tokenId=%d, amount=%d, fundingErr=%s): %s",
					s.relayInfo.UserId, s.relayInfo.TokenId, s.tokenConsumed, err.Error(), rollbackErr.Error()))
			}
			s.tokenConsumed = 0
		}
		// TODO: model 层应定义哨兵错误（如 ErrNoActiveSubscription），用 errors.Is 替代字符串匹配
		if errors.Is(err, ErrInsufficientWalletQuota) {
			userQuota, quotaErr := model.GetUserQuota(s.relayInfo.UserId, false)
			if quotaErr != nil {
				userQuota = 0
			}
			return types.NewErrorWithStatusCode(
				fmt.Errorf("用户额度不足, 剩余额度: %s", logger.FormatQuota(userQuota)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		errMsg := err.Error()
		if strings.Contains(errMsg, "no active subscription") || strings.Contains(errMsg, "subscription quota insufficient") {
			return types.NewErrorWithStatusCode(fmt.Errorf("订阅额度不足或未配置订阅: %s", errMsg), types.ErrorCodeInsufficientUserQuota, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}

	// A locked subscription transaction may skip an exhausted package and pick
	// a later package with a different purchased multiplier. Reconcile the token
	// reservation to that selected package before exposing the session.
	if subscriptionFunding, ok := s.funding.(*SubscriptionFunding); ok {
		selectedQuota := int(subscriptionFunding.preConsumed)
		if selectedQuota <= 0 {
			if refundErr := s.funding.Refund(); refundErr != nil {
				common.SysLog("error refunding invalid subscription reservation: " + refundErr.Error())
			}
			if s.tokenConsumed > 0 && !s.relayInfo.IsPlayground {
				_ = model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, s.tokenConsumed)
			}
			s.tokenConsumed = 0
			return types.NewError(errors.New("subscription returned an invalid reservation"), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
		}
		delta := selectedQuota - effectiveQuota
		if delta > 0 && !s.relayInfo.IsPlayground {
			if err := PreConsumeTokenQuota(s.relayInfo, delta); err != nil {
				if refundErr := s.funding.Refund(); refundErr != nil {
					common.SysLog("error refunding subscription after token reserve failure: " + refundErr.Error())
				}
				if s.tokenConsumed > 0 {
					if rollbackErr := model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, s.tokenConsumed); rollbackErr != nil {
						common.SysLog("error rolling back token quota after subscription repricing failure: " + rollbackErr.Error())
					}
				}
				s.tokenConsumed = 0
				return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
			}
			s.tokenConsumed += delta
		} else if delta < 0 && !s.relayInfo.IsPlayground {
			if err := model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, -delta); err != nil {
				if refundErr := s.funding.Refund(); refundErr != nil {
					common.SysLog("error refunding subscription after token release failure: " + refundErr.Error())
				}
				if s.tokenConsumed > 0 {
					if rollbackErr := model.IncreaseTokenQuota(s.relayInfo.TokenId, s.relayInfo.TokenKey, s.tokenConsumed); rollbackErr != nil {
						common.SysLog("error rolling back token quota after subscription repricing failure: " + rollbackErr.Error())
					}
				}
				s.tokenConsumed = 0
				return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
			}
			s.tokenConsumed += delta
		}
		effectiveQuota = selectedQuota
	}

	s.preConsumedQuota = effectiveQuota

	// ---- 同步 RelayInfo 兼容字段 ----
	s.syncRelayInfo()

	return nil
}

func (s *BillingSession) reserveFunding(delta int) error {
	switch funding := s.funding.(type) {
	case *WalletFunding:
		// 与结算补扣（SettleBilling 正差额 → WalletFunding.Settle）语义一致：
		// 全额无条件扣减，余额不足的部分记为欠费（余额可为负），不中断请求，
		// 保证日志记录的预扣额度与用户余额的实际变动始终对账一致。
		// DecreaseUserQuota 仅在数据库错误时失败。
		if err := model.DecreaseUserQuota(funding.userId, delta, false); err != nil {
			return types.NewError(err, types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
		}
		funding.consumed += delta
		return nil
	case *SubscriptionFunding:
		if err := funding.ReserveDelta(int64(delta)); err != nil {
			return types.NewErrorWithStatusCode(
				fmt.Errorf("订阅额度不足或未配置订阅: %s", err.Error()),
				types.ErrorCodeInsufficientUserQuota,
				http.StatusForbidden,
				types.ErrOptionWithSkipRetry(),
				types.ErrOptionWithNoRecordErrorLog(),
			)
		}
		return nil
	default:
		return types.NewError(fmt.Errorf("unsupported funding source: %s", s.funding.Source()), types.ErrorCodeUpdateDataError, types.ErrOptionWithSkipRetry())
	}
}

func (s *BillingSession) rollbackFundingReserve(delta int) {
	switch funding := s.funding.(type) {
	case *WalletFunding:
		if err := model.IncreaseUserQuota(funding.userId, delta, false); err != nil {
			common.SysLog("error rolling back wallet funding reserve: " + err.Error())
		} else {
			funding.consumed -= delta
		}
	case *SubscriptionFunding:
		if err := funding.ReserveDelta(-int64(delta)); err != nil {
			common.SysLog("error rolling back subscription funding reserve: " + err.Error())
		}
	}
}

func (s *BillingSession) reserveToken(delta int) error {
	if delta <= 0 || s.relayInfo.IsPlayground {
		return nil
	}
	if err := PreConsumeTokenQuota(s.relayInfo, delta); err != nil {
		return types.NewErrorWithStatusCode(err, types.ErrorCodePreConsumeTokenQuotaFailed, http.StatusForbidden, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}
	return nil
}

// shouldTrust 统一信任额度检查，适用于钱包和订阅。
func (s *BillingSession) shouldTrust(c *gin.Context) bool {
	// 异步任务（ForcePreConsume=true）必须预扣全额，不允许信任旁路
	if s.relayInfo.ForcePreConsume {
		return false
	}

	trustQuota := common.GetTrustQuota()
	if trustQuota <= 0 {
		return false
	}

	// 检查令牌是否充足
	tokenTrusted := s.relayInfo.TokenUnlimited
	if !tokenTrusted {
		tokenQuota := c.GetInt("token_quota")
		tokenTrusted = tokenQuota > trustQuota
	}
	if !tokenTrusted {
		return false
	}

	switch s.funding.Source() {
	case BillingSourceWallet:
		return s.relayInfo.UserQuota > trustQuota
	case BillingSourceSubscription:
		// 订阅不能启用信任旁路。原因：
		// 1. PreConsumeUserSubscription 要求 amount>0 来创建预扣记录并锁定订阅
		// 2. SubscriptionFunding.PreConsume 忽略参数，始终用 s.amount 预扣
		// 3. 若信任旁路将 effectiveQuota 设为 0，会导致 preConsumedQuota 与实际订阅预扣不一致
		return false
	default:
		return false
	}
}

// syncRelayInfo 将 BillingSession 的状态同步到 RelayInfo 的兼容字段上。
func (s *BillingSession) syncRelayInfo() {
	info := s.relayInfo
	info.FinalPreConsumedQuota = s.preConsumedQuota
	info.BillingSource = s.funding.Source()

	if sub, ok := s.funding.(*SubscriptionFunding); ok {
		info.SubscriptionId = sub.subscriptionId
		info.SubscriptionBillingRatio = sub.BillingRatio
		info.SubscriptionResourceKey, info.SubscriptionResourceType, info.SubscriptionResourceAmount = sub.ResourceIdentity()
		info.SubscriptionPreConsumed = sub.preConsumed + int64(s.extraReserved)
		info.SubscriptionPostDelta = 0
		info.SubscriptionAmountTotal = sub.AmountTotal
		info.SubscriptionAmountUsedAfterPreConsume = sub.AmountUsedAfter + int64(s.extraReserved)
		info.SubscriptionPlanId = sub.PlanId
		info.SubscriptionPlanTitle = sub.PlanTitle
	} else {
		info.SubscriptionId = 0
		info.SubscriptionBillingRatio = 0
		info.SubscriptionResourceKey = ""
		info.SubscriptionResourceType = ""
		info.SubscriptionResourceAmount = 0
		info.SubscriptionPreConsumed = 0
	}
}

// subscriptionPreConsumeQuota converts the live-group estimate into the
// purchased plan's snapshotted multiplier. The live estimate is retained for
// wallet fallback; only a successfully selected subscription uses this value.
func subscriptionPreConsumeQuota(relayInfo *relaycommon.RelayInfo, liveQuota int) (int, float64, bool) {
	if relayInfo == nil {
		return liveQuota, 0, false
	}
	ratio, ok, err := model.GetActiveSubscriptionBillingRatio(
		relayInfo.UserId,
		relayInfo.OriginModelName,
		relayInfo.UsingGroup,
	)
	if err != nil || !ok {
		return liveQuota, 0, false
	}
	quota, err := subscriptionQuotaForRatio(relayInfo, liveQuota, ratio)
	if err != nil || quota < 0 {
		return liveQuota, 0, false
	}
	return quota, ratio, true
}

// subscriptionQuotaForRatio converts the current request estimate to a
// package multiplier. The pre-consume quote may use one active subscription,
// while the locked transaction can select another (for example when the first
// package is exhausted), so this conversion is also used after pre-consume
// with the ratio returned by the selected subscription.
func subscriptionQuotaForRatio(relayInfo *relaycommon.RelayInfo, liveQuota int, ratio float64) (int, error) {
	if relayInfo == nil || ratio <= 0 {
		return liveQuota, nil
	}
	if ratio > model.SubscriptionMaxBillingRatio || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0, errors.New("invalid subscription billing ratio")
	}
	if relayInfo.PriceData.BaseQuotaBeforeGroup > 0 {
		return common.QuotaFromFloatStrict(relayInfo.PriceData.BaseQuotaBeforeGroup * ratio)
	}
	liveRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	raw := float64(liveQuota) * ratio
	if liveRatio > 0 {
		raw /= liveRatio
	} else {
		raw = relayInfo.PriceData.BaseQuotaBeforeGroup * ratio
	}
	return common.QuotaFromFloatStrict(raw)
}

// applySubscriptionPriceSnapshot updates the in-flight price only after the
// subscription funding source has successfully reserved quota. This keeps
// wallet fallback on the live group ratio while making all subsequent settle
// calculations use the purchased plan's fixed ratio.
func applySubscriptionPriceSnapshot(relayInfo *relaycommon.RelayInfo, liveQuota int, fixedQuota int, ratio float64) {
	if relayInfo == nil || ratio <= 0 || ratio > model.SubscriptionMaxBillingRatio || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return
	}
	if relayInfo.PriceData.GroupRatioInfo.GroupRatio != ratio {
		relayInfo.PriceData.GroupRatioInfo.GroupRatio = ratio
		relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio = ratio
		relayInfo.PriceData.GroupRatioInfo.HasSpecialRatio = true
	}
	if relayInfo.PriceData.QuotaToPreConsume == liveQuota {
		relayInfo.PriceData.QuotaToPreConsume = fixedQuota
	}
	if relayInfo.PriceData.Quota == liveQuota {
		relayInfo.PriceData.Quota = fixedQuota
	}
	if snap := relayInfo.TieredBillingSnapshot; snap != nil {
		snap.GroupRatio = ratio
		if quota, err := common.QuotaFromFloatStrict(snap.EstimatedQuotaBeforeGroup * ratio); err == nil {
			snap.EstimatedQuotaAfterGroup = quota
		}
	}
}

func liveGroupRatio(relayInfo *relaycommon.RelayInfo) float64 {
	if relayInfo == nil {
		return 1
	}
	if ratio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup); ok {
		return ratio
	}
	return ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
}

// restoreWalletPriceSnapshot switches an initially subscription-priced request
// back to the live group price when subscription funding is unavailable and
// wallet overflow is allowed.
func restoreWalletPriceSnapshot(relayInfo *relaycommon.RelayInfo, preConsumed int, fixedRatio float64) int {
	if relayInfo == nil || fixedRatio <= 0 {
		return preConsumed
	}
	liveRatio := liveGroupRatio(relayInfo)
	walletQuota := preConsumed
	if relayInfo.PriceData.BaseQuotaBeforeGroup > 0 {
		if quota, err := common.QuotaFromFloatStrict(relayInfo.PriceData.BaseQuotaBeforeGroup * liveRatio); err == nil {
			walletQuota = quota
		}
	} else if liveRatio <= 0 {
		walletQuota = 0
	} else if quota, err := common.QuotaFromFloatStrict(float64(preConsumed) * liveRatio / fixedRatio); err == nil {
		walletQuota = quota
	}
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = liveRatio
	if special, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup); ok {
		relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio = special
		relayInfo.PriceData.GroupRatioInfo.HasSpecialRatio = true
	} else {
		relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio = -1
		relayInfo.PriceData.GroupRatioInfo.HasSpecialRatio = false
	}
	relayInfo.SubscriptionBillingRatio = 0
	relayInfo.SubscriptionResourceKey = ""
	relayInfo.SubscriptionResourceType = ""
	relayInfo.SubscriptionResourceAmount = 0
	if relayInfo.PriceData.QuotaToPreConsume == preConsumed {
		relayInfo.PriceData.QuotaToPreConsume = walletQuota
	}
	if relayInfo.PriceData.Quota == preConsumed {
		relayInfo.PriceData.Quota = walletQuota
	}
	if liveRatio <= 0 {
		relayInfo.PriceData.FreeModel = true
	}
	return walletQuota
}

// ---------------------------------------------------------------------------
// NewBillingSession 工厂 — 根据计费偏好创建会话并处理回退
// ---------------------------------------------------------------------------

// NewBillingSession 根据用户计费偏好创建 BillingSession，处理 subscription_first / wallet_first 的回退。
func NewBillingSession(c *gin.Context, relayInfo *relaycommon.RelayInfo, preConsumedQuota int) (*BillingSession, *types.NewAPIError) {
	if relayInfo == nil {
		return nil, types.NewError(fmt.Errorf("relayInfo is nil"), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	pref := common.NormalizeBillingPreference(relayInfo.UserSetting.BillingPreference)
	subscriptionQuota, fixedRatio, hasFixedRatio := preConsumedQuota, 0.0, false
	if pref != "wallet_first" && pref != "wallet_only" {
		subscriptionQuota, fixedRatio, hasFixedRatio = subscriptionPreConsumeQuota(relayInfo, preConsumedQuota)
	}
	walletQuota := preConsumedQuota

	// 钱包路径需要先检查用户额度
	tryWallet := func() (*BillingSession, *types.NewAPIError) {
		if hasFixedRatio {
			walletQuota = restoreWalletPriceSnapshot(relayInfo, preConsumedQuota, fixedRatio)
		}
		userQuota, err := model.GetUserQuota(relayInfo.UserId, false)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
		}
		if walletQuota > 0 && userQuota <= 0 {
			return nil, types.NewErrorWithStatusCode(
				fmt.Errorf("用户额度不足, 剩余额度: %s", logger.FormatQuota(userQuota)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		if userQuota-walletQuota < 0 {
			return nil, types.NewErrorWithStatusCode(
				fmt.Errorf("预扣费额度失败, 用户剩余额度: %s, 需要预扣费额度: %s", logger.FormatQuota(userQuota), logger.FormatQuota(walletQuota)),
				types.ErrorCodeInsufficientUserQuota, http.StatusForbidden,
				types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		relayInfo.UserQuota = userQuota

		session := &BillingSession{
			relayInfo: relayInfo,
			funding:   &WalletFunding{userId: relayInfo.UserId},
		}
		if apiErr := session.preConsume(c, walletQuota); apiErr != nil {
			return nil, apiErr
		}
		return session, nil
	}

	trySubscription := func() (*BillingSession, *types.NewAPIError) {
		// wallet_first intentionally leaves the first attempt on live pricing;
		// recompute the fixed snapshot only after that wallet attempt fails.
		subQuota, subRatio, subHasFixed := subscriptionPreConsumeQuota(relayInfo, preConsumedQuota)
		if !subHasFixed {
			subQuota = subscriptionQuota
			subRatio = fixedRatio
			subHasFixed = hasFixedRatio
		}
		subConsume := int64(subQuota)
		if subConsume <= 0 {
			subConsume = 1
		}
		resource := subscriptionResourceRequest(relayInfo, subConsume)
		session := &BillingSession{
			relayInfo: relayInfo,
			funding: &SubscriptionFunding{
				requestId:      relayInfo.RequestId,
				userId:         relayInfo.UserId,
				modelName:      relayInfo.OriginModelName,
				effectiveGroup: relayInfo.UsingGroup,
				amount:         subConsume,
				resource:       resource,
				pricing: &model.SubscriptionQuotaPricing{
					BaseQuotaBeforeGroup: relayInfo.PriceData.BaseQuotaBeforeGroup,
					LiveQuota:            int64(preConsumedQuota),
				},
			},
		}
		// 必须传 subConsume 而非 preConsumedQuota，保证 SubscriptionFunding.amount、
		// preConsume 参数和 FinalPreConsumedQuota 三者一致，避免订阅多扣费。
		if apiErr := session.preConsume(c, int(subConsume)); apiErr != nil {
			return nil, apiErr
		}
		// Use the ratio returned by the locked pre-consume transaction. The
		// earlier read-only quote can point at an exhausted subscription with a
		// different multiplier, or miss a plan reached through a resource grant.
		actualRatio := subRatio
		if funding, ok := session.funding.(*SubscriptionFunding); ok {
			// The locked transaction is authoritative because it may skip an
			// exhausted package that had a different fixed multiplier.
			actualRatio = funding.BillingRatio
		}
		if actualRatio > 0 {
			fixedQuota := session.preConsumedQuota
			applySubscriptionPriceSnapshot(relayInfo, preConsumedQuota, fixedQuota, actualRatio)
		}
		return session, nil
	}

	switch pref {
	case "subscription_only":
		return trySubscription()
	case "wallet_only":
		return tryWallet()
	case "wallet_first":
		session, err := tryWallet()
		if err != nil {
			if err.GetErrorCode() == types.ErrorCodeInsufficientUserQuota {
				return trySubscription()
			}
			return nil, err
		}
		return session, nil
	case "subscription_first":
		fallthrough
	default:
		hasSub, subCheckErr := model.HasActiveUserSubscriptionForModel(relayInfo.UserId, relayInfo.OriginModelName, relayInfo.UsingGroup)
		if subCheckErr != nil {
			return nil, types.NewError(subCheckErr, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
		}
		if !hasSub {
			return tryWallet()
		}
		session, apiErr := trySubscription()
		if apiErr != nil {
			if apiErr.GetErrorCode() == types.ErrorCodeInsufficientUserQuota {
				// 仅当用户的活跃订阅允许钱包回退时才回退到钱包，否则返回订阅额度不足错误
				allowOverflow, overflowErr := model.UserActiveSubscriptionsAllowWalletOverflowForModel(relayInfo.UserId, relayInfo.OriginModelName, relayInfo.UsingGroup)
				if overflowErr != nil {
					return nil, types.NewError(overflowErr, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
				}
				if allowOverflow {
					return tryWallet()
				}
				return nil, apiErr
			}
			return nil, apiErr
		}
		return session, nil
	}
}

func subscriptionResourceRequest(relayInfo *relaycommon.RelayInfo, quotaAmount int64) *model.SubscriptionResourceRequest {
	if relayInfo == nil || quotaAmount <= 0 {
		return nil
	}
	modelName := strings.TrimSpace(strings.ToLower(relayInfo.OriginModelName))
	if modelName == "" {
		return nil
	}
	if isSubscriptionImageModel(modelName) {
		count := int64(1)
		if imageRequest, ok := relayInfo.Request.(*dto.ImageRequest); ok {
			if imageRequest.N != nil && *imageRequest.N > 0 {
				count = int64(*imageRequest.N)
			}
		} else if geminiRequest, ok := relayInfo.Request.(*dto.GeminiChatRequest); ok {
			if candidateCount := geminiRequest.GenerationConfig.CandidateCount; candidateCount != nil && *candidateCount > 0 {
				count = int64(*candidateCount)
			}
		}
		if count > int64(dto.MaxImageN) {
			count = int64(dto.MaxImageN)
		}
		return &model.SubscriptionResourceRequest{
			ResourceKey:  subscriptionImageResourceKey(modelName),
			ResourceType: model.SubscriptionResourceTypeImageCount,
			Amount:       count,
		}
	}
	return &model.SubscriptionResourceRequest{
		ResourceKey:  modelName,
		ResourceType: model.SubscriptionResourceTypeQuota,
		Amount:       quotaAmount,
	}
}

func isSubscriptionImageModel(modelName string) bool {
	return strings.HasPrefix(modelName, "gpt-image-2") ||
		strings.HasPrefix(modelName, "gemini-3-pro-image") ||
		strings.HasPrefix(modelName, "gemini-3.1-flash-image") ||
		modelName == "gemini-2.5-flash-image" ||
		modelName == "gemini-2.0-flash-exp-image-generation" ||
		modelName == "gemini-2.0-flash-exp" ||
		modelName == "nano-banana-pro-preview"
}

func subscriptionImageResourceKey(modelName string) string {
	if strings.HasPrefix(modelName, "gpt-image-2") {
		return "gpt-image-2"
	}
	return "nano-banana"
}
