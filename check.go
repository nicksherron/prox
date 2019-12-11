package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/icrowley/fake"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type HttpBin struct {
	Headers struct {
		Accept          string `json:"Accept"`
		Host            string `json:"Host"`
		UserAgent       string `json:"User-Agent"`
		Via             string `json:"Via"`
		XForwardedFor   string `json:"X-Forwarded-For"`
		XForwardedPort  string `json:"X-Forwarded-Port"`
		XForwardedProto string `json:"X-Forwarded-Proto"`
		XProxyID        string `json:"X-Proxy-Id"`
		XRealIP         string `json:"X-Real-Ip"`
	} `json:"headers"`
	Origin      string `json:"origin"`
	URL         string `json:"url"`
	Proxy       string `json:"proxy"`
	Transparent bool   `json:"transparent"`
	Elite       bool   `json:"elite"`
}

var (
	wgC         sync.WaitGroup
	good        []string
	mutex       = &sync.Mutex{}
	goodCount   uint64
	badCount    uint64
	toCount     uint64
	reqCount    uint64
	barTemplate = `{{string . "message"}}{{counters . }} {{bar . }} {{percent . }} {{speed . "%s req/sec" }}`
	realIp      string
)

func hostIp() string {
	req, err := http.NewRequest("GET", "https://ipinfo.io/ip", nil)
	check(err)
	curl := &http.Client{}
	resp, err := curl.Do(req)
	check(err)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return strings.ReplaceAll(string(body), "\n", "")
}

func proxyCheck(addr string, bar *pb.ProgressBar) {
	defer func() {
		bar.Increment()
		wgC.Done()
		if r := recover(); r != nil {
			atomic.AddUint64(&badCount, 1)
			return
		}
	}()
	atomic.AddUint64(&reqCount, 1)
	prox := strings.Split(addr, "*")[0]
	proxyUrl, err := url.Parse(prox)
	check(err)
	tr := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
		Dial: (&net.Dialer{
			Timeout: 60 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 60 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	req, err := http.NewRequest("GET", testUrl, nil)
	check(err)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Forwarded-For", fake.IPv4())
	req.Header.Set("User-Agent", fake.UserAgent())
	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "Client.Timeout") {
			atomic.AddUint64(&toCount, 1)
		} else {
			check(err)
		}
	}
	if limit > 0 {
		if atomic.CompareAndSwapUint64(&goodCount, limit, limit) {
			return
		}
	}
	if resp.StatusCode == 200 {
		if testUrl == "http://httpbin.org/get?show_env" {
			body, err := ioutil.ReadAll(resp.Body)
			check(err)
			var jsonBody HttpBin
			err = json.Unmarshal(body, &jsonBody)
			check(err)
			jsonBody.Proxy = addr
			jsonBody.Transparent = false
			jsonBody.Elite = false
			if !strings.Contains(realIp, jsonBody.Headers.XRealIP) {
				jsonBody.Transparent = true
				if !strings.Contains(realIp,jsonBody.Origin){
					jsonBody.Elite = true
				}
			}
			b, err := json.MarshalIndent(&jsonBody, ``, `   `)
			check(err)
			atomic.AddUint64(&goodCount, 1)
			mutex.Lock()
			good = append(good, string(b))
			mutex.Unlock()
		} else {
			atomic.AddUint64(&goodCount, 1)
			mutex.Lock()
			good = append(good, addr)
			mutex.Unlock()
		}
	} else {
		atomic.AddUint64(&badCount, 1)
	}
}

func checkInit(addresses []string) {
	realIp = hostIp()
	_, _ = fmt.Fprintf(os.Stderr,`Host ip identified as %v\n`,realIp)
	//log.SetOutput(nil)
	start := time.Now()
	counter := 0
	bar := pb.ProgressBarTemplate(barTemplate).Start(len(addresses)).SetMaxWidth(80)
	bar.Set("message", "Testing proxies\t")
	var wgLoop sync.WaitGroup
	wgLoop.Add(1)
	go func() {
		defer wgLoop.Done()
		for _, addr := range addresses {
			wgC.Add(1)
			go proxyCheck(addr, bar)
			counter++
			if limit > 0 {
				if atomic.CompareAndSwapUint64(&goodCount, limit, limit) {
					return
				}
			}
			if counter >= workers {
				wgC.Wait()
				counter = 0
			}
		}
		if limit == 0 {
			wgC.Wait()
		}
	}()
	wgLoop.Wait()
	bar.Finish()
	done := time.Since(start)
	_, _ = fmt.Fprintf(os.Stderr,
		"\nGood:\t%v\tBad:\t%v\tTimed out:\t%v\tTook:\t%v\t\n\n",
		goodCount, badCount, toCount, done)
}
