package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
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

	urlsNum := len(args)

	resultsCh := make(chan Result, urlsNum)

	for _, url := range args {
		go func(url string) {
			result := RunGetRequest(ctx, url)

			select {
			case resultsCh <- result:
			case <-ctx.Done():
			}
		}(url)
	}

	for range urlsNum {
		select {
		case res := <-resultsCh:
			if res.Error != nil {
				log.Printf("[ERROR] - [%s] - %s\n", res.URL, res.Error.Error())
				continue
			}

			info, err := res.Info()
			if err != nil {
				log.Println(err.Error())
				return
			}

			fmt.Println(info)

			return

		case <-ctx.Done():
			log.Printf("[Timeout] - none of the servers responded within %v", timeout)
			os.Exit(228)
		}
	}

	log.Fatalln("all requests ended with an error")
}

func (r Result) Info() (string, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "[%s]\n%s\n\u001B[1mResponse Header\033[0m:\n", r.URL, r.Status)

	if err := r.ResponseHeader.Write(&b); err != nil {
		return "", fmt.Errorf("write response headers: %w", err)
	}

	fmt.Fprintf(&b, "\n\u001B[1mResponse Body\033[0m:\n%s", r.ResponseBody)

	return b.String(), nil
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
