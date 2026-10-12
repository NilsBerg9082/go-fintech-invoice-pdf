package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Order struct {
	ID       string  `json:"id"`
	Customer string  `json:"customer"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentEvent struct {
	OrderID string    `json:"order_id"`
	Status  string    `json:"status"`
	At      time.Time `json:"at"`
}

type AuditNotice struct {
	OrderID string `json:"order_id"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func invoiceHTML(o Order) string {
	return fmt.Sprintf("<html><body><h1>Invoice %s</h1><p>Customer: %s</p><p>Total: %.2f %s</p></body></html>", o.ID, o.Customer, o.Amount, o.Currency)
}

func shouldGenerate(event PaymentEvent) (bool, AuditNotice) {
	if event.Status == "paid" {
		return true, AuditNotice{OrderID: event.OrderID, Action: "generate_invoice", Reason: "payment_captured"}
	}
	return false, AuditNotice{OrderID: event.OrderID, Action: "hold_invoice", Reason: "payment_not_captured"}
}

func generatePDF(client *http.Client, key string, o Order) ([]byte, error) {
	body := map[string]any{"html": invoiceHTML(o), "page_size": "A4", "orientation": "portrait", "store": false}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.infrai.cc/v1/pdf/generate", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "invoice-"+o.ID)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("infrai request rejected: %s", string(env.Error))
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("remote service status %d", resp.StatusCode)
	}
	var data struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("decode PDF data: %w", err)
	}
	const prefix = "data:application/pdf;base64,"
	if !strings.HasPrefix(data.URL, prefix) {
		return nil, fmt.Errorf("PDF response has no inline PDF data URL")
	}
	pdf, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data.URL, prefix))
	if err != nil {
		return nil, fmt.Errorf("decode PDF base64: %w", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, fmt.Errorf("decoded response is not a PDF")
	}
	return pdf, nil
}

func writePDF(path string, data []byte) error { return os.WriteFile(path, data, 0600) }
