package feishu

import (
	"context"
	"errors"
	"github.com/chyroc/lark"
	"github.com/go-zoox/chatbot-feishu"
	"log"
	"net/url"
	"time"
)

type FeishuClient struct {
	conf        *chatbot.Config
	Ctx         context.Context
	LarkClient  *lark.Lark
	RedirectUrl string
	UserTokens  map[string]*string
	ImgDir      string
}

func NewFeishuClient(ctx context.Context, conf *chatbot.Config) *FeishuClient {
	return &FeishuClient{
		LarkClient: lark.New(
			lark.WithAppCredential(conf.AppID, conf.AppSecret),
			lark.WithOpenBaseURL("https://open.feishu.cn"),
			lark.WithTimeout(60*time.Second),
		),
		Ctx:        ctx,
		conf:       conf,
		UserTokens: make(map[string]*string),
	}
}

func (c FeishuClient) GenAuthToken(userId string) (lark.MethodOptionFunc, error) {
	token := c.UserTokens[userId]
	if token == nil {
		return nil, errors.New("not found user token")
	}
	return lark.WithUserAccessToken(*token), nil
}

func (c FeishuClient) WithAuthToken() {

}

func (c *FeishuClient) GetAccessToken(urlStr string) (error, string) {
	urlParse, _ := url.Parse(urlStr)
	code := urlParse.Query().Get("code")
	req := lark.GetAccessTokenReq{Code: code, GrantType: "authorization_code"}
	resp, _, err := c.LarkClient.Auth.GetAccessToken(c.Ctx, &req)
	if err != nil {
		return err, ""
	}
	c.UserTokens[resp.UserID] = &resp.AccessToken
	return nil, resp.AccessToken
}

func (c *FeishuClient) SendMessageToSomeOne(mail string, msg string) error {
	req := lark.SendRawMessageReq{
		ReceiveID:     mail,
		ReceiveIDType: lark.IDTypeEmail,
		MsgType:       lark.MsgTypeText,
		Content:       msg,
	}
	_, _, err := c.LarkClient.Message.SendRawMessage(c.Ctx, &req)
	if err != nil {
		log.Printf("send msg error:%v", err)
		return err
	}
	return nil
}
