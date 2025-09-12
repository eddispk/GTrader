package listen

import (
	"bot/bybits/get"
	"bot/bybits/post"
	"bot/bybits/print"
	"bot/data"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	notify "bot/bybits/bot"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var runnerEnabled = map[string]bool{}

func GetPosition(api data.BybitApi, symbol string, url_bybite string) (get.Position, error) {
	var position get.Position
	settle := strings.TrimSpace(os.Getenv("SETTLE_COIN"))
	if settle == "" {
		settle = "USDT"
	}

	q := map[string]string{
		"category":   "linear",
		"symbol":     symbol,
		"openOnly":   "1",
		"settleCoin": settle,
	}
	body, err := get.PrivateGET(url_bybite, "/v5/position/list", q, api.Api, api.Api_secret)
	if err != nil {
		log.Println(err)
		return position, err
	}
	if err := json.Unmarshal(body, &position); err != nil {
		log.Println(err)
		return position, err
	}
	return position, nil
}

func BuyTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url string, idChannel string) error {
	price := get.GetPrice(symbol, url)
	if len(price.Result.List) == 0 {
		return nil
	}

	parseMaybe := func(s string) (float64, bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, false
		}
		v, err := strconv.ParseFloat(s, 64)
		return v, err == nil
	}

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	slStr := trade.GetSl(symbol)
	sl, _ := strconv.ParseFloat(slStr, 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)

	tp1, hasTP1 := parseMaybe(trade.GetTp1(symbol))
	tp2, hasTP2 := parseMaybe(trade.GetTp2(symbol))
	tp3, hasTP3 := parseMaybe(trade.GetTp3(symbol))
	tp4, hasTP4 := parseMaybe(trade.GetTp4(symbol))

	// SL touched → close
	if last <= sl {
		msg := fmt.Sprintf("🔴 [SL] %s BUY: SL touched → closed (last=%.6f, SL=%s)", symbol, last, slStr)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	// PRIORITY (highest first):
	// 1) TP4 hit  -> SL = TP3
	// 2) TP2 hit  -> SL = TP1
	// 3) TP1 hit  -> SL = BE (entry)
	switch {

	case hasTP4 && hasTP3 && last >= tp4:
		// step 1: lock to TP3 (only if it actually raises SL)
		newSL := fmt.Sprintf("%.6f", tp3)
		if newSL != currentSL {
			if err := post.ChangeLs(api, symbol, newSL, trade.GetType(symbol), url); err == nil {
				trade.SetSl(symbol, newSL)
				notify.SendToChannel(order, idChannel,
					fmt.Sprintf("😎 [TP] %s BUY: TP4 reached (%.6f) -> SL moved to TP3 (%s)", symbol, tp4, newSL))
			} else {
				log.Printf("[WARN] ChangeLs failed for %s: %v", symbol, err)
			}
		}
		// step 2: enable trailing stop (one-time per session)
		if !runnerEnabled[symbol] {
			pct := 1.0
			if s := strings.TrimSpace(os.Getenv("RUNNER_TRAIL_PCT")); s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
					pct = v
				}
			}
			dist := last * pct / 100.0
			if err := post.SetTrailingStop(api, symbol, fmt.Sprintf("%.6f", dist), fmt.Sprintf("%.6f", last), url); err == nil {
				runnerEnabled[symbol] = true
				notify.SendToChannel(order, idChannel,
					fmt.Sprintf("🏃 [Runner] %s BUY: Trailing stop enabled (%.2f%%, dist=%s) • active=%s",
						symbol, pct, fmt.Sprintf("%.6f", dist), fmt.Sprintf("%.6f", last)))
			} else {
				log.Printf("[WARN] SetTrailingStop failed for %s: %v", symbol, err)
			}

		}
		// nothing else to do in this tick
		return nil

	case hasTP3 && hasTP2 && last >= tp3:
		wantSL = fmt.Sprintf("%.6f", tp2)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP3 reached (%.6f) -> SL moved to TP2 (%s)", symbol, tp3, wantSL)

	case hasTP2 && hasTP1 && last >= tp2:
		wantSL = fmt.Sprintf("%.6f", tp1)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP2 reached (%.6f) -> SL moved to TP1 (%s)", symbol, tp2, wantSL)

	case hasTP1 && last >= tp1:
		wantSL = fmt.Sprintf("%.6f", entry) // default BE after TP1
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP1 reached (%.6f) -> SL moved to BREAKEVEN (%s)", symbol, tp1, wantSL)
	}

	if wantSL != "" && wantSL != currentSL {
		if err := post.ChangeLs(api, symbol, wantSL, trade.GetType(symbol), url); err == nil {
			trade.SetSl(symbol, wantSL)
			log.Println(hitMsg)
			notify.SendToChannel(order, idChannel, hitMsg)
		} else {
			log.Printf("[WARN] ChangeLs failed for %s: %v", symbol, err)
		}
	}
	return nil
}

func SellTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url string, idChannel string) error {
	price := get.GetPrice(symbol, url)
	if len(price.Result.List) == 0 {
		return nil
	}

	parseMaybe := func(s string) (float64, bool) {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, false
		}
		v, err := strconv.ParseFloat(s, 64)
		return v, err == nil
	}

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	slStr := trade.GetSl(symbol)
	sl, _ := strconv.ParseFloat(slStr, 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)

	tp1, hasTP1 := parseMaybe(trade.GetTp1(symbol))
	tp2, hasTP2 := parseMaybe(trade.GetTp2(symbol))
	tp3, hasTP3 := parseMaybe(trade.GetTp3(symbol))
	tp4, hasTP4 := parseMaybe(trade.GetTp4(symbol))

	// SL touched for shorts → last >= SL
	if last >= sl {
		msg := fmt.Sprintf("🔴 [SL] %s SELL: SL touched → closed (last=%.6f, SL=%s)", symbol, last, slStr)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	// PRIORITY (highest first):
	// 1) TP4 hit  -> SL = TP3
	// 2) TP2 hit  -> SL = TP1
	// 3) TP1 hit  -> SL = BE (entry)
	switch {
	case hasTP4 && hasTP3 && last <= tp4:
		// step 1: lock to TP3 (for shorts, TP3 is ABOVE last; retrace to TP3 stops out with profit)
		newSL := fmt.Sprintf("%.6f", tp3)
		if newSL != currentSL {
			if err := post.ChangeLs(api, symbol, newSL, trade.GetType(symbol), url); err == nil {
				trade.SetSl(symbol, newSL)
				notify.SendToChannel(order, idChannel,
					fmt.Sprintf("😎 [TP] %s SELL: TP4 reached (%.6f) -> SL moved to TP3 (%s)", symbol, tp4, newSL))
			} else {
				log.Printf("[WARN] ChangeLs failed for %s: %v", symbol, err)
			}
		}
		// step 2: enable trailing stop once
		if !runnerEnabled[symbol] {
			pct := 1.0
			if s := strings.TrimSpace(os.Getenv("RUNNER_TRAIL_PCT")); s != "" {
				if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
					pct = v
				}
			}
			dist := last * pct / 100.0
			if err := post.SetTrailingStop(api, symbol, fmt.Sprintf("%.6f", dist), fmt.Sprintf("%.6f", last), url); err == nil {
				runnerEnabled[symbol] = true
				notify.SendToChannel(order, idChannel,
					fmt.Sprintf("🏃 [Runner] %s SELL: Trailing stop enabled (%.2f%%, dist=%s) • active=%s",
						symbol, pct, fmt.Sprintf("%.6f", dist), fmt.Sprintf("%.6f", last)))
			} else {
				log.Printf("[WARN] SetTrailingStop failed for %s: %v", symbol, err)
			}
		}
		return nil

	case hasTP3 && hasTP2 && last >= tp3:
		wantSL = fmt.Sprintf("%.6f", tp2)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP3 reached (%.6f) -> SL moved to TP2 (%s)", symbol, tp3, wantSL)

	case hasTP2 && hasTP1 && last <= tp2:
		wantSL = fmt.Sprintf("%.6f", tp1)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP2 reached (%.6f) -> SL moved to TP1 (%s)", symbol, tp2, wantSL)

	case hasTP1 && last <= tp1:
		wantSL = fmt.Sprintf("%.6f", entry) // default BE after TP1
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP1 reached (%.6f) -> SL moved to BREAKEVEN (%s)", symbol, tp1, wantSL)
	}

	if wantSL != "" && wantSL != currentSL {
		if err := post.ChangeLs(api, symbol, wantSL, trade.GetType(symbol), url); err == nil {
			trade.SetSl(symbol, wantSL)
			log.Println(hitMsg)
			notify.SendToChannel(order, idChannel, hitMsg)
		} else {
			log.Printf("[WARN] ChangeLs failed for %s: %v", symbol, err)
		}
	}
	return nil
}

