package main

import (
	"context"
	"feishu-gpt-search/src/config"
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/llama"
	"feishu-gpt-search/src/server"
	"fmt"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
)

func main() {
	ctx := context.Background()
	feishuConf := config.ReadFeishuConfig()
	feishuApiClient := feishu.NewFeishuClient(ctx, feishuConf)
	searchClient := &llama.SearchChainClient{}
	println(fmt.Sprintf("info:%v", feishuConf))
	bot, err := server.FeishuServer(feishuConf, searchClient, feishuApiClient)
	if err != nil {
		logger.Fatalf("bot error:%v", err)
	}
	if err := bot.Run(); err != nil {
		logger.Fatalf("bot error:%v", err)
	}

	authPage := func(c *zoox.Context) {
		server.AuthPage(c, c.Request.URL, feishuApiClient)
	}

	app := defaults.Application()
	app.Post(feishuConf.Path, bot.Handler())
	app.Get(feishuConf.Path+"/auth", authPage)
	app.Run(fmt.Sprintf(":%d", feishuConf.Port))
}
