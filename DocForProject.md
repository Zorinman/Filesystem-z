# 文件存储系统 - 技术文档

## 1. 项目概述

本项目是一个基于微服务架构的分布式文件存储系统,支持文件的上传、下载、秒传、分块上传等功能。系统采用 Go 语言开发,使用 Kubernetes 进行容器编排,支持多种存储后端(本地存储、OSS、Ceph)。

**核心特性**:
- 微服务架构,服务独立部署
- 支持同步/异步文件上传
- 支持大文件分块上传
- 文件秒传功能(基于 SHA1 去重)
- 多存储后端支持
- 基于 Token 的用户认证

---

## 2. 系统架构

### 2.1 整体架构

系统采用微服务架构,主要包含以下层次:

```
用户层 → Ingress层 → 微服务层 → 数据层 → 存储层
```

**微服务模块**:
- **API Gateway**: 统一入口,HTTP 协议转换和路由分发
- **Account Service**: 用户注册、登录、Token 管理
- **Upload Service**: 文件上传、秒传、分块上传
- **Download Service**: 文件下载、断点续传
- **DBProxy Service**: 数据库操作统一代理
- **Transfer Service**: 文件异步转移(未完成)

**通信协议**:
- 外部: HTTP/HTTPS
- 内部: gRPC(微服务间通信)

**部署方式**:
- Kubernetes 容器编排
- Traefik Ingress 流量入口
- Docker 容器化部署

---

## 3. 核心模块功能

### 3.1 API Gateway(API 网关)

**端口**: 8080  
**协议**: HTTP

**功能**:
- 统一对外入口
- HTTP 请求路由分发
- 协议转换(HTTP → gRPC)
- 返回上传/下载服务地址

**主要接口**:
- `POST /user/signup`: 用户注册
- `POST /user/signin`: 用户登录
- 登录成功后返回 Token 和上传/下载服务入口地址

---

### 3.2 Account Service(账户服务)

**端口**: 8080  
**协议**: gRPC

**功能**:
- 用户注册(用户名、密码、邮箱、手机号)
- 用户登录验证
- Token 生成与管理(MD5(username+timestamp+salt))
- 用户信息查询

**核心流程**:
1. 接收 API Gateway 的 gRPC 请求
2. 调用 DBProxy 验证用户凭证
3. 生成 Token 并存储到数据库
4. 返回 Token 给客户端

---

### 3.3 Upload Service(上传服务)

**端口**: 28080  
**协议**: HTTP

**功能**:
- **普通上传**: 单文件直接上传
- **秒传**: 基于文件 SHA1 哈希值检测,文件已存在则秒传
- **分块上传**: 大文件分块上传,支持断点续传
- **异步上传**: 通过 RabbitMQ 异步转移文件到 OSS/Ceph

**主要接口**:
- `POST /file/upload`: 文件上传
- `POST /file/fastupload`: 秒传检测
- `POST /file/mpupload/init`: 初始化分块上传
- `POST /file/mpupload/uppart`: 上传文件分块
- `POST /file/mpupload/complete`: 完成分块上传

**上传流程**:
1. 计算文件 SHA1 哈希值
2. 检查文件是否已存在(秒传)
3. 保存文件到存储后端(本地/OSS/Ceph)
4. 调用 DBProxy 保存文件元信息
5. 如果是异步上传,发送消息到 RabbitMQ

---

### 3.4 Download Service(下载服务)

**端口**: 38080  
**协议**: HTTP

**功能**:
- 文件下载
- 断点续传
- 支持多种存储后端(本地/OSS/Ceph)

**主要接口**:
- `GET /file/download`: 文件下载
- `GET /file/downloadurl`: 获取文件下载 URL(OSS 签名 URL)

**下载流程**:
1. 验证用户 Token
2. 调用 DBProxy 查询文件元信息
3. 根据存储类型读取文件
4. 返回文件流或重定向到 OSS URL

---

### 3.5 DBProxy Service(数据库代理服务)

**协议**: gRPC

**功能**:
- 统一数据库操作接口
- 封装 MySQL 操作
- 提供 ORM 功能

