package post

import (
	"bot/bybits/get"
	"bot/bybits/print"
	"bot/data"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"strconv"
)

// CHANGE: PostOrder -> v5 /v5/order/create with tpslMode=Partial
func PostOrder(symbol string, api data.BybitApi, trade *data.Trades, url_bybit string, debug bool) error {
	baseBody := map[string]interface{}{
		"category":       "linear",
		"symbol":         symbol,
		"side":           trade.GetType(symbol), // Buy|Sell
		"orderType":      "Limit",
		"price":          trade.GetEntry(symbol),
		"timeInForce":    "GTC", // v5
		"reduceOnly":     false,
		"closeOnTrigger": false,
		"stopLoss":       trade.GetSl(symbol),
		"tpslMode":       "Partial", // each split gets its own TP
		"positionIdx":    0,         // one-way mode default
	}

	tps := []struct{ tp, qty string }{
		{trade.GetTp1(symbol), trade.GetTp1Order(symbol)},
		{trade.GetTp2(symbol), trade.GetTp2Order(symbol)},
		{trade.GetTp3(symbol), trade.GetTp3Order(symbol)},
		{trade.GetTp4(symbol), trade.GetTp4Order(symbol)},
	}

	for i, x := range tps {
		if x.tp != "" && x.qty != "" && x.qty != "0" && x.qty != "0.0000" {
			body := maps.Clone(baseBody) // Go 1.21; if not available, copy manually
			body["takeProfit"] = x.tp
			body["qty"] = x.qty

			if debug {
				log.Printf("[ORDER] TP%d body: %s", i+1, print.PrettyPrint(body))
			}

			respBytes, err := get.PrivatePOST(url_bybit, "/v5/order/create", body, api.Api, api.Api_secret)
			if err != nil {
				return err
			}
			var res Post
			if err := json.Unmarshal(respBytes, &res); err != nil {
				return err
			}
			if res.RetCode != 0 {
				return errors.New(res.RetMsg)
			}
			trade.SetId(symbol, res.Result.OrderID)
		}
	}
	return nil
}

// CHANGE: PostIsoled -> v5 /v5/position/switch-isolated + set-leverage
func PostIsoled(api data.BybitApi, symbol string, trade *data.Trades, url_bybit string, debug bool) error {
	levStr := trade.GetLeverage(symbol)
	lev, _ := strconv.Atoi(levStr)
	if lev <= 0 {
		lev = 10
	}

	// 1) switch isolated
	body := map[string]interface{}{
		"category":     "linear",
		"symbol":       symbol,
		"tradeMode":    1, // 1 = isolated
		"buyLeverage":  fmt.Sprint(lev),
		"sellLeverage": fmt.Sprint(lev),
	}
	resp1, err := get.PrivatePOST(url_bybit, "/v5/position/switch-isolated", body, api.Api, api.Api_secret)
	if err != nil {
		return err
	}
	if debug {
		log.Printf("[POST] switch-isolated: %s", string(resp1))
	}

	// 2) set leverage (required by docs to ensure leverage is applied)
	body2 := map[string]interface{}{
		"category":     "linear",
		"symbol":       symbol,
		"buyLeverage":  fmt.Sprint(lev),
		"sellLeverage": fmt.Sprint(lev),
	}
	resp2, err := get.PrivatePOST(url_bybit, "/v5/position/set-leverage", body2, api.Api, api.Api_secret)
	if err != nil {
		return err
	}
	if debug {
		log.Printf("[POST] set-leverage: %s", string(resp2))
	}

	log.Printf("[POST] Isolated ON, leverage=%d", lev)
	return nil
}

// CHANGE: CancelOrder / CancelAll -> v5 /v5/order/cancel-all
func CancelOrder(symbol string, api data.BybitApi, trade *data.Trades, url_bybit string) error {

	var cancel PostCancel
	b := map[string]interface{}{"category": "linear", "symbol": symbol}
	raw, err := get.PrivatePOST(url_bybit, "/v5/order/cancel-all", b, api.Api, api.Api_secret)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &cancel); err != nil {
		return err
	}
	if cancel.RetCode != 0 {
		return errors.New(cancel.RetMsg)
	}
	log.Printf("Cancel order success: %s", symbol)
	return nil
}

func CancelBySl(px get.Price, tr *data.Trade) string {
	if len(px.Result.List) == 0 {
		return ""
	}
	pBid := px.Result.List[0].Bid1Price
	pAsk := px.Result.List[0].Ask1Price
	if tr.Type == "Buy" {
		// close long quickly → place SL just below bid
		bid, _ := strconv.ParseFloat(pBid, 64)
		return fmt.Sprintf("%.4f", bid*0.99)
	}
	if tr.Type == "Sell" {
		// close short quickly → place SL just above ask
		ask, _ := strconv.ParseFloat(pAsk, 64)
		return fmt.Sprintf("%.4f", ask*1.01)
	}
	return ""
}

// CHANGE: ChangeLs -> v5 /v5/position/trading-stop
func ChangeLs(api data.BybitApi, symbol string, sl string, side string, url_bybit string) error {
	body := map[string]interface{}{
		"category":    "linear",
		"symbol":      symbol,
		"tpslMode":    "Full",
		"positionIdx": 0,
		"stopLoss":    sl,
	}
	raw, err := get.PrivatePOST(url_bybit, "/v5/position/trading-stop", body, api.Api, api.Api_secret)
	if err != nil {
		return err
	}
	var res EmptyResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}
	log.Printf("[ChangeLs] %s", print.PrettyPrint(res))
	return nil
}
