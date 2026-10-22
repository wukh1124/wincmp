// Package phpver 維護 PHP 各分支的官方支援狀態（生命週期）資訊。
// 資料來源：https://www.php.net/supported-versions.php 、 https://www.php.net/eol.php
// 官方調整支援期程時，請同步更新下方 eolDates。
package phpver

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// 分支支援狀態
const (
	StateSupported = "supported" // 仍在安全支援期
	StateWarning   = "warning"   // 已進入停止支援預警期
	StateEOL       = "eol"       // 已停止支援
)

// WarningDays 為「即將停止支援」預警的天數門檻
const WarningDays = 180

// Branch 描述單一 PHP 分支（主.次版本）的支援狀態
type Branch struct {
	MajorMin string `json:"MajorMin"`
	EOLDate  string `json:"EOLDate"`  // 停止支援日（YYYY-MM-DD）
	DaysLeft int    `json:"DaysLeft"` // 距停止支援天數，已停止支援為負值
	State    string `json:"State"`
}

// eolDates 為各分支的官方停止支援日期；未列出的分支視為未知，前端不顯示標記。
var eolDates = map[string]string{
	"5.6": "2018-12-31",
	"7.0": "2019-01-10",
	"7.1": "2019-12-01",
	"7.2": "2020-11-30",
	"7.3": "2021-12-06",
	"7.4": "2022-11-28",
	"8.0": "2023-11-26",
	"8.1": "2025-12-31",
	"8.2": "2026-12-31",
	"8.3": "2027-12-31",
	"8.4": "2028-12-31",
	"8.5": "2029-12-31",
}

// Lookup 查詢指定分支的支援狀態；未知分支回傳 false。
func Lookup(majorMin string, now time.Time) (Branch, bool) {
	dateStr, ok := eolDates[strings.TrimSpace(majorMin)]
	if !ok {
		return Branch{}, false
	}
	return build(majorMin, dateStr, now), true
}

// All 回傳所有已知分支，依版本由新到舊排序。
func All(now time.Time) []Branch {
	out := make([]Branch, 0, len(eolDates))
	for majorMin, dateStr := range eolDates {
		out = append(out, build(majorMin, dateStr, now))
	}
	sort.Slice(out, func(i, j int) bool {
		return compareMajorMin(out[i].MajorMin, out[j].MajorMin) > 0
	})
	return out
}

// build 依停止支援日推算狀態；以日期（而非時間）比較，避免同一天因時分秒產生誤差。
func build(majorMin, dateStr string, now time.Time) Branch {
	b := Branch{MajorMin: majorMin, EOLDate: dateStr, State: StateSupported}
	eol, err := time.ParseInLocation("2006-01-02", dateStr, now.Location())
	if err != nil {
		return b
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	b.DaysLeft = int(eol.Sub(today).Hours() / 24)
	switch {
	case b.DaysLeft < 0:
		b.State = StateEOL
	case b.DaysLeft <= WarningDays:
		b.State = StateWarning
	}
	return b
}

// compareMajorMin 以數值比較「主.次」版本（字串比較在 8.9 / 8.10 會出錯）
func compareMajorMin(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var na, nb int
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na != nb {
			if na < nb {
				return -1
			}
			return 1
		}
	}
	return 0
}
