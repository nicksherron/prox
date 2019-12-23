//TODO: Make tests

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	outFile      string
	noCheck      bool
	limit        uint64
	timeout      time.Duration
	workers      int
	testUrl      string
	urls         []string
	showRequest  bool
	serveProxies bool
	eliteOnly bool
	noTransparent bool
)

func main() {

	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
{{if .VisibleFlags}}
OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}
`

	app := &cli.App{
		Version: "v1.3.1",
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Nick Sherron",
				Email: "nsherron90@gmail.com",
			},
		},
		Name:  "prox",
		Usage: "Find and test proxies from the web.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "file",
				Aliases:     []string{"f"},
				Value:       "",
				Usage:       "File name to write to instead of stdout.",
				Destination: &outFile,
			},
			&cli.BoolFlag{
				Name:        "nocheck",
				Value:       false,
				Usage:       "Download only and skip proxy checks.",
				Destination: &noCheck,
			},
			&cli.Uint64Flag{
				Name:        "limit",
				Aliases:     []string{"l"},
				Value:       0,
				Usage:       "Limit number of good proxies to check before completing.",
				Destination: &limit,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Value:       15 * time.Second,
				Usage:       "Specify request time out for checking proxies.",
				Destination: &timeout,
			},
			&cli.IntFlag{
				Name:        "workers",
				Aliases:     []string{"w"},
				Value:       20,
				Usage:       "Number of (goroutines) concurrent requests to make for checking proxies.",
				Destination: &workers,
			},
			&cli.StringFlag{
				Name:        "url",
				Aliases:     []string{"u"},
				Value:       "http://httpbin.org/get?show_env",
				Usage:       "The url to test proxies against.",
				Destination: &testUrl,
			},
			&cli.BoolFlag{
				Name:        "show-response",
				Value:       false,
				Usage:       "If using default httpbin.org test url, add http response body to return results.",
				Destination: &showRequest,
			},
			&cli.BoolFlag{
				Name:        "serve",
				Value:       false,
				Usage:       "Serve proxies after checking",
				Destination: &serveProxies,
			},
			&cli.BoolFlag{
				Name:        "elite",
				Value:       false,
				Usage:       "Only use fully anonymous proxies. Only works if using default test url",
				Destination: &eliteOnly,
			},
			&cli.BoolFlag{
				Name:        "no-transparent",
				Aliases:     []string{"nt"},
				Value:       false,
				Usage:       "Filter out transparent proxies. Only works if using default test url",
				Destination: &noTransparent,
			},
		},
		Action: func(c *cli.Context) error {
			go func() {
				defer cli.OsExiter(0)
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
				<-sigs
			}()
			_, _ = fmt.Fprintln(os.Stderr, "Finding proxies ...")
			proxies := downloadProxies()
			if !noCheck {
				checkInit(proxies)
				if len(good) == 0 {
					_, _ = fmt.Fprintln(os.Stderr, "no good proxies found")
					return nil
				}
				if outFile != "" {
					g, err := os.Create(outFile)
					check(err)
					defer g.Close()
					if len(jsonProxies) != 0 {
						for _, v := range jsonProxies {
							fmt.Fprintln(g, v)
						}
					}else {
						for _, v := range good {
							fmt.Fprintln(g, v)
						}
					}
				}else if serveProxies{
					serve(good)
				} else {
					if len(jsonProxies) != 0 {
						for _, v := range jsonProxies {
							fmt.Println(v)
						}
					}else {
						for _, v := range good {
							fmt.Println(v)
						}
					}
				}
			} else {
				if outFile != "" {
					g, err := os.Create(outFile)
					check(err)
					defer g.Close()
					for _, v := range proxies {
						fmt.Fprintln(g, v)
					}
				}else if serveProxies{
					serve(proxies)
				} else {
					for _, v := range proxies {
						fmt.Println(v)
					}
				}
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
