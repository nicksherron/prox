proxyutil
===
proxyutil is a command line utility written in Go for quickly finding and testing proxies from the web.


### Downloading

```bash
GO111MODULE=on go get github.com/nicksherron/proxyutil
```
Keep in mind this will build master branch which is in active development. 
Versions and stable releases are coming, but until then there's no guarantees

### Supported platforms
  proxyutil has been tested on Linux(ubuntu) and OS X

### Usage
```bash
proxytuil --help
NAME:
   proxytuil - Find and test proxies from the web.

AUTHOR:
   Nick Sherron <nsherron90@gmail.com>


OPTIONS:
   --file value, -f value     File name to write to instead of stdout.
   --nocheck                  Download only and skip proxy checks. (default: false)
   --limit value, -l value    Limit number of good proxies to check before completing. (default: 0)
   --timeout value, -t value  Specify request time out for checking proxies. (default: 15s)
   --workers value, -w value  Number of (goroutines) concurrent requests to make for checking proxies. (default: 20)
   --url value, -u value      The url to test proxies against. (default: "https://httpbin.org/ip")
   --help, -h                 show help (default: false)
   --version, -v              print the version (default: false)

```
![Alt Text](https://github.com/nicksherron/proxyutil/blob/master/proxyutil_example.gif?raw=true)
