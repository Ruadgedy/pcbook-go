# 简单学习gRPC的使用

# 使用步骤
1. Makefile中定义了编译步骤，首先需要通过 `make gen`, 将proto文件翻译成go文件
2. `make server`与`make client` 分别启动服务器和客户端
> 注意：客户端中包含如下操作：
>       - 创建laptop（输入unary，输出unary）
>       - 查找laptop（输入unary，输出stream）
>       - 上传laptop图片（输入stream，输出unary）
>       - 给laptop打分（输入stream，输出stream）

具体操作见 `service/laptop_server.go`
server和client端的具体细节见 `cmd/client/main.go` 和 `cmd/server/main.go`
