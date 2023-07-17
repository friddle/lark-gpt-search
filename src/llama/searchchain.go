package llama

import (
	"errors"
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/utils"
	"fmt"
	"github.com/chyroc/lark"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type SearchProcessFunc = func(isAuth bool, content string, docToken []string, err error) error
type SearchChainClient struct {
	gptClient      chatgpt.Client
	feishuClient   *feishu.FeishuClient
	searchContexts map[string]*SearchContext
}

func NewSearchClient(gptClient chatgpt.Client, feishuClient *feishu.FeishuClient) (*SearchChainClient, error) {
	return &SearchChainClient{
		gptClient:      gptClient,
		feishuClient:   feishuClient,
		searchContexts: make(map[string]*SearchContext),
	}, nil
}

type SearchContext struct {
	//
	askHistory     []string
	searchKey      []string
	conversationId string
	baikeIds       []string
	background     string
	options        feishu.SearchOptions
	userId         string
}

// TODO
func (client *SearchChainClient) GetContext(conversationId string, userId string, args map[string]string) (*SearchContext, error) {
	context, isOk := client.searchContexts[conversationId]
	if !isOk {
		context = &SearchContext{
			conversationId: conversationId,
			baikeIds:       []string{},
			searchKey:      []string{},
			background:     "no info",
			userId:         userId,
		}
	}
	background, ok := args["background"]
	if ok {
		context.background = background
	}
	searchKey, ok := args["searchKey"]
	if ok {
		context.searchKey = append([]string{}, searchKey)
	}

	//update SearchOptions
	searchOption := &feishu.SearchOptions{
		DocTypes: []string{"doc", "docx"},
		Count:    int64(3),
		Offset:   int64(0),
		Wiki:     true,
		Exclude:  []string{},
		Self:     false,
	}
	context.options = *searchOption

	docTypes, ok := args["docTypes"]
	if docTypes != "" {
		docTypeList := strings.Split(docTypes, ",")
		context.options.DocTypes = append([]string{}, docTypeList...)
	}
	exclude, ok := args["exclude"]
	if exclude != "" {
		excludeTypeList := strings.Split(exclude, ",")
		context.options.Exclude = append([]string{}, excludeTypeList...)
	}
	count, ok := args["count"]
	if ok {
		countint, _ := strconv.Atoi(count)
		context.options.Count = int64(countint)
	}
	offset, ok := args["offset"]
	if ok {
		offsetnum, _ := strconv.Atoi(offset)
		context.options.Offset = int64(offsetnum)
	}
	isSelf, ok := args["isSelf"]
	if ok {
		context.options.Self, _ = strconv.ParseBool(isSelf)
	}
	isWiki, ok := args["isWiki"]
	if ok {
		context.options.Wiki, _ = strconv.ParseBool(isWiki)
	}

	return context, nil
}

// 暂时不做复杂的langchain.
func (client *SearchChainClient) Search(context *SearchContext, question string, reply func(isAuth bool,
	content string,
	moreQuestion map[string]string,
	links map[string]string, err error)) {
	var searchKey []string
	isAuth := client.feishuClient.IsAuthWork(context.userId)
	if !isAuth {
		reply(false, "需要登录", nil, nil, nil)
		return
	}
	if len(context.searchKey) != 0 {
		searchKey = context.searchKey
	} else {
		log.Printf(fmt.Sprintf("question:%s", question))
		_, keyWords, info, error := client.TranslateQuestionToKeyWord(context, question)
		if error != nil {
			reply(true, "翻译关键词失败:chatgpt 返回"+info, nil, nil, error)
			return
		}
		searchKey = keyWords
	}
	reply(true, fmt.Sprintf("搜索关键词:%s", strings.Join(searchKey, " ")), nil, nil, nil)
	_, documentContent, links, err := client.SearchFeishuDoc(context, searchKey)
	if err != nil {
		reply(true, fmt.Sprintf("搜索文档失败:%v", err), nil, nil, err)
		return
	}
	if len(documentContent) == 0 {
		reply(true, "没有搜索到相应的文档。请切换搜索词汇或者设置 --searchKey=", nil, nil, err)
		return
	}
	var titles []string
	for title, _ := range documentContent {
		titles = append(titles, title)
	}
	reply(true, fmt.Sprintf("搜索%d个文档,默认为%s:\r", len(documentContent), strings.Join(titles, "\r\n")), nil, nil, nil)

	_, answer, moreQuestion, info, err := client.TranslateAnswer(context, question, documentContent)
	if err != nil {
		reply(true, "理解答案失败(请不要使用太长的文档),ChatGPT返回:"+info, nil, nil, err)
		return
	}
	reply(true, "结果为:"+answer, moreQuestion, links, nil)
	logQuestionToFile(question, answer)
	context.askHistory = append(context.askHistory, question)
}

func logQuestionToFile(question string, answer string) {
	log.Printf("question:" + question + "answer:" + answer)

}

// 是否继续搜索,并且把问题转换为关键词
func (client *SearchChainClient) TranslateQuestionToKeyWord(context *SearchContext, question string) (bool, []string, string, error) {
	defaultQuestionTmpl := `
       You are a professional search engine optimization (SEO) expert. 
       Your task is to extract search keyword based on Question And Return Search KeyWord,
       If multiple keyword,split by ';'
       Question:%s

       And Output Info with Format,Not Change Format:
       关键词: KeyWord Here
    `
	if utils.Exists(".prompt_search.txt") {
		defaultQuestionTmplByte, _ := os.ReadFile(".prompt_search.txt")
		defaultQuestionTmpl = string(defaultQuestionTmplByte)
	}

	history := strings.Join(context.askHistory, "\r")
	if history == "" {
		history = "No History"
	}
	keyWordStr, info, _ := client.AskChatGpt(context, 1000, []string{"关键词"}, defaultQuestionTmpl,
		question,
	)
	keyWords := strings.Split(keyWordStr["关键词"], ";")
	return true, keyWords, info, nil
}

func (client *SearchChainClient) GetMethodOptions(context *SearchContext) (bool, lark.MethodOptionFunc, error) {
	option, err := client.feishuClient.GenAuthToken(context.userId)
	if err != nil {
		return false, nil, err
	} else {
		return true, option, nil
	}
}

func (client *SearchChainClient) GetBaike(context *SearchContext) (string, error) {
	if len(context.baikeIds) == 0 {
		return "no wiki information", nil
	}
	bkRsp, err := client.feishuClient.GetBaike(context.baikeIds)
	if err != nil {

	}
	bkString := ""
	for _, bk := range bkRsp {
		bkString = bkString + "\n" + bk.Entity.Description
	}
	return bkString, nil
}

// 搜索相应飞书的文档
func (client *SearchChainClient) SearchFeishuDoc(context *SearchContext, keywords []string) (bool, map[string]string, map[string]string, error) {
	_, option, err := client.GetMethodOptions(context)
	if err != nil {
		return true, nil, nil, err
	}
	contentMap, linksMap, err := client.feishuClient.SearchDocsWithResult(strings.Join(keywords, " "), context.userId, context.options, option)
	if err != nil {
		return true, nil, nil, err
	}
	return true, contentMap, linksMap, nil
}

func splitByLength(str string, length int) []string {
	var result []string
	for len(str) > 0 {
		if len(str) >= length {
			result = append(result, str[:length])
			str = str[length:]
		} else {
			result = append(result, str)
			str = ""
		}
	}
	return result
}

func (client *SearchChainClient) CleanMarkDownToText(ctx *SearchContext, text string) []string {
	maxLength := 5000
	output := text
	output = strings.Replace(output, "#", "", -1)
	output = strings.Replace(output, "<strong>", "", -1)
	output = strings.Replace(output, "</strong>", "", -1)
	re, _ := regexp.Compile(`\n\n+`)
	output = re.ReplaceAllString(output, "\n")
	return splitByLength(output, maxLength)
}

//可以把这个问题记录下来。向量化。并且提供商搜索回答

// TODO: 这边应该一个简单的for循环来实现效果
func (client *SearchChainClient) TranslateAnswer(ctx *SearchContext, query string, docs map[string]string) (bool, string, map[string]string, string, error) {
	defaultAnswerTmpl := `
You are a professional problem solve export  
Now you are required to answer a question based on the information provided below. Please try not to use related information to anwser user's question.
If you need more information, you can provide relevant search keywords and best related question"
Related Question Are Chinese
And Answer Should include Document Origin Information,If No Clue About The Answer.Return "查询的文档没有相关信息,请换一个文档搜索关键词"
Related Question Please Return Chinese

Documents: "%s"
Question:%s

And Output with format:
回答: Answer Here
`
	if utils.Exists(".prompt_answer.txt") {
		defaultAnswerBytes, _ := os.ReadFile(".prompt_answer.txt")
		defaultAnswerTmpl = string(defaultAnswerBytes)
	}

	questionMap := make(map[string]string)
	var info = ""
	for _, document := range docs {
		texts := client.CleanMarkDownToText(ctx, document)
		for _, text := range texts {
			_, info, _ := client.AskChatGpt(ctx, 4000, []string{"回答"}, defaultAnswerTmpl, text, query)
			if !strings.Contains(info, "没有相关") {
				return true, info, questionMap, info, nil
			}
		}
	}
	return true, info, questionMap, info, nil
}

func (client *SearchChainClient) AskWithDoc(ctx *SearchContext, getKeys []string, content string, args ...any) (map[string]string, error) {
	return nil, nil
}

func (client *SearchChainClient) AskChatGpt(ctx *SearchContext, requestToken int64, getKeys []string, content string, args ...any) (map[string]string, string, error) {
	prompt := fmt.Sprintf(content, args...)
	//这边有可能是巨坑
	conversation, err := client.gptClient.GetOrCreateConversation(ctx.conversationId+"_query", &chatgpt.ConversationConfig{
		MaxRequestTokens:  requestToken,
		MaxResponseTokens: 1000,
		Language:          "zh",
	})
	if err != nil {
		return nil, "", err
	}
	answerBytes, err := conversation.Ask([]byte(prompt), &chatgpt.ConversationAskConfig{})
	if err != nil {
		return nil, "", err
	}
	answer := string(answerBytes)
	answers := map[string]string{}
	log.Println(fmt.Sprintf("chatgpt的提问为:%s \r 结果为:%s", prompt, answer))
	for _, getKey := range getKeys {
		rx := regexp.MustCompile(fmt.Sprintf(`%s:(.*)`, getKey))
		// 在字符串中查找匹配项
		match := rx.FindStringSubmatch(answer)
		if len(match) == 2 {
			result := match[1]
			answers[getKey] = result
		} else {
			return answers, answer, errors.New(fmt.Sprintf("chatgpt返回的格式不对:%s", answer))
		}
	}
	return answers, answer, nil
}
