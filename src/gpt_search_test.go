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
	"os"
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
	url := "https://laiye.com/?code=febo7c021fb4458ba3ddd30fa87a1c1e&state=164981201"
	err, info := feishuApiClient.SetAccessTokenByUrl(url)
	if err != nil {
		panic("授权失败 " + info)
	}
	for userId, token := range feishuApiClient.UserTokens {
		println(userId, *token)
	}
}

func TestSearch(t *testing.T) {
	feishuClient, searchClient, _ := GetInitClient()
	conversationId := string(uuid.V4())
	userId := "164981201"
	token := "u-dSyhiQWxx018UznesaT9kH50jZkh1hz1OG00lls022pd"
	argsMap := map[string]string{}
	question := "2022Q4的RPA的产品规划"
	context, err := searchClient.GetContext(conversationId, userId, argsMap)
	if err != nil {
		panic(fmt.Sprintf("+%v", err))
	}
	feishuClient.UserTokens[userId] = &token
	searchClient.Search(context, question, func(isAuth bool, content string, moreQuestion map[string]string, links map[string]string, err error) {
		if !isAuth {
			panic("没有登录")
		}
		println(content)
	})
}

func TestGptInvoke(t *testing.T) {
	prompt := `
     You are a professional problem solve export
     Now you are required to answer a question based on the information provided below. 
     Please try not to use related information to answer user's question.
     Information:%s
    `

	docByte, _ := os.ReadFile("./src/sample.txt")
	doc := string(docByte)
	conf := config.ReadChatGptClient()
	conf.AzureDeployment = "lyd-test-davinci"
	gptClient, _ := chatgpt.New(conf)
	message := []*chatgpt.Message{
		&chatgpt.Message{
			Text:      "Question:Laipvt怎么关闭?",
			IsChatGPT: true,
			Role:      "user",
		},
	}
	config := chatgpt.AskConfig{
		Messages:                 message,
		Prompt:                   fmt.Sprintf(prompt, doc),
		Model:                    "text-davinci-003",
		MaxRequestResponseTokens: 7000,
	}
	rsp, err := gptClient.Ask(&config)
	if err != nil {
		panic(fmt.Sprintf("err %v", err))
	}
	logger.Info("rsp:" + string(rsp))
}

func TestGptConversationInvoke(t *testing.T) {
	template := `
You are a professional problem solve export  
Now you are required to answer a question based on the information provided below. Please try not to use related information to anwser user's question.
If you need more information, you can provide relevant search keywords and best related question"
Related Question Are Chinese
And Answer Should include Document Origin Information.Document will be split by "---"

-----------------
-----------------
Documents: "%s"
-----------------
-----------------

Question is %s
Give The answer
`

	docByte, _ := os.ReadFile("./src/sample.txt")
	doc := string(docByte)
	conf := config.ReadChatGptClient()
	//conf.AzureDeployment = "lyd-test-davinci"
	gptClient, _ := chatgpt.New(conf)
	question := "最新的Laipvt地址是什么?"
	ask := fmt.Sprintf(template, doc, question)

	conversation, _ := gptClient.GetOrCreateConversation(uuid.V4(), &chatgpt.ConversationConfig{
		MaxRequestTokens:  60000,
		MaxResponseTokens: 10000,
		Language:          "zh",
	})
	answerBytes, _ := conversation.Ask([]byte(ask), &chatgpt.ConversationAskConfig{})
	answer := string(answerBytes)
	logger.Info("answer:" + answer)
}
