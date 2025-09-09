package telegram

import (
	"bot/bybits/print"
	"errors"
	"log"
	"strings"
)

type Data struct {
	Currency   string `json:"currency"`
	Type       string `json:"type"` // Buy|Sell
	Entry      string `json:"entry"`
	Tp1        string `json:"tp_1"`
	Tp2        string `json:"tp_2"`
	Tp3        string `json:"tp_3"`
	Tp4        string `json:"tp_4"` // NEW
	Sl         string `json:"sl"`
	Level      string `json:"level"`
	SetUp      string `json:"set_up"`
	Order      string `json:"order"`
	Cancel     bool
	Trade      bool
	Spot       bool
	BEAfterTP1 bool `json:"be_after_tp1"` // NEW
}

func SetDataNil(data *Data) {
	data.Trade = false
	data.Cancel = false
	data.Spot = false
	data.BEAfterTP1 = false
}

func CancelParse(msg string, debug bool, data Data) (Data, error) {
	pos := strings.Index(msg, "#")
	SetDataNil(&data)
	if pos > 0 {
		data.Cancel = true
		data.Currency = msg[pos+1:]
		data.Currency = data.Currency[:strings.Index(data.Currency, " ")]
		data.Currency = strings.Replace(data.Currency, "/", "", 1)
	}
	if debug {
		log.Println("[PARSER] CancelParse:", print.PrettyPrint(data))
	}
	return data, nil
}

func FuturParse(msg string, debug bool, data Data) (Data, error) {
	SetDataNil(&data)
	lines := strings.Split(msg, "\n")

	for _, raw := range lines {
		l := strings.TrimSpace(raw)
		low := strings.ToLower(l)

		// 1) Header: "ENS / LONG" or "WAL / SHORT"
		if strings.Contains(l, "/") && (strings.Contains(low, "long") || strings.Contains(low, "short")) {
			sym := l
			if i := strings.Index(strings.ToUpper(sym), " LONG"); i != -1 {
				sym = sym[:i]
				data.Type = "Buy"
			}
			if i := strings.Index(strings.ToUpper(sym), " SHORT"); i != -1 {
				sym = sym[:i]
				data.Type = "Sell"
			}
			sym = strings.ReplaceAll(sym, " ", "")
			sym = strings.ReplaceAll(sym, "/", "")
			// assume USDT linear contracts
			if !strings.HasSuffix(strings.ToUpper(sym), "USDT") {
				sym = sym + "USDT"
			}
			data.Currency = strings.ToUpper(sym)
			log.Println("[PARSER] symbol/type:", data.Currency, data.Type)
			continue
		}

		// 2) Leverage: 20x
		if strings.HasPrefix(low, "leverage") && strings.Contains(l, ":") {
			x := strings.TrimSpace(l[strings.Index(l, ":")+1:])
			x = strings.TrimSuffix(x, "x")
			data.Level = strings.TrimSpace(x)
			log.Println("[PARSER] leverage:", data.Level)
			continue
		}

		// 3) Entry
		if strings.Contains(low, "entry") && strings.Contains(l, ":") {
			data.Entry = strings.TrimSpace(l[strings.Index(l, ":")+1:])
			log.Println("[PARSER] entry:", data.Entry)
			continue
		}

		// 4) Targets: comma list (TP1..TP4)
		if strings.Contains(low, "targets") && strings.Contains(l, ":") {
			trg := strings.TrimSpace(l[strings.Index(l, ":")+1:])
			parts := strings.Split(trg, ",")
			if len(parts) > 0 {
				data.Tp1 = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				data.Tp2 = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				data.Tp3 = strings.TrimSpace(parts[2])
			}
			if len(parts) > 3 {
				data.Tp4 = strings.TrimSpace(parts[3])
			}
			log.Println("[PARSER] targets:", data.Tp1, data.Tp2, data.Tp3, data.Tp4)
			continue
		}

		// 5) Stop or SL
		if (strings.Contains(low, "stop") || strings.Contains(low, "sl")) && strings.Contains(l, ":") {
			data.Sl = strings.TrimSpace(l[strings.Index(l, ":")+1:])
			log.Println("[PARSER] stop:", data.Sl)
			continue
		}

		// 6) Breakeven instruction
		if strings.Contains(low, "once tp1 is hit") && strings.Contains(low, "breakeven") {
			data.BEAfterTP1 = true
			log.Println("[PARSER] breakeven after TP1: true")
			continue
		}
	}

	// Sanity checks
	if data.Currency != "" && data.Entry != "" && data.Sl != "" && data.Tp1 != "" && data.Type != "" {
		data.Trade = true
		if debug {
			log.Println("[PARSER] OK:", print.PrettyPrint(data))
		}
		return data, nil
	}

	if debug {
		log.Println("[PARSER] FAIL:", print.PrettyPrint(data))
	}
	return data, errors.New("Error Parsing")
}

func ParseMsg(msg string, debug bool) (Data, error) {
	var data Data

	if strings.Index(msg, "Cancelled") > 0 {
		return CancelParse(msg, debug, data)
	}
	return FuturParse(msg, debug, data)
}
