package payhere

import (
	"crypto/md5" // PayHere mandates MD5 for its protocol checksum.
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

const (
	SandboxCheckoutURL = "https://sandbox.payhere.lk/pay/checkout"
	LiveCheckoutURL    = "https://www.payhere.lk/pay/checkout"
)

type Config struct {
	MerchantID, MerchantSecret, CheckoutURL string
}

type Notification struct {
	MerchantID, OrderID, PaymentID, Amount, Currency, StatusCode, Signature string
}

var callbackAmount = regexp.MustCompile(`^(0|[1-9][0-9]*)\.[0-9]{2}$`)
var callbackCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

func digest(value string) string {
	sum := md5.Sum([]byte(value))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func RequestHash(merchantID, orderID, amount, currency, secret string) string {
	return digest(merchantID + orderID + amount + currency + digest(secret))
}

func NotificationHash(n Notification, secret string) string {
	return digest(n.MerchantID + n.OrderID + n.Amount + n.Currency + n.StatusCode + digest(secret))
}

func VerifyNotification(n Notification, cfg Config) error {
	if n.MerchantID == "" || n.OrderID == "" || n.PaymentID == "" || n.Amount == "" || n.Currency == "" || n.StatusCode == "" || n.Signature == "" {
		return errors.New("missing required callback field")
	}
	if !callbackAmount.MatchString(n.Amount) || !callbackCurrency.MatchString(n.Currency) {
		return errors.New("invalid callback money")
	}
	switch n.StatusCode {
	case "2", "0", "-1", "-2", "-3":
	default:
		return errors.New("unsupported callback status")
	}
	if len(n.Signature) != 32 {
		return errors.New("invalid callback signature")
	}
	if subtle.ConstantTimeCompare([]byte(n.MerchantID), []byte(cfg.MerchantID)) != 1 {
		return errors.New("merchant mismatch")
	}
	expected := NotificationHash(n, cfg.MerchantSecret)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToUpper(n.Signature))) != 1 {
		return errors.New("signature mismatch")
	}
	return nil
}
