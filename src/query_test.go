package t_test

import (
	"feishu-gpt-search/src/server"
	"fmt"
	"github.com/chyroc/lark"
	"github.com/go-zoox/logger"
	"regexp"
	"strings"
	"testing"
)

func Test_TextParser(t *testing.T) {
	argsMap := map[string]string{}
	text := "laipvt怎么安装 --searchKey=LAIPVT详细地址"
	args := strings.Split(text, " ")
	for _, arg := range args {
		if strings.Contains(arg, "--") {
			argItem := strings.Split(arg, "=")
			if len(argItem) != 2 {
				panic("info")
			}
			argsMap[argItem[0]] = argItem[1]
		}
	}
	re := regexp.MustCompile(`--\w+=\w+`)
	text = re.ReplaceAllString(text, "")
	logger.Info(fmt.Sprintf("text:%s", text))

}

func Test_Extract(t *testing.T) {
	rx := regexp.MustCompile(fmt.Sprintf(`%s:(.*)`, "关键词"))
	text := "关键词: laipvt; 最新; 地址"
	// 在字符串中查找匹配项
	match := rx.FindStringSubmatch(text)
	logger.Infof("%+v", match)
}

func Test_SendMessage(t *testing.T) {
	feishuClient, _, _ := GetInitClient()
	userId := "164981201"
	reply := func(content string, msgType ...string) error {
		rsp1, rsp2, err := feishuClient.LarkClient.Message.SendRawMessage(feishuClient.Ctx, &lark.SendRawMessageReq{
			Content:       content,
			MsgType:       lark.MsgType(msgType[0]),
			ReceiveID:     userId,
			ReceiveIDType: lark.IDTypeUserID,
		},
		)
		logger.Infof("%v %v %v", rsp1, rsp2, err)
		return err
	}
	server.ReplyTextWithLinks(reply, "hellowrod", map[string]string{"A": "http://www.baidu.com"}, map[string]string{})

}

func Test_SheetParser(t *testing.T) {
	feishuClient, _, _ := GetInitClient()
	userId := "164981201"
	token := "u-cQqS7FGTt7dVjkTlu2jsdr1k3KQ41h13qo0055c022g_"
	feishuClient.SetAccessToken(userId, token)
	option, _ := feishuClient.GenAuthToken(userId)
	text, _ := feishuClient.GetFileContentByApi("shtcngoRXRRplvSVuMeVli8hGRc", "sheet", "IDP服务进程说明", option)
	print(text)

}
