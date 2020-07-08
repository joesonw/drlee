package http

import (
	"net"
	"net/http"
	"testing"
	"time"
)

func runPlainTest(b *testing.B, size int, timeout bool, concurrency int) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		b.Error(err)
	}
	addr := lis.Addr().(*net.TCPAddr)
	for i := 0; i < size; i++ {
		server := &http.Server{}
		server.SetKeepAlivesEnabled(false)
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body.Close()
			if timeout {
				time.Sleep(time.Millisecond * 100)
			}
			w.Write([]byte("OK"))
		})
		go func() {
			server.Serve(lis)
		}()
	}

	time.Sleep(time.Millisecond * 100)
	runClient(b, addr.Port, concurrency)
	lis.Close()
}

func BenchmarkPlain(b *testing.B) {
	runPlainTest(b, 1, false, 1)
}

func BenchmarkPlainParallel4(b *testing.B) {
	runPlainTest(b, 4, false, 1)
}

func BenchmarkPlainSleep(b *testing.B) {
	runPlainTest(b, 1, true, 1)
}

func BenchmarkPlainSleepParallel4(b *testing.B) {
	runPlainTest(b, 4, true, 1)
}

func BenchmarkPlainConcurrent4(b *testing.B) {
	runPlainTest(b, 1, false, 4)
}

func BenchmarkPlainParallel4Concurrent4(b *testing.B) {
	runPlainTest(b, 4, false, 4)
}

func BenchmarkPlainSleepConcurrent4(b *testing.B) {
	runPlainTest(b, 1, true, 4)
}

func BenchmarkPlainSleepParallel4Concurrent4(b *testing.B) {
	runPlainTest(b, 4, true, 4)
}
