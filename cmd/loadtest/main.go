// Command loadtest is a small, reproducible load generator for the cakecake
// backend. It has two modes:
//
//	loadtest http -url http://127.0.0.1:8080/api/v1/hot-search -c 50 -d 30s
//	loadtest ws   -url ws://127.0.0.1:8080/api/v1/ws/danmaku -video 6 -clients 100 -d 25s
//
// HTTP mode reports QPS, p50/p90/p99 latency, and the status-code
// distribution. WS mode connects N danmaku clients and, optionally, generates
// real danmaku traffic with minted JWTs (JWT_SECRET must be set then).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"cakecake/internal/pkg/jwttoken"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	switch os.Args[1] {
	case "http":
		runHTTP(ctx, os.Args[2:])
	case "ws":
		runWS(ctx, os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  loadtest http -url URL -c CONCURRENCY -d DURATION [-qps RATE] [-header "K: V"] [-out FILE]
  loadtest ws -url WSURL -clients N -video VIDEO_ID -d DURATION [-token TOKEN]
             [-sender-users N] [-send-interval 200ms] [-out FILE]`)
}

// ---------------------------------------------------------------- HTTP ----

func runHTTP(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("http", flag.ExitOnError)
	urls := fs.String("url", "", "target URL")
	conc := fs.Int("c", 50, "concurrency")
	dur := fs.Duration("d", 30*time.Second, "duration")
	qps := fs.Float64("qps", 0, "target rate; 0 = unlimited")
	header := fs.String("header", "", "extra header, repeat by separating with ';'")
	out := fs.String("out", "", "write JSON report to file")
	_ = fs.Parse(args)

	if *urls == "" {
		fmt.Fprintln(os.Stderr, "http: -url is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var start atomic.Int64
	var finish atomic.Int64
	var statuses sync.Map // code -> count
	latencies := make(chan time.Duration, 262144)
	var errCount atomic.Int64

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        *conc * 4,
			MaxIdleConnsPerHost: *conc * 4,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	headers := http.Header{}
	if *header != "" {
		for _, part := range strings.Split(*header, ";") {
			k, v, ok := strings.Cut(strings.TrimSpace(part), ":")
			if ok {
				headers.Set(strings.TrimSpace(k), strings.TrimSpace(v))
			}
		}
	}

	var wg sync.WaitGroup
	interval := time.Duration(0)
	if *qps > 0 {
		interval = time.Duration(float64(time.Second) / *qps)
	}
	next := time.Now()
	var mu sync.Mutex

	for i := 0; i < *conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if *qps > 0 {
					mu.Lock()
					wait := time.Until(next)
					if wait > 0 {
						next = next.Add(interval)
						mu.Unlock()
						select {
						case <-ctx.Done():
							return
						case <-time.After(wait):
						}
					} else {
						next = next.Add(interval)
						mu.Unlock()
					}
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				start.Add(1)
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, *urls, nil)
				if err != nil {
					finish.Add(1)
					continue
				}
				req.Header = headers.Clone()
				begin := time.Now()
				resp, err := client.Do(req)
				lat := time.Since(begin)
				finish.Add(1)
				if err != nil {
					errCount.Add(1)
					continue
				}
				io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				resp.Body.Close()
				code := fmt.Sprintf("%d", resp.StatusCode)
				v, _ := statuses.LoadOrStore(code, 0)
				statuses.Store(code, v.(int)+1)
				select {
				case latencies <- lat:
				default:
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
	case <-time.After(*dur):
	}
	cancel()
	wg.Wait()
	close(latencies)

	report := finishHTTPReport(start.Load(), finish.Load(), latencies, &statuses, *dur)
	report.Errors = errCount.Load()
	printHTTP(report)
	if *out != "" {
		writeJSON(*out, report)
	}
}

type httpReport struct {
	Mode     string           `json:"mode"`
	URL      string           `json:"url"`
	Duration string           `json:"duration"`
	Started  int64            `json:"started"`
	Finished int64            `json:"finished"`
	QPS      float64          `json:"qps"`
	P50      string           `json:"p50"`
	P90      string           `json:"p90"`
	P99      string           `json:"p99"`
	Min      string           `json:"min"`
	Max      string           `json:"max"`
	Errors   int64            `json:"errors"`
	Statuses map[string]int64 `json:"statuses"`
}

func finishHTTPReport(started, finished int64, latencies <-chan time.Duration, statuses *sync.Map, dur time.Duration) httpReport {
	var vals []time.Duration
	for d := range latencies {
		vals = append(vals, d)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	p := func(q float64) time.Duration {
		if len(vals) == 0 {
			return 0
		}
		return vals[int(math.Ceil(q*float64(len(vals)))-1)]
	}
	codes := map[string]int64{}
	statuses.Range(func(k, v any) bool {
		if ks, ok := k.(string); ok {
			codes[ks] = int64(v.(int))
		}
		return true
	})
	var min, max time.Duration
	if len(vals) > 0 {
		min, max = vals[0], vals[len(vals)-1]
	}
	return httpReport{
		Mode: "http", URL: "", Duration: dur.String(),
		Started: started, Finished: finished,
		QPS: float64(finished) / dur.Seconds(),
		P50: p(0.50).String(), P90: p(0.90).String(), P99: p(0.99).String(),
		Min: min.String(), Max: max.String(), Statuses: codes,
	}
}

func printHTTP(r httpReport) {
	fmt.Printf("HTTP 压测完成\n")
	fmt.Printf("  发起请求: %d  完成请求: %d\n", r.Started, r.Finished)
	fmt.Printf("  QPS: %.1f/s\n", r.QPS)
	fmt.Printf("  P50: %s  P90: %s  P99: %s  min: %s  max: %s\n", r.P50, r.P90, r.P99, r.Min, r.Max)
	fmt.Printf("  状态码分布: %v\n", r.Statuses)
}

// ----------------------------------------------------------------- WS ----

type wsFlags struct {
	url          string
	clients      int
	video        int
	dur          time.Duration
	token        string
	senderUsers  int
	sendInterval time.Duration
	out          string
}

func runWS(ctx context.Context, args []string) {
	fs := flag.NewFlagSet("ws", flag.ExitOnError)
	f := &wsFlags{}
	fs.StringVar(&f.url, "url", "", "ws URL")
	fs.IntVar(&f.clients, "clients", 100, "number of ws clients")
	fs.IntVar(&f.video, "video", 0, "video id")
	fs.DurationVar(&f.dur, "d", 25*time.Second, "duration")
	fs.StringVar(&f.token, "token", "", "optional JWT token")
	fs.IntVar(&f.senderUsers, "sender-users", 0, "generate danmaku with N minted users (needs JWT_SECRET)")
	fs.DurationVar(&f.sendInterval, "send-interval", 200*time.Millisecond, "danmaku send interval")
	fs.StringVar(&f.out, "out", "", "write JSON report to file")
	_ = fs.Parse(args)
	if f.url == "" || f.video == 0 {
		fmt.Fprintln(os.Stderr, "ws: -url and -video are required")
		os.Exit(2)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tokens := make([]string, 0, f.senderUsers)
	if f.senderUsers > 0 {
		secret := os.Getenv("JWT_SECRET")
		mgr, err := jwttoken.NewManager(secret)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ws: JWT_SECRET required when -sender-users > 0:", err)
			os.Exit(2)
		}
		for i := 1; i <= f.senderUsers; i++ {
			access, _, _, err := mgr.IssuePair(uint64(i))
			if err != nil {
				fmt.Fprintln(os.Stderr, "ws: mint token:", err)
				os.Exit(2)
			}
			tokens = append(tokens, access)
		}
	}

	wsURL := f.url
	sep := "?"
	if strings.Contains(wsURL, "?") {
		sep = "&"
	}
	wsURL += fmt.Sprintf("%svideo_id=%d", sep, f.video)
	if f.token != "" {
		wsURL += "&token=" + url.QueryEscape(f.token)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
	}

	var connected atomic.Int64
	var failed atomic.Int64
	var msgs atomic.Int64
	var readTimeouts atomic.Int64
	var readErrs atomic.Int64
	var cleanEnds atomic.Int64
	var sendOK atomic.Int64
	var sendFail atomic.Int64
	var perConn []int64
	var perMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < f.clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, _, err := dialer.Dial(wsURL, nil)
			if err != nil {
				failed.Add(1)
				return
			}
			defer conn.Close()
			connected.Add(1)
			var n int64
			defer func() {
				perMu.Lock()
				perConn = append(perConn, n)
				perMu.Unlock()
			}()
			_ = conn.SetReadDeadline(time.Now().Add(f.dur + 5*time.Second))
			for {
				if ctx.Err() != nil {
					cleanEnds.Add(1)
					return
				}
				if _, _, err := conn.ReadMessage(); err != nil {
					if ctx.Err() != nil {
						cleanEnds.Add(1)
						return
					}
					var ne net.Error
					if errors.As(err, &ne) && ne.Timeout() {
						readTimeouts.Add(1)
					} else {
						readErrs.Add(1)
					}
					return
				}
				n++
				msgs.Add(1)
			}
		}()
	}

	if f.senderUsers > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}
			ticker := time.NewTicker(f.sendInterval)
			defer ticker.Stop()
			i := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				tok := tokens[i%len(tokens)]
				i++
				body := fmt.Sprintf(`{"content":"lt-%d","video_time":1.5,"type":"scroll","color":"#ffffff","font_size":"medium"}`, i)
				req, _ := http.NewRequest(http.MethodPost,
					fmt.Sprintf("http://%s/api/v1/videos/%d/danmaku", hostOf(wsURL), f.video),
					strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+tok)
				resp, err := client.Do(req)
				if err == nil {
					io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
					resp.Body.Close()
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						sendOK.Add(1)
					} else {
						sendFail.Add(1)
					}
				} else {
					sendFail.Add(1)
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
	case <-time.After(f.dur):
	}
	cancel()
	wg.Wait()

	perMu.Lock()
	conns := append([]int64(nil), perConn...)
	perMu.Unlock()
	sort.Slice(conns, func(i, j int) bool { return conns[i] < conns[j] })
	perQ := func(q float64) int64 {
		if len(conns) == 0 {
			return 0
		}
		return conns[int(math.Ceil(q*float64(len(conns)))-1)]
	}
	var perAvg float64
	if len(conns) > 0 {
		var sum int64
		for _, v := range conns {
			sum += v
		}
		perAvg = float64(sum) / float64(len(conns))
	}

	report := wsReport{
		Mode: "ws", URL: wsURL, Duration: f.dur.String(),
		Clients:      f.clients,
		Connected:    connected.Load(),
		Failed:       failed.Load(),
		Messages:     msgs.Load(),
		MsgPerSec:    float64(msgs.Load()) / f.dur.Seconds(),
		ReadTimeouts: readTimeouts.Load(),
		ReadErrors:   readErrs.Load(),
		CleanEnds:    cleanEnds.Load(),
		PerConnAvg:   perAvg,
		PerConnP50:   perQ(0.50),
		PerConnP99:   perQ(0.99),
		SenderOK:     sendOK.Load(),
		SenderFail:   sendFail.Load(),
	}
	printWS(report)
	if f.out != "" {
		writeJSON(f.out, report)
	}
}

type wsReport struct {
	Mode         string  `json:"mode"`
	URL          string  `json:"url"`
	Duration     string  `json:"duration"`
	Clients      int     `json:"clients"`
	Connected    int64   `json:"connected"`
	Failed       int64   `json:"failed"`
	Messages     int64   `json:"messages"`
	MsgPerSec    float64 `json:"msg_per_sec"`
	ReadTimeouts int64   `json:"read_timeouts"`
	ReadErrors   int64   `json:"read_errors"`
	CleanEnds    int64   `json:"clean_ends"`
	PerConnAvg   float64 `json:"per_conn_avg"`
	PerConnP50   int64   `json:"per_conn_p50"`
	PerConnP99   int64   `json:"per_conn_p99"`
	SenderOK     int64   `json:"sender_ok"`
	SenderFail   int64   `json:"sender_fail"`
}

func printWS(r wsReport) {
	fmt.Printf("WS 压测完成\n")
	fmt.Printf("  客户端: %d 连接成功: %d 失败: %d\n", r.Clients, r.Connected, r.Failed)
	fmt.Printf("  收到弹幕: %d (%.1f msg/s) 每连接平均 %.1f 条 (P50 %d / P99 %d)\n",
		r.Messages, r.MsgPerSec, r.PerConnAvg, r.PerConnP50, r.PerConnP99)
	fmt.Printf("  读错误: 意外 %d / 超时 %d / 测试正常结束 %d\n",
		r.ReadErrors, r.ReadTimeouts, r.CleanEnds)
	if r.SenderOK+r.SenderFail > 0 {
		fmt.Printf("  弹幕发送: 成功 %d / 失败 %d\n", r.SenderOK, r.SenderFail)
	}
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}

func writeJSON(path string, v any) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	_ = w.Flush()
	fmt.Println("report written:", path)
}
