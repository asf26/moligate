package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// NormalizeVideoAccountGroups turns the administrator's group selection into
// the compact representation stored in video_accounts.groups. The account is
// intentionally scoped by the gateway's user group, not by generic Channel
// rows, so an account can be enabled for one or more dashboard groups without
// being mistaken for a relay channel.
func NormalizeVideoAccountGroups(groups []string) string {
	seen := make(map[string]struct{}, len(groups))
	normalized := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		key := strings.ToLower(group)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, group)
	}
	if len(normalized) == 0 {
		return "default"
	}
	return strings.Join(normalized, ",")
}

func GenerateVideoAccountTokenID() (string, error) {
	key, err := common.GenerateRandomCharsKey(40)
	if err != nil {
		return "", fmt.Errorf("generate video account token id: %w", err)
	}
	return "vca_" + key, nil
}

// ValidateVideoAccount performs validation shared by create and update. The
// upstream endpoint is deliberately fixed; accepting an arbitrary base URL
// here would turn this dedicated integration back into a generic channel.
func ValidateVideoAccount(account *VideoAccount, requireAPIKey bool) error {
	if account == nil {
		return errors.New("video account is nil")
	}
	account.Name = strings.TrimSpace(account.Name)
	if account.Name == "" {
		return errors.New("video account name is required")
	}
	if len(account.Name) > 128 {
		return errors.New("video account name is too long")
	}
	account.ApiKey = strings.TrimSpace(account.ApiKey)
	if requireAPIKey && account.ApiKey == "" {
		return errors.New("video account api key is required")
	}
	if len(account.ApiKey) > 8192 {
		return errors.New("video account api key is too long")
	}
	account.Groups = NormalizeVideoAccountGroups(account.GroupList())
	if len(account.Groups) > 255 {
		return errors.New("video account groups are too long")
	}
	account.Proxy = strings.TrimSpace(account.Proxy)
	if account.Proxy != "" {
		if _, err := common.ParseProxyURLStrict(account.Proxy); err != nil {
			return fmt.Errorf("invalid video account proxy: %w", err)
		}
	}
	if account.Status != VideoAccountStatusDisabled && account.Status != VideoAccountStatusEnabled {
		return errors.New("invalid video account status")
	}
	account.Remark = strings.TrimSpace(account.Remark)
	return nil
}

// CreateVideoAccount persists an already validated account. Callers should
// populate PublicKey before this function; keeping persistence free of HTTP
// calls makes the model layer deterministic and easy to test.
func CreateVideoAccount(account *VideoAccount) error {
	if err := ValidateVideoAccount(account, true); err != nil {
		return err
	}
	if strings.TrimSpace(account.PublicKey) == "" {
		key, err := GenerateVideoAccountTokenID()
		if err != nil {
			return err
		}
		account.PublicKey = key
	}
	account.PublicKey = strings.TrimSpace(account.PublicKey)
	return DB.Create(account).Error
}

func UpdateVideoAccount(account *VideoAccount) error {
	if err := ValidateVideoAccount(account, true); err != nil {
		return err
	}
	if account.Id <= 0 {
		return errors.New("invalid video account id")
	}
	if strings.TrimSpace(account.PublicKey) == "" {
		return errors.New("video account token id is required")
	}
	return DB.Save(account).Error
}

func DeleteVideoAccount(id int) error {
	if id <= 0 {
		return errors.New("invalid video account id")
	}
	result := DB.Delete(&VideoAccount{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
