package data

import (
	"bot/bybits/get"
	"bot/bybits/print"
	"bot/bybits/telegram"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Trade struct {
	Symbol      string   `json:"symbol"`
	Type        string   `json:"type"`
	Order       string   `json:"order"`
	SymbolPrice string   `json:"symbolPrice"`
	Wallet      string   `json:"wallet"`
	Price       string   `json:"price"`
	Entry       string   `json:"entry"`
	Leverage    string   `json:"leverage"`
	Tp1Order    string   `json:"tp_1Order"`
	Tp2Order    string   `json:"tp_2Order"`
	Tp3Order    string   `json:"tp_3Order"`
	Tp4Order    string   `json:"tp_4Order"` // NEW
	Tp1         string   `json:"tp1"`
	Tp2         string   `json:"tp2"`
	Tp3         string   `json:"tp3"`
	Tp4         string   `json:"tp4"` // NEW
	Sl          string   `json:"Sl"`
	Id          []string `json:"id"`
	Active      []string `json:"active"`
	BEAfterTP1  bool     `json:"be_after_tp1"` // NEW
}

type (
	Trades []Trade
)

type BybitApi struct {
	Api        string
	Api_secret string
}

type Env struct {
	Api          []BybitApi
	Admin        []string
	Api_telegram string
	Url          string
	BotName      string
	IdCHannel    string
}

type Bot struct {
	Active  []Start
	Debeug  bool
	Botapi  *tgbotapi.BotAPI
	Updates tgbotapi.UpdatesChannel
	Db      *sql.DB
}

type Start struct {
	Symbol string
	Active bool
}

func (t *Env) AddApi(api string, api_secret string) {
	check := false
	elem := BybitApi{
		Api:        api,
		Api_secret: api_secret,
	}
	for _, ls := range (*t).Api {
		if ls.Api == elem.Api {
			check = true
		}
	}
	if !check {
		(*t).Api = append((*t).Api, elem)
	}
}

func (t *Env) Delette(api string) string {
	ret := false
	ls := (*t).Api
	var tmp []BybitApi

	for i := 0; i < len(ls); i++ {
		if ls[i].Api != api {
			tmp = append(tmp, ls[i])
		} else {
			ret = true
		}
	}
	(*t).Api = tmp
	if !ret {
		return "Api not found cannot be deletted"
	}
	return "Api deletted"
}

func (t *Env) DeletteAdmin(adm string) string {
	ret := false
	ls := (*t).Admin
	var tmp []string

	for i := 0; i < len(ls); i++ {
		if ls[i] != adm {
			tmp = append(tmp, ls[i])
		} else {
			ret = true
		}
	}
	(*t).Admin = tmp
	if !ret {
		return "Admin not found cannot be deletted"
	}
	return "Admin deletted"
}

func (t Env) ListApi() {
	for i := 0; i < len(t.Api); i++ {
		log.Println(print.PrettyPrint(t.Api[i]))
	}
}

func (t *Bot) NewBot(api *Env, debeug bool) error {
	elem := Bot{
		Active: nil,
		Debeug: debeug,
	}
	*t = elem
	return nil
}

func (t *Bot) CheckPositon(pos get.Position) {
	if len(pos.Result.List) == 0 {
		return
	}
	parse := func(s string) float64 {
		f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return f
	}

	// v5: position list items are strings; consider a position "open" if any
	// leg has size > 0 (primary signal). We also allow avg/liq/bust > 0 as a fallback.
	hasOpen := false
	sym := pos.Result.List[0].Symbol
	for _, p := range pos.Result.List {
		if p.Size != "" && p.Size != "0" && p.Size != "0.0000" {
			hasOpen = true
			break
		}
		if parse(p.Size) > 0 || parse(p.AvgPrice) > 0 || parse(p.LiqPrice) > 0 || parse(p.BustPrice) > 0 {
			hasOpen = true
			break
		}
	}

	for i := range t.Active {
		if t.Active[i].Symbol == sym {
			t.Active[i].Active = hasOpen
		}
	}
}

func (t Bot) GetActive() []string {
	var tmp []string
	for i := 0; i < len(t.Active); i++ {
		tmp = append(tmp, t.Active[i].Symbol)
	}
	return tmp
}

func (t *Bot) GetActiveSymbol(symbol string) bool {
	ret := false
	for _, ls := range (*t).Active {
		if ls.Symbol == symbol {
			ret = ls.Active
		}
	}
	return ret
}

func (t *Bot) AddActive(symbol string) {
	ls := (*t).Active
	elem := Start{
		Symbol: symbol,
		Active: false,
	}

	ls = append(ls, elem)
	(*t).Active = ls
}

func (t *Bot) Delete(symbol string) {
	var tmp []Start
	ls := (*t).Active

	for i := 0; i < len(ls); i++ {
		if symbol != ls[i].Symbol {
			tmp = append(tmp, ls[i])
		}
	}
	(*t).Active = tmp
}

func (t *Trades) SetId(symbol string, id string) {
	ls := *t

	for i := 0; i < len(ls); i++ {
		if ls[i].Symbol == symbol {
			ls[i].Id = append(ls[i].Id, id)
		}
	}
}

func (t *Trades) GetTrades() *Trades {
	return t
}

func (t *Trades) SetSl(symbol string, sl string) {
	for i := 0; i < len(*t); i++ {
		if (*t)[i].Symbol == symbol {
			(*t)[i].Sl = sl
		}
	}
}

func (t *Trades) GetSymbolOrder() []string {
	ls := *t
	var ret []string
	var check bool

	for i := 0; i < len(ls); i++ {
		check = true
		if ret == nil {
			ret = append(ret, ls[i].Symbol)
		} else {
			for j := 0; j < len(ret); j++ {
				if ret[j] == ls[i].Symbol {
					check = false
				}
			}
			if check {
				ret = append(ret, ls[i].Symbol)
			}
		}
	}
	return ret
}

func (t *Trades) CheckSymbol(symbol string) bool {
	ls := *t

	for i := 0; i < len(ls); i++ {
		if ls[i].Symbol == symbol {
			return false
		}
	}
	return true
}

func GetTrade(symbol string, t *Trades) *Trade {
	ls := *t

	for i := 0; i < len(ls); i++ {
		if ls[i].Symbol == symbol {
			return &ls[i]
		}
	}
	return nil
}

func (t *Trades) Add(api BybitApi, data telegram.Data, price get.Price, url_bybit string) bool {
	// fixed margin per trade from env
	stakeEnv := os.Getenv("STAKE_USDT")
	if stakeEnv == "" {
		stakeEnv = "20"
	}
	stake, _ := strconv.ParseFloat(stakeEnv, 64)

	lev, _ := strconv.Atoi(data.Level)
	if lev <= 0 {
		lev = 10
	}

	// --- decide entry price ---
	// Prefer the policy for Channel-2 *even if* parser pre-filled midpoint.
	var entry float64
	if strings.EqualFold(data.Source, "CH2") && data.EntryLow != "" && data.EntryHigh != "" {
		lo, _ := strconv.ParseFloat(data.EntryLow, 64)
		hi, _ := strconv.ParseFloat(data.EntryHigh, 64)
		if hi < lo {
			lo, hi = hi, lo
		}

		last := parseF(price.Result.List[0].LastPrice)

		pol := strings.ToLower(strings.TrimSpace(os.Getenv("ENTRY_POLICY_CH2")))
		if pol == "" {
			pol = "mid"
		}

		switch pol {
		case "lowest":
			// for longs, lowest = lower bound; for shorts, highest = upper bound
			if strings.EqualFold(data.Type, "Buy") {
				entry = lo
			} else {
				entry = hi
			}
		case "closest":
			// pick the boundary nearer to current price
			if math.Abs(last-hi) < math.Abs(last-lo) {
				entry = hi
			} else {
				entry = lo
			}
		default: // "mid"
			entry = (lo + hi) / 2.0
		}
		data.Entry = fmt.Sprintf("%.6f", entry) // override parser midpoint
	} else if data.Entry != "" {
		entry, _ = strconv.ParseFloat(data.Entry, 64)
	} else if data.EntryLow != "" && data.EntryHigh != "" {
		// Fallback: any other ranged source
		lo, _ := strconv.ParseFloat(data.EntryLow, 64)
		hi, _ := strconv.ParseFloat(data.EntryHigh, 64)
		if hi < lo {
			lo, hi = hi, lo
		}

		last := parseF(price.Result.List[0].LastPrice)

		pol := strings.ToLower(strings.TrimSpace(os.Getenv("ENTRY_POLICY_CH2")))
		if pol == "" {
			pol = "mid"
		}

		switch pol {
		case "lowest":
			if strings.EqualFold(data.Type, "Buy") {
				entry = lo
			} else {
				entry = hi
			}
		case "closest":
			if math.Abs(last-hi) < math.Abs(last-lo) {
				entry = hi
			} else {
				entry = lo
			}
		default:
			entry = (lo + hi) / 2.0
		}
		data.Entry = fmt.Sprintf("%.6f", entry)
	} else {
		entry, _ = strconv.ParseFloat(data.Entry, 64)
	}

	// qty = (stake * leverage) / entry
	qty := (stake * float64(lev)) / entry

	// Split
	tp1Qty := qty * 0.50
	tp2Qty := qty * 0.25
	tp3Qty := qty * 0.15
	tp4Qty := 0.0
	if data.Tp4 != "" {
		tp4Qty = qty * 0.10
	} else {
		// if no TP4, distribute 50/30/20
		tp2Qty = qty * 0.30
		tp3Qty = qty * 0.20
	}

	// --- NEW: get filters and quantize ---
	inst, _ := get.GetInstrument(url_bybit, data.Currency)
	qtyStep, minQty, tick := 0.0, 0.0, 0.0
	if inst.RetCode == 0 && len(inst.Result.List) > 0 {
		f := inst.Result.List[0]
		qtyStep = parseF(f.LotSizeFilter.QtyStep)
		minQty = parseF(f.LotSizeFilter.MinOrderQty)
		tick = parseF(f.PriceFilter.TickSize)
	}

	// Quantize prices
	entry = RoundFloatF(RoundToTick(entry, tick), 6) // store as string later
	tp1 := RoundToTick(parseF(data.Tp1), tick)
	tp2 := RoundToTick(parseF(data.Tp2), tick)
	tp3 := RoundToTick(parseF(data.Tp3), tick)
	tp4 := RoundToTick(parseF(data.Tp4), tick)
	sl := RoundToTick(parseF(data.Sl), tick)

	// Recompute strings with tick applied
	data.Entry = RoundFloat(tp(entry), 6) // see helper below, or fmt.Sprintf
	data.Tp1 = RoundFloat(tp1, 6)
	data.Tp2 = RoundFloat(tp2, 6)
	data.Tp3 = RoundFloat(tp3, 6)
	if data.Tp4 != "" {
		data.Tp4 = RoundFloat(tp4, 6)
	}
	data.Sl = RoundFloat(sl, 6)

	// Snap qty splits to qtyStep and enforce minQty
	snap := func(q float64) float64 {
		if qtyStep > 0 {
			q = RoundDownToStep(q, qtyStep)
		}
		if minQty > 0 && q > 0 && q < minQty {
			q = minQty
		}
		return q
	}
	tp1Qty = snap(tp1Qty)
	tp2Qty = snap(tp2Qty)
	tp3Qty = snap(tp3Qty)
	tp4Qty = snap(tp4Qty)

	r := func(v float64) string { return RoundFloat(v, 3) }

	elem := Trade{
		Symbol:      data.Currency,
		Type:        data.Type,
		Order:       "Limit",
		SymbolPrice: price.Result.List[0].Bid1Price,
		Wallet:      fmt.Sprint(RoundFloat(stake, 2)), // store margin used
		Entry:       data.Entry,
		Leverage:    fmt.Sprint(lev),
		Tp1Order:    r(tp1Qty),
		Tp2Order:    r(tp2Qty),
		Tp3Order:    r(tp3Qty),
		Tp4Order:    r(tp4Qty),
		Tp1:         data.Tp1,
		Tp2:         data.Tp2,
		Tp3:         data.Tp3,
		Tp4:         data.Tp4,
		Sl:          data.Sl,
		BEAfterTP1:  data.BEAfterTP1,
	}

	log.Println("[TRADE.ADD] stake:", stakeEnv, "lev:", lev, "entry:", entry, "qty:", r(qty))
	log.Println("[TRADE.ADD] qty split:", elem.Tp1Order, elem.Tp2Order, elem.Tp3Order, elem.Tp4Order)
	log.Println("[TRADE.ADD] tp/SL:", elem.Tp1, elem.Tp2, elem.Tp3, elem.Tp4, "SL:", elem.Sl)

	if !t.CheckSymbol(data.Currency) {
		log.Printf("[TRADE.ADD] Trade already active for %s", data.Currency)
		return false
	}
	*t = append(*t, elem)
	return true
}

func (t *Trades) Delete(symbol string) bool {
	tmp := &Trades{}
	check := 0
	for i := 0; i < len(*t); i++ {
		if (*t)[i].Symbol != symbol {
			*tmp = append(*tmp, (*t)[i])
		} else {
			check = 1
		}
	}
	*t = *tmp
	return check != 1
}

func RoundDownToStep(x, step float64) float64 {
	if step <= 0 {
		return x
	}
	return math.Floor(x/step) * step
}
func RoundToTick(x, tick float64) float64 {
	if tick <= 0 {
		return x
	}
	// Bybit usually wants prices on tick; round to nearest
	return math.Round(x/tick) * tick
}

func parseF(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

func RoundFloatF(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func tp(x float64) float64 { return x }

func (t *Trades) Print() {
	ls := *t

	for i := 0; i < len(ls); i++ {
		log.Println(print.PrettyPrint(ls[i]))
	}
}

func (t *Trades) GetLen() int {
	ls := *t

	return len(ls)
}

// get Trade
func (t *Trades) GetSymbol(index int) string {
	ls := *t
	for i := 0; i < len(ls); i++ {
		if i == index {
			return ls[i].Symbol
		}
	}
	return ""
}

func (t *Trades) GetType(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Type
	}
	return ""
}

func (t *Trades) GetOrder(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Order
	}
	return ""
}

func (t *Trades) GetSymbolPrice(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.SymbolPrice
	}
	return ""
}

