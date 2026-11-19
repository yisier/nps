package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

const (
	// logBufferCap 每个客户端内存中最多保留的日志条数
	logBufferCap = 3000
	// logFileMaxSize 单个日志文件超过该大小后轮转（5MB）
	logFileMaxSize = 5 << 20
	// logFileMaxBackups 轮转时最多保留的历史文件个数（.1 .2 .3）
	logFileMaxBackups = 3
)

// initAppLogger 初始化 GUI 的日志系统。
// - beego 全局日志（nps 客户端库使用）接到自定义适配器，格式与 npc 进程一致
// - 标准库 log（lib/nps_mux 等使用）也写入同一份运行日志
// - 应用自身日志仍使用 slog（JSON），写入 logs/npc-gui.log
func initAppLogger() {
	logs.Reset()
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)
	logs.SetLevel(logs.LevelDebug)
	_ = logs.SetLogger("npcgui")

	// lib/nps_mux 等使用标准库 log，收进同一份运行日志。
	log.SetFlags(0)
	log.SetOutput(runLogStdWriter{})

	openRunLogFile()

	var w io.Writer = os.Stderr
	if fw, err := newRotatingFileWriter(filepath.Join(getLogsPath(), "npc-gui.log")); err == nil {
		w = io.MultiWriter(fw, os.Stderr)
	}
	logger := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

func init() {
	logs.Register("npcgui", func() logs.Logger { return &runLogAdapter{} })
}

// runLogAdapter 把 beego 日志写入全局运行日志，格式与 npc 控制台一致：
// 2026/11/19 16:00:00 [I] [client.go:104] Successful connection ...
type runLogAdapter struct{}

func (*runLogAdapter) Init(string) error { return nil }
func (*runLogAdapter) Destroy()          {}
func (*runLogAdapter) Flush()            {}

func (*runLogAdapter) WriteMsg(when time.Time, msg string, level int) error {
	logType := "info"
	switch level {
	case logs.LevelEmergency, logs.LevelAlert, logs.LevelCritical, logs.LevelError:
		logType = "error"
	case logs.LevelWarning:
		logType = "warning"
	}
	appendRunEntry(logType, when, msg)
	return nil
}

// runLogStdWriter 收纳标准库 log 的输出。
type runLogStdWriter struct{}

func (runLogStdWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg != "" {
		appendRunMessage("info", "[I] "+msg)
	}
	return len(p), nil
}

type runLogStore struct {
	mu   sync.Mutex
	ring *ringBuffer
	file *rotatingFileWriter
}

var appRunLog runLogStore

func openRunLogFile() {
	appRunLog.mu.Lock()
	defer appRunLog.mu.Unlock()
	if appRunLog.ring == nil {
		appRunLog.ring = newRingBuffer(logBufferCap)
	}
	if appRunLog.file == nil {
		if fw, err := newRotatingFileWriter(filepath.Join(getLogsPath(), "npc-client.log")); err == nil {
			appRunLog.file = fw
		}
	}
}

func appendRunMessage(logType, msg string) {
	appendRunEntry(logType, time.Now(), msg)
}

func appendRunEntry(logType string, when time.Time, msg string) {
	if logType == "info" && isSuccessMessage(msg) {
		logType = "success"
	}
	e := ConnectionLog{
		Timestamp: when.Format("2006/01/02 15:04:05"),
		Message:   msg,
		Type:      logType,
	}

	appRunLog.mu.Lock()
	defer appRunLog.mu.Unlock()
	if appRunLog.ring == nil {
		appRunLog.ring = newRingBuffer(logBufferCap)
	}
	appRunLog.ring.add(e)
	if appRunLog.file != nil {
		_, _ = appRunLog.file.Write([]byte(e.Timestamp + " " + msg + "\n"))
	}
}

func runLogSnapshot() []ConnectionLog {
	appRunLog.mu.Lock()
	defer appRunLog.mu.Unlock()
	if appRunLog.ring == nil {
		return nil
	}
	return appRunLog.ring.all()
}

func clearRunLogs() {
	appRunLog.mu.Lock()
	defer appRunLog.mu.Unlock()
	if appRunLog.ring != nil {
		appRunLog.ring.reset()
	}
	if appRunLog.file != nil {
		_ = appRunLog.file.truncate()
	}
}

// rotatingFileWriter 带大小轮转的日志文件写入器（线程安全）。
// 写入超过 maxSize 时把当前文件顺延为 .1 .2 ...，保留 maxBackups 份历史。
type rotatingFileWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingFileWriter(path string) (*rotatingFileWriter, error) {
	w := &rotatingFileWriter{path: path, maxSize: logFileMaxSize, maxBackups: logFileMaxBackups}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *rotatingFileWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.size = st.Size()
	return nil
}

func (w *rotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		w.rotateLocked()
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingFileWriter) rotateLocked() {
	_ = w.file.Close()
	w.file = nil
	for i := w.maxBackups - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", w.path, i)
		if _, err := os.Stat(src); err == nil {
			_ = os.Remove(fmt.Sprintf("%s.%d", w.path, i+1))
			_ = os.Rename(src, fmt.Sprintf("%s.%d", w.path, i+1))
		}
	}
	_ = os.Remove(w.path + ".1")
	_ = os.Rename(w.path, w.path+".1")
	w.size = 0
	_ = w.open()
}

// truncate 清空当前文件内容
func (w *rotatingFileWriter) truncate() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	f, err := os.Create(w.path)
	if err != nil {
		return err
	}
	w.file = f
	w.size = 0
	return nil
}

func (w *rotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}

