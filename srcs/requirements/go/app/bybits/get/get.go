package get

import (
	"bot/bybits/print"
	"bot/bybits/sign"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func GetRequetJson(url string) ([]byte, error) {
	req, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return body, err
}
func v5Headers(apiKey, apiSecret string, queryOrBody string) (http.Header, string) {
	const recvWindow = "5000"
	ts := print.GetTimestamp()
	// signature = HMAC_SHA256(timestamp+apiKey+recvWindow+queryOrBody)
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(ts + apiKey + recvWindow + queryOrBody))
	sig := hex.EncodeToString(mac.Sum(nil))

	h := http.Header{}
	h.Set("X-BAPI-API-KEY", apiKey)
	h.Set("X-BAPI-TIMESTAMP", ts)
	h.Set("X-BAPI-RECV-WINDOW", recvWindow)
	h.Set("X-BAPI-SIGN", sig)
	// optional but common:
	// h.Set("X-BAPI-SIGN-TYPE", "2")
	return h, ts
}

// ADD: v5 private GET
func PrivateGET(base, path string, q map[string]string, apiKey, apiSecret string) ([]byte, error) {
	qs := sign.BuildQueryString(q)
	url := fmt.Sprintf("%s%s?%s", base, path, qs)
	h, _ := v5Headers(apiKey, apiSecret, qs)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// ADD: v5 private POST with JSON body
func PrivatePOST(base, path string, body map[string]interface{}, apiKey, apiSecret string) ([]byte, error) {
	bts, _ := json.Marshal(body)
	h, _ := v5Headers(apiKey, apiSecret, string(bts))
	url := fmt.Sprintf("%s%s", base, path)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bts))
	req.Header = h
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// ---------- Endpoints ----------

func GetPrice(symbol string, url_bybit string) Price {
	var curr Price
	url := fmt.Sprint(url_bybit, "/v5/market/tickers?category=linear&symbol=", symbol)
	body, err := GetRequetJson(url)
	if err != nil {
		log.Println(err)
		return curr
	}
	if err := json.Unmarshal(body, &curr); err != nil {
		log.Println(err)
	}
	return curr
}

func GetWallet(apiKey string, apiSecret string, url_bybit string) Wallet {
	var wall Wallet
	url := fmt.Sprint(url_bybit, "/v5/account/wallet-balance?accountType=UNIFIED&coin=USDT")
	h, _ := v5Headers(apiKey, apiSecret, "accountType=UNIFIED&coin=USDT")
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header = h
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return wall
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return wall
	}
	if err := json.Unmarshal(body, &wall); err != nil {
		log.Println(err)
	}
	return wall
}
