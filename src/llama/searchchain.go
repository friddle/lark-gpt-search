package llama

import (
	"errors"
	"feishu-gpt-search/src/feishu"
	"fmt"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"regexp"
	"strings"
)

type SearchProcessFunc = func(isAuth bool, content string, docToken []string, err error) error
type SearchChainClient struct {
	gptClient      *chatgpt.Client
	feishuClient   *feishu.FeishuClient
	searchContexts map[string]*SearchContext
}

type SearchContext struct {
	//
	askHistory     []string
	userId         string
	conversationId string
	background     string
}

func (client *SearchChainClient) GetContext(conversationId string, userId string) (*SearchContext, error) {
	context, isOk := client.searchContexts[conversationId]
	if !isOk {
		return &SearchContext{
			conversationId: conversationId,
			userId:         userId,
		}, nil
	}
	return context, nil
}

// 暂时不做复杂的langchain.
func (client *SearchChainClient) Search(context *SearchContext, question string, reply func(isAuth bool, content string, links []string, err error)) {
}

// 是否继续搜索,并且把问题转换为关键词
func (client *SearchChainClient) TranslateQuestionToKeyWord(context *SearchContext, question string) (bool, []string, error) {
	defaultQuestionTmpl := `
       You are a professional search engine optimization (SEO) expert. 
       Your task is to extract relevant search terms based on the following background Infromation and Wiki and user Question
       And Return Search KeyWord,if mutiple keyword,split by ,
       Must KeyWord give both English and  Chinese
       Background Information: %s
       Wiki: %s
       Question:%s
       PreviewInfo: %s

       And Output with format:
       keyword: KeyWord Here
    `
	Wiki := ""
	history := strings.Join(context.askHistory, "\r")
	client.AskChatGpt(context, "KeyWord", defaultQuestionTmpl, question, Wiki, context.background, history)
	return false, nil, nil
}

// 搜索相应飞书的文档
func (client *SearchChainClient) SearchFeishuDoc(context *SearchContext, keywords []string) ([]string, error) {
	client.feishuClient.SearchDoc(context.userId, keywords)

	return nil, nil
}

func (client *SearchChainClient) TranslateAnswer(ctx *SearchContext, docs []string) (string, []string, error) {
	return "", nil, nil
}

func (client *SearchChainClient) AskChatGpt(ctx *SearchContext, getKey string, content string, args ...any) (string, error) {
	prompt := fmt.Sprintf(content, args...)
	conversation, err := client.gptClient.GetOrCreateConversation(ctx.conversationId+"_query", &chatgpt.ConversationConfig{
		MaxRequestTokens:  1000,
		MaxResponseTokens: 1000,
		Language:          "zh",
	})
	if err != nil {
		return "", err
	}
	answerBytes, err := conversation.Ask([]byte(prompt), &chatgpt.ConversationAskConfig{})
	if err != nil {
		return "", err
	}
	answer := string(answerBytes)
	rx := regexp.MustCompile(fmt.Sprintf(`%s:(.*)`, getKey))
	// 在字符串中查找匹配项
	match := rx.FindStringSubmatch(answer)
	if len(match) == 2 {
		result := match[1]
		return result, nil
	} else {
		return "", errors.New(fmt.Sprintf("chatgpt返回的格式不对:%s", answer))
	}

}
