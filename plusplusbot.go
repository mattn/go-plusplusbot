package main

import (
	"database/sql"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var db *sql.DB
var lock1 = new(sync.Mutex)
var lock2 = new(sync.Mutex)
var plus = regexp.MustCompile(`^\s*([a-zA-Z0-9_{^}]+)\+\+\s*$`)
var minus = regexp.MustCompile(`^\s*([a-zA-Z0-9_{^}]+)--\s*$`)
var pluseq = regexp.MustCompile(`^\s*([a-zA-Z0-9_{^}]+)\+=([0-9])\s*$`)
var minuseq = regexp.MustCompile(`^\s*([a-zA-Z0-9_{^}]+)\-=([0-9])\s*$`)
var ref = 0

func atoi(a string) int {
	i, _ := strconv.Atoi(a)
	return i
}

func parse(message string, callback func(nick string, plus int)) {
	if plus.MatchString(message) {
		m := plus.FindStringSubmatch(message)
		callback(m[1], 1)
	} else if minus.MatchString(message) {
		m := minus.FindStringSubmatch(message)
		callback(m[1], -1)
	} else if pluseq.MatchString(message) {
		m := pluseq.FindStringSubmatch(message)
		callback(m[1], atoi(m[2]))
	} else if minuseq.MatchString(message) {
		m := minuseq.FindStringSubmatch(message)
		callback(m[1], -atoi(m[2]))
	}
}

func incr() {
	lock2.Lock()
	defer lock2.Unlock()
	ref++
}

func decr() {
	lock2.Lock()
	defer lock2.Unlock()
	ref--
}

func plusplus(c *irc.Conn, line *irc.Line, nick string, plus int) {
	score := 0

	incr()
	lock1.Lock()

	defer func() {
		lock1.Unlock()
		<-time.After(1 * time.Second)
		decr()
		if ref == 0 {
			c.Notice(line.Args[0], fmt.Sprintf("%s (%d)", nick, score))
		}
	}()

	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return
	}
	defer tx.Rollback()

	row, err := tx.Query(`select score from plusplus where nick = ?`, strings.ToLower(nick))
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return
	}

	if row.Next() {
		err = row.Scan(&score)
		if err != nil {
			fmt.Printf("Database error: %v\n", err)
			row.Close()
			return
		}
	}
	score += plus
	row.Close()

	stmt, err := tx.Prepare(`insert or replace into plusplus (nick, score) values (?, ?)`)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(strings.ToLower(nick), score)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return
	}
	tx.Commit()
}

func ranking(c *irc.Conn, line *irc.Line) {
	lock1.Lock()
	defer lock1.Unlock()

	rows, err := db.Query(`select nick, score from plusplus order by score desc`)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		return
	}
	defer rows.Close()

	rank, nick, score := 1, "", 0
	for rows.Next() {
		rows.Scan(&nick, &score)
		c.Notice(line.Args[0], fmt.Sprintf("%03d: %s (%d)\n", rank, nick, score))
		rank++
		if rank > 5 {
			break
		}
	}
}

func main() {
	var err error

	db, err = sql.Open("sqlite3", "./plusplus.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	c := irc.SimpleClient("plusplusbot", "plusplusbot")
	c.EnableStateTracking()

	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		for _, room := range os.Args[1:] {
			c.Join("#" + room)
		}
	})

	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})

	c.AddHandler("privmsg", func(conn *irc.Conn, line *irc.Line) {
		println(line.Src, line.Args[0], line.Args[1])
		if line.Args[1] == "!++" {
			go ranking(c, line)
		} else {
			parse(line.Args[1], func(nick string, plus int) {
				go plusplus(c, line, nick, plus)
			})
		}
	})

	for {
		if err := c.Connect("irc.freenode.net:6667"); err != nil {
			fmt.Printf("Connection error: %s\n", err)
			return
		}
		<-quit
	}
}
