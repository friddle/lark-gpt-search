package feishu

import (
	"fmt"
	"github.com/chyroc/lark"
	"testing"
)

//memory的问题

func TestFeishuClient(test *testing.T) {
	feishuConfig := ReadFeishuConfig()
	feishuClient := client.NewFeishuClient(context.Background(), feishuConfig)
	ctx := context.Background()
	query := "IDP3.5部署手册"
	//query := "部署系统 设计"
	token := "u-cpYGwuIMF6dHNajzw55EXkh54XXxgh3bogw0gk.803d8"
	//friddle
	//ownerId := "ou_ded20bb705e6e7339a9abd8570b76d44"
	feishuClient.SetAuthToken(token)
	method := feishuClient.WithAuthToken()
	ownerId := "ou_2402f18326c96951792e802b4c1dbcb5"
	parser := client.NewParser(feishuClient.Ctx)
	//pageNums := int64(10)
	entiyRsp, _, err := feishuClient.LarkClient.Drive.SearchDriveFile(ctx, &lark.SearchDriveFileReq{
		SearchKey: query,
		OwnerIDs:  []string{ownerId},
	}, method)
	if err != nil {
		fmt.Print(err)
	} else {
		fmt.Print(entiyRsp)
	}

	contents := []string{}
	for index, entity := range entiyRsp.DocsEntities {
		if index < 10 {
			if entity.DocsType == "docx" {
				documents, block, _ := feishuClient.GetDocxContent(entity.DocsToken)
				contents = append(contents, parser.ParseDocxContent(documents, block))
			}
			if entity.DocsType == "doc" {
				documents, _ := feishuClient.GetDocContent(entity.DocsToken)
				contents = append(contents, parser.ParseDocContent(documents))
			}
			if entity.DocsType == "sheet" {
				//documents, _ := feishuClient.GetSheetClient(entity.DocsToken)
				//parser.ParseSheetContent(entity.DocsToken)
			}
		}
	}
	feishuClient.SendMessageToSomeOne("liaoyuandong@laiye.com", "need_auth")
	feishuClient.GetSheetClient()

}
