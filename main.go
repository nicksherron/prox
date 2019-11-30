package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	cpuProfile   = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile   = flag.String("memprofile", "", "write memory profile to `file`")
	traceProfile = flag.String("traceprofile", "", "write trace profile to `file`")
	outFile      = flag.String("file", "", "File name to write to instead of stdout.")
	noCheck      = flag.Bool("nocheck", false, "Download only and skip proxy checks.")
	limit        = flag.Uint64("limit", 0, "Limit number of good proxies to check before completing.")
	timeout      = flag.Duration("t", 15*time.Second, "Specify request time out for checking proxies.")
	workers      = flag.Int("w", 5, "Number of concurrent requests to make for checking proxies.")
	testUrl      = flag.String("u", "https://httpbin.org/ip", "The url to test proxies against.")
	urls         []string
)

func main() {
	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *traceProfile != "" {
		f, err := os.Create(*traceProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := trace.Start(f); err != nil {
			log.Fatal("could not start Trace profile: ", err)
		}
		defer trace.Stop()
	}

	_, _ = fmt.Fprintln(os.Stderr, "Finding proxies ...")
	proxies := downloadProxies()
	if !*noCheck {
		checkInit(proxies)
		if len(good) == 0 {
			_, _ = fmt.Fprintln(os.Stderr, "no good proxies found")
			return
		}
	}

	if !*noCheck {
		if *outFile != "" {
			g, err := os.Create(*outFile)
			check(err)
			defer g.Close()

			for _, v := range good {
				fmt.Fprintln(g, v)
			}
		} else {
			for _, v := range good {
				fmt.Println(v)
			}
		}
	} else {
		if *outFile != "" {
			g, err := os.Create(*outFile)
			check(err)
			defer g.Close()
			for _, v := range proxies {
				fmt.Fprintln(g, v)
			}
		} else {
			for _, v := range proxies {
				fmt.Println(v)
			}
		}
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
