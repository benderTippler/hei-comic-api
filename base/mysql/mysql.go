package mysql

import (
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var engine *gorm.DB
var debug bool

func init() {
	debug = true
}

func InitMysql() error {
	ctx := gctx.New()
	databaseVar, err := g.Cfg().Get(ctx, "database")
	if err != nil {
		return err
	}
	databaseCfg := databaseVar.MapStrVar()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		databaseCfg["user"].String(),
		databaseCfg["pass"].String(),
		databaseCfg["host"].String(),
		databaseCfg["port"].Int(),
		databaseCfg["dbname"].String(),
		databaseCfg["charset"].String(),
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	SetDebug(databaseCfg["debug"].Bool())

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxIdleConns(800)
	sqlDB.SetMaxOpenConns(3000)
	sqlDB.SetConnMaxLifetime(time.Hour)

	engine = db
	return nil
}

func CloseMysql() {
	if engine != nil {
		sqlDB, _ := engine.DB()
		sqlDB.Close()
	}
}

func NewDb() *gorm.DB {
	if debug {
		return engine.Debug()
	}
	return engine
}

func SetDebug(b bool) {
	debug = b
}
