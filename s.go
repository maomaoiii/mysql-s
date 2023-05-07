package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/auth"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"

	"github.com/maomaoiii/mysql-s/memory2"
)

const (
	dbName    = "benchmark"
	tableName = "bench_tab"
)

func main() {

	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	driver := sqle.NewDefault()
	driver.AddDatabase(createTestDatabase())

	ips := get_inner_ip()

	user, pass, host, port := "root", "supersecret", "localhost", 3306
	host = ips["inner"]
	config := server.Config{
		Protocol: "tcp",
		Address:  fmt.Sprintf("%s:%d", host, port),
		Auth:     auth.NewNativeSingle(user, pass, auth.AllPermissions),
	}

	s, err := server.NewDefaultServer(config, driver)
	if err != nil {
		panic(err)
	}
	fmt.Printf("started, can use:\n")
	fmt.Printf("mysql -h%s -P%d -u%s -p%s -D%s\n", host, port, user, pass, dbName)
	s.Start()
}

func createTestDatabase() *memory2.Database {

	db := memory2.NewDatabase(dbName)
	table := memory2.NewTable(tableName, sql.Schema{
		{Name: "id", Type: sql.Uint64, Nullable: false, Source: tableName, PrimaryKey: true, AutoIncrement: true},
		{Name: "extra_data", Type: sql.Text, Nullable: false, Source: tableName},
		{Name: "create_time", Type: sql.Uint32, Nullable: false, Source: tableName},
	})

	db.AddTable(tableName, table)
	ctx := sql.NewEmptyContext()

	rand.Seed(100)

	const l = 512
	for i := 1; i <= 10000; i++ {
		data := generateRandomString(l - 1)
		table.Insert(ctx, sql.NewRow(i, data, time.Now().Unix()))
	}

	sm := os.Getenv("SLEEP_MS")
	if len(sm) > 0 {
		memory2.SleepMs, _ = strconv.Atoi(sm)
	}

	return db
}

func get_inner_ip() map[string]string {
	addrs, err := net.InterfaceAddrs()
	ret := make(map[string]string)
	ret["loopback"] = "127.0.0.1"
	if err != nil {
		fmt.Println(err)
		return ret
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil && isPrivateIP(ipnet.IP) {
				ret["inner"] = ipnet.IP.String()
				fmt.Println(ipnet.IP.String())
			}
		}
	}
	return ret
}

// 判断是否为内网 IP 地址
func isPrivateIP(ip net.IP) bool {
	privateIPBlocks := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateRandomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
