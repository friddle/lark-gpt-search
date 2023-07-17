package feishu

import (
	"context"
	"errors"
	"fmt"
	"github.com/chyroc/lark"
	"github.com/go-zoox/chatbot-feishu"
	"log"
	"net/url"
	"os"
	"time"
)

type FeishuClient struct {
	conf        *chatbot.Config
	Ctx         context.Context
	LarkClient  *lark.Lark
	RedirectUrl string
	UserTokens  map[string]*string
	CacheDir    string
}

func NewFeishuClient(ctx context.Context, conf *chatbot.Config) *FeishuClient {
	client := &FeishuClient{
		LarkClient: lark.New(
			lark.WithAppCredential(conf.AppID, conf.AppSecret),
			lark.WithOpenBaseURL("https://open.feishu.cn"),
			lark.WithTimeout(60*time.Second),
		),
		Ctx:         ctx,
		conf:        conf,
		RedirectUrl: os.Getenv("FEISHU_REDIRECT_URL") + os.Getenv("FEISHU_AUTH_PATH"),
		CacheDir:    os.Getenv("FEISHU_CACHE_FOLDER"),
		UserTokens:  make(map[string]*string),
	}
	if client.CacheDir == "" {
		client.CacheDir = "./cache"
	}
	return client
}

func (c FeishuClient) GenAuthToken(userId string) (lark.MethodOptionFunc, error) {
	token := c.UserTokens[userId]
	if token == nil {
		return nil, errors.New("not found user token")
	}
	return lark.WithUserAccessToken(*token), nil
}

func (c FeishuClient) IsAuthWork(userId string) bool {
	option, err := c.GenAuthToken(userId)
	if err != nil {
		log.Printf(fmt.Sprintf("auth error %v", err))
		return false
	}
	_, _, err = c.LarkClient.Auth.GetUserInfo(c.Ctx, &lark.GetUserInfoReq{}, option)
	//TODO: check is user work
	if err != nil {
		log.Printf(fmt.Sprintf("auth error %v", err))
		return false
	}
	return true
}

func (c *FeishuClient) SetAccessToken(userId string, token string) (error, string) {
	c.UserTokens[userId] = &token
	return nil, token
}

func (c *FeishuClient) SetAccessTokenByUrl(urlStr string) (error, string) {
	urlParse, _ := url.Parse(urlStr)
	code := urlParse.Query().Get("code")
	req := lark.GetAccessTokenReq{Code: code, GrantType: "authorization_code"}
	resp, _, err := c.LarkClient.Auth.GetAccessToken(c.Ctx, &req)
	if err != nil {
		return err, ""
	}
	log.Println(fmt.Sprintf("userId:%s,token:%s", resp.UserID, resp.AccessToken))
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
