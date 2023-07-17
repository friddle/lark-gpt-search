package server

import (
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/llama"
	"fmt"
	"github.com/chyroc/lark"
	"github.com/go-zoox/chatbot-feishu"
	regexp2 "github.com/go-zoox/core-utils/regexp"
	"github.com/go-zoox/feishu/contact/user"
	"github.com/go-zoox/feishu/event"
	feishuEvent "github.com/go-zoox/feishu/event"
	mc "github.com/go-zoox/feishu/message/content"
	"github.com/go-zoox/logger"
	"regexp"
	"strings"
)

func getUser(request *feishuEvent.EventRequest) (*user.RetrieveResponse, error) {
	sender := request.Sender()
	return &user.RetrieveResponse{
		User: user.UserEntity{
			Name:    sender.SenderID.UserID,
			OpenID:  sender.SenderID.OpenID,
			UnionID: sender.SenderID.UnionID,
			UserID:  sender.SenderID.UserID,
		},
	}, nil
}

func getText(client *feishu.FeishuClient, text string, request *feishuEvent.EventRequest) string {
	var newText string
	// group chat
	if request.IsGroupChat() {
		botRsp, _, _ := client.LarkClient.Bot.GetBotInfo(client.Ctx, &lark.GetBotInfoReq{})
		if ok := regexp2.Match("^@_user_1", text); ok {
			for _, mention := range request.Event.Message.Mentions {
				if mention.Key == "@_user_1" && mention.ID.OpenID == botRsp.OpenID {
					newText = strings.Replace(text, "@_user_1", "", 1)
					logger.Info("chat command %s", text)
					break
				}
			}
		}
	} else if request.IsP2pChat() {
		newText = text
	}
	logger.Info("question get  %s", newText)
	return newText
}

func ReplyText(reply func(context string, msgType ...string) error, text string) error {
	if text == "" {
		text = "服务没有返回"
	}

	msgType, content, err := mc.
		NewContent().
		Post(&mc.ContentTypePost{
			ZhCN: &mc.ContentTypePostBody{
				Content: [][]mc.ContentTypePostBodyItem{
					{
						{
							Tag:      "text",
							UnEscape: false,
							Text:     text,
						},
					},
				},
			},
		}).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build content: %v", err)
	}
	if err := reply(string(content), msgType); err != nil {
		logger.Info(fmt.Sprintf("failed to reply: %v", err))
	}
	return nil
}

func ReplyTextWithLinks(reply func(context string, msgType ...string) error, text string, links map[string]string, questions map[string]string) error {
	if text == "" {
		text = "服务没有返回"
	}
	contentBody := &mc.ContentTypePost{
		ZhCN: &mc.ContentTypePostBody{
			Title: "返回中",
			Content: [][]mc.ContentTypePostBodyItem{
				{
					{
						Tag:      "text",
						UnEscape: false,
						Text:     text,
					},
				},
			},
		},
	}
	if len(links) > 0 {
		linkItems := make([]mc.ContentTypePostBodyItem, 0)
		for title, link := range links {
			linkItems = append(linkItems, mc.ContentTypePostBodyItem{
				Tag:  "a",
				Text: "参考:" + title + "\r\n",
				Href: "" + string(link),
			})
			logger.Info(fmt.Sprintf("text %s link %s", title, link))
		}
		contentBody.ZhCN.Content = append(contentBody.ZhCN.Content, linkItems)
	}
	if len(questions) > 0 {
		linkItems := make([]mc.ContentTypePostBodyItem, 0)
		for title, _ := range questions {
			linkItems = append(linkItems, mc.ContentTypePostBodyItem{
				Tag:  "text",
				Text: title,
			})
		}
		contentBody.ZhCN.Content = append(contentBody.ZhCN.Content, linkItems)
	}
	msgType, content, err := mc.NewContent().Post(contentBody).Build()
	if err != nil {
		return fmt.Errorf("failed to build content: %v", err)
	}
	if err := reply(string(content), msgType); err != nil {
		logger.Info(fmt.Sprintf("failed to reply: %v", err))
	}

	return nil
}

func FeishuServer(feishuConf *chatbot.Config, searchClient *llama.SearchChainClient, feishuClient *feishu.FeishuClient) (chatbot.ChatBot, error) {
	bot, err := chatbot.New(feishuConf)
	if err != nil {
		logger.Errorf("failed to create bot: %v", err)
		return nil, err
	}

	bot.OnCommand("ping", &chatbot.Command{
		Handler: func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
			if err := ReplyText(reply, "pong"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}
			return nil
		},
	})

	bot.OnCommand("help", &chatbot.Command{
		Handler: func(args []string, request *event.EventRequest, reply chatbot.MessageReply) error {
			helpText := ""
			if err := ReplyText(reply, helpText); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}
			return nil
		},
	})

	authHandler := func(args []string, request *feishuEvent.EventRequest, reply func(content string, msgType ...string) error) error {
		user, err := getUser(request)
		if err != nil {
			if err := ReplyText(reply, "系统故障无法使用。用户信息缺失"); err != nil {
				return fmt.Errorf("failed to reply: %v", err)
			}
		}
		genAuth := lark.GenOAuthURLReq{RedirectURI: feishuClient.RedirectUrl, State: user.User.UserID}
		genRsp := feishuClient.LarkClient.Auth.GenOAuthURL(feishuClient.Ctx, &genAuth)
		response := fmt.Sprintf("请点击链接进行授权: %s,授权完成后请重新输入", genRsp)
		if err := ReplyText(reply, response); err != nil {
			return fmt.Errorf("failed to reply: %v", err)
		}
		return nil
	}

	bot.OnCommand("auth", &chatbot.Command{
		Handler: authHandler,
	})

	bot.OnCommand("token", &chatbot.Command{
		Handler: func(args []string, request *event.EventRequest, reply chatbot.MessageReply) error {
			logger.Info("set auth token %s %s", request.Event.Sender.SenderID.UserID, args[0])
			feishuClient.SetAccessToken(request.Event.Sender.SenderID.UserID, args[0])
			ReplyText(reply, "set auth token ok")
			return nil
		},
	})

	bot.OnMessage(func(text string, request *event.EventRequest, reply chatbot.MessageReply) error {
		question := getText(feishuClient, text, request)
		if question == "" {
			return nil
		}
		if strings.HasPrefix(question, "/") {
			logger.Infof("ignore empty command message")
			return nil
		}

		argsMap := map[string]string{}
		args := strings.Split(question, " ")
		for _, arg := range args {
			if strings.Contains(arg, "--") {
				argItem := strings.Split(strings.Replace(arg, "--", "", -1), "=")
				if len(argItem) != 2 {
					ReplyText(reply, fmt.Sprintf("参数错误 %s,参数设置应该为 --searchKey=xxx", arg))
					return nil
				}
				argsMap[argItem[0]] = argItem[1]
			}
		}
		re := regexp.MustCompile(`--\w+=\w+`)
		questionPure := re.ReplaceAllString(question, "")
		context, _ := searchClient.GetContext(request.Event.Message.RootID, request.Event.Sender.SenderID.UserID, argsMap)
		searchClient.Search(context, questionPure, func(isAuth bool, content string, question map[string]string, links map[string]string, err error) {
			if !isAuth {
				authHandler([]string{}, request, reply)
			}
			ReplyTextWithLinks(reply, content, links, question)
		})
		return nil
	})
	return bot, nil
}
