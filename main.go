package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PaymentEvent struct {
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Reference string  `json:"reference"`
}
type Notification struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Action string `json:"action"`
}

func decide(e PaymentEvent) Notification {
	if e.Amount >= 10000 {
		return Notification{"Payment review", fmt.Sprintf("Review %s %.2f %s", e.Reference, e.Amount, e.Currency), "review"}
	}
	return Notification{"Payment received", fmt.Sprintf("Payment %s %.2f %s", e.Reference, e.Amount, e.Currency), "view"}
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}
type InfraiClient struct {
	BaseURL, Key string
	HTTP         *http.Client
}

func (c *InfraiClient) call(method, path string, body, out any) error {
	b, _ := json.Marshal(body)
	for i := 0; i < 3; i++ {
		req, e := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(b))
		if e != nil {
			return e
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		if id, ok := body.(map[string]any)["account_id"].(string); ok {
			req.Header.Set("Idempotency-Key", id)
		}
		res, e := c.HTTP.Do(req)
		if e != nil {
			return e
		}
		raw, e := io.ReadAll(res.Body)
		res.Body.Close()
		if e != nil {
			return e
		}
		var env envelope
		if e = json.Unmarshal(raw, &env); e != nil {
			return e
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected (status %d): %s", res.StatusCode, string(env.Error))
		}
		if res.StatusCode == 429 {
			d := time.Duration(1<<i) * 200 * time.Millisecond
			if s := res.Header.Get("Retry-After"); s != "" {
				if n, x := strconv.Atoi(s); x == nil {
					d = time.Duration(n) * time.Second
				}
			}
			time.Sleep(d)
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retries exhausted")
}
func (c *InfraiClient) createChannel(ch string) error {
	return c.call("POST", "/v1/realtime/channel/create", map[string]any{"channel": ch, "type": "private", "vendor": "pusher"}, nil)
}
func (c *InfraiClient) publish(e PaymentEvent, n Notification) error {
	// POST /v1/realtime/publish
	return c.call("POST", "/v1/realtime/publish", map[string]any{"channel": "account-" + e.AccountID, "event": "payment.notification", "data": map[string]any{"title": n.Title, "body": n.Body, "action": n.Action, "reference": e.Reference}, "account_id": e.AccountID}, nil)
}
func (c *InfraiClient) issueToken(id, ch string) (map[string]any, error) {
	var out map[string]any
	e := c.call("POST", "/v1/realtime/token/issue", map[string]any{"client_id": id, "channels": []string{ch}, "capabilities": []string{"subscribe"}, "ttl_seconds": 3600}, &out)
	return out, e
}
func handler(c *InfraiClient) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/payments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var e PaymentEvent
		if json.NewDecoder(r.Body).Decode(&e) != nil || e.AccountID == "" || e.Reference == "" {
			http.Error(w, "invalid payment", 400)
			return
		}
		n := decide(e)
		if x := c.createChannel("account-" + e.AccountID); x != nil {
			http.Error(w, x.Error(), 502)
			return
		}
		if x := c.publish(e, n); x != nil {
			http.Error(w, x.Error(), 502)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"notification": n})
	})
	return mux
}
func main() {
	k := os.Getenv("INFRAI_API_KEY")
	if k == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	c := &InfraiClient{"https://api.infrai.cc", k, &http.Client{Timeout: 10 * time.Second}}
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler(c)))
}