func (t *Trades) GetWallet(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Wallet
	}
	return ""
}

func (t *Trades) GetPrice(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Price
	}
	return ""
}

func (t *Trades) GetEntry(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Entry
	}
	return ""
}

func (t *Trades) GetTp1Order(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp1Order
	}
	return ""
}

func (t *Trades) GetLeverage(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Leverage
	}
	return ""
}

func (t *Trades) GetTp2Order(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp2Order
	}
	return ""
}

func (t *Trades) GetTp3Order(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp3Order
	}
	return ""
}

func (t *Trades) GetTp1(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp1
	}
	return ""
}

func (t *Trades) GetTp2(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp2
	}
	return ""
}

func (t *Trades) GetTp3(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp3
	}
	return ""
}

func (t *Trades) GetTp4(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp4
	}
	return ""
}

func (t *Trades) GetTp4Order(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Tp4Order
	}
	return ""
}

func (t *Trades) GetBEAfterTP1(symbol string) bool {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.BEAfterTP1
	}
	return false
}

func (t *Trades) GetSl(symbol string) string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Sl
	}
	return ""
}

func (t *Trades) GetId(symbol string) []string {
	ret := GetTrade(symbol, t)
	if ret != nil {
		return ret.Id
	}
	return nil
}

func RoundFloat(val float64, precision uint) string {
	ratio := math.Pow(10, float64(precision))
	ret := math.Round(val*ratio) / ratio
	rets := fmt.Sprint(ret)
	return rets
}

