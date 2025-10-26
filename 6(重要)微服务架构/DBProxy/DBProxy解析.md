# DBProxy 模块解析
之前的`db文件夹`下面的功能全部作为微服务迁移到当前模块的orm和conn，并使用DBProxy作为数据库的中间层代理处理与数据库交互的操作。客户端因此无需之间操作对应数据库
`dbproxy` 文件夹的作用是实现一个数据库代理服务，用于处理与数据库的交互操作。它通过微服务架构提供了一个中间层，客户端可以通过 RPC 调用的方式与数据库交互，而无需直接操作数据库。以下是 `dbproxy` 文件夹的主要功能和结构解析：

---

### **1. 文件夹的整体作用**
- **数据库代理服务**: `dbproxy` 作为一个独立的服务，封装了对数据库的操作逻辑，提供了统一的接口供其他服务调用。
- **微服务架构**: 使用了 `go-micro` 框架，通过 RPC 实现服务间通信。
- **功能模块化**: 将数据库操作、ORM 映射、RPC 接口等功能分模块实现，便于维护和扩展。

---

### **2. 文件夹结构**
```plaintext
dbproxy/
├── main.go          // 服务的入口文件
├── client/          // 客户端封装，用于调用 dbproxy 服务
├── config/          // 配置文件，例如数据库连接配置
├── conn/            // 数据库连接管理
├── mapper/          // 函数映射，用于动态调用数据库操作
├── orm/             // ORM 映射，定义数据库表结构和操作
├── proto/           // Protobuf 定义和生成的代码
└── rpc/             // RPC 服务的具体实现
```

---

### **3. 各子模块的作用**
#### **3.1 [`main.go`](main.go )**
- 服务的入口文件，启动 `dbproxy` 服务。
- 注册 RPC 服务处理器，启动微服务框架。
- 示例代码：
  ```go
  func main() {
      startRpcService()
  }
  ```

#### **3.2 [`client/`](../../E:/filestore-server/service/dbproxy/proto/proxy.micro.go )**
- 封装了对 `dbproxy` 服务的客户端调用逻辑。
- 提供了文件元信息、用户信息等操作的高层接口。
- 示例功能：
  - 用户注册、登录（[`UserSignup`](../../E:/filestore-server/service/dbproxy/client/client.go )、[`UserSignin`](../../E:/filestore-server/service/dbproxy/client/client.go )）。
  - 文件元信息管理（[`GetFileMeta`](../../E:/filestore-server/service/dbproxy/client/client.go )、[`OnFileUploadFinished`](../../E:/filestore-server/service/dbproxy/client/client.go )）。
  - RPC 调用封装（[`execAction`](../../E:/filestore-server/service/dbproxy/client/client.go )）。
- 示例代码：
  ```go
  func execAction(funcName string, paramJson []byte) (*dbProto.RespExec, error) {
      return dbCli.ExecuteAction(context.TODO(), &dbProto.ReqExec{
          Action: []*dbProto.SingleAction{
              &dbProto.SingleAction{
                  Name:   funcName,
                  Params: paramJson,
              },
          },
      })
  }
  ```

#### **3.3 [`config`](config )**
- 存放服务的配置文件，例如数据库连接字符串。
- 示例代码：
  ```go
  const MySQLSource = "root:123456@tcp(192.168.0.105:13306)/fileserver?charset=utf8"
  ```

#### **3.4 `conn/`**
- 管理数据库连接，提供数据库连接池。
- 提供通用的结果集解析工具（[`ParseRows`](../../E:/filestore-server/service/dbproxy/conn/conn.go )）。
- 示例代码：
  ```go
  func DBConn() *sql.DB {
      return db
  }
  ```

#### **3.5 [`mapper/`](../../E:/filestore-server/service/dbproxy/rpc/proxy.go )**
- 提供动态函数调用功能，通过函数名映射到具体的数据库操作。
- 示例代码：
  ```go
  func FuncCall(name string, params ...interface{}) (result []reflect.Value, err error) {
      // 动态调用数据库操作
  }
  ```

#### **3.6 `orm/`**
- 定义数据库表结构和对应的操作方法。
- 示例功能：
  - 文件元信息操作（[`GetFileMeta`](../../E:/filestore-server/service/dbproxy/client/client.go )、[`GetFileMetaList`](../../E:/filestore-server/service/dbproxy/client/client.go )）。
  - 用户信息操作（[`GetUserInfo`](../../E:/filestore-server/service/dbproxy/client/client.go )）。
- 示例代码：
  ```go
  func GetFileMeta(filehash string) (res ExecResult) {
      // 查询文件元信息
  }
  ```

#### **3.7 [`proto/`](../../E:/filestore-server/service/dbproxy/proto/proxy.micro.go )**
- 定义了 `dbproxy` 服务的 Protobuf 文件，并生成对应的 Go 代码。
- 定义了 RPC 请求和响应的结构体（[`ReqExec`](../../E:/filestore-server/service/dbproxy/proto/proxy.pb.go )、[`RespExec`](../../E:/filestore-server/service/dbproxy/proto/proxy.pb.go )）。
- 示例代码：
  ```proto
  message ReqExec {
      bool sequence = 1;
      bool transaction = 2;
      int32 resultType = 3;
      repeated SingleAction action = 4;
  }
  ```

#### **3.8 `rpc/`**
- 实现了 `dbproxy` 服务的 RPC 接口。
- 处理客户端的 RPC 请求，调用对应的数据库操作。
- 示例代码：
  ```go
  func (db *DBProxy) ExecuteAction(ctx context.Context, req *dbProxy.ReqExec, res *dbProxy.RespExec) error {
      // 执行数据库操作
  }
  ```

---

### **4. 工作流程**
1. **客户端调用**:
   - 客户端通过 [`client`](../../E:/filestore-server/service/dbproxy/proto/proxy.micro.go ) 模块调用 `dbproxy` 服务的接口。
   - 例如，调用 [`GetFileMeta`](../../E:/filestore-server/service/dbproxy/client/client.go ) 获取文件元信息。

2. **RPC 请求**:
   - [`client`](../../E:/filestore-server/service/dbproxy/proto/proxy.micro.go ) 模块通过 RPC 将请求发送到 `dbproxy` 服务。

3. **服务处理**:
   - `rpc` 模块接收请求，解析参数，调用 [`mapper`](../../E:/filestore-server/service/dbproxy/rpc/proxy.go ) 模块执行对应的数据库操作。

4. **数据库操作**:
   - `orm` 模块执行具体的数据库查询或更新操作。
   - 使用 `conn` 模块管理数据库连接。

5. **返回结果**:
   - 将操作结果通过 RPC 返回给客户端。

---

### **5. 总结**
`dbproxy` 文件夹的主要作用是作为数据库操作的中间层，提供统一的接口供其他服务调用。它通过微服务架构实现了模块化设计，封装了数据库连接、ORM 映射、RPC 通信等功能，简化了客户端与数据库的交互逻辑，同时提高了系统的可维护性和扩展性。