package feishu

import (
	"errors"
	"fmt"
	"github.com/chyroc/lark"
	"log"
)

//memory的问题

func (client *FeishuClient) SearchDocs(query string, count int64, ownerId []string, docTypes []string, options ...lark.MethodOptionFunc) (*lark.SearchDriveFileResp, error) {
	entityRsp, _, err := client.LarkClient.Drive.SearchDriveFile(client.Ctx, &lark.SearchDriveFileReq{
		SearchKey: query,
		OwnerIDs:  ownerId,
		DocsTypes: docTypes,
		Count:     &count,
	}, options...)
	if err != nil {
		return nil, err
	}
	return entityRsp, nil
}

// 以后加缓存功能
func (client *FeishuClient) GetFileContent(token string, docType string, title string, options ...lark.MethodOptionFunc) (string, error) {
	parser := NewParser(client.Ctx)
	if docType == "docx" {
		documents, block, _ := client.GetDocxContent(token, options...)
		return parser.ParseDocxContent(documents, block), nil
	}
	if docType == "doc" {
		documents, _ := client.GetDocContent(token, options...)
		return parser.ParseDocContent(documents), nil
	}
	if docType == "sheet" {
		//documents, _ := feishuClient.GetSheetClient(entity.DocsToken)
		//parser.ParseSheetContent(entity.DocsToken)
	}
	return "", errors.New(fmt.Sprintf("没有支持的格式,或者格式不正确 %s", token))

}

func (client *FeishuClient) SearchDocsWithResult(query string, count int64, ownerId []string, docTypes []string, options ...lark.MethodOptionFunc) (map[string]string, map[string]string, error) {
	entityRsp, err := client.SearchDocs(query, count, ownerId, docTypes, options...)
	if err != nil {
		return nil, nil, err
	}
	contents := make(map[string]string, 0)
	for index, entity := range entityRsp.DocsEntities {
		if index < 10 {
			rsp, err := client.GetFileContent(entity.DocsToken, entity.DocsType, entity.Title, options...)
			if err != nil {
				log.Printf(fmt.Sprintf("get doc type rsp:%v", rsp))
			}
			contents[entity.Title] = rsp
		}
	}
	return contents, make(map[string]string), nil
}

func (client *FeishuClient) GetBaike(entityIds []string, options ...lark.MethodOptionFunc) (map[string]lark.GetBaikeEntityResp, error) {
	entityRsp := make(map[string]lark.GetBaikeEntityResp)
	for _, entityId := range entityIds {
		baikeRsp, _, err := client.LarkClient.Baike.GetBaikeEntity(client.Ctx, &lark.GetBaikeEntityReq{
			EntityID: entityId,
		}, options...)
		if err != nil {
			log.Printf(fmt.Sprintf("get baike entity err:%v", err))
		}
		entityRsp[entityId] = *baikeRsp
	}
	return entityRsp, nil
}
