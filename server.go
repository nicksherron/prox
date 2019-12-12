package main

import (
	"github.com/elazarl/goproxy"
	"github.com/go-redis/redis/v7"
	"log"
	"net/http"
	"net/url"
	"time"
)

func serve(proxyQueue []string) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   5,
	})

	client.Del("proxy")
	for _, v := range proxyQueue {
		client.RPush("proxy", v)
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	http.HandleFunc("/", handler(client, proxy))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func handler(client *redis.Client, p *goproxy.ProxyHttpServer) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		purl, err := client.BRPopLPush("proxy", "proxy", 1*time.Second).Result()
		if err != nil {
			log.Fatal(err)
		}
		log.Println(purl)
		proxyUrl, err := url.Parse(purl)
		if err != nil {
			panic(err)
		}
		p.Tr = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			//Dial: (&net.Dialer{
			//	Timeout: 15 * time.Second,
			//}).Dial,
			//DisableKeepAlives:   false,
			//MaxIdleConnsPerHost: 200,
		}
		r.Header.Del("X-Forwarded-For")
		p.ServeHTTP(w, r)
	}
}
