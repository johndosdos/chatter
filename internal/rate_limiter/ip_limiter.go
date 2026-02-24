package ratelimiter

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	viewAuth "github.com/johndosdos/chatter/components/auth"
	"golang.org/x/time/rate"
)

type CleanupOpts struct {
	TTL      time.Duration
	Interval time.Duration
}

type ipAddr string

type IPRateLimiter struct {
	limiters map[ipAddr]*rate.Limiter
	lastSeen map[ipAddr]time.Time
	mu       sync.Mutex
	Cancel   context.CancelFunc
	rate     rate.Limit
	burst    int
	CleanupOpts
}

func NewIPRateLimiter(requests int, window time.Duration, cleanupOpts CleanupOpts) *IPRateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &IPRateLimiter{
		limiters:    make(map[ipAddr]*rate.Limiter),
		lastSeen:    make(map[ipAddr]time.Time),
		Cancel:      cancel,
		mu:          sync.Mutex{},
		rate:        rate.Every(window / time.Duration(requests)),
		burst:       requests,
		CleanupOpts: cleanupOpts,
	}

	go rl.cleanup(ctx)

	return rl
}

func (rl *IPRateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(rl.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()

			for ip, ls := range rl.lastSeen {
				if time.Since(ls) > rl.TTL {
					delete(rl.limiters, ip)
					delete(rl.lastSeen, ip)
				}
			}

			rl.mu.Unlock()
		}
	}
}

func (rl *IPRateLimiter) GetClientIP(r *http.Request) ipAddr {
	xff := http.Header.Get(r.Header, "X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return ipAddr(strings.TrimSpace(ips[len(ips)-1]))
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		//nolint:gosec
		slog.Warn("invalid argument for net.SplitHostPort()",
			slog.String("remote_addr", r.RemoteAddr))
		return ipAddr(r.RemoteAddr)
	}

	return ipAddr(host)
}

func (rl *IPRateLimiter) Allow(ip ipAddr) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.limiters[ip]
	if !ok {
		bucket = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = bucket
	}

	rl.lastSeen[ip] = time.Now()
	return bucket.Allow()
}

func (rl *IPRateLimiter) Middleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.GetClientIP(r)

		if !rl.Allow(ip) {
			slog.WarnContext(r.Context(), "rate limit exceeded",
				"ip", ip,
				"path", r.URL.Path,
				"method", r.Method)

			if r.Header.Get("HX-Request") == "true" {
				err := viewAuth.ErrorMsgAuth("Too many requests. Try again later.").Render(r.Context(), w)
				if err != nil {
					slog.ErrorContext(r.Context(), "failed to render error component",
						"error", err,
						"ip", ip)
				}
				return
			}

			http.Error(w, "Too many requests. Try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
