package main

import (
  "bufio"
  "fmt"
  "log"
  "os"
  "net/http"
  "github.com/mfonda/simhash"
  "io/ioutil"
  "time"
)

func readLines(path string) ([]string, error) {
  file, err := os.Open(path)
  if err != nil {
    return nil, err
  }
  defer file.Close()

  var lines []string
  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return lines, scanner.Err()
}

func getResponseHash(domain string, ip string) uint64{
	client := &http.Client{
		Timeout: time.Duration(3 * time.Second),
	}

	req, err := http.NewRequest("GET", "http://" + ip, nil)
	if err != nil {
		return 0
	}
	if domain != "" {
		req.Host = domain
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0
	}
	return simhash.Simhash(simhash.NewWordFeatureSet(body))
}

/**
func populateInitialHashes(domains []string, ips []string, knownHashes []uint64) []uint64 {
	var responseHash uint64
	fmt.Println(append(domains,ips...))
	for _, host := range append(domains,ips...) {
		responseHash = getResponseHash("", host)
		fmt.Println(responseHash)
		knownHashes = AppendIfMissing(knownHashes, responseHash)
	}
	return knownHashes
} **/

// http://jmoiron.net/blog/limiting-concurrency-in-go/
func populateInitialHashes(domains []string, ips []string) []uint64 {
	//var responseHash uint64
	var knownHashes []uint64
	hosts := append(domains, ips...)

	concurrency := 10
	sem := make(chan bool, concurrency)

	for _, host := range hosts {
    	sem <- true
    	go func(host string) {
        	defer func() { <-sem }()
        	// get the url
        	client := &http.Client{
				Timeout: time.Duration(3 * time.Second),
			}
			req, err := http.NewRequest("GET", "http://" + host, nil)
			if err != nil {
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}
			fmt.Println(host)
			knownHashes = AppendIfMissing(knownHashes, simhash.Simhash(simhash.NewWordFeatureSet(body)))
    	}(host)
	}
	for i := 0; i < cap(sem); i++ {
    	sem <- true
	}
	fmt.Println(knownHashes)
	return knownHashes
}

func AppendIfMissing(slice []uint64, i uint64) []uint64 {
    for _, ele := range slice {
        if ele == i {
            return slice
        }
    }
    return append(slice, i)
}

func isHashUnique(responseHash uint64, knownHashes []uint64) bool{
	for _, hash := range knownHashes {
		if simhash.Compare(hash, responseHash) < 10 {
			return false
		}
	}
	return true
}

func main() {
	//var threshold = 5
	var knownHashes []uint64
	var responseHash uint64

	ips, err := readLines("ips.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	domains, err := readLines("domains.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	//fmt.Println(ips)
	//fmt.Println(domains)

	knownHashes = populateInitialHashes(ips, domains)
	fmt.Println(knownHashes)

	for _, ip := range ips {
		for _, domain := range domains {
			responseHash = getResponseHash(domain, ip)
			if responseHash != 0 {
				if isHashUnique(responseHash, knownHashes) == true {
					fmt.Printf("%s:%s:%x\n", domain, ip, responseHash)
				}
			}
		}
	}
}
