package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Timeout int `short:"t" long:"timeout" description:"Sets the timeout for all HTTP requests in seconds" default:"15"`
}

type Result struct {
	URL            string
	Status         string
	ResponseHeader http.Header
	ResponseBody   []byte
	Error          error
}

func main() {
	var opts Options

	args, err := flags.Parse(&opts)
	if err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
	
		log.Fatalf("parse flags: %v", err)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "at least one URL is required")
		os.Exit(1)
	}

	if opts.Timeout <= 0 {
		fmt.Fprintln(os.Stderr, "timeout must be greater than zero")
		os.Exit(1)
	}

	var timeout = time.Duration(opts.Timeout) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultCh := make(chan Result, 1)

	urlsNum := len(args)
	errNum := 0
	errDone := make(chan struct{}, 1)

	for _, url := range args {
		go func(url string) {
			result := RunGetRequest(ctx, url)
			if result.Error != nil {
				log.Printf("[ERROR] - [%s] - %s\n", result.URL, result.Error.Error())
				errNum++
				if errNum == urlsNum {
					errDone <- struct{}{}
				}
			} else {
				select {
				case resultCh <- result:
				case <-ctx.Done():
				}
			}
		}(url)
	}

	select {
	case res := <-resultCh:
		fmt.Println(res.Info())
		return
	case <-ctx.Done():
		log.Printf("[Timeout] - none of the servers responded within %v", timeout)
		os.Exit(228)
	case <-errDone:
		log.Fatalln("all requests ended with an error")
	}
}

func (r Result) Info() string {
	return fmt.Sprintf("[%s]\n%s\n\u001B[1mResponse Header\033[0m:\n%v\n\u001B[1mResponse Body\033[0m:\n%v",
		r.URL,
		r.Status,
		r.ResponseHeader,
		string(r.ResponseBody),
	)
}

func RunGetRequest(ctx context.Context, url string) Result {
	result := Result{URL: url}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		result.Error = fmt.Errorf("create request: %w", err)
		return result
	}

	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("do request: %w", err)
		return result

	}
	defer resp.Body.Close()

	result.Status = resp.Proto + " " + resp.Status
	result.ResponseHeader = resp.Header

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("read from response body: %w", err)
		return result
	}

	result.ResponseBody = respBody

	return result
}
