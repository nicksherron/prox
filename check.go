package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/icrowley/fake"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
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
	//Speed string `json:"speed"`
}

var (
	wgC         sync.WaitGroup
	jsonProxies []string
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
	req, err := http.NewRequest("GET", "http://httpbin.org/get?show_env", nil)
	req.Header.Set("Accept", "application/json")
	check(err)
	curl := &http.Client{}
	resp, err := curl.Do(req)
	check(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	var jsonBody HttpBin
	err = json.Unmarshal(body, &jsonBody)
	return jsonBody.Headers.XRealIP
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
		TLSHandshakeTimeout: 60 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	check(err)

	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
		Jar:       jar,
	}
	req, err := http.NewRequest("GET", testUrl, nil)
	check(err)
	req.Header.Set("User-Agent", fake.UserAgent())
	resp, err := client.Do(req)
	defer resp.Body.Close()
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
			jsonBody.Elite = true
			if realIp == jsonBody.Headers.XRealIP {
				jsonBody.Transparent = true
				if noTransparent {
					return
				}
			}
			for _, ips := range strings.Fields(strings.ReplaceAll(jsonBody.Origin, `,`, ``)) {
				if ips == realIp {
					jsonBody.Elite = false
				}
			}
			if eliteOnly && !jsonBody.Elite {
				return
			}

			var results interface{}
			if !showRequest {
				out := make(map[string]interface{})
				out["proxy"] = jsonBody.Proxy
				out["transparent"] = jsonBody.Transparent
				out["elite"] = jsonBody.Elite
				results = out
			} else {
				results = &jsonBody
			}

			b, err := json.MarshalIndent(results, ``, `   `)
			check(err)
			atomic.AddUint64(&goodCount, 1)
			mutex.Lock()
			jsonProxies = append(jsonProxies, string(b))
			good = append(good, addr)
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
