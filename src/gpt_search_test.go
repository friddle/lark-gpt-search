package t_test

import (
	"context"
	"feishu-gpt-search/src/config"
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/llama"
	"feishu-gpt-search/src/server"
	"fmt"
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/logger"
	"github.com/go-zoox/uuid"
	"testing"
)

func GetInitClient() (*feishu.FeishuClient, *llama.SearchChainClient, *chatbot.ChatBot) {
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
	return feishuApiClient, searchClient, &bot
}

func TestAuthUrl(t *testing.T) {
	ctx := context.Background()
	feishuConf := config.ReadFeishuConfig()
	feishuApiClient := feishu.NewFeishuClient(ctx, feishuConf)
	url := "https://laiye.com/?code=6a9t5e3d68cb4ce0ad6bf92a6e6b3a17&state=164981201"
	feishuApiClient.SetAccessTokenByUrl(url)
	for userId, token := range feishuApiClient.UserTokens {
		println(userId, *token)
	}
}

func TestSearch(t *testing.T) {
	feishuClient, searchClient, _ := GetInitClient()
	conversationId := string(uuid.V4())
	userId := "164981201"
	token := "u-fHBUmRLwR6QGZscRZErkmV1l6BflkhbxNww0h4E001us"
	argsMap := map[string]string{}
	context, err := searchClient.GetContext(conversationId, userId, argsMap)
	if err != nil {
		panic(fmt.Sprintf("+%v", err))
	}
	feishuClient.UserTokens[userId] = &token
	searchClient.Search(context, "IDP3.5如何部署", func(isAuth bool, content string, moreQuestion map[string]string, links map[string]string, err error) {
		if !isAuth {
			panic("没有登录")
		}
		println(content)
		println(moreQuestion)
		println(links)
	})

}
