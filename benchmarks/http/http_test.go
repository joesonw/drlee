package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

func init() {
}

func runClient(b *testing.B, port, concurrency int) {
	ch := make(chan struct{}, 100)
	wg := sync.WaitGroup{}

	for i := 0; i < concurrency; i++ {
		go func() {
			client := http.Client{}
			for range ch {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d", port), nil)
				if err != nil {
					b.Error(err)
				}
				res, err := client.Do(req)
				if err != nil {
					b.Error(err)
				}
				_, err = ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					b.Error(err)
				}
				wg.Done()
			}
		}()
	}

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		ch <- struct{}{}
	}
	wg.Wait()
}
