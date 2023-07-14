# 此工具目标
模拟NewBing的搜索操作,搜索飞书内网文档，并返回结果

# 备注
没有用langchian。是go的langchain库(golangchian)等同于残废。
确实只有python的库比较好  

# 工具流程
0. 用户进行登录
1. 用户的输入进入ChatGPT进行翻译(辅助API文档)
2. 调用飞书的API进行搜索
3. 把结果分析给ChatGPT
4. 得出结果返回给用户

# 权限
p2p:chat:bot
待补充
...

# 配置
参考相应的  
.feishu.env  
.chatgpt.env   
