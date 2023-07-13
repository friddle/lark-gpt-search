package server

import (
	"feishu-gpt-search/src/feishu"
	"feishu-gpt-search/src/llama"
	"fmt"
	"github.com/chyroc/lark"
	"github.com/go-zoox/chatbot-feishu"
	"github.com/go-zoox/feishu/contact/user"
	"github.com/go-zoox/feishu/event"
	feishuEvent "github.com/go-zoox/feishu/event"
	mc "github.com/go-zoox/feishu/message/content"
	"github.com/go-zoox/logger"
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
				Tag:  "link",
				Text: title,
				Href: link,
			})
		}
		contentBody.ZhCN.Content = append(contentBody.ZhCN.Content, linkItems)
	}
	if len(questions) > 0 {
		linkItems := make([]mc.ContentTypePostBodyItem, 0)
		for title, link := range questions {
			linkItems = append(linkItems, mc.ContentTypePostBodyItem{
				Tag:  "link",
				Text: title,
				Href: link,
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
		response := fmt.Sprintf("请点击链接进行授权: %s", genRsp)
		if err := ReplyText(reply, response); err != nil {
			return fmt.Errorf("failed to reply: %v", err)
		}
		return nil
	}

	bot.OnCommand("auth", &chatbot.Command{
		Handler: authHandler,
	})

	bot.OnMessage(func(text string, request *event.EventRequest, reply chatbot.MessageReply) error {
		if strings.HasPrefix(text, "/") {
			logger.Infof("ignore empty command message")
			return nil
		}
		argsMap := map[string]string{}
		args := strings.Split(text, " ")
		for _, arg := range args {
			if strings.Contains("--", arg) {
				argItem := strings.Split(arg, "--")
				if len(argItem) != 2 {
					ReplyText(reply, fmt.Sprintf("参数错误 %s", arg))
				}
				argsMap[argItem[0]] = argItem[1]
			}
		}
		context, _ := searchClient.GetContext(request.Event.Message.RootID, request.Event.Sender.SenderID.UserID, argsMap)
		searchClient.Search(context, text, func(isAuth bool, content string, question map[string]string, links map[string]string, err error) {
			if !isAuth {
				authHandler([]string{}, request, reply)
			}
			ReplyTextWithLinks(reply, content, links, question)
		})
		return nil
	})
	return bot, nil
}
