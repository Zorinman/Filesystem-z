> 本文本用于回答与该项目相关的问题
# 项目中Redis和Mysql分别完成了哪些功能？是如何完成的？

### 1. Redis 的核心功能与实现
Redis 在本项目中主要作为 **高性能的中间状态缓存**，专门用于处理 **分块上传（Multipart Upload）** 的临时状态管理。

*   **功能定位**：分块上传的状态记录与协调。
*   **具体实现逻辑**：
    *   **初始化上传 (`InitialMultipartUploadHandler`)**：
        *   生成唯一的 `UploadID`。
        *   使用 Redis Hash 结构存储上传任务的元数据。
        *   **Key**: `MP_<UploadID>`
        *   **Fields**:
            *   `chunkcount`: 分块总数
            *   `filehash`: 文件哈希
            *   `filesize`: 文件总大小
    *   **上传分块 (`UploadPartHandler`)**：
        *   每当一个分块上传成功，就在对应的 Hash 中更新该分块的状态。
        *   **Key**: `MP_<UploadID>`
        *   **Field**: `chkidx_<index>` 设置为 `1`（标记该索引的分块已完成）。
    *   **合并分块 (`CompleteUploadHandler`)**：
        *   获取 Hash 中所有的 `chkidx_` 字段，统计已上传的分块数量。
        *   对比已上传数量与 `chunkcount` 是否一致，若一致则触发合并操作，随后写入 MySQL 并清理 Redis 数据。

### 2. MySQL 的核心功能与实现
MySQL 在本项目中作为 **核心持久化存储**，负责存储所有的业务关系数据和元数据。

*   **功能定位**：元数据存储、用户关系管理、文件索引。
*   **数据表设计与功能**：
    *   **文件唯一表 (`tbl_file`)**：
        *   **核心字段**：`file_sha1` (主键), `file_addr` (存储路径), `file_size`。
        *   **作用**：实现**文件秒传**和**去重**的核心。无论多少用户上传同一个文件，物理存储中只保存一份，数据库中只存一条记录。
    *   **用户文件表 (`tbl_user_file`)**：
        *   **核心字段**：`user_name`, `file_sha1`, `file_name` (用户自定义文件名)。
        *   **作用**：建立“用户”与“文件”的映射关系。不同用户可以拥有同一个文件（指向相同的 `file_sha1`），但可以重命名为不同的 `file_name`，互不影响。
    *   **用户表 (`tbl_user`)**：
        *   **作用**：存储用户名、密码（加盐 SHA1 加密）、注册时间等基本信息。
    *   **用户 Token 表 (`tbl_user_token`)**：
        *   **作用**：存储用户的登录会话 Token。虽然代码中 `IsTokenValid` 校验逻辑尚待完善，但架构设计上是将 Session 持久化在 MySQL 中。

# 分块上传从上传到合并具体是如何实现的？Redis具体进行了哪些操作？

### 一、 分块上传：从上传到合并的完整实现逻辑
分块上传的核心思想是将一个大文件切分成多个小块（Chunk）并发上传，最后在服务器端将它们合并。Redis 在这个过程中充当了“记账员”的角色。

#### 1. 初始化阶段 (`InitialMultipartUploadHandler`)
*   **逻辑**：客户端告诉服务器要上传一个大文件（文件名、Hash、大小）。服务器计算需要切分成多少块，并生成一个全局唯一的 `UploadID`。
*   **Redis 操作**：创建一个 Hash 表来记录这次上传任务。
    *   **Key**: `MP_<UploadID>`
    *   **操作**：
        *   `HSET MP_<UploadID> chunkcount <数量>` (记录总分块数)
        *   `HSET MP_<UploadID> filehash <Hash值>`
        *   `HSET MP_<UploadID> filesize <总大小>`

#### 2. 上传分块阶段 (`UploadPartHandler`)
*   **逻辑**：客户端携带 `UploadID` 和 `index` (分块序号) 上传具体的数据块。服务器将数据块保存为临时文件（路径通常包含 `UploadID` 以区分）。
*   **Redis 操作**：每上传成功一个分块，就在 Redis 中“打个勾”。
    *   **Key**: `MP_<UploadID>`
    *   **操作**：`HSET MP_<UploadID> chkidx_<index> 1`
    *   **作用**：标记第 `index` 块已经上传完成。

#### 3. 合并阶段 (`CompleteUploadHandler`)
*   **逻辑**：
    1.  客户端通知服务器：“我传完了，你检查一下”。
    2.  服务器查询 Redis，检查是否所有分块都已就位。
    3.  如果检查通过，执行文件合并 (`mergeChunks`)。
    4.  将合并后的文件信息写入 MySQL (`tbl_file` 和 `tbl_user_file`)。
