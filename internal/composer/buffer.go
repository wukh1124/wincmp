package composer

import (
	"bytes"
	"sync"
	"unicode/utf8"

	"wincmp/internal/i18n"
)

const (
	// HeadLimitBytes 保留輸出開頭的最大位元組數 (8 KB)
	HeadLimitBytes = 8 * 1024
	// TailLimitBytes 保留輸出結尾的最大位元組數 (24 KB)
	TailLimitBytes = 24 * 1024
)

// HeadTailBuffer 固定記憶體上限的頭尾輸出緩衝區，並精確統計原始位元組數與行數
type HeadTailBuffer struct {
	mu            sync.Mutex
	headLimit     int
	tailLimit     int
	head          []byte
	tail          []byte
	tailPos       int
	tailFull      bool
	truncated     bool
	originalBytes int
	originalLines int
	endedWithLF   bool
}

// NewHeadTailBuffer 建立頭尾截斷緩衝區（預設 8 KB + 24 KB）
func NewHeadTailBuffer(headLimit, tailLimit int) *HeadTailBuffer {
	if headLimit <= 0 {
		headLimit = HeadLimitBytes
	}
	if tailLimit <= 0 {
		tailLimit = TailLimitBytes
	}
	return &HeadTailBuffer{
		headLimit: headLimit,
		tailLimit: tailLimit,
	}
}

// Write 寫入一段輸出串流並更新統計資訊
func (b *HeadTailBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	b.originalBytes += len(p)
	b.originalLines += bytes.Count(p, []byte{'\n'})
	b.endedWithLF = p[len(p)-1] == '\n'

	// 先填滿頭部緩衝
	if len(b.head) < b.headLimit {
		remain := b.headLimit - len(b.head)
		if len(p) <= remain {
			b.head = append(b.head, p...)
			return
		}
		b.head = append(b.head, p[:remain]...)
		p = p[remain:]
	}

	// 頭部已滿，剩餘資料寫入尾部環形緩衝
	if len(p) == 0 {
		return
	}
	if b.tail == nil {
		b.tail = make([]byte, b.tailLimit)
	}
	if len(p) >= b.tailLimit {
		b.truncated = true
		copy(b.tail, p[len(p)-b.tailLimit:])
		b.tailPos = 0
		b.tailFull = true
		return
	}

	for _, ch := range p {
		b.tail[b.tailPos] = ch
		b.tailPos++
		if b.tailPos >= b.tailLimit {
			b.tailPos = 0
			if b.tailFull {
				b.truncated = true
			}
			b.tailFull = true
		} else if b.tailFull {
			b.truncated = true
		}
	}
}

// Result 回傳最終格式化輸出、是否截斷、原始位元組數與原始行數
func (b *HeadTailBuffer) Result() (output string, truncated bool, origBytes int, origLines int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	origBytes = b.originalBytes
	origLines = b.originalLines
	if origBytes > 0 && !b.endedWithLF {
		origLines++
	}

	if !b.truncated {
		var buf bytes.Buffer
		buf.Write(b.head)
		if b.tailFull {
			buf.Write(b.tail[b.tailPos:])
			buf.Write(b.tail[:b.tailPos])
		} else if b.tailPos > 0 {
			buf.Write(b.tail[:b.tailPos])
		}
		return buf.String(), false, origBytes, origLines
	}

	// 重組環形尾部資料
	tailData := make([]byte, b.tailLimit)
	copy(tailData, b.tail[b.tailPos:])
	copy(tailData[b.tailLimit-b.tailPos:], b.tail[:b.tailPos])

	// 1. 對齊頭部至 UTF-8 邊界（並盡量對齊最後一個換行符，避免截斷半行）
	headSlice := trimValidUTF8End(b.head)
	if idx := bytes.LastIndexByte(headSlice, '\n'); idx > len(headSlice)/2 {
		headSlice = headSlice[:idx+1]
	}

	// 2. 對齊尾部至有效 UTF-8 字元起點，並跳過開頭可能殘留的半行/半截 ANSI 序列
	tailSlice := trimValidUTF8Start(tailData)
	if idx := bytes.IndexByte(tailSlice, '\n'); idx >= 0 && idx < 512 && idx+1 < len(tailSlice) {
		tailSlice = tailSlice[idx+1:]
	}

	omittedBytes := origBytes - len(headSlice) - len(tailSlice)
	if omittedBytes < 0 {
		omittedBytes = 0
	}
	keptLines := bytes.Count(headSlice, []byte{'\n'}) + bytes.Count(tailSlice, []byte{'\n'})
	omittedLines := origLines - keptLines
	if omittedLines < 0 {
		omittedLines = 0
	}

	marker := "\x1b[0m\n" + i18n.Tfmt("... [已省略 %d bytes / %d 行，原始總計 %d bytes / %d 行] ...", omittedBytes, omittedLines, origBytes, origLines) + "\n\x1b[0m"

	var out bytes.Buffer
	out.Grow(len(headSlice) + len(marker) + len(tailSlice))
	out.Write(headSlice)
	out.WriteString(marker)
	out.Write(tailSlice)

	return out.String(), true, origBytes, origLines
}

func trimValidUTF8End(b []byte) []byte {
	for len(b) > 0 && !utf8.Valid(b) {
		// 最多回退 3 bytes 即可排除結尾不完整 UTF-8 序列
		lastStart := len(b) - 1
		for lastStart > 0 && lastStart > len(b)-4 && !utf8.RuneStart(b[lastStart]) {
			lastStart--
		}
		if lastStart >= 0 && !utf8.FullRune(b[lastStart:]) {
			return b[:lastStart]
		}
		break
	}
	return b
}

func trimValidUTF8Start(b []byte) []byte {
	for len(b) > 0 && !utf8.RuneStart(b[0]) {
		b = b[1:]
	}
	return b
}
