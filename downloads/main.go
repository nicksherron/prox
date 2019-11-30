package main

import (
	"fmt"
	"github.com/icrowley/fake"
	"io/ioutil"
	"net/http"
	"regexp"
)
var (
	reIpPort = regexp.MustCompile(`(?P<ip>(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))(?:.*?(?:(?:(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?))|(?P<port>\d{2,5})))`)
	reID = regexp.MustCompile(`(?m)href\s*=\s*['"](?P<t>[^'"]*)/(?P<uts>\d{10})[^'"]*['"]`)
	fplUrls = []string{
		"http://www.freeproxylists.com/anonymous.html",
		"http://www.freeproxylists.com/elite.html",
	}
)
func check(e error) {
	if e != nil {
		panic(e)
	}
}
func findNamedMatches(regex *regexp.Regexp, str string) map[string]string {
	match := regex.FindStringSubmatch(str)

	results := map[string]string{}
	for i, name := range match {
		results[regex.SubexpNames()[i]] = name
	}
	return results
}

func get(u string) string{
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	check(err)
	req.Header.Set("X-Forwarded-For", fake.IPv4())
	req.Header.Set("User-Agent", fake.UserAgent())
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)

}

func main()  {
	for _, u := range fplUrls {
		body := get(u)
		for _, match := range reID.FindAllString(body, -1) {
			m := findNamedMatches(reID, match)
			u := fmt.Sprintf("http://www.freeproxylists.com/load_%v_%v.html", m["t"], m["uts"])
			body2 := get(u)
			for _, match := range reIpPort.FindAllString(body2, -1) {
				m := findNamedMatches(reIpPort, match)
				ip := fmt.Sprintf("http://%v:%v", m["ip"], m["port"])
				fmt.Println(ip)
			}
		}
	}

}