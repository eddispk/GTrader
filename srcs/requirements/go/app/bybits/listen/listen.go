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

func GetPosition(api data.BybitApi, symbol string, url_bybite string) (get.Position, error) {
	var position get.Position
	q := map[string]string{
		"category": "linear",
		"symbol":   symbol,
	}
	body, err := get.PrivateGET(url_bybite, "/v5/position/list", q, api.Api, api.Api_secret)
	if err != nil {
		log.Println(err)
		return position, err
	}
	json.Unmarshal(body, &position)
	return position, nil
}

func BuyTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url string, idChannel string) error {
	price := get.GetPrice(symbol, url)
	if len(price.Result.List) == 0 {
		return nil
	}

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	slStr := trade.GetSl(symbol)
	sl, _ := strconv.ParseFloat(slStr, 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	hasTP4 := trade.GetTp4(symbol) != ""
	var tp4 float64
	if hasTP4 {
		tp4, _ = strconv.ParseFloat(trade.GetTp4(symbol), 64)
	}

	// SL touched → close (exchange closes; we just notify & cleanup)
	if last <= sl {
		msg := fmt.Sprintf("🔴 [SL] %s BUY: SL touched → closed (last=%.6f, SL=%s)", symbol, last, slStr)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	// All targets done → close (TP4 if present, else TP3)
	if (hasTP4 && last >= tp4) || (!hasTP4 && last >= tp3) {
		msg := fmt.Sprintf("😎 [TP] %s BUY: All take-profit targets achieved", symbol)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	switch {
	case hasTP4 && last >= tp3:
		wantSL = trade.GetTp2(symbol)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP3 reached (%.6f) -> SL moved to TP2 (%s)", symbol, tp3, wantSL)
	case last >= tp2:
		wantSL = trade.GetTp2(symbol)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP2 reached (%.6f) -> SL moved to TP2 (%s)", symbol, tp2, wantSL)
	case last >= tp1:
		if trade.GetBEAfterTP1(symbol) {
			wantSL = fmt.Sprintf("%.6f", entry)
			hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP1 reached (%.6f) -> SL moved to BREAKEVEN (%s)", symbol, tp1, wantSL)
		} else {
			wantSL = trade.GetTp1(symbol)
			hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP1 reached (%.6f) -> SL moved to TP1 (%s)", symbol, tp1, wantSL)
		}
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

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	slStr := trade.GetSl(symbol)
	sl, _ := strconv.ParseFloat(slStr, 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	hasTP4 := trade.GetTp4(symbol) != ""
	var tp4 float64
	if hasTP4 {
		tp4, _ = strconv.ParseFloat(trade.GetTp4(symbol), 64)
	}

	// SL touched for shorts → last >= SL
	if last >= sl {
		msg := fmt.Sprintf("🔴 [SL] %s SELL: SL touched → closed (last=%.6f, SL=%s)", symbol, last, slStr)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	// All targets done
	if (hasTP4 && last <= tp4) || (!hasTP4 && last <= tp3) {
		msg := fmt.Sprintf("😎 [TP] %s SELL: All take-profit targets achieved", symbol)
		log.Println(msg)
		notify.SendToChannel(order, idChannel, msg)
		trade.Delete(symbol)
		order.Delete(symbol)
		return nil
	}

	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	if hasTP4 && last <= tp3 {
		// after TP3, raise SL to TP2 but not below entry
		newSL := tp2
		if newSL < entry {
			newSL = entry
		}
		wantSL = fmt.Sprintf("%.6f", newSL)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP3 reached (%.6f) -> SL moved to TP2/BE floor (%s)", symbol, tp3, wantSL)
	} else if last <= tp2 {
		newSL := tp1
		if newSL < entry {
			newSL = entry
		}
		wantSL = fmt.Sprintf("%.6f", newSL)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP2 reached (%.6f) -> SL moved to TP1/BE floor (%s)", symbol, tp2, wantSL)
	} else if last <= tp1 && trade.GetBEAfterTP1(symbol) {
		wantSL = fmt.Sprintf("%.6f", entry)
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

func UpdateChannel(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		}
	}
}

// ReloadOpenPositions scans Bybit for any open linear positions and rebuilds
// in-memory state so the watcher resumes after restarts.
func ReloadOpenPositions(api *data.Env, order *data.Bot, trade *data.Trades) int {
	seen := map[string]bool{}
	recovered := 0

	parseF := func(s string) float64 {
		f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return f
	}

	for _, keys := range api.Api {
		// list *all* positions for linear; symbol is optional
		body, err := get.PrivateGET(api.Url, "/v5/position/list",
			map[string]string{"category": "linear"},
			keys.Api, keys.Api_secret)
		if err != nil {
			log.Printf("[Reload] error: %v", err)
			continue
		}
		var pos get.Position
		if err := json.Unmarshal(body, &pos); err != nil {
			log.Printf("[Reload] unmarshal: %v", err)
			continue
		}
		if pos.RetCode != 0 || len(pos.Result.List) == 0 {
			continue
		}

		for _, p := range pos.Result.List {
			if p.Symbol == "" || p.Size == "" || p.Size == "0" || p.Size == "0.0000" {
				continue
			}
			if seen[p.Symbol] {
				continue
			}
			seen[p.Symbol] = true

			// ensure Active list has the symbol and mark it active
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

			// if we already track a trade for it, skip creating another
			if !trade.CheckSymbol(p.Symbol) {
				continue
			}

			// minimal Trade reconstruction (we may not know original TPs)
			entry := parseF(p.AvgPrice)
			sl := p.StopLoss
			if sl == "" || sl == "0" || sl == "0.0000" {
				// fallback 2% away from entry
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

	if recovered > 0 {
		log.Printf("[Reload] recovered %d open position(s): %v", recovered, order.GetActive())
	}
	return recovered
}
