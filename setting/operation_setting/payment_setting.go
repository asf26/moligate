package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额折扣阈值，例如 100 元 0.9 表示充值满 100 元享受 9 折优惠
	RechargeBonus  RechargeBonus   `json:"recharge_bonus"`
	InviteRanking  InviteRanking   `json:"invite_ranking"`

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

type RechargeBonus struct {
	Enabled        bool    `json:"enabled"`
	MinAmount      int64   `json:"min_amount"`
	BonusRate      float64 `json:"bonus_rate"`
	StartTime      int64   `json:"start_time"`
	EndTime        int64   `json:"end_time"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	ShowOnTopup    bool    `json:"show_on_topup"`
	ShowBonusRatio bool    `json:"show_bonus_ratio"`
}

type InviteRanking struct {
	Enabled     bool   `json:"enabled"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	Title       string `json:"title"`
	ShowTopN    int    `json:"show_top_n"`
	MaskUsers   bool   `json:"mask_users"`
	ShowOnTopup bool   `json:"show_on_topup"`
}

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:  []int{10, 20, 50, 100, 200, 500},
	AmountDiscount: map[int]float64{},
	RechargeBonus: RechargeBonus{
		BonusRate:      0.1,
		ShowOnTopup:    true,
		ShowBonusRatio: true,
	},
	InviteRanking: InviteRanking{
		Title:       "Invite Leaderboard",
		ShowTopN:    5,
		MaskUsers:   true,
		ShowOnTopup: true,
	},
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	if paymentSetting.RechargeBonus.BonusRate < 0 {
		paymentSetting.RechargeBonus.BonusRate = 0
	}
	if paymentSetting.InviteRanking.ShowTopN <= 0 {
		paymentSetting.InviteRanking.ShowTopN = 5
	}
	if paymentSetting.InviteRanking.ShowTopN > 20 {
		paymentSetting.InviteRanking.ShowTopN = 20
	}
	return &paymentSetting
}

func ResolveAmountDiscount(amount int64, discounts map[int]float64) float64 {
	discount := 1.0
	matchedThreshold := int64(0)

	for threshold, rate := range discounts {
		thresholdAmount := int64(threshold)
		if thresholdAmount <= 0 || thresholdAmount > amount || rate <= 0 {
			continue
		}
		if thresholdAmount > matchedThreshold {
			matchedThreshold = thresholdAmount
			discount = rate
		}
	}

	return discount
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}

func (bonus RechargeBonus) IsActive(now int64) bool {
	if !bonus.Enabled || bonus.BonusRate <= 0 {
		return false
	}
	if bonus.StartTime > 0 && now < bonus.StartTime {
		return false
	}
	if bonus.EndTime > 0 && now > bonus.EndTime {
		return false
	}
	return true
}

func (bonus RechargeBonus) AppliesToAmount(amount int64, now int64) bool {
	if !bonus.IsActive(now) {
		return false
	}
	return bonus.MinAmount <= 0 || amount >= bonus.MinAmount
}

func (ranking InviteRanking) IsActive(now int64) bool {
	if !ranking.Enabled {
		return false
	}
	if ranking.StartTime > 0 && now < ranking.StartTime {
		return false
	}
	if ranking.EndTime > 0 && now > ranking.EndTime {
		return false
	}
	return true
}
