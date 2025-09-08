package listen

import (
	"bot/bybits/get"
	"bot/bybits/post"
	"bot/bybits/print"
	"bot/bybits/sign"
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
	params := map[string]string{
		"api_key":   api.Api,
		"symbol":    symbol,
		"timestamp": print.GetTimestamp(),
	}
	params["sign"] = sign.GetSigned(params, api.Api_secret)
	url := fmt.Sprint(
		url_bybite,
		"/private/linear/position/list?api_key=",
		params["api_key"],
		"&symbol=", symbol,
		"&timestamp=",
		params["timestamp"],
		"&sign=",
		params["sign"],
	)
	body, err := get.GetRequetJson(url)
	if err != nil {
		log.Println(err)
		return position, err
	}
	json.Unmarshal(body, &position)
	return position, nil
}

func BuyTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url_bybite string) error {
	price := get.GetPrice(symbol, url_bybite)
	lastPrice, _ := strconv.ParseFloat(price.Result[0].LastPrice, 64)
	sl, _ := strconv.ParseFloat(trade.GetSl(symbol), 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	if lastPrice <= sl {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s BUY: SL touched -> closed", symbol)
		return nil
	}
	if lastPrice >= tp3 {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s BUY: All targets hit -> closed", symbol)
		return nil
	}
	if lastPrice >= tp2 {
		// trail SL to TP2
		if err := post.ChangeLs(api, symbol, trade.GetTp2(symbol), trade.GetType(symbol), url_bybite); err == nil {
			trade.SetSl(symbol, trade.GetTp2(symbol))
			log.Printf("[TP] %s BUY: TP2 hit -> SL moved to TP2", symbol)
		}
		return nil
	}
	if lastPrice >= tp1 {
		if trade.GetBEAfterTP1(symbol) {
			be := fmt.Sprintf("%.4f", entry)
			if err := post.ChangeLs(api, symbol, be, trade.GetType(symbol), url_bybite); err == nil {
				trade.SetSl(symbol, be)
				log.Printf("[TP] %s BUY: TP1 hit -> SL moved to BREAKEVEN (%s)", symbol, be)
			}
		} else {
			if err := post.ChangeLs(api, symbol, trade.GetTp1(symbol), trade.GetType(symbol), url_bybite); err == nil {
				trade.SetSl(symbol, trade.GetTp1(symbol))
				log.Printf("[TP] %s BUY: TP1 hit -> SL moved to TP1", symbol)
			}
		}
		return nil
	}
	return nil
}

func SellTp(api data.BybitApi, trade *data.Trades, symbol string, order *data.Bot, url_bybite string) error {
	price := get.GetPrice(symbol, url_bybite)
	lastPrice, _ := strconv.ParseFloat(price.Result[0].LastPrice, 64)
	sl, _ := strconv.ParseFloat(trade.GetSl(symbol), 64)
	entry, _ := strconv.ParseFloat(trade.GetEntry(symbol), 64)
	tp1, _ := strconv.ParseFloat(trade.GetTp1(symbol), 64)
	tp2, _ := strconv.ParseFloat(trade.GetTp2(symbol), 64)
	tp3, _ := strconv.ParseFloat(trade.GetTp3(symbol), 64)

	if lastPrice >= sl {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s SELL: SL touched -> closed", symbol)
		return nil
	}
	if lastPrice <= tp3 {
		trade.Delete(symbol)
		order.Delete(symbol)
		log.Printf("[TP] %s SELL: All targets hit -> closed", symbol)
		return nil
	}
	if lastPrice <= tp2 && tp2 != sl {
		// trail SL to TP2
		if err := post.ChangeLs(api, symbol, trade.GetTp2(symbol), trade.GetType(symbol), url_bybite); err == nil {
			trade.SetSl(symbol, trade.GetTp2(symbol))
			log.Printf("[TP] %s SELL: TP2 hit -> SL moved to TP2", symbol)
		}
		return nil
	}
	if lastPrice <= tp1 && tp1 != sl {
		if trade.GetBEAfterTP1(symbol) {
			be := fmt.Sprintf("%.4f", entry)
			if err := post.ChangeLs(api, symbol, be, trade.GetType(symbol), url_bybite); err == nil {
				trade.SetSl(symbol, be)
				log.Printf("[TP] %s SELL: TP1 hit -> SL moved to BREAKEVEN (%s)", symbol, be)
			}
		} else {
			if err := post.ChangeLs(api, symbol, trade.GetTp1(symbol), trade.GetType(symbol), url_bybite); err == nil {
				trade.SetSl(symbol, trade.GetTp1(symbol))
				log.Printf("[TP] %s SELL: TP1 hit -> SL moved to TP1", symbol)
			}
		}
		return nil
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
						if order.GetActiveSymbol(symbol) == true && trade.GetType(symbol) == "Sell" {
							err := SellTp(apis, trade, ord.Symbol, order, api.Url)
							if err != nil {
								log.Println(err)
							}
						} else if order.GetActiveSymbol(symbol) == true && trade.GetType(symbol) == "Buy" {
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
