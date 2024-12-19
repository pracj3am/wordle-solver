package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

var (
	replace = map[rune]string{
		'a': "[aá]",
		'c': "[cč]",
		'd': "[dď]",
		'e': "[eéě]",
		'i': "[ií]",
		'n': "[nň]",
		'o': "[oó]",
		'r': "[rř]",
		's': "[sš]",
		't': "[tť]",
		'u': "[uúů]",
		'y': "[yý]",
		'z': "[zž]",
	}

	tc = tls.Config{
		InsecureSkipVerify: true,
	}
	tr = http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 300 * time.Second,
		TLSClientConfig: &tc,
	}
	client = http.Client{Transport: &tr}
	u      = url.URL{
		Scheme: "https",
		Host:   "ssjc.ujc.cas.cz",
		Path:   "/search.php",
	}
	query = url.Values{
		"hledej":  []string{"Hledat"},
		"sti":     []string{"EMPTY"},
		"where":   []string{"hesla"},
		"hsubstr": []string{"no"},
	}
)

func loadDictionary(filePath string) ([]string, error) {
	words := make([]string, 0, 2000)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	for s.Scan() {
		word := s.Text()
		words = append(words, word)
	}

	return words, s.Err()
}

func findWords(pattern string) ([]string, error) {
	// https: //ssjc.ujc.cas.cz/search.php?hledej=Hledat&heslo=%5Ba%C3%A1%5D%5Bc%C4%8D%5Dh%5Ba%C3%A1%5D%5Bt%C5%A5%5D&sti=EMPTY&where=hesla&hsubstr=no
	query.Set("heslo", pattern)
	u.RawQuery = query.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	list := htmlquery.Find(doc, "//table//td[1]//span[@class='entry']")

	if len(list) > 0 {
		words := make([]string, len(list))
		for i := range list {
			words[i] = htmlquery.InnerText(list[i])
		}
		return words, nil
	}

	elem := htmlquery.FindOne(doc, "//*[@class='entry']/p/span[1]")

	if elem != nil {
		return []string{htmlquery.InnerText(elem)}, nil
	}

	return nil, nil
}

func main() {
	words, err := loadDictionary("../db.txt")
	if err != nil {
		fmt.Println("loading words failed", err)
		os.Exit(1)
	}

	for _, word := range words {
		var sb strings.Builder
		for _, r := range word {
			if repl, ok := replace[r]; ok {
				sb.WriteString(repl)
			} else {
				sb.WriteRune(r)
			}
		}

		time.Sleep(500 * time.Millisecond)
		fWords, err := findWords(sb.String())
		if err != nil {
			fmt.Println(word, err)
			continue
		}

		if len(fWords) > 0 {
			for _, w := range fWords {
				fmt.Println(w)
			}
		} else {
			fmt.Println("-", word)
		}
	}
}
