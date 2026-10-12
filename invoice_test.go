package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestGeneratePDFDecodesInlineData(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"data":{"url":"data:application/pdf;base64,JVBERi0xLjQK"}}`)), Header: make(http.Header)}, nil
	})}
	pdf, err := generatePDF(client, "test-key", Order{ID: "o-1"})
	if err != nil || string(pdf) != "%PDF-1.4\n" {
		t.Fatalf("got %q, err %v", pdf, err)
	}
}

func TestShouldGenerateOnlyAfterPayment(t *testing.T) {
	tests := []struct {
		name, status string
		want         bool
		action       string
	}{
		{"captured", "paid", true, "generate_invoice"},
		{"pending", "pending", false, "hold_invoice"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, notice := shouldGenerate(PaymentEvent{OrderID: "o-1", Status: tc.status})
			if got != tc.want || notice.Action != tc.action {
				t.Fatalf("got %v/%s, want %v/%s", got, notice.Action, tc.want, tc.action)
			}
		})
	}
}
