package feishu

import (
	"errors"
	"feishu-gpt-search/src/utils"
	"fmt"
	"github.com/chyroc/lark"
	"log"
	"os"
)

//memory的问题

type SearchWikiFileReq struct {
	Query   string `json:"query,omitempty"`
	SpaceId string `json:"space_id,omitempty"`
	NodeId  string `json:"node_id,omitempty"`
}

type searchWikiFileResp struct {
	Code int64               `json:"code,omitempty"`
	Msg  string              `json:"msg,omitempty"`
	Data *SearchWikiFileResp `json:"data,omitempty"`
}

type SearchWikiFileRespItemsEntity struct {
	NodeID  string `json:"node_id"`
	SpaceID string `json:"space_id"`
	ObjType int64  `json:"obj_type"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Icon    string `json:"icon"`
}

type SearchWikiFileResp struct {
	Items     []*SearchWikiFileRespItemsEntity `json:"items,omitempty"`      // 搜索匹配文档列表
	HasMore   bool                             `json:"has_more,omitempty"`   // 搜索偏移位结果列表后是否还有数据
	PageToken string                           `json:"page_token,omitempty"` // 搜索匹配文档总数量
}

func newMethodOption(options []lark.MethodOptionFunc) *lark.MethodOption {
	opt := new(lark.MethodOption)
	for _, v := range options {
		v(opt)
	}
	return opt
}

func (client *FeishuClient) SearchDocs(query string, count int64, ownerId []string, docTypes []string, options ...lark.MethodOptionFunc) ([]*SearchWikiFileRespItemsEntity, error) {
	request := &SearchWikiFileReq{
		Query: query,
	}
	req := &lark.RawRequestReq{
		Scope:                 "Wiki",
		API:                   "NodesSearch",
		Method:                "POST",
		URL:                   "https://open.feishu.cn/open-apis/wiki/v1/nodes/search",
		Body:                  request,
		MethodOption:          newMethodOption(options),
		NeedTenantAccessToken: true,
		NeedUserAccessToken:   true,
	}
	entityRsp := new(searchWikiFileResp)
	_, err := client.LarkClient.RawRequest(client.Ctx, req, entityRsp)
	if err != nil {
		return nil, err
	}
	return entityRsp.Data.Items, nil
}

func (client *FeishuClient) GetFileContent(token string, docType string, title string, options ...lark.MethodOptionFunc) (string, error) {
	cachePath := fmt.Sprintf("%s/%s.%s.txt", "cache", token, docType)
	if utils.Exists(cachePath) {
		text, _ := os.ReadFile(cachePath)
		if string(text) == "" {
			os.Remove(cachePath)
		} else {
			return string(text), nil
		}
	}
	content, err := client.getFileContentByApi(token, docType, title, options...)
	if err != nil {
		return content, nil
	}
	os.MkdirAll("cache", os.ModePerm)
	os.WriteFile(cachePath, []byte(content), os.ModePerm)
	return content, nil
}

// 以后加缓存功能
func (client *FeishuClient) getFileContentByApi(token string, docType string, title string, options ...lark.MethodOptionFunc) (string, error) {

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
	for index, entity := range entityRsp {
		if index < 10 {
			node, err := client.GetWikiNodeInfo(entity.NodeID, options...)
			rsp, err := client.GetFileContent(node.ObjToken, node.ObjType, node.Title, options...)
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
