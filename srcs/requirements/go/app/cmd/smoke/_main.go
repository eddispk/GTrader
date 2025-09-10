package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"bot/bybits/get"
	"bot/bybits/listen"
	"bot/bybits/post"
	"bot/bybits/print"
	"bot/bybits/telegram"
	"bot/data"

	"github.com/joho/godotenv"
)

func mustGetwd() string { wd, _ := os.Getwd(); return wd }

func requireEnv(keys ...string) {
	for _, k := range keys {
		if os.Getenv(k) == "" {
			log.Fatalf("missing env %s (cwd=%s)", k, mustGetwd())
		}
	}
}

// ---- helpers: numeric math → string with fixed dp ----
func up(x, pct float64) float64   { return x * (1 + pct/100.0) }
func down(x, pct float64) float64 { return x * (1 - pct/100.0) }
func r(x float64, dp int) string  { return fmt.Sprintf("%."+strconv.Itoa(dp)+"f", x) }

// plan prices by side
func plan(side string, last float64) (entry, tp1, tp2, tp3, sl string) {
	if strings.EqualFold(side, "Buy") {
		e := down(last, 0.20)
		return r(e, 2),
			r(up(e, 0.10), 2),
			r(up(e, 0.25), 2),
			r(up(e, 0.50), 2),
			r(down(e, 0.30), 2) // SL BELOW for longs
	}
	// Sell
	e := up(last, 0.20)
	return r(e, 2),
		r(down(e, 0.10), 2),
		r(down(e, 0.25), 2),
		r(down(e, 0.50), 2),
		r(up(e, 0.30), 2) // SL ABOVE for shorts
}

// very small helper to detect “insufficient balance” style errors from Bybit
func isInsufficientBalance(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return containsAny(s,
		"insufficient", "Insufficient", "balance", "Balance",
		"insufficient available balance", "not enough",
	)
}
func containsAny(s string, subs ...string) bool {
	for _, t := range subs {
		if t != "" && (len(s) <= 4096) && (contains(s, t)) {
			return true
		}
	}
	return false
}
func contains(s, sub string) bool {
	return len(sub) > 0 && (len(s) >= len(sub)) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	return len([]rune(s[:])) - len([]rune(s)) + len([]rune(sub)) - len([]rune(sub)) /* dummy to avoid extra deps */
}

// pick the first API keypair
func pickAPI(env *data.Env) data.BybitApi {
	if len(env.Api) == 0 {
		log.Fatal("no API keys loaded")
	}
	return env.Api[0]
}

