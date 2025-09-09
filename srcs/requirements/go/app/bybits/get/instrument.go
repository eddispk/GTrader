package get

import (
	"encoding/json"
	"fmt"
)

type InstrumentInfo struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Time    int64  `json:"time"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol        string `json:"symbol"`
			Status        string `json:"status"`
			BaseCoin      string `json:"baseCoin"`
			QuoteCoin     string `json:"quoteCoin"`
			LotSizeFilter struct {
				BasePrecision string `json:"basePrecision"`
				QtyStep       string `json:"qtyStep"`
				MinOrderQty   string `json:"minOrderQty"`
				MaxOrderQty   string `json:"maxOrderQty"`
			} `json:"lotSizeFilter"`
			PriceFilter struct {
				TickSize string `json:"tickSize"`
			} `json:"priceFilter"`
		} `json:"list"`
	} `json:"result"`
}

func GetInstrument(baseURL, symbol string) (InstrumentInfo, error) {
	var out InstrumentInfo
	u := fmt.Sprintf("%s/v5/market/instruments-info?category=linear&symbol=%s", baseURL, symbol)
	b, err := GetRequetJson(u)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}
