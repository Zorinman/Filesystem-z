# ⭐以下所有功能都将整合到每个微服务中单独实现（见微服务架构解析以及`service`文件夹）



### DoUploadHandler ： 处理post文件上传

**过程**：客户端上传 → 内存缓冲区(buf) → 构建文件元信息 → 本地临时文件(newFile) → io.ReadAll(newFile)存储到data变量 → Ceph存储/OSS存储 → 更新元数据与用户文件记录

**问题**：这里将文件内容从 file 句柄读取到内存缓冲区 buf如何是大文件会导致内存爆炸（如上传 1GB 文件会占用 1GB 内存）  

**优化建议**：流式处理（直接写入磁盘或云存储）

**例：跳过内存缓冲区(buf) 、本地临时文件(newFile)，io.ReadAll(newFile)存储到data变量，从HTTP请求直接流式上传到ceph存储**

中间的元信息构建等不再赘述
```go
file, head, err := c.Request.FormFile("file")
if err != nil {
    // 处理错误
}
defer file.Close()

// 直接上传文件流，避免内存缓存
_, err = bucket.PutReader(cephPath, file, head.Size, "octet-stream", s3.PublicRead)
if err != nil {
    // 处理错误
}
```


### DownloadHandler ： 处理文件下载


**过程** 

客户端请求下载文件 → 

获取请求表单的文件hash值以及查询用户的用户名 →

 `meta.GetFileMetaDB`检查数据库是否有该文件 → 
 
 `dblayer.QueryUserFileMeta`检查用户是否关联了该文件（是否有权下载该文件）→ 
 
 
 `strings.HasPrefix`判断文件元信息来决定从哪里下载→
 
 从本地或者ceph/OSS获取文件内容→
 
 设置强制浏览器将服务端返回的内容视为附件下载→
 
 服务端返回内容响应给浏览器→
 
 用户下载

### FileMetaUpdateHandler ： 更新元信息接口(重命名)

客户端请求修改文件信息→

服务端获取HTTP请求中的各项内容(用户名，新的文件名，文件哈希值，op值)→

判断新文件名以及op字段的值是否合规→

`dblayer.RenameFileName`更新数据库中用户文件表tbl_user_file中的文件名的文件内容→

`dblayer.QueryUserFileMeta`从数据库获取更新后的文件信息，并通过`c.JSON`转换为JSON格式返回响应内容给客户端

**op字段** :
op字段有多种用途，如操作类型标识，权限控制标志，多步骤流程控制

这里是操作类型标识：
作用：区分客户端请求的具体操作（如增删改查）。

示例值：

"0" → 更新操作

"1" → 删除操作

"2" → 重命名操作


### FileDeleteHandler ：  删除文件及元信息

客户端请求删除文件信息→

服务端获取HTTP请求中的各项内容（哈希值，用户名）→

`meta.GetFileMetaDB`通过哈希值从数据库中获取文件信息→

`os.Remove`通过获取的文件信息从本地删除该文件→
TODO: 可考虑删除Ceph/OSS上的文件

`dblayer.DeleteUserFile`删除数据库文件表中关于这个文件的记录→

`c.Status`返回成功删除的响应给客户端


### UploadHandler ： 响应上传页面

**过程**：

客户端请求上传页面 → 

服务端通过`os.ReadFile`读取本地文件`static/view/index.html`的内容 → 

如果文件读取失败，返回状态码`404`，并通过`c.String`返回错误信息`"网页不存在"` → 

如果文件读取成功，通过`c.Data`返回状态码`200`，并将HTML内容以`text/html; charset=utf-8`的格式返回给客户端，展示上传页面。


**错误处理**：

1. 如果读取文件`static/view/index.html`失败，返回状态码`404`，并通过`c.String`返回错误信息`"网页不存在"`。

### TryFastUploadHandler ： 尝试秒传接口（场景：其它用户已经上传过相同的文件）

**过程**：

1. 客户端请求秒传接口，上传文件的相关信息（用户名、文件哈希值、文件名、文件大小）。
2. 服务端通过`meta.GetFileMetaDB`从数据库中查询是否存在相同哈希值的文件记录。
3. 如果查询不到记录，返回秒传失败的响应，提示客户端访问普通上传接口。
4. 如果查询到记录，调用`dblayer.OnUserFileUploadFinished`将文件信息写入该用户文件表。
5. 根据写入结果返回秒传成功或失败的响应。

**优点**：

- 避免重复上传相同文件，节省带宽和存储资源。
- 提高用户体验，秒传成功时响应速度快。

**错误处理**：

1. 如果数据库查询文件元信息失败，返回状态码`500`。
2. 如果秒传失败，返回状态码`200`，并通过JSON格式返回失败信息。

### FileQueryHandler ： 查询批量文件元信息并展示在网页端

**过程**：

1. 用户登录后，客户端请求查询接口，传递用户名和查询限制数量（limit）。
2. 服务端通过`dblayer.QueryUserFileMetas`查询数据库中该与该用户的关联的所有文件元信息。
3. 如果查询失败，返回状态码`500`，并通过JSON格式返回错误信息`"Query failed!"`。
4. 如果查询成功，将查询结果序列化为JSON格式。
5. 返回状态码`200`，并将JSON数据作为响应内容返回给客户端进行文件列表展示。

**错误处理**：

1. 如果数据库查询失败，返回状态码`500`，并通过JSON格式返回错误信息。
2. 如果JSON序列化失败，返回状态码`500`，并通过JSON格式返回错误信息。

