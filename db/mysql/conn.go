package mysql

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB // 声明一个全局变量 db，类型为 *sql.DB，用于存储数据库连接对象

func init() {

	var err error
	// 尝试打开数据库连接并捕获错误
	db, err = sql.Open("mysql", "root:123456@tcp(192.168.0.105:13306)/fileserver?charset=utf8")
	if err != nil {
		fmt.Printf("Failed to connect to mysql, err: %v\n", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1000)
	err = db.Ping()
	if err != nil {
		fmt.Printf("Failed to ping mysql, err: %v\n", err)
		os.Exit(1)
	}
}

// DBCoon:返回数据库链接的对象
func DBConn() *sql.DB {
	return db
}
