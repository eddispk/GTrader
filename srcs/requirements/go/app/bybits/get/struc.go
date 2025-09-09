package get

import "time"

type Price struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	RetExtInfo any    `json:"retExtInfo"`
	Time       int64  `json:"time"`
	Result     struct {
		Category string `json:"category"`
		List     []struct {
			Symbol                 string `json:"symbol"`
			LastPrice              string `json:"lastPrice"`
			IndexPrice             string `json:"indexPrice"`
			MarkPrice              string `json:"markPrice"`
			PrevPrice24h           string `json:"prevPrice24h"`
			Price24hPcnt           string `json:"price24hPcnt"`
			HighPrice24h           string `json:"highPrice24h"`
			LowPrice24h            string `json:"lowPrice24h"`
			PrevPrice1h            string `json:"prevPrice1h"`
			OpenInterest           string `json:"openInterest"`
			OpenInterestValue      string `json:"openInterestValue"`
			Turnover24h            string `json:"turnover24h"`
			Volume24h              string `json:"volume24h"`
			FundingRate            string `json:"fundingRate"`
			NextFundingTime        string `json:"nextFundingTime"`
			PredictedDeliveryPrice string `json:"predictedDeliveryPrice"`
			BasisRate              string `json:"basisRate"`
			Basis                  string `json:"basis"`
			DeliveryFeeRate        string `json:"deliveryFeeRate"`
			DeliveryTime           string `json:"deliveryTime"`
			Ask1Size               string `json:"ask1Size"`
			Bid1Price              string `json:"bid1Price"`
			Ask1Price              string `json:"ask1Price"`
			Bid1Size               string `json:"bid1Size"`
			PreOpenPrice           string `json:"preOpenPrice"`
			PreQty                 string `json:"preQty"`
			CurPreListingPhase     string `json:"curPreListingPhase"`
		} `json:"list"`
	} `json:"result"`
}

type Wallet struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	RetExtInfo any    `json:"retExtInfo"`
	Time       int64  `json:"time"`
	Result     struct {
		List []struct {
			AccountType            string `json:"accountType"`
			TotalEquity            string `json:"totalEquity"`
			TotalWalletBalance     string `json:"totalWalletBalance"`
			TotalAvailableBalance  string `json:"totalAvailableBalance"`
			TotalMarginBalance     string `json:"totalMarginBalance"`
			TotalPerpUPL           string `json:"totalPerpUPL"`
			TotalInitialMargin     string `json:"totalInitialMargin"`
			TotalMaintenanceMargin string `json:"totalMaintenanceMargin"`
			Coin                   []struct {
				Coin            string `json:"coin"`
				Equity          string `json:"equity"`
				WalletBalance   string `json:"walletBalance"`
				UnrealisedPnl   string `json:"unrealisedPnl"`
				CumRealisedPnl  string `json:"cumRealisedPnl"`
				TotalOrderIM    string `json:"totalOrderIM"`
				TotalPositionIM string `json:"totalPositionIM"`
				TotalPositionMM string `json:"totalPositionMM"`
				Locked          string `json:"locked"`
				UsdValue        string `json:"usdValue"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
}

type Position struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	RetExtInfo any    `json:"retExtInfo"`
	Time       int64  `json:"time"`
	Result     struct {
		Category       string `json:"category"`
		NextPageCursor string `json:"nextPageCursor"`
		List           []struct {
			PositionIdx            int    `json:"positionIdx"`
			Symbol                 string `json:"symbol"`
			Side                   string `json:"side"` // Buy|Sell
			Size                   string `json:"size"`
			AvgPrice               string `json:"avgPrice"`
			PositionValue          string `json:"positionValue"`
			TradeMode              int    `json:"tradeMode"`
			AutoAddMargin          int    `json:"autoAddMargin"`
			PositionStatus         string `json:"positionStatus"`
			Leverage               string `json:"leverage"`
			MarkPrice              string `json:"markPrice"`
			LiqPrice               string `json:"liqPrice"`
			BustPrice              string `json:"bustPrice"`
			PositionIM             string `json:"positionIM"`
			PositionIMByMp         string `json:"positionIMByMp"`
			PositionMM             string `json:"positionMM"`
			PositionMMByMp         string `json:"positionMMByMp"`
			PositionBalance        string `json:"positionBalance"`
			TakeProfit             string `json:"takeProfit"`
			StopLoss               string `json:"stopLoss"`
			TrailingStop           string `json:"trailingStop"`
			SessionAvgPrice        string `json:"sessionAvgPrice"`
			UnrealisedPnl          string `json:"unrealisedPnl"`
			CurRealisedPnl         string `json:"curRealisedPnl"`
			CumRealisedPnl         string `json:"cumRealisedPnl"`
			AdlRankIndicator       int    `json:"adlRankIndicator"`
			CreatedTime            string `json:"createdTime"`
			UpdatedTime            string `json:"updatedTime"`
			Seq                    int64  `json:"seq"`
			IsReduceOnly           bool   `json:"isReduceOnly"`
			MmrSysUpdateTime       string `json:"mmrSysUpdateTime"`
			LeverageSysUpdatedTime string `json:"leverageSysUpdatedTime"`
		} `json:"list"`
	} `json:"result"`
}

type PositonTrade struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	ExtInfo string `json:"ext_info"`
	Result  struct {
		CurrentPage int `json:"current_page"`
		Data        []struct {
			OrderID        string    `json:"order_id"`
			UserID         int       `json:"user_id"`
			Symbol         string    `json:"symbol"`
			Side           string    `json:"side"`
			OrderType      string    `json:"order_type"`
			Price          float64   `json:"price"`
			Qty            int       `json:"qty"`
			TimeInForce    string    `json:"time_in_force"`
			OrderStatus    string    `json:"order_status"`
			LastExecPrice  float64   `json:"last_exec_price"`
			CumExecQty     int       `json:"cum_exec_qty"`
			CumExecValue   float64   `json:"cum_exec_value"`
			CumExecFee     float64   `json:"cum_exec_fee"`
			ReduceOnly     bool      `json:"reduce_only"`
			CloseOnTrigger bool      `json:"close_on_trigger"`
			OrderLinkID    string    `json:"order_link_id"`
			CreatedTime    time.Time `json:"created_time"`
			UpdatedTime    time.Time `json:"updated_time"`
			TakeProfit     float64   `json:"take_profit"`
			StopLoss       float64   `json:"stop_loss"`
			TpTriggerBy    string    `json:"tp_trigger_by"`
			SlTriggerBy    string    `json:"sl_trigger_by"`
		} `json:"data"`
	} `json:"result"`
	TimeNow          string `json:"time_now"`
	RateLimitStatus  int    `json:"rate_limit_status"`
	RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
	RateLimit        int    `json:"rate_limit"`
}
