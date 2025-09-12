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

func PostOrder(symbol string, api data.BybitApi, trade *data.Trades, url_bybit string, debug bool) error {
	baseBody := map[string]interface{}{
		"category":       "linear",
		"symbol":         symbol,
		"side":           trade.GetType(symbol), // Buy|Sell
		"orderType":      "Limit",
		"price":          trade.GetEntry(symbol),
		"timeInForce":    "GTC",
		"reduceOnly":     false,
		"closeOnTrigger": false,
		"stopLoss":       trade.GetSl(symbol),
		"tpslMode":       "Partial",
		"positionIdx":    0,
	}

	type tpSplit struct{ tp, qty string }
	tps := []tpSplit{
		{trade.GetTp1(symbol), trade.GetTp1Order(symbol)},
		{trade.GetTp2(symbol), trade.GetTp2Order(symbol)},
		{trade.GetTp3(symbol), trade.GetTp3Order(symbol)},
		{trade.GetTp4(symbol), trade.GetTp4Order(symbol)},
	}

	ok := 0
	var lastErr error

	for i, x := range tps {
		if x.tp == "" || x.qty == "" || x.qty == "0" || x.qty == "0.0000" {
			continue
		}
		body := maps.Clone(baseBody)
		body["takeProfit"] = x.tp
		body["qty"] = x.qty

		if debug {
			log.Printf("[ORDER] TP%d body: %s", i+1, print.PrettyPrint(body))
		}

		respBytes, err := get.PrivatePOST(url_bybit, "/v5/order/create", body, api.Api, api.Api_secret)
		if err != nil {
			lastErr = err
			continue
		}
		var res Post
		if err := json.Unmarshal(respBytes, &res); err != nil {
			lastErr = err
			continue
		}
		if res.RetCode != 0 {
			lastErr = errors.New(res.RetMsg)
			continue
		}
		trade.SetId(symbol, res.Result.OrderID)
		ok++
	}

	if ok == 0 && lastErr != nil {
		return lastErr
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
		bid, _ := strconv.ParseFloat(pBid, 64)
		return fmt.Sprintf("%.6f", bid*0.99)
	}
	if tr.Type == "Sell" {
		ask, _ := strconv.ParseFloat(pAsk, 64)
		return fmt.Sprintf("%.6f", ask*1.01)
	}
	return ""
}

func ChangeLs(api data.BybitApi, symbol string, sl string, side string, url_bybit string) error {
	body := map[string]interface{}{
		"category":    "linear",
		"symbol":      symbol,
		"tpslMode":    "Full",
		"positionIdx": 0,
		"stopLoss":    sl,
		"slTriggerBy": "LastPrice", // <<< make local checks & exchange behavior consistent
	}
	raw, err := get.PrivatePOST(url_bybit, "/v5/position/trading-stop", body, api.Api, api.Api_secret)
	if err != nil {
		return err
	}
	var res EmptyResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return err
	}
	if res.RetCode != 0 {
		return fmt.Errorf("trading-stop failed for %s: %s (%d)", symbol, res.RetMsg, res.RetCode)
	}
	log.Printf("[ChangeLs] OK %s -> stopLoss=%s", symbol, sl)
	return nil
}
