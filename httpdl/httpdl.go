package httpdl

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gonejack/semagroup"
)

type HTTPDl struct {
	Options
	ctx context.Context
}

func (h *HTTPDl) Run(ctx context.Context) (err error) {
	if h.Download == "" {
		h.PrintUsage()
		return
	}

	h.ctx = ctx
	err = h.fetch(h.Download)
	log.Println("下载结束")

	return
}

func (h *HTTPDl) fetch(ref string) (err error) {
	req, err := http.NewRequestWithContext(h.ctx, http.MethodGet, ref, nil)
	if err != nil {
		return fmt.Errorf("构建请求出错: %s", err)
	}
	req.SetBasicAuth(h.Username, h.Password)

	log.Printf("fetch %s", req.URL.Path)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求出错: %s", err)
	}
	defer rsp.Body.Close()

	_ = os.MkdirAll(h.savePath(req.URL.Path), 0766)

	if rsp.StatusCode < 200 || rsp.StatusCode > 300 {
		return fmt.Errorf("状态码不正确: %s", rsp.Status)
	}
	ct := rsp.Header.Get("content-type")
	ty, _, _ := mime.ParseMediaType(ct)
	if ty != "text/html" {
		return fmt.Errorf("返回内容不是HTML")
	}
	doc, err := goquery.NewDocumentFromReader(rsp.Body)
	if err != nil {
		return fmt.Errorf("解析HTML出错: %s", err)
	}

	var dirs []string
	var files []string
	doc.Find("a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		switch {
		case href == "":
			return
		case href == "../":
			return
		case strings.HasSuffix(href, "/"):
			dirs = append(dirs, href)
		default:
			files = append(files, href)
		}
	})

	for i, file := range files {
		files[i] = absolute(ref, file)
	}
	h.downloadList(files)

	for _, dir := range dirs {
		select {
		case <-h.ctx.Done():
			return
		default:
		}
		dir = absolute(ref, dir)
		err := h.fetch(dir)
		if err != nil {
			unescaped, e := url.QueryUnescape(dir)
			if e != nil {
				unescaped = dir
			}
			log.Printf("访问目录[%s]出错: %s", unescaped, err)
		}
	}

	return
}
func (h *HTTPDl) downloadList(list []string) {
	g := semagroup.New(3)
	for _, v := range list {
		ref := v
		err := g.AcquireAndGo(h.ctx, func() {
			err := h.download(ref)
			if err != nil {
				log.Printf("下载出错: %s", err)
			}
		})
		if err != nil {
			log.Printf("取消下载: %s", v)
			break
		}
	}
	g.Wait(context.TODO())
}
func (h *HTTPDl) download(ref string) (err error) {
	u, err := url.Parse(ref)
	if err != nil {
		return fmt.Errorf("无效的地址: %s %s", ref, err)
	}

	output := h.savePath(u.Path)
	err = os.MkdirAll(path.Dir(output), 0766)
	if err != nil {
		return fmt.Errorf("创建文件夹出错: %s", err)
	}

	if stat, fe := os.Stat(output); fe == nil && stat.Size() > 0 {
		log.Printf("skip %s", u.Path)
		return
		//req, err := http.NewRequest(http.MethodHead, ref, nil)
		//if err != nil {
		//	return fmt.Errorf("创建请求出错: %s", err)
		//}
		//req.SetBasicAuth(h.Username, h.Password)
		//rsp, err := http.DefaultClient.Do(req)
		//if err != nil {
		//	return fmt.Errorf("请求出错: %s", err)
		//}
		//
		//io.Copy(io.Discard, rsp.Body)
		//rsp.Body.Close()
		//
		//switch {
		//case rsp.StatusCode < 200 || rsp.StatusCode > 300:
		//	return fmt.Errorf("请求%s状态码不正确: %s", ref, rsp.Status)
		//case stat.Size() == rsp.ContentLength:
		//	log.Printf("skip %s", req.URL.Path)
		//	return nil
		//default:
		//}
	}

	timeout, cancel := context.WithTimeout(h.ctx, time.Minute*5)
	defer cancel()
	req, err := http.NewRequestWithContext(timeout, http.MethodGet, ref, nil)
	if err != nil {
		return fmt.Errorf("创建请求出错: %s", err)
	}
	req.SetBasicAuth(h.Username, h.Password)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求出错: %s", err)
	}
	if rsp.StatusCode < 200 || rsp.StatusCode > 300 {
		return fmt.Errorf("请求%s状态码不正确: %s", ref, rsp.Status)
	}
	defer rsp.Body.Close()

	fd, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("创建文件出错: %s", err)
	}
	if h.Verbose {
		log.Printf("save %s => %s", ref, output)
	} else {
		log.Printf("save %s", req.URL.Path)
	}
	n, err := io.Copy(fd, rsp.Body)
	_ = fd.Close()
	if err != nil {
		_ = os.Remove(fd.Name())
		return fmt.Errorf("downolad %s => %s failed: %s", ref, output, err)
	}
	if rsp.ContentLength > n {
		_ = os.Remove(fd.Name())
		err = fmt.Errorf("short read %d/%d", n, rsp.ContentLength)
		return fmt.Errorf("downolad %s => %s failed: %s", ref, output, err)
	}
	log.Printf("save %s done", req.URL.Path)

	return
}
func (h *HTTPDl) savePath(p string) string {
	return path.Join("download", p)
}

func absolute(base string, href string) string {
	switch {
	case strings.HasPrefix(href, "http://"):
		return href
	case strings.HasPrefix(href, "https://"):
		return href
	}
	u, _ := url.Parse(base)
	u.Path = path.Join(u.Path, href)
	x, err := url.QueryUnescape(u.String())
	if err == nil {
		return x
	} else {
		return u.String()
	}
}