*   **Redis 操作**：
    *   **操作**：`HGETALL MP_<UploadID>`
    *   **检查逻辑**：遍历取出的所有字段，统计以 `chkidx_` 开头的字段数量。如果 `chkidx_` 的数量等于 `chunkcount`，说明所有分块齐全，可以合并。
*   **合并操作细节 (`mergeChunks`)**：
    *   **依赖**：依赖于**文件系统的追加写入**或**流式拷贝**以及**分块序号的有序性**。
    *   **具体步骤**：
        1.  创建一个新的空文件作为最终的目标文件。
        2.  使用一个 `for` 循环，从 `0` 到 `chunkcount-1` 遍历索引。
        3.  根据 `UploadID` 和索引 `i` 构建每个分块文件的路径（例如：`/data/<UploadID>/<index>`）。
        4.  依次打开每个分块文件，将其内容通过 `io.Copy` 追加写入到目标文件中。
        5.  每写入一个分块，就关闭并删除该分块文件（清理临时文件）。
    *   **关键点**：必须严格按照 `index` 顺序（0, 1, 2...）进行合并，否则文件内容会错乱。

# 这里MySQL是存储token信息的 那么这个token有过期时间吗？这里可以把token放入Redis中保存吗？ 过期时间如何设置？

### 1. 当前 MySQL 中的 Token 有过期时间吗？
**目前没有。**
目前 Token 只是简单地存储在数据库中，并没有字段记录 `expire_time`（过期时间），也没有定时清理机制。这意味着只要用户不重新登录，Token 理论上是永久有效的，这存在安全隐患。

### 2. 可以把 Token 放入 Redis 保存吗？
**完全可以，且非常推荐。**
Redis 天然支持 Key 的过期机制（TTL），是存储 Session/Token 的绝佳场所。这样做有两个好处：
1.  **性能更好**：鉴权是高频操作（每个请求都要查），Redis 内存读取速度远快于 MySQL 磁盘 I/O。
2.  **自动过期**：无需编写复杂的定时任务清理数据库，Redis 会自动删除过期的 Token。

**过期时间设置示例 (Go):**
```go
_, err := conn.Do("SETEX", "TOKEN_"+token, 86400, username)
```

# 文件SHA1哈希值是基于文件的什么得出的？
严格基于 **文件的二进制内容（Content）**。

```go
// util/util.go
func Sha1(data []byte) string {
	_sha1 := sha1.New()
	_sha1.Write(data) // 写入文件的所有二进制字节数据
	return hex.EncodeToString(_sha1.Sum([]byte("")))
}

func FileSha1(file *os.File) string {
	_sha1 := sha1.New()
	io.Copy(_sha1, file) // 将文件流中的全部内容拷贝到哈希计算器中
	return hex.EncodeToString(_sha1.Sum(nil))
}
```

# 断点续传是如何实现的？
断点续传并不是一个单独的“黑科技”接口，而是**分块上传机制的自然延伸**。它的实现依赖于**Redis 中保存的分块上传状态**。

*   **核心原理**：**查询状态 -> 跳过已传 -> 续传未传**。
*   **具体流程**：
    1.  **中断发生**：用户上传了大文件的前 50 个分块后，网络中断或页面关闭。
    2.  **再次上传**：用户刷新页面，再次选择同一个文件进行上传。
    3.  **初始化检查 (`InitialMultipartUploadHandler`)**：
        *   客户端发送初始化请求（携带文件 Hash）。
        *   服务器根据 Hash 计算出 `UploadID`（或者客户端提供之前的 `UploadID`）。
        *   服务器查询 Redis 中 `MP_<UploadID>` 的状态。
    4.  **状态反馈**：
        *   虽然代码示例中未完整展示这部分前端逻辑，但服务器端的 Redis 此时依然保留着 `chkidx_0` 到 `chkidx_49` 的记录（假设 TTL 未过期）。
    5.  **客户端续传**：
        *   客户端（前端 JS）在开始上传前，会先检查哪些分块已经“打勾”了。
        *   客户端**只发送**那些在 Redis 中没有记录的分块（例如从第 50 块开始）。
    6.  **完成**：所有分块传完后，发送合并请求，流程与普通分块上传一致。

**一句话总结**：Redis 记住了你“传到了哪里”，下次你直接从“断掉的地方”继续传，这就是断点续传。