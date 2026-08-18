package videoclient

import (
	"context"
	"crypto/tls"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"
)

type qnaNetworkTrace struct {
	mu sync.Mutex

	dnsStart         time.Time
	dnsDone          time.Time
	connectStart     time.Time
	connectDone      time.Time
	tlsStart         time.Time
	tlsDone          time.Time
	firstByte        time.Time
	remoteAddr       string
	tlsVersion       uint16
	negotiated       string
	connectionReused bool
	connectionIdle   bool
}

type qnaNetworkSnapshot struct {
	dnsMS            int64
	connectMS        int64
	tlsMS            int64
	timeToFirstByte  int64
	remoteAddr       string
	tlsVersion       string
	protocol         string
	connectionReused bool
	connectionIdle   bool
}

func newQNARequestTrace() *qnaNetworkTrace {
	return &qnaNetworkTrace{}
}

func (t *qnaNetworkTrace) clientTrace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			t.mu.Lock()
			if t.dnsStart.IsZero() {
				t.dnsStart = time.Now()
			}
			t.mu.Unlock()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			t.mu.Lock()
			t.dnsDone = time.Now()
			t.mu.Unlock()
		},
		ConnectStart: func(_, addr string) {
			t.mu.Lock()
			if t.connectStart.IsZero() {
				t.connectStart = time.Now()
			}
			t.remoteAddr = addr
			t.mu.Unlock()
		},
		ConnectDone: func(_, addr string, _ error) {
			t.mu.Lock()
			t.connectDone = time.Now()
			if addr != "" {
				t.remoteAddr = addr
			}
			t.mu.Unlock()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Conn == nil {
				return
			}
			t.mu.Lock()
			t.remoteAddr = info.Conn.RemoteAddr().String()
			t.connectionReused = info.Reused
			t.connectionIdle = info.WasIdle
			t.mu.Unlock()
		},
		TLSHandshakeStart: func() {
			t.mu.Lock()
			t.tlsStart = time.Now()
			t.mu.Unlock()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, _ error) {
			t.mu.Lock()
			t.tlsDone = time.Now()
			t.tlsVersion = state.Version
			t.negotiated = state.NegotiatedProtocol
			t.mu.Unlock()
		},
		GotFirstResponseByte: func() {
			t.mu.Lock()
			t.firstByte = time.Now()
			t.mu.Unlock()
		},
	}
}

func (t *qnaNetworkTrace) snapshot(startedAt time.Time, protocol string) qnaNetworkSnapshot {
	if t == nil {
		return qnaNetworkSnapshot{protocol: protocol}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return qnaNetworkSnapshot{
		dnsMS:            durationMilliseconds(t.dnsStart, t.dnsDone),
		connectMS:        durationMilliseconds(t.connectStart, t.connectDone),
		tlsMS:            durationMilliseconds(t.tlsStart, t.tlsDone),
		timeToFirstByte:  durationMilliseconds(startedAt, t.firstByte),
		remoteAddr:       t.remoteAddr,
		tlsVersion:       tlsVersionName(t.tlsVersion),
		protocol:         firstNonEmptyTrace(protocol, t.negotiated),
		connectionReused: t.connectionReused,
		connectionIdle:   t.connectionIdle,
	}
}

func durationMilliseconds(start, end time.Time) int64 {
	if start.IsZero() {
		return 0
	}
	if end.IsZero() {
		end = time.Now()
	}
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return ""
	}
}

func firstNonEmptyTrace(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func withQNAHTTPTrace(ctx context.Context, trace *qnaNetworkTrace) context.Context {
	if trace == nil {
		return ctx
	}
	return httptrace.WithClientTrace(ctx, trace.clientTrace())
}
