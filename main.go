package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

const (
	workersCount = 2
)

type countInfo struct {
	URL   string
	Count int
}

func main() {
	ctx := context.Background()
	urlChan := produce(ctx, bufio.NewReader(os.Stdin))
	resultChan := consume(ctx, urlChan)
	for res := range resultChan {
		fmt.Printf("count for %s: %d\n", res.URL, res.Count)
	}
}

func produce(ctx context.Context, reader *bufio.Reader) chan string {
	urlChan := make(chan string, workersCount)
	go func() {
		defer close(urlChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			url, _, err := reader.ReadLine()
			if err == io.EOF {
				return
			}
			urlChan <- string(url)
		}
	}()
	return urlChan
}

func consume(ctx context.Context, urlChan <-chan string) chan countInfo {
	resultChan := make(chan countInfo, workersCount)
	go func() {
		var wg sync.WaitGroup
		defer close(resultChan)

		throttle := make(chan struct{}, workersCount)
		for url := range urlChan {
			select {
			case <-ctx.Done():
				return
			default:
			}
			throttle <- struct{}{}
			wg.Add(1)
			go func(url string) {
				count, err := getCount(ctx, url)
				if err == nil {
					resultChan <- countInfo{URL: url, Count: count}
				}
				wg.Done()
				<-throttle
			}(url)
		}
		wg.Wait()
	}()
	return resultChan
}

func getCount(ctx context.Context, url string) (int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	count := bytes.Count(body, []byte("Go"))
	return count, nil
}