func GetPositionOrder(api *data.Env, order *data.Bot, trade *data.Trades) {
	for {
		for _, apis := range api.Api {
			// refresh active flags for all symbols we track
			for _, s := range order.GetActive() {
				if pos, err := GetPosition(apis, s, api.Url); err == nil {
					order.CheckPositon(pos)
				}
			}

			// drive TP/SL logic per *trade* symbol (not per order.Active entry)
			for i := 0; i < trade.GetLen(); i++ {
				symbol := trade.GetSymbol(i)
				if !order.GetActiveSymbol(symbol) {
					continue
				}
				switch trade.GetType(symbol) {
				case "Sell":
					if err := SellTp(apis, trade, symbol, order, api.Url, api.IdCHannel); err != nil {
						log.Println(err)
					}
				case "Buy":
					if err := BuyTp(apis, trade, symbol, order, api.Url, api.IdCHannel); err != nil {
						log.Println(err)
					}
				}
			}

			if order.Debeug {
				log.Println(print.PrettyPrint(trade))
				log.Println(print.PrettyPrint(order))
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// ReloadOpenPositions scans Bybit for any open linear positions and rebuilds
// in-memory state so the watcher resumes after restarts.
func ReloadOpenPositions(api *data.Env, order *data.Bot, trade *data.Trades) int {
	seen := map[string]bool{}
	recovered := 0

	settle := strings.TrimSpace(os.Getenv("SETTLE_COIN"))
	if settle == "" {
		settle = "USDT"
	}

	parseF := func(s string) float64 {
		f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return f
	}

	for _, keys := range api.Api {
		// ask for ALL open linear positions (no symbol), filtered by openOnly & settleCoin
		body, err := get.PrivateGET(
			api.Url,
			"/v5/position/list",
			map[string]string{
				"category":   "linear",
				"openOnly":   "1",
				"settleCoin": settle,
			},
			keys.Api, keys.Api_secret,
		)
		if err != nil {
			log.Printf("[Reload] HTTP error: %v", err)
			continue
		}

		var pos get.Position
		if err := json.Unmarshal(body, &pos); err != nil {
			log.Printf("[Reload] JSON error: %v body=%s", err, string(body))
			continue
		}
		if pos.RetCode != 0 {
			log.Printf("[Reload] retCode=%d retMsg=%s body=%s", pos.RetCode, pos.RetMsg, string(body))
			continue
		}
		if len(pos.Result.List) == 0 {
			log.Printf("[Reload] no open positions (settle=%s) for this key", settle)
			continue
		}

		for _, p := range pos.Result.List {
			// consider only actual open size
			if p.Symbol == "" || p.Size == "" || p.Size == "0" || p.Size == "0.0000" {
				continue
			}
			if seen[p.Symbol] {
				continue
			}
			seen[p.Symbol] = true

			// ensure Active has the symbol and mark active
			found := false
			for i := range order.Active {
				if order.Active[i].Symbol == p.Symbol {
					order.Active[i].Active = true
					found = true
					break
				}
			}
			if !found {
				order.Active = append(order.Active, data.Start{Symbol: p.Symbol, Active: true})
			}

			// if we already track it, don't add a duplicate Trade
			if !trade.CheckSymbol(p.Symbol) {
				continue
			}

			entry := parseF(p.AvgPrice)
			sl := strings.TrimSpace(p.StopLoss)
			if sl == "" || sl == "0" || sl == "0.0000" {
				// fallback 2% away from entry to have a usable value for watcher logs
				if strings.EqualFold(p.Side, "Buy") {
					sl = fmt.Sprintf("%.6f", entry*0.98)
				} else {
					sl = fmt.Sprintf("%.6f", entry*1.02)
				}
			}

			elem := data.Trade{
				Symbol:   p.Symbol,
				Type:     p.Side, // "Buy" or "Sell"
				Order:    "Limit",
				Wallet:   os.Getenv("STAKE_USDT"),
				Entry:    fmt.Sprintf("%.6f", entry),
				Leverage: p.Leverage,
				Tp1Order: "0", Tp2Order: "0", Tp3Order: "0", Tp4Order: "0",
				Tp1: "", Tp2: "", Tp3: "", Tp4: "",
				Sl:         sl,
				BEAfterTP1: false,
			}
			*trade = append(*trade, elem)
			recovered++
		}
	}

	if recovered == 0 {
		log.Printf("[Reload] found 0 open positions (settle=%s). If you HAVE open trades, check API key/account and settle coin.", settle)
	} else {
		log.Printf("[Reload] recovered %d open position(s): %v", recovered, order.GetActive())
	}
	return recovered
}

func UpdateChannel(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		}
	}
}
