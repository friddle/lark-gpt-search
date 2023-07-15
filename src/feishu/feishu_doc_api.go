package feishu

import (
	"encoding/json"
	"fmt"
	"github.com/chyroc/lark"
	"io"
	"os"
	"path/filepath"
)

func (c *FeishuClient) GetDocContent(docToken string, options ...lark.MethodOptionFunc) (*lark.DocContent, error) {
	resp, _, err := c.LarkClient.Drive.GetDriveDocContent(c.Ctx, &lark.GetDriveDocContentReq{
		DocToken: docToken,
	}, options...)
	if err != nil {
		return nil, err
	}
	doc := &lark.DocContent{}
	err = json.Unmarshal([]byte(resp.Content), doc)
	if err != nil {
		return doc, err
	}

	return doc, nil
}

func (c *FeishuClient) GetSheetDoc(docToken string, options ...lark.MethodOptionFunc) ([]*lark.GetSheetRespSheet, error) {
	metaRsp, _, _ := c.LarkClient.Drive.GetSheetMeta(c.Ctx, &lark.GetSheetMetaReq{
		SpreadSheetToken: docToken,
	})
	sheetContents := []*lark.GetSheetRespSheet{}
	for _, sheet := range metaRsp.Sheets {
		resp, _, err := c.LarkClient.Drive.GetSheet(c.Ctx, &lark.GetSheetReq{
			SpreadSheetToken: docToken,
			SheetID:          sheet.SheetID,
		}, options...)
		if err != nil {
			continue
		}
		sheetContents = append(sheetContents, resp.Sheet)
	}

	return sheetContents, nil

}

func (c *FeishuClient) DownloadImage(imgToken string, options ...lark.MethodOptionFunc) (string, error) {
	resp, _, err := c.LarkClient.Drive.DownloadDriveMedia(c.Ctx, &lark.DownloadDriveMediaReq{
		FileToken: imgToken,
	}, options...)
	if err != nil {
		return imgToken, err
	}
	imgDir := c.CacheDir + "/img"
	fileext := filepath.Ext(resp.Filename)
	filename := fmt.Sprintf("%s/%s%s", imgDir, imgToken, fileext)
	err = os.MkdirAll(filepath.Dir(filename), 0o755)
	if err != nil {
		return imgToken, err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return imgToken, err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.File)
	if err != nil {
		return imgToken, err
	}
	return filename, nil
}

func (c *FeishuClient) GetDocxContent(docToken string, options ...lark.MethodOptionFunc) (*lark.DocxDocument, []*lark.DocxBlock, error) {
	resp, _, err := c.LarkClient.Drive.GetDocxDocument(c.Ctx, &lark.GetDocxDocumentReq{
		DocumentID: docToken,
	}, options...)
	if err != nil {
		return nil, nil, err
	}
	docx := &lark.DocxDocument{
		DocumentID: resp.Document.DocumentID,
		RevisionID: resp.Document.RevisionID,
		Title:      resp.Document.Title,
	}
	var blocks []*lark.DocxBlock
	var pageToken *string
	for {
		resp2, _, err := c.LarkClient.Drive.GetDocxBlockListOfDocument(c.Ctx, &lark.GetDocxBlockListOfDocumentReq{
			DocumentID: docx.DocumentID,
			PageToken:  pageToken,
		}, options...)
		if err != nil {
			return docx, nil, err
		}
		blocks = append(blocks, resp2.Items...)
		pageToken = &resp2.PageToken
		if !resp2.HasMore {
			break
		}
	}

	return docx, blocks, nil
}

func (c *FeishuClient) GetWikiNodeInfo(token string, options ...lark.MethodOptionFunc) (*lark.GetWikiNodeRespNode, error) {
	resp, _, err := c.LarkClient.Drive.GetWikiNode(c.Ctx, &lark.GetWikiNodeReq{
		Token: token,
	}, options...)
	if err != nil {
		return nil, err
	}
	return resp.Node, nil
}

func (c *FeishuClient) GetWikiNodeList(spaceId string, parentNodeToken *string, pageToken *string, options ...lark.MethodOptionFunc) ([]*lark.GetWikiNodeListRespItem, *string, error) {
	size := int64(50)
	resp, _, err := c.LarkClient.Drive.GetWikiNodeList(c.Ctx, &lark.GetWikiNodeListReq{
		SpaceID:         spaceId,
		PageSize:        &size,
		PageToken:       pageToken,
		ParentNodeToken: parentNodeToken,
	}, options...)
	if err != nil {
		return nil, nil, err
	}
	return resp.Items, nil, nil
}

func TestDocuments() {
	//u-cpYGwuIMF6dHNajzw55EXkh54XXxgh3bogw0gk.803d8

}
