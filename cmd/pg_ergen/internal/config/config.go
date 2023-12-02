package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Config struct {
	Host       string
	User       string
	Password   string
	Port       uint16
	Database   string
	AcceptPort uint16
}

func GetConfig() (conf Config, err error) {
	confPtr := flag.String("f", "config.json", "config filename")
	hostPtr := flag.String("h", "localhost", "db address")
	portPtr := flag.Uint("p", 5432, "db port")
	userPtr := flag.String("u", "postgres", "db username")
	pwPtr := flag.String("w", "", "db password")
	dbPtr := flag.String("d", "", "database name")
	acceptPtr := flag.Uint("a", 20000, "[server mode] accept port")
	flag.Parse()

	conf, err = readConfig("./" + *confPtr)
	if err != nil {
		log.Fatal(err)
	}

	if len(conf.Host) == 0 || *hostPtr != "localhost" {
		conf.Host = *hostPtr
	}
	if conf.Port == 0 || *portPtr != 5432 {
		conf.Port = uint16(*portPtr)
	}
	if len(conf.User) == 0 || *userPtr != "postgres" {
		conf.User = *userPtr
	}
	if len(conf.Password) == 0 || len(*pwPtr) > 0 {
		conf.Password = *pwPtr
	}
	if len(conf.Database) == 0 || len(*dbPtr) > 0 {
		conf.Database = *dbPtr
	}
	if conf.AcceptPort == 0 || *acceptPtr != 20000 {
		conf.AcceptPort = uint16(*acceptPtr)
	}

	return
}

func readConfig(fn string) (conf Config, err error) {
	conf = Config{}

	file, err := os.Open(fn)
	if err != nil {
		log.Println(err.Error())
		return conf, nil
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&conf)
	if err != nil {
		return
	}

	return
}
