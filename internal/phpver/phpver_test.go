package phpver

import (
	"testing"
	"time"
)

// fixedNow 固定時間點，避免測試隨實際日期漂移
func fixedNow() time.Time {
	return time.Date(2026, 10, 22, 15, 30, 0, 0, time.UTC)
}

// TestLookupState 驗證各分支狀態判定（EOL / 預警 / 支援中）
func TestLookupState(t *testing.T) {
	cases := []struct {
		majorMin string
		want     string
	}{
		{"7.4", StateEOL},       // 2022-11-28 已停止支援
		{"8.2", StateWarning},   // 2026-12-31，距 2026-10-22 約 70 天，進入預警期
		{"8.3", StateSupported}, // 2027-12-31，仍在支援期
		{"8.5", StateSupported}, // 2029-12-31
	}
	for _, c := range cases {
		got, ok := Lookup(c.majorMin, fixedNow())
		if !ok {
			t.Fatalf("Lookup(%q) 應存在", c.majorMin)
		}
		if got.State != c.want {
			t.Errorf("Lookup(%q).State = %q，預期 %q", c.majorMin, got.State, c.want)
		}
	}
}

// TestLookupBoundary 驗證停止支援當天仍算支援、隔天才轉為 EOL
func TestLookupBoundary(t *testing.T) {
	onEOLDay := time.Date(2026, 12, 31, 9, 0, 0, 0, time.UTC)
	if got, _ := Lookup("8.2", onEOLDay); got.State == StateEOL {
		t.Errorf("停止支援當天不應判定為 EOL，實際 %q", got.State)
	}
	nextDay := time.Date(2027, 1, 1, 0, 1, 0, 0, time.UTC)
	if got, _ := Lookup("8.2", nextDay); got.State != StateEOL {
		t.Errorf("停止支援隔天應為 EOL，實際 %q", got.State)
	}
}

// TestLookupUnknown 驗證未知分支不誤報
func TestLookupUnknown(t *testing.T) {
	if _, ok := Lookup("9.9", fixedNow()); ok {
		t.Error("未知分支不應回傳支援資訊")
	}
}

// TestAllSorted 驗證清單依版本由新到舊排序
func TestAllSorted(t *testing.T) {
	list := All(fixedNow())
	if len(list) == 0 {
		t.Fatal("生命週期清單不可為空")
	}
	if list[0].MajorMin != "8.5" {
		t.Errorf("排序後第一筆應為 8.5，實際 %q", list[0].MajorMin)
	}
	if last := list[len(list)-1]; last.MajorMin != "5.6" {
		t.Errorf("排序後最後一筆應為 5.6，實際 %q", last.MajorMin)
	}
}
