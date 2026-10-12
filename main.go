package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	order := Order{ID: "ord_1042", Customer: "Northwind Labs", Amount: 1840.50, Currency: "USD"}
	event := PaymentEvent{OrderID: order.ID, Status: "paid", At: time.Now().UTC()}
	approved, notice := shouldGenerate(event)
	encoded, _ := json.Marshal(notice)
	fmt.Printf("audit=%s\n", encoded)
	if !approved {
		return
	}
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "INFRAI_API_KEY is required")
		os.Exit(1)
	}
	pdf, err := generatePDF(http.DefaultClient, key, order)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writePDF("invoice-"+order.ID+".pdf", pdf); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("invoice written")
}
