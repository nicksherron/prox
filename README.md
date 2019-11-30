prox
===
prox is a command line utility for quickly finding and testing proxies from the web.


### Downloading

```bash
GO111MODULE=on go get github.com/nicksherron/prox
```
Keep in mind this will build master branch which is in active development. 
To get the stable release run
```bash
GO111MODULE=on go get github.com/nicksherron/prox@v1.2.0
```

### Supported platforms
  prox has been tested on Linux(ubuntu) and OS X

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
prox  --workers 400 --limit 100
```
![example](https://github.com/nicksherron/prox/blob/master/example.gif?raw=true)
