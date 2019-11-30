proxyutil
===
proxyutil is a command line utility for quickly finding and testing proxies from the web.


### Downloading

```bash
GO111MODULE=on go get github.com/nicksherron/proxyutil
```
Keep in mind this will build master branch which is in active development. 
To get the stable release run
```bash
GO111MODULE=on go get github.com/nicksherron/proxyutil@v1.1.0
```


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


### Example

Here's and example of downloading 20,000 proxies and returning the first 100 that returned 200 status code 
during testing. 
```bash
proxyutil --limit 100 --workers 500
```
![example](https://github.com/nicksherron/proxyutil/blob/master/example.gif?raw=true)
