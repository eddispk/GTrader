package listen

import (
	"bot/data"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func nowMs() int64 { return time.Now().UnixMilli() }

func parseMs(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		// if seconds, upscale to ms
		if v < 1_000_000_000_000 {
			return v * 1000
		}
		return v
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		if f < 1e12 {
			return int64(f * 1000)
		}
		return int64(f)
	}
	return 0
}

func humanizeDuration(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, h)
}

func sinceStart(tr *data.Trades, symbol string) string {
	start := tr.GetStartMs(symbol)
	if start <= 0 {
		return "—"
	}
	return humanizeDuration(nowMs() - start)
}

func pnlPct(side string, entry, price float64) float64 {
	if entry == 0 {
		return 0
	}
	if strings.EqualFold(side, "Buy") {
		return (price - entry) / entry * 100.0
	}
	return (entry - price) / entry * 100.0
}
