package server

import (
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/html"
	"fmt"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"net/http"
	"net/url"
)

func AuthPage(c *zoox.Context, url *url.URL, feishuApiClient *feishu.FeishuClient) {
	err, token := feishuApiClient.SetAccessTokenByUrl(url.Path + "?" + url.RawQuery)
	fileStr, _ := html.HtmlFs.ReadFile("html/index.html")
	if err != nil {
		logger.Error("+v", err)
		fileStr, _ = html.HtmlFs.ReadFile("html/error.html")
	}
	logger.Info(fmt.Sprintf("token:%v", token))
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, "%s", string(fileStr))
}
