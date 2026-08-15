package controller

import (
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/setting/system_setting"
)

func paymentReturnPath(suffix string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	return base + suffix
}

func paymentResultPath(kind string, status string) string {
	query := url.Values{}
	query.Set("kind", kind)
	query.Set("status", status)
	return paymentReturnPath("/payment/result?" + query.Encode())
}
