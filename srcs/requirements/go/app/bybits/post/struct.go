package post

// REPLACE Post (order create) with v5 minimal
type Post struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Time    int64  `json:"time"`
	Result  struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
	} `json:"result"`
	RetExtInfo any `json:"retExtInfo"`
}

// REPLACE PostCancel with v5 minimal
type PostCancel struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Time    int64  `json:"time"`
	Result  struct {
		List []struct {
			OrderID     string `json:"orderId"`
			OrderLinkID string `json:"orderLinkId"`
		} `json:"list"`
		Success string `json:"success,omitempty"`
	} `json:"result"`
	RetExtInfo any `json:"retExtInfo"`
}

// Isolated / SetLeverage / TradingStop on v5 just return empty result on success
type EmptyResult struct {
	RetCode    int    `json:"retCode"`
	RetMsg     string `json:"retMsg"`
	Time       int64  `json:"time"`
	Result     any    `json:"result"`
	RetExtInfo any    `json:"retExtInfo"`
}
