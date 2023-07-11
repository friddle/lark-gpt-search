package config

import (
	"feishu-gpt-search/src/utils"
	"github.com/go-zoox/chatbot-feishu"
	chatgpt "github.com/go-zoox/chatgpt-client"
	"github.com/go-zoox/logger"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

var logs = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

func ReadFeishuConfig() *chatbot.Config {
	//假如有文件。读取文件
	if utils.Exists(".feishu.env") {
		err := godotenv.Load(".feishu.env")
		if err != nil {
			logger.Fatal("read err %v", err)
		}
		logger.Info("load local feishu.env files")
	}

	port, _ := strconv.Atoi(os.Getenv("FEISHU_BOT_PORT"))
	//然后判断文件
	conf := chatbot.Config{
		AppID:             os.Getenv("FEISHU_APP_ID"),
		AppSecret:         os.Getenv("FEISHU_APP_SECRET"),
		EncryptKey:        os.Getenv("FEISHU_ENCRYPT_KEY"),
		VerificationToken: os.Getenv("FEISHU_VERIFICATION_TOKEN"),
		Port:              int64(port),
		Path:              os.Getenv("FEISHU_BOT_PATH"),
	}
	if conf.AppID == "" || conf.AppSecret == "" {
		logs.Fatalln("请配置APP_ID和APP_SECRET")
		os.Exit(2)
	}
	logs.Printf("配置读取成功 %v", conf)
	return &conf
}

func ReadChatGptClient() *chatgpt.Config {
	if utils.Exists(".chatgpt.env") {
		err := godotenv.Load(".chatgpt.env")
		if err != nil {
			logs.Fatalf("read err %v", err)
		}
		logs.Println("load local chatgpt.env files")
	}
	//然后判断文件
	conf := chatgpt.Config{
		APIKey:               os.Getenv("CHATGPT_API_KEY"),
		APIServer:            os.Getenv("CHATGPT_API_SERVER"),
		APIType:              os.Getenv("CHATGPT_API_TYPE"),
		AzureResource:        os.Getenv("CHATGPT_AZURE_RESOURCE"),
		AzureDeployment:      os.Getenv("CHATGPT_AZURE_DEPLOYMENT"),
		AzureAPIVersion:      os.Getenv("CHATGPT_AZURE_API_VERSION"),
		ConversationContext:  os.Getenv("CHATGPT_CONVERSATION_CONTEXT"),
		ConversationLanguage: os.Getenv("CHATGPT_CONVERSATION_LANGUAGE"),
		ChatGPTName:          os.Getenv("CHATGPT_NAME"),
		Proxy:                os.Getenv("CHATGPT_PROXY"),
	}
	if conf.APIKey == "" {
		return nil
	}
	return &conf
}
