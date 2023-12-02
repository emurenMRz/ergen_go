package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/canvas"
	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/config"
	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/db"
)

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		return
	}

	param := db.DBConnect{Host: conf.Host, User: conf.User, Password: conf.Password}

	if len(conf.Database) > 0 {
		c := connectDatabase(param, conf.Database)

		today := time.Now().Format("2006-01-02_150405")
		fn := fmt.Sprintf("ER %s %s.svg", conf.Database, today)

		f, err := os.Create(fn)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		c.OutputSVG(f)
	} else {
		_, err = param.Connect()
		if err != nil {
			panic(err.Error())
		}

		names, err := param.Databasenames()
		if err != nil {
			panic(err.Error())
		}

		server(param, names, conf.AcceptPort)
	}
}

func connectDatabase(conn db.DBConnect, dbName string) (c *canvas.Canvas) {
	log.Println("DB: " + dbName)

	conn.Dbname = dbName

	_, err := conn.Connect()
	if err != nil {
		log.Println(err.Error())
		return
	}

	tableNames, err := conn.Tablenames()
	if err != nil {
		log.Println(err.Error())
		return
	}

	tableInfos := []db.TableInfo{}
	for _, tableName := range tableNames {
		info, err := conn.GetTableInfo(tableName)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		tableInfos = append(tableInfos, info)
	}

	c = canvas.NewCanvas()
	for _, info := range tableInfos {
		c.RegisterEntity(canvas.NewEntityFromTableInfo(&info))
	}

	return
}
