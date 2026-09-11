package main

import "testing"

func TestDecidePaymentNotification(t *testing.T) {
	tests := []struct {
		name          string
		amount        float64
		action, title string
	}{{"ordinary", 125.5, "view", "Payment received"}, {"high value", 10000, "review", "Payment review"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := decide(PaymentEvent{Amount: tt.amount, Currency: "USD", Reference: "pay-1"})
			if n.Action != tt.action || n.Title != tt.title {
				t.Fatalf("got %#v", n)
			}
		})
	}
}
