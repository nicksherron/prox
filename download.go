package main

import (
	"encoding/base64"
	"fmt"
	"github.com/icrowley/fake"
	cuckoo "github.com/seiflotfy/cuckoofilter"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	proxies []string
	wgD     sync.WaitGroup
	// Matches ip and port
	reProxy       = regexp.MustCompile(`(?ms)(?P<ip>(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))(?:.*?(?:(?:(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))|(?P<port>\d{2,5})))`)
	templateProxy = "http://${ip}:${port}\n"
)

func findSubmatchRange(regex *regexp.Regexp, str string) []string {
	var matched []string
	for _, matches := range regex.FindAllString(str, -1) {
		match := regex.FindStringSubmatch(matches)[1]
		matched = append(matched, match)
	}
	return matched
}

func findAllTemplate(pattern *regexp.Regexp, html string, template string) []string {
	var (
		results []string
		result  []byte
	)

	for _, matches := range pattern.FindAllStringSubmatchIndex(html, -1) {
		result = pattern.ExpandString(result, template, html, matches)
	}
	for _, newLine := range strings.Split(string(result), "\n") {
		results = append(results, newLine)
	}
	return results
}

func get(u string) (string, error) {
	client := &http.Client{
		Timeout: 20 * time.Second,
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Forwarded-For", fake.IPv4())
	req.Header.Set("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36`)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil

}

func counter(quit chan int) {
	for {
		select {
		case <-quit:
			return
		default:
			n := uint(len(proxies))
			cf := cuckoo.NewFilter(n)
			i := 0
			for _, v := range proxies {
				if cf.InsertUnique([]byte(v)) {
					i++
				}
			}
			fmt.Fprintf(os.Stderr, "\rFound %d proxies", i)
			cf.Reset()
			time.Sleep(1 * time.Second)
		}
	}
}

func downloadProxies() []string {
	wgD.Add(13)
	// freeproxylists.com  tested:  1/16/20 found: 1051
	go func() {
		defer wgD.Done()
		var (
			w       sync.WaitGroup
			fplReID = regexp.MustCompile(`(?m)href\s*=\s*['"](?P<type>[^'"]*)/(?P<id>\d{10})[^'"]*['"]`)
			fplUrls = []string{
				"http://www.freeproxylists.com/anonymous.html",
				"http://www.freeproxylists.com/elite.html",
			}
		)
		for _, u := range fplUrls {
			body, err := get(u)
			if err != nil {
				continue
			}
			template := "http://www.freeproxylists.com/load_${type}_${id}.html\n"
			matches := findAllTemplate(fplReID, body, template)
			for _, match := range matches {
				w.Add(1)
				go func(endpoint string) {
					defer w.Done()
					ipList, err := get(endpoint)
					if err != nil {
						return
					}
					matched := findAllTemplate(reProxy, ipList, templateProxy)
					for _, proxy := range matched {
						mutex.Lock()
						proxies = append(proxies, proxy)
						mutex.Unlock()
					}
				}(match)
			}
			w.Wait()
		}
	}()
	// webanetlabs.net  tested:  1/16/20 found: 2871
	go func() {
		defer wgD.Done()
		var (
			w   sync.WaitGroup
			re  = regexp.MustCompile(`(?m)href\s*=\s*['"]([^'"]*proxylist_at_[^'"]*)['"]`)
			url = "https://webanetlabs.net/publ/24"
		)
		body, err := get(url)
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			w.Add(1)
			go func(page string) {
				defer w.Done()
				// https://webanetlabs.net/freeproxyweb/proxylist_at_02.11.2019.txt
				u := "https://webanetlabs.net" + page
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(href)
		}
		w.Wait()
	}()
	// checkerproxy.net  tested:  1/16/20 found: 0
	go func() {
		defer wgD.Done()
		var (
			w   sync.WaitGroup
			re  = regexp.MustCompile(`(?m)href\s*=\s*['"](/archive/\d{4}-\d{2}-\d{2})['"]`)
			url = "https://checkerproxy.net/"
		)
		body, err := get(url)
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			w.Add(1)
			go func(endpoint string) {
				defer w.Done()
				u := "https://checkerproxy.net/api" + endpoint
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(href)
		}
		w.Wait()
	}()
	// proxy-list.org tested:  1/16/20 found: 83
	go func() {
		defer wgD.Done()
		var (
			ipBase64 = regexp.MustCompile(`Proxy\('([\w=]+)'\)`)
			w        sync.WaitGroup
		)
		w.Add(10)
		for i := 1; i < 11; i++ {
			go func(page int) {
				defer w.Done()
				u := fmt.Sprintf("http://proxy-list.org/english/index.php?p=%v", page)
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, match := range findSubmatchRange(ipBase64, ipList) {
					decoded, err := base64.StdEncoding.DecodeString(match)
					check(err)
					ip := fmt.Sprintf("http://%v", string(decoded))
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(i)
		}
		w.Wait()
	}()
	// aliveproxy.com 1/16/20 found: 367
	go func() {
		defer wgD.Done()
		var (
			w        sync.WaitGroup
			suffixes = []string{
				//"socks5-list",
				"high-anonymity-proxy-list",
				"anonymous-proxy-list",
				"fastest-proxies",
				"us-proxy-list",
				"gb-proxy-list",
				"fr-proxy-list",
				"de-proxy-list",
				"jp-proxy-list",
				"ca-proxy-list",
				"ru-proxy-list",
				"proxy-list-port-80",
				"proxy-list-port-81",
				"proxy-list-port-3128",
				"proxy-list-port-8000",
				"proxy-list-port-8080",
			}
		)

		for _, href := range suffixes {
			w.Add(1)
			u := fmt.Sprintf("http://www.aliveproxy.com/%v/", href)
			go func(endpoint string) {
				defer w.Done()
				ipList, err := get(endpoint)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(u)
		}
		w.Wait()
	}()
	//proxylist.me tested:  1/16/20 found: 23118/24000
	go func() {
		defer wgD.Done()
		var (
			w    sync.WaitGroup
			ints []int
			re   = regexp.MustCompile(`(?m)href\s*=\s*['"][^'"]*/?page=(\d+)['"]`)
			url  = "https://proxylist.me/"
		)
		body, err := get(url)
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			i, err := strconv.Atoi(href)
			if err != nil {
				continue
			}
			ints = append(ints, i)
		}
		if len(ints) == 0 {
			return
		}
		sort.Ints(ints)
		largest := ints[len(ints)-1]
		largest++
		counter := 0
		for i := 1; i < largest; i++ {
			w.Add(1)
			counter++
			go func(page int) {
				defer w.Done()
				u := fmt.Sprintf("https://proxylist.me/?page=%v", page)
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(i)
			// only 25 goroutines at a time. (1170 urls to get)
			if counter >= 25 {
				w.Wait()
				counter = 0
			}
		}
		w.Wait()
	}()
	// proxy-list.download tested:  1/16/20 found: 0
	go func() {
		defer wgD.Done()
		body, err := get("https://www.proxy-list.download/api/v1/get?type=http")
		if err != nil {
			return
		}
		for _, ip := range findAllTemplate(reProxy, body, templateProxy) {
			mutex.Lock()
			proxies = append(proxies, ip)
			mutex.Unlock()
		}
	}()
	// blogspot.com tested:  1/16/20 found: 10312
	go func() {
		defer wgD.Done()
		var (
			w       sync.WaitGroup
			re      = regexp.MustCompile(`(?m)<a href\s*=\s*['"]([^'"]*\.\w+/\d{4}/\d{2}/[^'"#]*)['"]>`)
			domains = []string{
				"sslproxies24.blogspot.com",
				"proxyserverlist-24.blogspot.com",
				"freeschoolproxy.blogspot.com",
				"googleproxies24.blogspot.com",
			}
		)
		for _, domain := range domains {
			w.Add(1)
			go func(endpoint string) {
				u := fmt.Sprintf("http://%v/", endpoint)
				defer w.Done()
				urlList, err := get(u)
				if err != nil {
					return
				}
				for _, href := range findSubmatchRange(re, urlList) {
					w.Add(1)
					go func(endpoint string) {
						ipList, err := get(endpoint)
						if err != nil {
							return
						}
						defer w.Done()
						for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
							mutex.Lock()
							proxies = append(proxies, ip)
							mutex.Unlock()
						}
					}(href)
				}
			}(domain)
		}
		w.Wait()
	}()
	// prox.com tested:  1/16/20 found: 25
	go func() {
		defer wgD.Done()
		var (
			w   sync.WaitGroup
			re  = regexp.MustCompile(`href\s*=\s*['"]([^'"]?proxy_list_high_anonymous_[^'"]*)['"]`)
			url = "http://www.proxz.com/proxy_list_high_anonymous_0.html"
		)
		urlList, err := get(url)
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, urlList) {
			w.Add(1)
			go func(endpoint string) {
				u := fmt.Sprintf("http://www.proxz.com/%v", endpoint)
				ipList, err := get(u)
				if err != nil {
					return
				}
				defer w.Done()
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(href)
		}
		w.Wait()
	}()
	// my-proxy.com tested:  1/16/20 found: 1026
	go func() {
		defer wgD.Done()
		var (
			w   sync.WaitGroup
			re  = regexp.MustCompile(`(?m)href\s*=\s*['"]([^'"]?free-[^'"]*)['"]`)
			url = "https://www.my-proxy.com/free-proxy-list.html"
		)

		urlList, err := get(url)
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, urlList) {
			w.Add(1)
			go func(endpoint string) {
				u := fmt.Sprintf("https://www.my-proxy.com/%v", endpoint)
				defer w.Done()
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(href)
		}
		w.Wait()
	}()
	// list.proxylistplus.com tested:  1/16/20 found: 0
	go func() {
		defer wgD.Done()
		var (
			w    sync.WaitGroup
			re   = regexp.MustCompile(`(?ms)<td>(?P<ip>(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))(?:.*?(?:(?:(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))|(?P<port>\d{2,5})))</td>`)
			urls = []string{
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-1",
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-2",
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-3",
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-4",
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-5",
				"https://list.proxylistplus.com/Fresh-HTTP-Proxy-List-6",
				"https://list.proxylistplus.com/ssl-List-1",
				"https://list.proxylistplus.com/ssl-List-2",
			}
		)
		for _, url := range urls {
			w.Add(1)
			u := url
			go func(endpoint string) {
				defer w.Done()
				ipList, err := get(endpoint)
				if err != nil {
					return
				}
				for _, ip := range findAllTemplate(re, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(u)
		}
		w.Wait()
	}()
	// xseo.in tested:  1/16/20 found: 0
	go func() {
		defer wgD.Done()
		var (
			url = "http://xseo.in/freeproxy"
		)
		ipList, err := get(url)
		if err != nil {
			return
		}
		for _, ip := range findAllTemplate(reProxy, ipList, templateProxy) {
			mutex.Lock()
			proxies = append(proxies, ip)
			mutex.Unlock()
		}
	}()
	// free-proxy.cz tested:  1/16/20 found: 800
	go func() {
		defer wgD.Done()
		var (
			w              sync.WaitGroup
			re             = regexp.MustCompile(`(?U)document.write\(Base64.decode\("(?P<base64>(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?)"\)\).*(?P<port>\d{2,5})</span`)
			templateBase64 = "${base64}:${port}\n"
			urls           []string
			baseUrls       = []string{
				"http://free-proxy.cz/en/proxylist/main",
				"http://free-proxy.cz/en/proxylist/country/all/http/ping/all",
				"http://free-proxy.cz/en/proxylist/country/all/https/ping/all",
				"http://free-proxy.cz/en/proxylist/country/all/http/uptime/level1",
				"http://free-proxy.cz/en/proxylist/country/all/https/uptime/level1",
				"http://free-proxy.cz/en/proxylist/country/all/http/uptime/level2",
				"http://free-proxy.cz/en/proxylist/country/all/https/uptime/level2",
			}
		)
		for _, url := range baseUrls {
			urls = append(urls, url)
			for i := 1; i < 7; i++ {
				u := fmt.Sprintf("%v/%v", url, i)
				urls = append(urls, u)
			}
		}
		for _, url := range urls {
			w.Add(1)
			go func(endpoint string) {
				defer w.Done()
				ipList, err := get(endpoint)
				if err != nil {
					return
				}
				if len(ipList) < 500 {
					return
				}
				for _, encodedIp := range findAllTemplate(re, ipList, templateBase64) {
					split := strings.Split(encodedIp, `:`)
					if len(split) < 2 {
						continue
					}
					proxyIp, err := base64.StdEncoding.DecodeString(split[0])
					if err != nil {
						continue
					}
					ip := fmt.Sprintf("http://%v:%v", string(proxyIp), split[1])
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}(url)
		}
		w.Wait()
	}()

	quit := make(chan int)
	go counter(quit)
	if deadline != 0 {
		time.Sleep(time.Duration(deadline) * time.Second)
	}else {
		wgD.Wait()
	}
	quit <- 0

	n := uint(len(proxies))
	cf := cuckoo.NewFilter(n)
	var unique []string
	for _, v := range proxies {
		if cf.InsertUnique([]byte(v)) {
			unique = append(unique, v)
		}
	}
	// clear filter and proxies
	cf.Reset()
	proxies = nil
	fmt.Fprintf(os.Stderr, "\rFound %d proxies\n", len(unique))
	fmt.Fprintln(os.Stderr, "\nStarting test ...")
	return unique
}
