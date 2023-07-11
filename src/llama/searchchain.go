package llama

import (
	"context"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/feishu"
)

type SearchProcessFunc = func(isAuth bool, content string, docToken []string, err error) error
type SearchChainClient struct {
	gptClient      chatgpt.Client
	feishuClient   feishu.Client
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

       And Output with format:
       keyword: KeyWord Here
    `
	return false, nil, nil
}

// 搜索相应飞书的文档
func (client *SearchChainClient) SearchFeishuDoc(context *SearchContext, keywords []string) ([]string, error) {
	return nil, nil
}

func (client *SearchChainClient) TranslateAnswer(ctx *context.Context, docs []string) (string, []string, error) {
	return "", nil, nil
}
