package telegram

import (
	"bot/bybits/print"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
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
	BEAfterTP1 bool   `json:"be_after_tp1"` // NEW
	EntryLow   string `json:"entry_low,omitempty"`
	EntryHigh  string `json:"entry_high,omitempty"`
	Source     string `json:"source,omitempty"`
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

// Channel 1: @Ethan_Signals PARSER
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
			data.Source = "CH1"
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
	return data, errors.New("error parsing")
}

// CHANNEL 2: @leaks_vip_signals  SIGNALS

/*
var (
	reHeaderSymbol  = regexp.MustCompile(`(?m)^\s*(?:🪙|🔹|•)?\s*([A-Z0-9]{2,20}USDT)\s*$`)
	reSideRangeLine = regexp.MustCompile(`(?im)^\s*(long|short)\s*:\s*([0-9][0-9.,]*)\s*-\s*([0-9][0-9.,]*)\s*$`)
	reLeverageLine  = regexp.MustCompile(`(?im)^\s*(?:cross\s+)?(\d+)\s*x\s*$`)
	reTargetsHeader = regexp.MustCompile(`(?im)^\s*👉?\s*targets?\s*$`)
	reTargetLine    = regexp.MustCompile(`(?im)^\s*\d+\)\s*([0-9][0-9.,]*)(?:\+)?\s*$`)
	reSLHeader      = regexp.MustCompile(`(?im)^\s*⛔?\s*sl\s*:?\s*$`)
	rePercentAny    = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*%`)
	reFirstNumber   = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`)
)
*/

