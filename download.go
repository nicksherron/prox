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
	wgD sync.WaitGroup
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

func FindAllTemplate(pattern *regexp.Regexp, html string, template string) []string {
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
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Forwarded-For", fake.IPv4())
	req.Header.Set("User-Agent", fake.UserAgent())
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
			fmt.Fprintf(os.Stderr,"\rFound %d proxies", i)
			cf.Reset()
			time.Sleep(1 * time.Second)
		}
	}
}

func downloadProxies() []string {
	wgD.Add(7)
	// http://www.freeproxylists.com
	go func() {
		defer wgD.Done()
		var (
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
			matches := FindAllTemplate(fplReID, body, template)
			for _, match := range matches {
				wgD.Add(1)
				go func() {
					defer wgD.Done()
					ipList, err := get(match)
					if err != nil {
						return
					}
					matched := FindAllTemplate(reProxy, ipList, templateProxy)
					for _, proxy := range matched {
						mutex.Lock()
						proxies = append(proxies, proxy)
						mutex.Unlock()
					}

				}()
			}
		}
	}()
	// https://webanetlabs.net
	go func() {
		defer wgD.Done()
		var (
			re = regexp.MustCompile(`(?m)href\s*=\s*['"]([^'"]*proxylist_at_[^'"]*)['"]`)
		)
		body, err := get("https://webanetlabs.net/publ/24")
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			wgD.Add(1)
			go func() {
				defer wgD.Done()
				// https://webanetlabs.net/freeproxyweb/proxylist_at_02.11.2019.txt
				u := "https://webanetlabs.net" + href
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range FindAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}()
		}
	}()
	//// https://checkerproxy.net/
	go func() {
		defer wgD.Done()
		var (
			re = regexp.MustCompile(`(?m)href\s*=\s*['"](/archive/\d{4}-\d{2}-\d{2})['"]`)
		)
		body, err := get("https://checkerproxy.net/")
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			wgD.Add(1)
			go func() {
				defer wgD.Done()
				u := "https://checkerproxy.net/api" + href
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range FindAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}()
		}
	}()
	//	proxy-list.org
	go func() {
		defer wgD.Done()
		var (
			re       = regexp.MustCompile(`href\s*=\s*['"]\./([^'"]?index\.php\?p=\d+[^'"]*)['"]`)
			ipBase64 = regexp.MustCompile(`Proxy\('([\w=]+)'\)`)
		)
		body, err := get("http://proxy-list.org/english/index.php?p=1")
		if err != nil {
			return
		}
		for _, href := range findSubmatchRange(re, body) {
			wgD.Add(1)
			go func() {
				defer wgD.Done()
				u := "http://proxy-list.org/english/" + href
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
			}()
		}
	}()
	//// http://www.aliveproxy.com
	go func() {
		defer wgD.Done()
		var (
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
			wgD.Add(1)
			go func() {
				defer wgD.Done()
				u := fmt.Sprintf("http://www.aliveproxy.com/%v/", href)
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range FindAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}()
		}
	}()
	// https://proxylist.me/
	go func() {
		defer wgD.Done()
		var (
			ints []int
			re   = regexp.MustCompile(`(?m)href\s*=\s*['"][^'"]*/?page=(\d+)['"]`)
		)
		body, err := get("https://proxylist.me/")
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
		for i := 0; i < largest; i++ {
			wgD.Add(1)
			go func() {
				defer wgD.Done()
				u := fmt.Sprintf("https://proxylist.me/?page=%v", i)
				ipList, err := get(u)
				if err != nil {
					return
				}
				for _, ip := range FindAllTemplate(reProxy, ipList, templateProxy) {
					mutex.Lock()
					proxies = append(proxies, ip)
					mutex.Unlock()
				}
			}()
		}
	}()
	// https://www.proxy-list.download
	go func() {
		defer wgD.Done()
		body, err := get("https://www.proxy-list.download/api/v1/get?type=http")
		if err != nil {
			return
		}
		for _, ip := range FindAllTemplate(reProxy, body, templateProxy) {
			mutex.Lock()
			proxies = append(proxies, ip)
			mutex.Unlock()
		}
	}()

	quit := make(chan int)
	go counter(quit)

	wgD.Wait()
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
	fmt.Fprintf(os.Stderr,"\rFound %d proxies\n", len(unique))
	fmt.Fprintln(os.Stderr, "\nStarting test ...")
	return unique
}