**主要方法**:
- `UserSignin`: 用户登录验证
- `UserSignup`: 用户注册
- `UpdateToken`: 更新用户 Token
- `GetFileMeta`: 查询文件元信息
- `OnFileUploadFinished`: 文件上传完成后更新数据库
- `UpdateFileLocation`: 更新文件存储位置

---

### 3.6 分块上传模块(Redis)

**技术栈**: Redis 缓存

**功能**:
- 大文件分块上传
- 分块状态管理
- 断点续传支持(未完全实现)

**流程**:
1. **初始化**: 生成 UploadID,计算分块数量,存储到 Redis
2. **上传分块**: 逐个上传分块,更新 Redis 状态
3. **合并分块**: 验证所有分块完成后,合并为完整文件
4. **清理**: 删除分块文件,清除 Redis 缓存

**Redis 数据结构**:
```
Key: MP_<UploadID>
Hash:
  - chunkcount: 分块总数
  - filehash: 文件哈希值
  - filesize: 文件大小
  - chkidx_0: 分块0状态(1=已上传)
  - chkidx_1: 分块1状态
  - ...
```

---

### 3.7 异步上传模块(RabbitMQ)

**技术栈**: RabbitMQ 消息队列

**功能**:
- 异步文件转移(临时存储 → OSS/Ceph)
- 解耦上传和转移逻辑
- 提高上传响应速度

**流程**:
1. **生产者**: 文件上传到临时存储后,发送转移消息到 RabbitMQ
2. **消费者**: Transfer Service 监听队列,读取消息
3. **转移**: 从临时存储读取文件,上传到 OSS/Ceph
4. **更新**: 调用 DBProxy 更新文件存储位置

**消息结构**:
```go
type TransferData struct {
    FileHash      string        // 文件哈希值
    CurLocation   string        // 当前存储位置
    DestLocation  string        // 目标存储位置
    DestStoreType StoreType     // 目标存储类型
}
```

---

## 4. 数据库设计

### 4.1 tbl_file(文件表)

**作用**: 存储系统中所有唯一文件的元信息

| 字段      | 类型          | 说明                             |
| --------- | ------------- | -------------------------------- |
| id        | int           | 主键,自增                        |
| file_sha1 | char(40)      | 文件 SHA1 哈希值(唯一索引)       |
| file_name | varchar(256)  | 文件名                           |
| file_size | bigint        | 文件大小(字节)                   |
| file_addr | varchar(1024) | 文件存储路径                     |
| create_at | datetime      | 创建时间                         |
| update_at | datetime      | 更新时间                         |
| status    | int           | 文件状态(0=可用,1=禁用,2=已删除) |

**索引**:
- 主键: `id`
- 唯一索引: `file_sha1`
- 普通索引: `status`

---

### 4.2 tbl_user(用户表)

**作用**: 存储用户账户信息

| 字段            | 类型         | 说明                                    |
| --------------- | ------------ | --------------------------------------- |
| id              | int          | 主键,自增                               |
| user_name       | varchar(64)  | 用户名                                  |
| user_pwd        | varchar(256) | 用户密码(加密存储)                      |
| email           | varchar(64)  | 邮箱                                    |
| phone           | varchar(128) | 手机号(唯一索引)                        |
| email_validated | tinyint(1)   | 邮箱是否验证                            |
| phone_validated | tinyint(1)   | 手机号是否验证                          |
| signup_at       | datetime     | 注册时间                                |
| last_active     | datetime     | 最后活跃时间                            |
| profile         | text         | 用户属性(JSON)                          |
| status          | int          | 账户状态(0=启用,1=禁用,2=锁定,3=已删除) |

**索引**:
- 主键: `id`
- 唯一索引: `phone`
- 普通索引: `status`

---

### 4.3 tbl_user_token(用户令牌表)

**作用**: 存储用户登录 Token

| 字段       | 类型        | 说明             |
| ---------- | ----------- | ---------------- |
| id         | int         | 主键,自增        |
| user_name  | varchar(64) | 用户名(唯一索引) |
| user_token | char(40)    | 用户 Token       |

**索引**:
- 主键: `id`
- 唯一索引: `user_name`

