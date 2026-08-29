package payhere

import "testing"

func TestRequestHash(t *testing.T) {
	got := RequestHash("123", "order-1", "1000.00", "LKR", "secret")
	want := "1595A0F471FAC19C0CACBD885E3A9973"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestVerifyNotification(t *testing.T) {
	cfg := Config{MerchantID: "123", MerchantSecret: "secret"}
	n := Notification{MerchantID: "123", OrderID: "order-1", PaymentID: "p-1", Amount: "1000.00", Currency: "LKR", StatusCode: "2"}
	n.Signature = NotificationHash(n, cfg.MerchantSecret)
	if err := VerifyNotification(n, cfg); err != nil {
		t.Fatal(err)
	}
	n.Amount = "1.00"
	if err := VerifyNotification(n, cfg); err == nil {
		t.Fatal("expected tampering rejection")
	}
}
