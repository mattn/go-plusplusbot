package main

import (
	"testing"
)

var plusplusTests = []struct {
	input string
	nick  string
	plus  int
}{
	{"mattn++", "mattn", 1},
	{" mattn++", "mattn", 1},
	{"mattn++ ", "mattn", 1},
	{"m_att_n--", "m_att_n", -1},
	{"m_att_n ++", "", 0},
	{"mattn+=5", "mattn", 5},
	{"mattn-=4", "mattn", -4},
}

func TestPlusplus(t *testing.T) {
	for _, e := range plusplusTests {
		nick := ""
		plus := 0
		plusplus(e.input, func(n string, p int) {
			nick = n
			plus = p
		})
		if nick != e.nick {
			t.Errorf("nick == %q, want %q", nick, e.nick)
		}
		if plus != e.plus {
			t.Errorf("plus == %d, want %d", plus, e.plus)
		}
	}
}
