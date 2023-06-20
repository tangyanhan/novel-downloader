package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var client http.Client

func init() {
	client = http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
}

func GetUrlWithCache(url string, fileName string) (filePath string, content []byte, err error) {
	const cacheFolder = "cache"
	sum := md5.Sum([]byte(url))
	dst := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(dst, sum[:])
	if fileName != "" {
		filePath = path.Join(cacheFolder, fileName)
	} else {
		filePath = path.Join(cacheFolder, string(dst)+".html")
	}

	if _, err = os.Stat(filePath); err == nil {
		log.Println("Cache Hit: ", url, "->", filePath)
		content, err = ioutil.ReadFile(filePath)
		return
	}

	if _, err = os.Stat(cacheFolder); err != nil {
		os.Mkdir(cacheFolder, fs.ModePerm)
	}
	var req *http.Request
	var resp *http.Response
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/47.0")
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode/100 != 2 {
		return
	}

	content, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err = ioutil.WriteFile(filePath, content, fs.ModePerm); err != nil {
		return
	}
	return
}

func main() {
	var contentsUrl string
	var baseUrl string
	var outFileName string
	flag.StringVar(&contentsUrl, "contents", "", "Url path of novel contents page")
	flag.StringVar(&baseUrl, "base", "", "Base url of the novel site to combine chapter links")
	flag.StringVar(&outFileName, "out", "", "Filename of the output file")
	flag.Parse()

	filePath, content, err := GetUrlWithCache(contentsUrl, "")
	if err != nil {
		log.Fatalln("Failed to get page", contentsUrl, ":", err)
	}

	log.Println(contentsUrl, "->", filePath)

	reader := bytes.NewReader(content)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	// Find the review items
	doc.Find("#chapterlist a").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the title
		log.Println("Found chapter=", s.Text())
		link, ok := s.Attr("href")
		if !ok {
			return
		}
		chapterName := s.Text()
		if match, err := regexp.MatchString("第.*章", chapterName); !match || err != nil {
			log.Println("Cannot match:", chapterName)
			return
		}

		buf.WriteString("\n\n=======================\n")
		buf.WriteString(chapterName)
		buf.WriteString("\n=======================\n\n")

		if err := DownloadChapter(baseUrl, link, chapterName, 0, &buf); err != nil {
			log.Fatalln("Failed to download chapter:", err)
			return
		}
	})

	if err := ioutil.WriteFile(outFileName, buf.Bytes(), os.ModePerm); err != nil {
		log.Fatalln("Failed to write file to", outFileName, "Error:", err)
	}
}

func DownloadChapter(baseUrl, link, chapterName string, page int, buf *bytes.Buffer) error {
	fmt.Println("Chapter=", chapterName, "Url=", link)
	chapterURL := baseUrl + link
	filePath, content, err := GetUrlWithCache(chapterURL, fmt.Sprintf("%s-%d.html", chapterName, page))
	if err != nil {
		log.Fatalln("Failed to get page", chapterURL, ":", err)
	}
	log.Println(chapterName, "->", filePath)

	reader := bytes.NewReader(content)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	chapterContent := doc.Find("#chaptercontent")
	if chapterContent == nil {
		return nil
	}
	rootDiv := chapterContent.Nodes[0]
	var hasLineEnd bool
	for node := rootDiv.FirstChild; node != nil; node = node.NextSibling {
		switch node.Type {
		case html.TextNode:
			buf.WriteString(strings.TrimSpace(node.Data))
			hasLineEnd = false
		case html.ElementNode:
			if node.Data == "br" && !hasLineEnd {
				buf.WriteString("\n")
				hasLineEnd = true
			}
		default:
		}
	}

	// Find the review items
	nextPage := doc.Find("#pt_next")
	if nextPage == nil {
		return nil
	}
	nextPageUrl, ok := nextPage.Attr("href")
	if !ok {
		log.Println("No next page link for ", chapterName, filePath)
		return nil
	}

	if match, err := regexp.MatchString("_\\d+.html", nextPageUrl); !match || err != nil {
		return nil
	}

	return DownloadChapter(baseUrl, nextPageUrl, chapterName, page+1, buf)
}