---

### 4.4 tbl_user_file(用户文件表)

**作用**: 存储用户与文件的关联关系(多对多)

| 字段        | 类型         | 说明                             |
| ----------- | ------------ | -------------------------------- |
| id          | int          | 主键,自增                        |
| user_name   | varchar(64)  | 用户名                           |
| file_sha1   | varchar(64)  | 文件 SHA1 哈希值                 |
| file_size   | bigint       | 文件大小                         |
| file_name   | varchar(256) | 用户自定义文件名                 |
| upload_at   | datetime     | 上传时间                         |
| last_update | datetime     | 最后修改时间                     |
| status      | int          | 文件状态(0=正常,1=已删除,2=禁用) |

**索引**:
- 主键: `id`
- 唯一索引: `(user_name, file_sha1)`
- 普通索引: `user_name`、`status`

---

## 5. 技术栈

### 5.1 后端技术

| 技术             | 用途           |
| ---------------- | -------------- |
| Go 1.x           | 开发语言       |
| Gin              | HTTP Web 框架  |
| go-micro         | 微服务框架     |
| gRPC             | 微服务通信协议 |
| Protocol Buffers | 数据序列化     |

### 5.2 数据存储

| 技术     | 用途               |
| -------- | ------------------ |
| MySQL    | 关系型数据库       |
| Redis    | 缓存、分块上传状态 |
| RabbitMQ | 消息队列、异步任务 |

### 5.3 存储后端

| 技术       | 用途         |
| ---------- | ------------ |
| 本地存储   | 临时文件存储 |
| 阿里云 OSS | 对象存储     |
| Ceph       | 分布式存储   |

### 5.4 部署运维

| 技术       | 用途           |
| ---------- | -------------- |
| Docker     | 容器化         |
| Kubernetes | 容器编排       |
| Traefik    | Ingress 控制器 |

---

## 6. 核心流程

### 6.1 用户登录流程

```
用户 → API Gateway → Account Service → DBProxy → MySQL
                                          ↓
                        生成 Token ← 验证成功
                                          ↓
                        返回 Token + 上传/下载地址
```

### 6.2 文件上传流程

```
用户 → Upload Service → 计算 SHA1 → 检查秒传
                              ↓
                        保存到存储后端
                              ↓
                        DBProxy → MySQL(保存元信息)
                              ↓
                    (可选)RabbitMQ → 异步转移
```

### 6.3 文件下载流程

```
用户 → Download Service → DBProxy → MySQL(查询元信息)
                              ↓
                        读取存储后端
                              ↓
                        返回文件流 / OSS URL
```

---

## 7. 项目特点

### 7.1 优势

- **高可扩展性**: 微服务架构,各服务独立扩容
- **高性能**: Redis 缓存 + RabbitMQ 异步处理
- **高可用性**: Kubernetes 自动故障恢复
- **存储灵活**: 支持多种存储后端
- **秒传优化**: 基于 SHA1 去重,节省存储和带宽

### 7.2 待完善功能

- Token 验证逻辑未完全实现
- 服务间认证机制缺失
- 断点续传功能未完全实现
- Transfer Service 未完成
- 缺少日志、监控、链路追踪

---

## 8. 部署架构

### 8.1 Kubernetes 部署

**命名空间**: default

**服务列表**:
- `svc-apigw:8080`
- `svc-account:8080`
- `svc-upload:28080`
- `svc-download:38080`
- `svc-dbproxy`

**Ingress 规则**:
- `apigw.fileserver.com` → API Gateway
- `upload.fileserver.com` → Upload Service
- `download.fileserver.com` → Download Service

**服务发现**:
- 使用 Kubernetes API 作为服务注册中心
- go-micro 框架自动服务发现

---

## 9. 总结

本项目是一个功能完整的分布式文件存储系统,采用微服务架构,支持文件的上传、下载、秒传、分块上传等功能。系统使用 Go 语言开发,基于 Kubernetes 部署,具有高可扩展性和高可用性。数据库设计合理,支持文件去重和用户文件关联。系统整体架构清晰,模块职责明确,适合作为学习微服务架构和分布式存储的参考项目。



