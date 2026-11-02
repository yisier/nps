package install

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	progressBarWidth = 30
	progressInterval = 200 * time.Millisecond
)

// progressReader 在读取下载数据时输出下载进度
type progressReader struct {
	reader io.Reader
	total  int64
	read   int64
	last   time.Time
	done   bool
}

func newProgressReader(r io.Reader, total int64) *progressReader {
	return &progressReader{reader: r, total: total, last: time.Now()}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.reader.Read(b)
	p.read += int64(n)
	if p.total > 0 && p.read >= p.total {
		p.finish()
	} else if time.Since(p.last) >= progressInterval {
		p.print()
		p.last = time.Now()
	}
	return n, err
}

// finish 输出最终进度并换行
func (p *progressReader) finish() {
	if p.done {
		return
	}
	p.print()
	fmt.Println()
	p.done = true
}

func (p *progressReader) print() {
	if p.total <= 0 {
		fmt.Printf("\r下载进度：%9s", humanSize(p.read))
		return
	}

	percent := float64(p.read) * 100 / float64(p.total)
	if percent > 100 {
		percent = 100
	}
	filled := int(percent / 100 * progressBarWidth)
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", progressBarWidth-filled)
	fmt.Printf("\r下载进度：[%s] %5.1f%% %9s / %9s", bar, percent, humanSize(p.read), humanSize(p.total))
}

func humanSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	size := float64(n) / 1024
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}
