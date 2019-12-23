package main

import (
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var (
	queue []string
)

func serve(proxyQueue []string) {

	queue = proxyQueue
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	http.HandleFunc("/", handler(proxy))
	log.Println("starting proxy server and listening at ", serverAddr)
	log.Fatal(http.ListenAndServe(serverAddr, nil))
}

func handler(p *goproxy.ProxyHttpServer) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var purl string
		mutex.Lock()
		purl, queue = queue[0], queue[1:]
		queue = append(queue, purl)
		mutex.Unlock()
		log.Println(purl)
		proxyUrl, err := url.Parse(purl)
		if err != nil {
			panic(err)
		}
		p.Tr = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
		proxIp := strings.Split(strings.ReplaceAll(purl, `http://`, ``), `:`)[0]
		//r.Header.Del("X-Forwarded-For")
		r.Header.Set("X-Forwarded-For", proxIp)
		p.ServeHTTP(w, r)
	}
}
