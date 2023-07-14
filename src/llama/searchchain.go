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
	ownerId        []string
	docTypes       []string
	userId         string
	searchKey      []string
	conversationId string
	wikiIds        []string
	background     string
	count          int64
	offset         int64
}

// TODO
func (client *SearchChainClient) GetContext(conversationId string, userId string, args map[string]string) (*SearchContext, error) {
	context, isOk := client.searchContexts[conversationId]
	if !isOk {
		context = &SearchContext{
			conversationId: conversationId,
			userId:         userId,
			wikiIds:        []string{},
			searchKey:      []string{},
			background:     "no info",
			docTypes:       []string{},
			count:          int64(1),
			offset:         int64(1),
		}
	}
	wikiIds, ok := args["wikiIds"]
	if ok {
		wikiIdList := strings.Split(wikiIds, ",")
		context.wikiIds = wikiIdList
	}
	background, ok := args["background"]
	if ok {
		context.background = background
	}
	searchKey, ok := args["searchKey"]
	if ok {
		context.searchKey = append([]string{}, searchKey)
	}
	docTypes, ok := args["docTypes"]
	if docTypes != "" {
		docTypeList := strings.Split(docTypes, ",")
		context.docTypes = append([]string{}, docTypeList...)
	}
	count, ok := args["count"]
	if ok {
		countint, _ := strconv.Atoi(count)
		context.count = int64(countint)
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
		_, keyWords, error := client.TranslateQuestionToKeyWord(context, question)
		if error != nil {
			reply(true, "翻译关键词失败", nil, nil, error)
			return
		}
		searchKey = keyWords
	}
	reply(true, fmt.Sprintf("搜索关键词:%s", strings.Join(searchKey, ";")), nil, nil, nil)
	_, documentContent, links, err := client.SearchFeishuDoc(context, searchKey)
	if err != nil {
		reply(true, "搜索文档失败", nil, nil, err)
		return
	}
	if len(documentContent) == 0 {
		reply(true, "没有搜索到相应的文档。请切换搜索词汇或者设置 --searchKey=", nil, nil, err)
		return
	}
	_, answer, moreQuestion, err := client.TranslateAnswer(context, question, documentContent)
	if err != nil {
		reply(true, "理解答案失败(请不要使用太长的文档)"+answer, nil, nil, err)
		return
	}
	reply(true, "结果为:"+answer, moreQuestion, links, nil)
	context.askHistory = append(context.askHistory, question)
}

// 是否继续搜索,并且把问题转换为关键词
func (client *SearchChainClient) TranslateQuestionToKeyWord(context *SearchContext, question string) (bool, []string, error) {
	defaultQuestionTmpl := `
       You are a professional search engine optimization (SEO) expert. 
       Your task is to extract relevant search terms based on the following background Infromation and Wiki and user Question
       And Return Search KeyWord,if multiple keyword,split by ';'
       Background Information: %s
       Wiki: %s
       PreviewInfo: %s
       Question:%s

       And Output with format:
       KeyWord: KeyWord Here
    `
	if utils.Exists(".prompt_search.txt") {
		defaultQuestionTmplByte, _ := os.ReadFile(".prompt_search.txt")
		defaultQuestionTmpl = string(defaultQuestionTmplByte)
	}

	Wiki, err := client.GetBaike(context)
	if err != nil {
		return true, nil, err
	}
	history := strings.Join(context.askHistory, "\r")
	if history == "" {
		history = "No History"
	}
	keyWordStr, _ := client.AskChatGpt(context, 1000, []string{"KeyWord"}, defaultQuestionTmpl,
		context.background, Wiki, history, question,
	)
	keyWords := strings.Split(keyWordStr["KeyWord"], ";")
	return true, keyWords, nil
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
	if len(context.wikiIds) == 0 {
		return "no wiki information", nil
	}
	bkRsp, err := client.feishuClient.GetBaike(context.wikiIds)
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
	contentMap, linksMap, err := client.feishuClient.SearchDocsWithResult(strings.Join(keywords, " "), context.count, context.ownerId, context.docTypes, option)
	if err != nil {
		return true, nil, nil, err
	}
	return true, contentMap, linksMap, nil
}

func (client *SearchChainClient) CleanMarkDownToText(ctx *SearchContext, text string) string {
	maxLength := 4000
	output := text
	output = strings.Replace(output, "#", "", -1)
	output = strings.Replace(output, "<strong>", "", -1)
	output = strings.Replace(output, "</strong>", "", -1)
	re, _ := regexp.Compile(`\n\n+`)
	output = re.ReplaceAllString(output, "\n")
	if len(output) <= maxLength {
		return output
	} else {
		return string(output)[:maxLength]
	}
}

// 翻译答案:TODO
func (client *SearchChainClient) TranslateAnswer(ctx *SearchContext, query string, docs map[string]string) (bool, string, map[string]string, error) {
	defaultAnswerTmpl := `
You are a professional problem solve export  
Now you are required to answer a question based on the information provided below. Please try not to use related information to anwser user's question.
If you need more information, you can provide relevant search keywords and best related question"
Related Question Are Chinese
And Answer Should include Document Origin Information

Documents: "%s"
Question:%s

And Output with format:
Answer: Answer Here
RelatedQuestion: Related Question Here
`
	if utils.Exists(".prompt_answer.txt") {
		defaultAnswerBytes, _ := os.ReadFile(".prompt_answer.txt")
		defaultAnswerTmpl = string(defaultAnswerBytes)
	}

	Documents := ""
	for title, document := range docs {
		Documents = Documents + "\n" + title + ":" + document
	}
	Documents = client.CleanMarkDownToText(ctx, Documents)
	answerMap, _ := client.AskChatGpt(ctx, 5000, []string{"Answer", "RelatedQuestion"}, defaultAnswerTmpl, Documents, query)
	answer := answerMap["Answer"]
	relatedQuestion := answerMap["RelatedQuestion"]
	questionMap := make(map[string]string)
	for index, question := range strings.Split(relatedQuestion, ",") {
		questionMap[string(index)] = question
	}
	return true, answer, questionMap, nil
}

func (client *SearchChainClient) AskWithDoc(ctx *SearchContext, getKeys []string, content string, args ...any) (map[string]string, error) {
	return nil, nil
}

func (client *SearchChainClient) AskChatGpt(ctx *SearchContext, requestToken int64, getKeys []string, content string, args ...any) (map[string]string, error) {
	prompt := fmt.Sprintf(content, args...)
	//这边有可能是巨坑
	conversation, err := client.gptClient.GetOrCreateConversation(ctx.conversationId+"_query", &chatgpt.ConversationConfig{
		MaxRequestTokens:  requestToken,
		MaxResponseTokens: 1000,
		Language:          "zh",
	})
	if err != nil {
		return nil, err
	}
	answerBytes, err := conversation.Ask([]byte(prompt), &chatgpt.ConversationAskConfig{})
	if err != nil {
		return nil, err
	}
	answer := string(answerBytes)
	answers := map[string]string{}
	log.Println(fmt.Sprintf("chatgpt的提问为:%s结果为:%s", prompt, answer))
	for _, getKey := range getKeys {
		rx := regexp.MustCompile(fmt.Sprintf(`%s:(.*)`, getKey))
		// 在字符串中查找匹配项
		match := rx.FindStringSubmatch(answer)
		if len(match) == 2 {
			result := match[1]
			answers[getKey] = result
		} else {
			return answers, errors.New(fmt.Sprintf("chatgpt返回的格式不对:%s", answer))
		}
	}
	return answers, nil
}
