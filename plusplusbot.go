package main

import (
	"database/sql"
	"fmt"
	"github.com/thoj/go-ircevent"
	_ "github.com/mattn/go-sqlite3"
	"regexp"
	"strconv"
)

var db *sql.DB
var plus = regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)\+\+\s*$`)
var minus = regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)--\s*$`)
var pluseq = regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)\+=([0-9])\s*$`)
var minuseq = regexp.MustCompile(`^\s*([a-zA-Z0-9_-]+)\-=([0-9])\s*$`)

func atoi(a string) int {
	i, _ := strconv.Atoi(a)
	return i
}

func plusplus(message string, callback func(nick string, plus int)) {
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

func main() {
	room := "#subtech"
	var err error

	db, err = sql.Open("sqlite3", "./plusplus.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	c := irc.IRC("plusplusbot", "plusplusbot")
	if err := c.Connect("irc.freenode.net:6667"); err != nil {
		fmt.Printf("Connection error: %v\n", err)
		return
	}

	c.AddCallback("PRIVMSG", func(e *irc.Event) {
		plusplus(e.Message, func(nick string, plus int) {
			println(nick, plus)

			tx, err := db.Begin()
			if err != nil {
				fmt.Printf("Database error: %v\n", err)
				return
			}
			defer tx.Rollback()

			score := 0
			row, err := tx.Query(`select score from plusplus where nick = ?`, nick)
			if err != nil {
				fmt.Printf("Database error: %v\n", err)
				return
			}
			if row.Next() {
				err = row.Scan(&score)
				if err != nil {
					fmt.Printf("Database error: %v\n", err)
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

			_, err = stmt.Exec(nick, score)
			if err != nil {
				fmt.Printf("Database error: %v\n", err)
				return
			}
			tx.Commit()

			c.Privmsg(room, fmt.Sprintf("%s (%d)", nick, score))
		})
	})

	c.Join(room)

	c.Loop()
}