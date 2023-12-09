package db

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"godzilla/env"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var Db *gorm.DB

func Open() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s", env.MysqlUser, env.MysqlPassword, env.MysqlHost,
		env.MysqlPort, env.MysqlDatabase, "utf8mb4&parseTime=True&loc=Asia%2FShanghai")
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		logrus.Fatal(err)
	}
	Db = conn
}
