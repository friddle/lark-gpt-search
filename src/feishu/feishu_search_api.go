package feishu

import (
	"errors"
	"feishu-gpt-search/src/utils"
	"fmt"
	"github.com/chyroc/lark"
	"log"
	"os"
	"strings"
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

type SearchOptions struct {
	DocTypes []string
	Exclude  []string
	Wiki     bool
	Self     bool
	Count    int64
	Offset   int64
}

type FeishuDocMeta struct {
	DocsToken string `json:"docs_token,omitempty"` // 文档token
	DocsType  string `json:"docs_type,omitempty"`  // 文档类型
	Title     string `json:"title,omitempty"`      // 标题
	OwnerID   string `json:"owner_id,omitempty"`   // 文件所有者
	NodeID    string `json:"node_id"`
	SpaceID   string `json:"space_id"`
	URL       string `json:"url"`
	SheetId   string `json:"sheet_id"`
	isWiki    bool
}

func newMethodOption(options []lark.MethodOptionFunc) *lark.MethodOption {
	opt := new(lark.MethodOption)
	for _, v := range options {
		v(opt)
	}
	return opt
}

func (client *FeishuClient) SearchWikiDocs(query string, userId string, option SearchOptions, options ...lark.MethodOptionFunc) ([]*FeishuDocMeta, error) {
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
	entitys := []*FeishuDocMeta{}
	for _, item := range entityRsp.Data.Items {
		node, err := client.GetWikiNodeInfo(item.NodeID, options...)
		if err != nil {
			continue
		}
		if option.Self {
			if node.Owner != userId {
				continue
			}
		}
		if len(option.DocTypes) > 0 {
		}

		if len(option.Exclude) > 0 {
			for _, entity := range entitys {
				for _, execlude := range option.Exclude {
					if strings.Contains(entity.Title, execlude) {
						continue
					}
				}
			}

		}
		entitys = append(entitys, &FeishuDocMeta{})
	}
	return entitys, nil
}

func (client *FeishuClient) SearchDriveDocs(query string, userId string, option SearchOptions, options ...lark.MethodOptionFunc) ([]*FeishuDocMeta, error) {
	req := &lark.SearchDriveFileReq{
		SearchKey: query,
		Count:     &option.Count,
		Offset:    &option.Offset,
		DocsTypes: option.DocTypes,
	}
	if option.Self {
		req.OwnerIDs = append([]string{}, userId)
	}
	rsp, _, err := client.LarkClient.Drive.SearchDriveFile(client.Ctx, req, options...)
	if err != nil {
		return nil, err
	}
	entitys := []*FeishuDocMeta{}
	for _, entity := range rsp.DocsEntities {
		// 删除所有包含关键字的文档
		if len(option.Exclude) > 0 {
			for _, entity := range entitys {
				for _, execlude := range option.Exclude {
					if strings.Contains(entity.Title, execlude) {
						continue
					}
				}
			}
		}
		entitys = append(entitys, &FeishuDocMeta{
			DocsToken: entity.DocsToken,
			DocsType:  entity.DocsType,
			Title:     entity.Title,
			OwnerID:   entity.OwnerID,
		})
	}

	return entitys, nil
}

func (client *FeishuClient) GetFileContent(token string, docType string, title string, options ...lark.MethodOptionFunc) (string, error) {
	cachePath := fmt.Sprintf("%s/%s.%s.txt", client.CacheDir, token, docType)
	if utils.Exists(cachePath) {
		text, _ := os.ReadFile(cachePath)
		if string(text) == "" {
			os.Remove(cachePath)
		} else {
			return string(text), nil
		}
	}
	content, err := client.GetFileContentByApi(token, docType, title, options...)
	if err != nil {
		return content, nil
	}
	os.MkdirAll("cache", os.ModePerm)
	os.WriteFile(cachePath, []byte(content), os.ModePerm)
	return content, nil
}

// 以后加缓存功能
func (client *FeishuClient) GetFileContentByApi(token string, docType string, title string, options ...lark.MethodOptionFunc) (string, error) {
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
		sheetContent, _ := client.GetSheetDoc(token, options...)
		parser.ParseSheetContent(sheetContent)
	}

	return "", errors.New(fmt.Sprintf("没有支持的格式,或者格式不正确 %s", token))

}

func (client *FeishuClient) SearchDocsWithResult(query string, userId string, option SearchOptions, options ...lark.MethodOptionFunc) (map[string]string, map[string]string, error) {
	log.Printf(fmt.Sprintf("search doc query:%s", query))
	var entityRsp []*FeishuDocMeta
	var err error
	if option.Wiki {
		entityRsp, err = client.SearchWikiDocs(query, userId, option, options...)
	} else {
		entityRsp, err = client.SearchDriveDocs(query, userId, option, options...)
	}
	linkMap := make(map[string]string)
	if err != nil {
		return nil, nil, err
	}
	contents := make(map[string]string, 0)
	for _, entity := range entityRsp {
		rsp, err := client.GetFileContent(entity.DocsToken, entity.DocsType, entity.Title, options...)
		linkMap[entity.Title] = entity.URL
		if err != nil {
			log.Printf(fmt.Sprintf("get doc type rsp:%v", rsp))
			continue
		}
		contents[entity.Title] = rsp
	}
	return contents, linkMap, nil
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