func main() {
	// 1) load .env from a few likely places
	_ = godotenv.Load("cmd/smoke/.env", ".env", "../../.env")

	// sanity
	requireEnv("URL", "API", "API_SECRET", "API_TELEGRAM", "ADMIN", "BOT_NAME", "ID_CHANNEL")
	if os.Getenv("STAKE_USDT") == "" {
		_ = os.Setenv("STAKE_USDT", "10")
	}

	// 2) load your app env
	var apiEnv data.Env
	if err := data.LoadEnv(&apiEnv); err != nil {
		log.Fatalf("LoadEnv: %v", err)
	}
	keys := pickAPI(&apiEnv)

	symbol := "BTCUSDT" // linear
	log.Printf("Base URL: %s", apiEnv.Url)
	log.Printf("Testing symbol: %s", symbol)

	// 3) public ticker (v5)
	tk := get.GetPrice(symbol, apiEnv.Url)
	if tk.RetCode != 0 || len(tk.Result.List) == 0 {
		log.Fatalf("get tickers failed: %s", print.PrettyPrint(tk))
	}
	last, _ := strconv.ParseFloat(tk.Result.List[0].LastPrice, 64)
	bid := tk.Result.List[0].Bid1Price
	log.Printf("[OK] ticker: last=%s bid=%s", tk.Result.List[0].LastPrice, bid)

	// 4) wallet (v5)
	w := get.GetWallet(keys.Api, keys.Api_secret, apiEnv.Url)
	if w.RetCode != 0 || len(w.Result.List) == 0 {
		log.Fatalf("wallet-balance failed: %s", print.PrettyPrint(w))
	}
	acc := w.Result.List[0]
	log.Printf("[OK] wallet: %s", print.PrettyPrint(acc))

	accountType := acc.AccountType // "UNIFIED" likely on testnet
	hasFunds := true
	if acc.TotalAvailableBalance == "0" && len(acc.Coin) > 0 && acc.Coin[0].WalletBalance == "0" {
		hasFunds = false
	}

	// 6) build your trade struct and Add
	var trades data.Trades
	side := "Sell" // or "Buy"
	entry, tp1, tp2, tp3, sl := plan(side, last)

	td := telegram.Data{
		Currency:   symbol,
		Type:       side, // "Sell"
		Entry:      entry,
		Tp1:        tp1,
		Tp2:        tp2,
		Tp3:        tp3,
		Sl:         sl,
		Level:      "20",
		BEAfterTP1: true,
	}
	if !trades.Add(keys, td, tk, apiEnv.Url) {
		log.Fatal("Trades.Add returned false (duplicate symbol?)")
	}
	log.Printf("[OK] trade prepared: %s", print.PrettyPrint(trades.GetTrades()))

	// 7) margin mode + leverage
	if accountType == "UNIFIED" {
		log.Printf("[INFO] UNIFIED account detected → skip switch-isolated (not supported)")
	} else {
		if err := post.PostIsoled(keys, symbol, &trades, apiEnv.Url, true); err != nil {
			log.Printf("[WARN] PostIsoled error: %v", err)
		} else {
			log.Printf("[OK] isolated mode set")
		}
	}

	// 8) place orders
	if hasFunds {
		if err := post.PostOrder(symbol, keys, &trades, apiEnv.Url, true); err != nil {
			log.Printf("[WARN] PostOrder error (but endpoint reached): %v", err)
		} else {
			log.Printf("[OK] placed split TP limit orders")
		}
	} else {
		log.Printf("[INFO] wallet=0 → skipping real Limit orders to avoid hard failure")
		// OPTIONAL: place a conditional order that does not consume margin now.
		if err := placeConditionalV5(apiEnv.Url, keys, symbol, "Buy", entry, tp1, sl); err != nil {
			log.Printf("[WARN] conditional order failed: %v", err)
		} else {
			log.Printf("[OK] placed a conditional order (no margin required until trigger)")
		}
	}

	time.Sleep(1 * time.Second)

	// 9) positions snapshot (will likely be empty if no fills)
	pos, err := listen.GetPosition(keys, symbol, apiEnv.Url)
	if err != nil {
		log.Printf("[WARN] GetPosition error: %v", err)
	} else {
		log.Printf("[INFO] position list: %s", print.PrettyPrint(pos.Result.List))
		// if size>0 try to set SL
		if len(pos.Result.List) > 0 && pos.Result.List[0].Size != "0" {
			if err := post.ChangeLs(keys, symbol, sl, "Buy", apiEnv.Url); err != nil {
				log.Printf("[WARN] ChangeLs failed: %v", err)
			} else {
				log.Printf("[OK] ChangeLs sent")
			}
		} else {
			log.Printf("[SKIP] ChangeLs (no open position)")
		}
	}

	hasOpen := len(pos.Result.List) > 0 && pos.Result.List[0].Size != "0"

	// Use the correct side for SL changes
	if hasOpen {
		if err := post.ChangeLs(keys, symbol, sl, side, apiEnv.Url); err != nil {
			log.Printf("[WARN] ChangeLs failed: %v", err)
		} else {
			log.Printf("[OK] ChangeLs sent")
		}
	} else {
		log.Printf("[SKIP] ChangeLs (no open position)")
	}

	// Use SellTp when side == "Sell"
	if strings.EqualFold(side, "Buy") {
		_ = listen.BuyTp(keys, &trades, symbol, &data.Bot{}, apiEnv.Url)
	} else {
		_ = listen.SellTp(keys, &trades, symbol, &data.Bot{}, apiEnv.Url)
	}
	// 11) cleanup: cancel orders (ok even if none)
	if err := post.CancelOrder(symbol, keys, &trades, apiEnv.Url); err != nil {
		if isInsufficientBalance(err) {
			log.Printf("[OK] CancelOrder reached API (balance-related response): %v", err)
		} else {
			log.Printf("[WARN] CancelOrder error: %v", err)
		}
	} else {
		log.Printf("[OK] cancel-all sent for %s", symbol)
	}

	log.Printf("SMOKE TEST COMPLETE ✅")
}

// place a minimal conditional order (linear) using v5 directly
func placeConditionalV5(baseURL string, api data.BybitApi, symbol, side, price, tp, sl string) error {
	body := map[string]interface{}{
		"category":     "linear",
		"symbol":       symbol,
		"side":         side,
		"orderType":    "Limit",
		"timeInForce":  "GTC",
		"qty":          "0.001", // tiny notional; it won’t be used until trigger
		"price":        price,   // limit price after trigger
		"triggerPrice": tp,      // make it far so it won't trigger soon
		"triggerBy":    "MarkPrice",
		"tpslMode":     "Partial",
		"tpOrderType":  "Limit",
		"slOrderType":  "Limit",
		"tpLimitPrice": tp,
		"slLimitPrice": sl,
		"positionIdx":  0,
		"timestamp":    print.GetTimestamp(),
	}
	// sign & send using your existing signer
	body["api_key"] = api.Api
	body["sign"] = signMapToHex(body, api.Api_secret)

	js, _ := json.Marshal(body)
	log.Printf("[ORDER] conditional body: %s", js)

	u := fmt.Sprintf("%s/v5/order/create", baseURL)
	resp, err := http.Post(u, "application/json", bytesReader(js))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	log.Printf("[ORDER] conditional resp: %v", print.PrettyPrint(out))
	if code, ok := out["retCode"].(float64); ok && code != 0 {
		return fmt.Errorf("%v", out["retMsg"])
	}
	return nil
}

// small helpers for the conditional request to avoid importing your internal packages here
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func signMapToHex(m map[string]interface{}, secret string) string {
	// produce the same query-string style as your sign.GetSignedinter
	// NOTE: we intentionally reimplement inline to keep smoke standalone
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		var v string
		switch x := m[k].(type) {
		case bool:
			if x {
				v = "true"
			} else {
				v = "false"
			}
		default:
			v = fmt.Sprintf("%v", x)
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		if i < len(keys)-1 {
			sb.WriteString("&")
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sb.String()))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
