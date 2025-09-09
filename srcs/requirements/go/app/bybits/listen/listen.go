package listen

import (
	"bot/bybits/get"
	"bot/bybits/post"
	"bot/bybits/print"
	"bot/data"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

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

func BuyTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url string) error {
	price := get.GetPrice(symbol, url)
	if len(price.Result.List) == 0 {
		return nil
	}

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	sl, _ := strconv.ParseFloat(trade.GetSl(symbol), 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	hasTP4 := trade.GetTp4(symbol) != ""
	var tp4 float64
	if hasTP4 {
		tp4, _ = strconv.ParseFloat(trade.GetTp4(symbol), 64)
	}

	// SL touched → close
	if last <= sl {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s BUY: SL touched -> closed", symbol)
		return nil
	}

	// All targets done → close (TP4 if present, else TP3)
	if (hasTP4 && last >= tp4) || (!hasTP4 && last >= tp3) {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("😎 [TP] %s BUY: All take-profit targets achieved 😎", symbol)
		return nil
	}

	// Decide desired new SL (idempotent)
	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	if hasTP4 && last >= tp3 {
		wantSL = trade.GetTp2(symbol)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP3 reached (%.4f) -> SL moved to TP2 (%s)", symbol, tp3, wantSL)
	} else if last >= tp2 {
		wantSL = trade.GetTp2(symbol)
		hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP2 reached (%.4f) -> SL moved to TP2 (%s)", symbol, tp2, wantSL)
	} else if last >= tp1 {
		if trade.GetBEAfterTP1(symbol) {
			wantSL = fmt.Sprintf("%.4f", entry)
			hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP1 reached (%.4f) -> SL moved to BREAKEVEN (%s)", symbol, tp1, wantSL)
		} else {
			wantSL = trade.GetTp1(symbol)
			hitMsg = fmt.Sprintf("😎 [TP] %s BUY: TP1 reached (%.4f) -> SL moved to TP1 (%s)", symbol, tp1, wantSL)
		}
	}

	if wantSL != "" && wantSL != currentSL {
		if err := post.ChangeLs(api, symbol, wantSL, trade.GetType(symbol), url); err == nil {
			trade.SetSl(symbol, wantSL)
			log.Println(hitMsg)
		}
	}
	return nil
}

func SellTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url string) error {
	price := get.GetPrice(symbol, url)
	if len(price.Result.List) == 0 {
		return nil
	}

	last, _ := strconv.ParseFloat(price.Result.List[0].LastPrice, 64)
	sl, _ := strconv.ParseFloat(trade.GetSl(symbol), 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	hasTP4 := trade.GetTp4(symbol) != ""
	var tp4 float64
	if hasTP4 {
		tp4, _ = strconv.ParseFloat(trade.GetTp4(symbol), 64)
	}

	// Short loses when price rises to/above SL
	if last >= sl {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s SELL: SL touched -> closed", symbol)
		return nil
	}

	// All targets done → close (TP4 if present, else TP3)
	if (hasTP4 && last <= tp4) || (!hasTP4 && last <= tp3) {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("😎 [TP] %s SELL: All take-profit targets achieved 😎", symbol)
		return nil
	}

	currentSL := trade.GetSl(symbol)
	wantSL := ""
	hitMsg := ""

	// With TP4: after TP3 → SL to TP2 (keep ≥ entry). Without TP4: after TP2.
	if hasTP4 && last <= tp3 {
		newSL := tp2
		if newSL < entry {
			newSL = entry
		}
		wantSL = fmt.Sprintf("%.4f", newSL)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP3 reached (%.4f) -> SL moved to TP2 (%s)", symbol, tp3, wantSL)
	} else if last <= tp2 {
		newSL := tp1
		if newSL < entry {
			newSL = entry
		}
		wantSL = fmt.Sprintf("%.4f", newSL)
		hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP2 reached (%.4f) -> SL moved to TP1/BE floor (%s)", symbol, tp2, wantSL)
	} else if last <= tp1 {
		if trade.GetBEAfterTP1(symbol) {
			be := entry
			if be < entry {
				be = entry
			}
			wantSL = fmt.Sprintf("%.4f", be)
			hitMsg = fmt.Sprintf("😎 [TP] %s SELL: TP1 reached (%.4f) -> SL moved to BREAKEVEN (%s)", symbol, tp1, wantSL)
		}
	}

	if wantSL != "" && wantSL != currentSL {
		if err := post.ChangeLs(api, symbol, wantSL, trade.GetType(symbol), url); err == nil {
			trade.SetSl(symbol, wantSL)
			log.Println(hitMsg)
		}
	}
	return nil
}

func GetPositionOrder(api *data.Env, order *data.Bot, trade *data.Trades) {
	for ok := true; ok; {
		for _, ord := range (*order).Active {
			for _, apis := range api.Api {
				pos, err := GetPosition(apis, ord.Symbol, api.Url)
				if err == nil {
					order.CheckPositon(pos)
					order.GetActive()
					for i := 0; i < trade.GetLen(); i++ {
						symbol := trade.GetSymbol(i)
						if order.GetActiveSymbol(symbol) && trade.GetType(symbol) == "Sell" {
							err := SellTp(apis, trade, ord.Symbol, order, api.Url)
							if err != nil {
								log.Println(err)
							}
						} else if order.GetActiveSymbol(symbol) && trade.GetType(symbol) == "Buy" {
							err := BuyTp(apis, trade, ord.Symbol, order, api.Url)
							if err != nil {
								log.Println(err)
							}
						}
					}
				} else {
					log.Println(err)
				}
				if order.Debeug {
					log.Println(print.PrettyPrint(trade))
					log.Println(print.PrettyPrint(order))
				}
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
