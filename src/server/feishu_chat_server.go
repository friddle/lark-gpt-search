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
		context, _ := searchClient.GetContext(request.Event.Message.RootID, request.Event.Sender.SenderID.UserID)
		searchClient.Search(context, text, func(isAuth bool, content string, links []string, err error) {
			if !isAuth {
				authHandler([]string{}, request, reply)
			}
			ReplyText(reply, content)
			if len(links) != 0 {
				ReplyText(reply, strings.Join(links, " "))
			}
		})
		return nil
	})
	return bot, nil
}