func (t *Env) AddAdmin(admin string) {
	t.Admin = append(t.Admin, admin)
}

func GetEnv(env *Env) error {
	api := os.Getenv("API")
	if api == "" {
		return errors.New("api not found")
	}
	api_secret := os.Getenv("API_SECRET")
	if api_secret == "" {
		return errors.New("api_secret not found")
	}
	env.Api_telegram = os.Getenv("API_TELEGRAM")
	if env.Api_telegram == "" {
		return errors.New("api_telegram not found")
	}
	env.Url = os.Getenv("URL")
	if env.Url == "" {
		return errors.New("url not found")
	}
	admin := os.Getenv("ADMIN")
	if admin == "" {
		return errors.New("admin not found")
	}
	env.BotName = os.Getenv("BOT_NAME")
	if env.BotName == "" {
		return errors.New("Bot name not found")
	}
	env.IdCHannel = os.Getenv("ID_CHANNEL")
	log.Println("env id ")
	log.Println(env.IdCHannel)
	if env.IdCHannel == "" {
		return errors.New("your channel name not found")
	}
	env.AddAdmin(admin)
	env.AddApi(api, api_secret)
	return nil
}

func LoadEnv(env *Env) error {
	err := GetEnv(env)
	if err != nil {
		return err
	}
	return nil
}
