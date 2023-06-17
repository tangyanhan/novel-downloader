package main

import (
	"encoding/xml"
	"flag"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
)

type PageBody struct {
	XMLName  xml.Name `xml:"body"`
	Chapters []struct {
		XMLName xml.Name `xml:"p"`
	} `xml:"p"`
	Body struct {
		Chapters []struct {
		}
	} `xml:"body"`
	// 	<body>
	// <header id="top" class="channelHea channelHea2">
	// <a href="javascript:history.go(-1);" class="iconback"><img src="/images/header-back.gif" alt="返回" /></a>
	// <span class="title">国民法医</span>
	// <a href="/" class="iconhome"><img src="/images/header-backhome.gif" alt="首页" /></a>
	// </header>
	// <div id="chapterlist" class="directoryArea">
	// <p><a href="#bottom" style="color:Red;">↓直达页面底部↓</a></p>
	// <p><a href="/112/112958/64643228.html">第一章：十七叔</a></p>
}

func main() {
	var baseUrl string
	flag.StringVar(&baseUrl, "url", "", "Url path of novel")
	flag.Parse()

	client := http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	resp, err := client.Get(baseUrl)
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode/100 != 2 {
		log.Fatalln("Status code failed with:", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("failed to read body:", err)
	}
	ioutil.WriteFile("dummy.html", raw, fs.ModePerm)

	xml.Unmarshal(raw)
}