// ringBuffer 固定容量的环形日志缓冲
type ringBuffer struct {
	buf   []ConnectionLog
	start int
	count int
}

func newRingBuffer(cap int) *ringBuffer {
	return &ringBuffer{buf: make([]ConnectionLog, cap)}
}

func (r *ringBuffer) add(e ConnectionLog) {
	if r.count == len(r.buf) {
		r.buf[r.start] = e
		r.start = (r.start + 1) % len(r.buf)
	} else {
		r.buf[(r.start+r.count)%len(r.buf)] = e
		r.count++
	}
}

func (r *ringBuffer) all() []ConnectionLog {
	out := make([]ConnectionLog, 0, r.count)
	for i := 0; i < r.count; i++ {
		out = append(out, r.buf[(r.start+i)%len(r.buf)])
	}
	return out
}

func (r *ringBuffer) reset() {
	r.start = 0
	r.count = 0
}

// clientLogStore 单个客户端的日志存储：内存环形缓冲 + JSON Lines 文件持久化。
// UI 直接读取内存缓冲，不再解析日志文件文本。
type clientLogStore struct {
	mu       sync.Mutex
	clientId string
	ring     *ringBuffer
	file     *rotatingFileWriter
}

func newClientLogStore(clientId string) *clientLogStore {
	s := &clientLogStore{
		clientId: clientId,
		ring:     newRingBuffer(logBufferCap),
	}
	if fw, err := newRotatingFileWriter(getClientLogFilePath(clientId)); err == nil {
		s.file = fw
		s.loadHistoryLocked()
	}
	return s
}

// loadHistoryLocked 启动时把已有日志文件尾部的 JSON 行载入内存缓冲（跨重启保留历史）。
func (s *clientLogStore) loadHistoryLocked() {
	data, err := os.ReadFile(s.file.path)
	if err != nil {
		return
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	var lines []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) > logBufferCap {
		lines = lines[len(lines)-logBufferCap:]
	}
	for _, line := range lines {
		var e ConnectionLog
		if json.Unmarshal([]byte(line), &e) == nil && e.Message != "" {
			s.ring.add(e)
		}
	}
}

// log 写入一条日志：先入内存缓冲，再持久化到 JSON Lines 文件。
func (s *clientLogStore) log(level, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logType := level
	if level == "info" && isSuccessMessage(msg) {
		logType = "success"
	}
	e := ConnectionLog{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Message:   msg,
		Type:      logType,
		ClientId:  s.clientId,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ring.add(e)
	if s.file != nil {
		if line, err := json.Marshal(e); err == nil {
			_, _ = s.file.Write(append(line, '\n'))
		}
	}
}

func (s *clientLogStore) snapshot() []ConnectionLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ring.all()
}

func (s *clientLogStore) clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ring.reset()
	if s.file != nil {
		return s.file.truncate()
	}
	return nil
}

func (s *clientLogStore) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
}

var (
	logStoresMu sync.Mutex
	logStores   = make(map[string]*clientLogStore)
)

// getClientLogStore 获取客户端的日志存储（懒创建，未启动也能查看历史日志）
func getClientLogStore(clientId string) *clientLogStore {
	logStoresMu.Lock()
	defer logStoresMu.Unlock()
	if s, ok := logStores[clientId]; ok {
		return s
	}
	s := newClientLogStore(clientId)
	logStores[clientId] = s
	return s
}

// closeAllLogStores 应用退出时关闭所有日志文件
func closeAllLogStores() {
	logStoresMu.Lock()
	defer logStoresMu.Unlock()
	for id, s := range logStores {
		s.close()
		delete(logStores, id)
	}
}

// trpcLogger 把 client.Logger 桥接到 clientLogStore，供 TRPClient 使用。
// 同时写入全局运行日志（带客户端前缀），与 npc 进程日志放在一起显示。
type trpcLogger struct {
	store *clientLogStore
}

func (l *trpcLogger) Info(format string, v ...interface{}) {
	msg := formatLogMessage(format, v...)
	l.store.log("info", "%s", msg)
	appendRunMessage("info", fmt.Sprintf("[I] [%s] %s", l.store.clientId, msg))
}

func (l *trpcLogger) Error(format string, v ...interface{}) {
	msg := formatLogMessage(format, v...)
	l.store.log("error", "%s", msg)
	appendRunMessage("error", fmt.Sprintf("[E] [%s] %s", l.store.clientId, msg))
}

func (l *trpcLogger) Warn(format string, v ...interface{}) {
	msg := formatLogMessage(format, v...)
	l.store.log("warning", "%s", msg)
	appendRunMessage("warning", fmt.Sprintf("[W] [%s] %s", l.store.clientId, msg))
}

func (l *trpcLogger) Trace(format string, v ...interface{}) {
	msg := formatLogMessage(format, v...)
	appendRunMessage("info", fmt.Sprintf("[D] [%s] %s", l.store.clientId, msg))
}

// formatLogMessage 与 beego logs 的格式化规则一致：
// 没有占位符时把参数依次接在消息后面，避免出现 %!(EXTRA ...)。
func formatLogMessage(format string, v ...interface{}) string {
	if len(v) == 0 {
		return format
	}
	msg := format
	if !strings.Contains(msg, "%") || strings.Contains(msg, "%%") {
		msg += strings.Repeat(" %v", len(v))
	}
	return fmt.Sprintf(msg, v...)
}

// isSuccessMessage 判断是否为“成功”类日志（用于 UI 高亮）
func isSuccessMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(msg, "成功") ||
		strings.Contains(lower, "success") ||
		strings.Contains(lower, "connected")
}
