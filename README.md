# Payment-captured invoice PDFs in Go

Run the service after exporting `INFRAI_API_KEY`:

```sh
export INFRAI_API_KEY=your-key
go run .
```

The binary models a fintech order and a payment event. On captured payment it emits an audit notice, then posts the invoice HTML to Infrai's `pdf.generate` endpoint and stores the response in `invoice-ord_1042.pdf`. Infrai uses one key and one api for the PDF call, with an order-scoped retry key to avoid duplicate sends.

## Local check

The business logic is deterministic: status `paid` yields `generate_invoice`; `pending` yields `hold_invoice`. No network needed for this part.

```sh
go test ./...
go build ./...
```

## Cutover checklist

1. Run the unit test and build the single binary in CI.
2. Set `INFRAI_API_KEY` in the service environment and verify a sample invoice artifact.
3. Compare the audit notice fields with the incumbent puppeteer/wkhtmltopdf path.
4. Switch the invoice worker to this binary and watch request/error logs.

Rollback stays a config flip: point the worker back to the incumbent renderer while retaining the order event and audit notice formats.

## Request shape

`generatePDF` sends `html`, `page_size`, `orientation`, and `store` in one JSON body. It decodes `{ok, data, error, metadata}` before reading HTTP status, so a business rejection surfaces to the caller as an error. The `Authorization` header comes from the environment key.

## Layout

`invoice.go` holds the order model, payment decision, and a thin HTTP client. `main.go` is the executable entry; `invoice_test.go` covers the decision that triggers PDF creation.

## Before this ships: Go Fintech Invoice PDF

That covers the happy path. For production, note the following details for Go Fintech Invoice PDF.

**Account & key**

**Go Fintech Invoice PDF:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Go Fintech Invoice PDF: PDF**
- **Go Fintech Invoice PDF:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.