package main

import (
	"context"
	"feishu-gpt-search/src/config"
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/llama"
	"feishu-gpt-search/src/server"
	"fmt"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/zoox"
	"github.com/go-zoox/zoox/defaults"
	"os"
)

func main() {
	ctx := context.Background()
	feishuConf := config.ReadFeishuConfig()
	feishuApiClient := feishu.NewFeishuClient(ctx, feishuConf)
	gptClient, _ := chatgpt.New(config.ReadChatGptClient())
	searchClient, _ := llama.NewSearchClient(gptClient, feishuApiClient)
	println(fmt.Sprintf("info:%v", feishuConf))
	bot, err := server.FeishuServer(feishuConf, searchClient, feishuApiClient)
	if err != nil {
		logger.Fatalf("bot error:%v", err)
	}

	authPage := func(c *zoox.Context) {
		server.AuthPage(c, c.Request.URL, feishuApiClient)
	}
	app := defaults.Application()
	logger.Info("registry auth path" + os.Getenv("FEISHU_AUTH_PATH"))
	logger.Info("registry event path" + feishuConf.Path)
	app.Get(os.Getenv("FEISHU_AUTH_PATH"), authPage)
	app.Post(feishuConf.Path, bot.Handler())
	app.Run(fmt.Sprintf(":%d", feishuConf.Port))
}