var (
	// allow any leading emoji/punctuation, then SYMBOLUSDT
	reHeaderSymbol = regexp.MustCompile(`(?m)^\s*(?:[^\w\s]+)?\s*([A-Z0-9]{2,20}USDT)\s*$`)

	// accept hyphen OR en/em dash, case-insensitive LONG/SHORT
	reSideRangeLine = regexp.MustCompile(`(?im)^\s*(long|short)\s*:\s*([0-9][0-9.,]*)\s*[-–—]\s*([0-9][0-9.,]*)\s*$`)

	// accept "Cross 10x", "Cross10x", or just "10x"
	reLeverageLine = regexp.MustCompile(`(?im)^\s*(?:cross\s*)?(\d+)\s*x\s*$`)

	// accept "👉 Targets" or "Targets:" with optional colon
	reTargetsHeader = regexp.MustCompile(`(?im)^\s*👉?\s*targets?\s*:?\s*$`)
	reTargetLine    = regexp.MustCompile(`(?im)^\s*\d+\)\s*([0-9][0-9.,]*)(?:\+)?\s*$`)

	// SL header: tolerate ⛔, ⛔️ (with variation selector), or no emoji at all
	reSLHeader   = regexp.MustCompile(`(?im)^\s*(?:⛔️?|[^\w\s]+)?\s*sl\s*:?\s*$`)
	rePercentAny = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)\s*%`)
)

func toF(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "")) // strip thousands commas just in case
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}
func f6(x float64) string { return fmt.Sprintf("%.6f", x) }

func SecondChannelParse(msg string, debug bool, data Data) (Data, error) {
	SetDataNil(&data)
	lines := strings.Split(msg, "\n")

	// 1) Symbol from header line only
	if m := reHeaderSymbol.FindStringSubmatch(msg); len(m) == 2 {
		data.Currency = strings.ToUpper(m[1])
	}

	// 2) Side + entry range (must be on its own line)
	var entry float64
	if m := reSideRangeLine.FindStringSubmatch(msg); len(m) == 4 {
		side := strings.ToLower(m[1])
		a, _ := toF(m[2])
		b, _ := toF(m[3])

		data.EntryLow = f6(math.Min(a, b))
		data.EntryHigh = f6(math.Max(a, b))

		entry = (a + b) / 2.0
		data.Entry = f6(entry)
		if side == "long" {
			data.Type = "Buy"
		} else {
			data.Type = "Sell"
		}
		data.Source = "CH2"

	}

	// 3) Leverage (prefer "Cross 10x" line, but accept plain "10x" line)
	for _, raw := range lines {
		l := strings.TrimSpace(raw)
		if m := reLeverageLine.FindStringSubmatch(l); len(m) == 2 {
			data.Level = m[1]
			break
		}
	}
	if data.Level == "" {
		data.Level = "10"
	} // sane default if missing

	// 4) Targets: read the first 4 numbered target lines AFTER the 👉 Targets header
	tps := []string{}
	seenTargets := false
	for _, raw := range lines {
		l := strings.TrimSpace(raw)
		if !seenTargets {
			if reTargetsHeader.MatchString(l) {
				seenTargets = true
			}
			continue
		}
		if m := reTargetLine.FindStringSubmatch(l); len(m) == 2 {
			val := strings.TrimSpace(strings.ReplaceAll(m[1], ",", ""))
			tps = append(tps, val)
			if len(tps) == 4 {
				break
			}
		}
	}
	if len(tps) > 0 {
		data.Tp1 = tps[0]
	}
	if len(tps) > 1 {
		data.Tp2 = tps[1]
	}
	if len(tps) > 2 {
		data.Tp3 = tps[2]
	}
	if len(tps) > 3 {
		data.Tp4 = tps[3]
	}

	// 5) SL: find SL header line, then read %s on the next non-empty line
	var slPct float64
	if entry > 0 {
		for i := 0; i < len(lines); i++ {
			if reSLHeader.MatchString(lines[i]) {
				// look ahead to the next non-empty line
				for j := i + 1; j < len(lines); j++ {
					nxt := strings.TrimSpace(lines[j])
					if nxt == "" {
						continue
					}
					pcts := rePercentAny.FindAllStringSubmatch(nxt, -1)
					if len(pcts) > 0 {
						// use the lower bound if a range like "5%-10%"
						min := 1e9
						for _, p := range pcts {
							if v, ok := toF(p[1]); ok && v < min {
								min = v
							}
						}
						if min < 1e9 {
							slPct = min
						}
						break
					}
					// fallback: absolute price
					if v, ok := toF(nxt); ok {
						data.Sl = f6(v)
						break
					}
					break
				}
				break
			}
		}
		if data.Sl == "" && slPct > 0 {
			if strings.EqualFold(data.Type, "Buy") {
				data.Sl = f6(entry * (1 - slPct/100.0))
			} else if strings.EqualFold(data.Type, "Sell") {
				data.Sl = f6(entry * (1 + slPct/100.0))
			}
		}
	}

	// 6) Sanity checks: require the essential fields
	if data.Currency == "" || data.Entry == "" || data.Sl == "" || data.Tp1 == "" || data.Type == "" {
		if debug {
			log.Println("[PARSER2] FAIL (missing fields):", print.PrettyPrint(data))
		}
		return data, errors.New("error parsing")
	}

	// 7) Directional sanity: targets must be on the correct side of entry
	nums := func(ss ...string) (out []float64) {
		for _, s := range ss {
			if v, ok := toF(s); ok {
				out = append(out, v)
			}
		}
		return
	}
	tpf := nums(data.Tp1, data.Tp2, data.Tp3, data.Tp4)
	if strings.EqualFold(data.Type, "Buy") {
		// all TPs should be >= entry (tolerate tiny eps)
		eps := entry * 1e-6
		for _, v := range tpf {
			if v+eps < entry {
				return data, errors.New("targets below entry for LONG")
			}
		}
	} else {
		// all TPs should be <= entry
		eps := entry * 1e-6
		for _, v := range tpf {
			if v-eps > entry {
				return data, errors.New("targets above entry for SHORT")
			}
		}
	}

	data.Trade = true
	if debug {
		log.Println("[PARSER2] OK:", print.PrettyPrint(data))
	}
	return data, nil
}

// CHANNEL 3: @Aman_crypto_vip  (each signal in a single message)
// Example lines:
// 🔴 TRADE -  PYTH / USDT ( Futures )
// 👉 Type -  LONG
// 👉 Leverage- 2X to 3X ( Recommend)
// 📌 Buy Zone -  0.18$ to 0.174$
// 🎯Target
// 1. 0.183$
// ...
// 🛑Stop loss 0.167$-( SL Must Use )
func ThirdChannelParse(msg string, debug bool, data Data) (Data, error) {
	SetDataNil(&data)
	text := strings.ReplaceAll(msg, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	// SYMBOL
	reHead := regexp.MustCompile(`(?i)trade\s*-\s*([A-Z0-9._-]+)\s*/\s*USDT`)
	if m := reHead.FindStringSubmatch(text); len(m) == 2 {
		data.Currency = strings.ToUpper(m[1]) + "USDT"
	}

	// TYPE
	reType := regexp.MustCompile(`(?i)type\s*[-:]*\s*(long|short)`)
	if m := reType.FindStringSubmatch(text); len(m) == 2 {
		if strings.EqualFold(m[1], "long") {
			data.Type = "Buy"
		} else {
			data.Type = "Sell"
		}
	}

	// LEVERAGE (pick the max in "2X to 3X")
	/*
		reLev := regexp.MustCompile(`(?i)leverage\s*[-:]*\s*([0-9]+)\s*x(?:\s*(?:to|-)\s*([0-9]+)\s*x)?`)
		if m := reLev.FindStringSubmatch(text); len(m) >= 2 {
			l1, _ := strconv.Atoi(m[1])
			lmax := l1
			if len(m) >= 3 && m[2] != "" {
				l2, _ := strconv.Atoi(m[2])
				if l2 > lmax {
					lmax = l2
				}
			}
			data.Level = fmt.Sprint(lmax)
		}*/
	data.Level = "10"

	// BUY ZONE range
	reZone := regexp.MustCompile(`(?i)buy\s*zone\s*[-:]*\s*([0-9][0-9.,]*)\$?\s*(?:to|[-–—])\s*([0-9][0-9.,]*)\$?`)
	if m := reZone.FindStringSubmatch(text); len(m) == 3 {
		a, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		b, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64)
		lo, hi := a, b
		if hi < lo {
			lo, hi = hi, lo
		}
		data.EntryLow = f6(lo)
		data.EntryHigh = f6(hi)
		data.Entry = f6((lo + hi) / 2.0) // midpoint; Add(...) may override per policy
	}

	// TARGETS (take first 4 if more)
	reTargetsHeader := regexp.MustCompile(`(?i)🎯?\s*target[s]?\b`)
	reTline := regexp.MustCompile(`(?i)^\s*\d+\s*[.)-]?\s*([0-9][0-9.,]*)\$?`)
	if reTargetsHeader.MatchString(text) {
		count := 0
		for _, raw := range lines {
			l := strings.TrimSpace(raw)
			if m := reTline.FindStringSubmatch(l); len(m) == 2 {
				val := strings.ReplaceAll(m[1], ",", "")
				switch count {
				case 0:
					data.Tp1 = val
				case 1:
					data.Tp2 = val
				case 2:
					data.Tp3 = val
				case 3:
					data.Tp4 = val // ignore 5th+
				}
				count++
				if count == 4 {
					break
				}
			}
		}
	}

	// STOP LOSS
	reSL := regexp.MustCompile(`(?i)stop\s*loss\s*([0-9][0-9.,]*)\$?`)
	if m := reSL.FindStringSubmatch(text); len(m) == 2 {
		data.Sl = strings.ReplaceAll(m[1], ",", "")
	}

	data.Source = "CH3"

	// Validate
	if data.Currency != "" &&
		(data.Entry != "" || (data.EntryLow != "" && data.EntryHigh != "")) &&
		data.Sl != "" && data.Tp1 != "" && data.Type != "" {
		data.Trade = true
		if debug {
			log.Println("[PARSER3] OK:", print.PrettyPrint(data))
		}
		return data, nil
	}
	if debug {
		log.Println("[PARSER3] FAIL:", print.PrettyPrint(data))
	}
	return data, errors.New("error parsing")
}

func ParseMsg(msg string, debug bool) (Data, error) {
	var data Data

	if strings.Contains(msg, "Cancelled") {
		return CancelParse(msg, debug, data)
	}

	// NEW: detect Channel 3 quickly
	low := strings.ToLower(msg)
	if strings.Contains(low, "trade -") &&
		strings.Contains(low, "buy zone") &&
		strings.Contains(low, "stop loss") {
		if out, err := ThirdChannelParse(msg, debug, data); err == nil {
			return out, nil
		}
	}

	// Channel 2?
	if reHeaderSymbol.MatchString(msg) && reSideRangeLine.MatchString(msg) {
		if out, err := SecondChannelParse(msg, debug, data); err == nil {
			return out, nil
		}
	}

	// Fallback to Channel 1
	return FuturParse(msg, debug, data)
}
