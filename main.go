package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/realth000/ToGoTool/html"
	"github.com/realth000/ToGoTool/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
)

const (
	mediaDownloadUrlPrefix = `https://res.wx.qq.com/voice/getvoice?mediaid=`
)

var (
	mediaIdRegexp = regexp.MustCompile(`mediaid=(?P<id>\w+)`)
)

func exit() {
	os.Exit(1)
}

func printId() {
	file, err := os.Open("./wechat_media.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if mediaIdRegexp.MatchString(scanner.Text()) {
			matches := mediaIdRegexp.FindStringSubmatch(scanner.Text())
			if len(matches) < mediaIdRegexp.NumSubexp()+1 {
				continue
			}
			id := matches[mediaIdRegexp.SubexpIndex("id")]
			id2, err := base64.StdEncoding.DecodeString(id)
			if err != nil {
				fmt.Println("failed to decode base64:", id, err)
				continue
			}
			fmt.Println("catch id:", id, string(id2))
		}
	}
}

func downloadFromMediaPlayPage(urlPath string, saveDir string) {
	doc, err := html.DocumentFromUrl(urlPath)
	if err != nil {
		fmt.Println("failed to download page:", err)
		exit()
	}
	selections := doc.Find(`mpvoice.js_editor_audio.js_uneditable`)

	if selections.Nodes == nil || len(selections.Nodes) == 0 {
		fmt.Println("failed to download: empty element grepped in", urlPath)
		return
	}
	for _, node := range selections.Nodes {
		path := html.NodeSearchAttr(node, "voice_encode_fileid")
		if path == "" {
			fmt.Println("failed to download: empty download path")
			exit()
		}
		name := html.NodeSearchAttr(node, "name")
		if name == "" {
			fmt.Println("failed to download: empty file name")
			exit()
		}
		fmt.Printf("download \"%s\" from %s\n", name, path)
		mediaData, err := http.GetRequest(fmt.Sprintf("%s%s", mediaDownloadUrlPrefix, path))
		if err != nil {
			fmt.Println("failed to download:", err)
			exit()
		}
		if err = os.WriteFile(fmt.Sprintf("%s/%s.mp3", saveDir, name), mediaData, 0644); err != nil {
			fmt.Println("failed to save media file:", err)
			exit()
		}
	}
}

func main() {
	/*
	   https://mp.weixin.qq.com/s?__biz=MzA4MzU2MjczOA==&mid=2247514662&idx=3&sn=bfa9d32d0e0ea0fe2717c8bbb886dddf&chksm=9ff663bba881eaad2b3feb1aa44fedda4ba5a292712cb2d74ef43f23283a0b5a664af84e7386&scene=21#wechat_redirect
	   http://mp.weixin.qq.com/s?__biz=MzA4MzU2MjczOA==&mid=2247505249&idx=5&sn=89769d90328cf26375a8dc516032fbab&chksm=9ff60cfca88185ea5aa196f7bfe6baffef655c59eb785a7d80409fd838a7ac027c65c090d272&scene=21#wechat_redirect
	*/
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s <url>", filepath.Base(os.Args[0]))
		exit()
	}
	if _, err := url.Parse(os.Args[1]); err != nil {
		fmt.Println("invalid url:", err)
		exit()
	}
	d, err := os.Getwd()
	if err != nil {
		fmt.Println("failed to get current path:", err)
		exit()
	}
	downloadDir := fmt.Sprintf("%s%ctmp", d, os.PathSeparator)
	info, err := os.Stat(downloadDir)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(downloadDir, 0755); err != nil {
			fmt.Println("failed to make download directory:", err)
			exit()
		}
	}
	if err == nil || (errors.Is(err, os.ErrExist) && !info.IsDir()) {
		if err = os.Remove(downloadDir); err != nil {
			fmt.Println("failed to remove file and make download directory:", err)
			exit()
		}
		if err = os.Mkdir("tmp", 0755); err != nil {
			fmt.Println("failed to make download directory:", err)
			exit()
		}
	}
	mainDoc, err := html.DocumentFromUrl(os.Args[1])
	if err != nil {
		fmt.Println("failed to get main doc:", err)
		exit()
	}
	selections := mainDoc.Find(`a`)
	if selections.Nodes == nil {
		fmt.Println("failed to get main doc:", err)
		exit()
	}
	for _, node := range selections.Nodes {
		/*
			<a target="_blank" href="http://mp.weixin.qq.com/s?__biz=MzA4MzU2MjczOA==&amp;mid=2247505862&amp;idx=2&amp;sn=fd85ac927850e8fd42fd7daf7d90dafe&amp;chksm=9ff60e5ba881874d860e35b6b9328eaaa912ab987ad81145fc689baba4460cbc2cdde748718e&amp;scene=21#wechat_redirect" data-itemshowtype="0" tab="innerlink" data-linktype="2" style="text-align: left; visibility: visible;" hasload="1">6.《三国》上</a>
		*/
		if !html.NodeSearchAttrEq(node, "data-linktype", "2") ||
			html.NodeSearchAttrEq(node, "class", "wx_tap_link") ||
			!html.NodeSearchAttrEq(node, "target", "_blank") ||
			html.NodeSearchAttr(node, "href") == "" {
			continue
		}
		url := html.NodeSearchAttr(node, "href")
		downloadFromMediaPlayPage(url, downloadDir)
	}
}
