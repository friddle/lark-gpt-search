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
	err, token := feishuApiClient.GetAccessToken(url.Path)
	if err != nil {
		logger.Error("+v", err)
		return
	}
	logger.Info(fmt.Sprintf("token:%v", token))
	fileStr, _ := html.HtmlFs.ReadFile("html/index.html")
	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Content-Types", "text/html; charset=utf-8")
	c.String(http.StatusOK, "%s", string(fileStr))
	c.Write(fileStr)
}
