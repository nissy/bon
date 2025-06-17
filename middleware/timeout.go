package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// timeoutWriter は ResponseWriter をラップして書き込み状態を追跡
type timeoutWriter struct {
	http.ResponseWriter
	written bool
	mu      sync.Mutex
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.written {
		tw.written = true
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.written {
		tw.written = true
		tw.ResponseWriter.WriteHeader(http.StatusOK)
	}
	return tw.ResponseWriter.Write(b)
}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			tw := &timeoutWriter{ResponseWriter: w}
			done := make(chan struct{})
			panicChan := make(chan interface{}, 1)
			
			go func() {
				defer func() {
					if p := recover(); p != nil {
						// パニックをキャプチャして転送
						select {
						case panicChan <- p:
						case <-ctx.Done():
							// タイムアウト済みの場合は無視
						}
					}
					close(done)
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			select {
			case <-done:
				// 正常に完了
			case p := <-panicChan:
				// パニックを再発生
				panic(p)
			case <-ctx.Done():
				// タイムアウト発生
				tw.mu.Lock()
				defer tw.mu.Unlock()
				if !tw.written {
					tw.written = true
					tw.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
					// タイムアウトメッセージを書き込み（エラーは無視）
					_, _ = tw.ResponseWriter.Write([]byte("Gateway Timeout"))
				}
				
				// goroutineの完了を待つ（リークを防ぐ）
				<-done
			}
		}

		return http.HandlerFunc(fn)
	}
}
