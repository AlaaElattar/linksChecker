package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/pelletier/go-toml/v2"
)

type WebSites struct {
	Sites map[string]struct {
		URL string `toml:"url"`
	} `toml:"sites"`
}

func ParseConfig(path string) (*WebSites, error) {
	var webSites WebSites
	sites, err := ioutil.ReadFile(path)
	if err != nil {
		return &WebSites{}, fmt.Errorf("failed to open config file: %w", err)
	}

	err = toml.Unmarshal(sites, &webSites)
	if err != nil {
		return &WebSites{}, fmt.Errorf("failed to read file: %w", err)
	}
	return &webSites, nil
}

func ValidateUrl(url string) string {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	return url
}

func IsReachable(url string, wg *sync.WaitGroup, ch chan<- string) bool {
	defer wg.Done()

	response, err := http.Get(url)
	if err != nil {
		ch <- fmt.Sprintf("%s is not reachable ", url)
		return false
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		ch <- fmt.Sprintf("%s is reachable.", url)
		return true
	} else {
		ch <- fmt.Sprintf("%s is not reachable.", url)

	}

	return false
}

func ExtractLinks(url string) ([]string, error) {
	url = ValidateUrl(url)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s url is not reachable.\n", url)
	}

	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return nil, err
	}

	links := []string{}
	document.Find("a").Each(func(index int, element *goquery.Selection) {
		link, exists := element.Attr("href")
		if exists {
			links = append(links, link)
		}
	})

	return links, nil

}

func main() {
	sites, err := ParseConfig("config.toml")
	if err != nil {
		fmt.Println(err)
		return
	}

	ch := make(chan string)
	var wg sync.WaitGroup

	for siteName, siteInfo := range sites.Sites {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()

			links, err := ExtractLinks(url)
			if err != nil {
				fmt.Printf(err.Error())
				return
			}

			for _, link := range links {
				fullURL := ValidateUrl(link)
				wg.Add(1)
				go IsReachable(fullURL, &wg, ch)
			}
		}(siteName, siteInfo.URL)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for msg := range ch {
		fmt.Println(msg)
	}
}
