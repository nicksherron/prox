package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var (
	outFile string
	noCheck bool
	limit   uint64
	timeout time.Duration
	workers int
	testUrl string
	urls    []string
)

func main() {

	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}
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
`

	app := &cli.App{
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Nick Sherron",
				Email: "nsherron90@gmail.com",
			},
		},
		Name:      "proxytuils",
		Usage:     "Find and test proxies from the web.",
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
				Value:       "https://httpbin.org/ip",
				Usage:       "The url to test proxies against.",
				Destination: &testUrl,
			},
		},
		Action: func(c *cli.Context) error {
			_, _ = fmt.Fprintln(os.Stderr, "Finding proxies ...")
			proxies := downloadProxies()
			if !noCheck {
				checkInit(proxies)
				if len(good) == 0 {
					_, _ = fmt.Fprintln(os.Stderr, "no good proxies found")
					return nil
				}
			}

			if !noCheck {
				if outFile != "" {
					g, err := os.Create(outFile)
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
				if outFile != "" {
					g, err := os.Create(outFile)
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
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
