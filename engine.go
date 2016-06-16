package jig

import (
	//"errors"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/nsf/termbox-go"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	PROMPT              = "[jig]>> "
	DefaultY     int    = 1
	FilterPrompt string = "[Filter]> "
)

var (
	f *[]rune
)

type Engine struct {
	json       *simplejson.Json
	orgJson    *simplejson.Json
	currenKeys []string
	query      bool
	pretty     bool
}

func NewEngine(s *os.File, q bool, p bool) *Engine {
	j := parse(s)
	e := &Engine{
		json:       j,
		orgJson:    j,
		currenKeys: []string{},
		query:      q,
		pretty:     p,
	}
	return e
}

func (e Engine) Run() int {

	if !e.render(e.json) {
		return 2
	}
	if e.query {
		fmt.Printf("%s", string(*f))
	} else if e.pretty {
		s, err := e.json.EncodePretty()
		if err != nil {
			return 1
		}
		fmt.Printf("%s", string(s))
	} else {
		s, err := e.json.Encode()
		if err != nil {
			return 1
		}
		fmt.Printf("%s", s)
	}
	return 0
}

func parse(content *os.File) *simplejson.Json {
	buf, err := ioutil.ReadAll(content)

	if err != nil {
		log.Fatal(err)
	}

	js, err := simplejson.NewJson(buf)

	if err != nil {
		log.Fatal(err)
	}

	return js
}

// fix:me
func (e *Engine) render(json *simplejson.Json) bool {

	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	f = &[]rune{}

	contents := e.prettyContents()
	keymode := false

	for {
		e.filterJson(string(*f))
		if keymode {
			contents = e.currenKeys
		} else {
			contents = e.prettyContents()
		}
		draw(contents)
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc, termbox.KeyCtrlC:
				return false
			case termbox.KeyCtrlK:
				keymode = !keymode
			case termbox.KeySpace:
				*f = append(*f, rune(' '))
			case termbox.KeyCtrlW:
				//delete whole word to period
				s := string(*f)
				kws := strings.Split(s, ".")
				lki := len(kws) - 1
				_, kws = kws[lki], kws[:lki]
				s = strings.Join(kws, ".")
				*f = ([]rune(s[0:len(s)]))
			case termbox.KeyBackspace, termbox.KeyBackspace2:
				if i := len(*f) - 1; i >= 0 {
					slice := *f
					*f = slice[0:i]
				}
			case termbox.KeyEnter:
				return true
			case 0:
				*f = append(*f, rune(ev.Ch))
				s := string(*f)
				kws := strings.Split(s, ".")
				lki := len(kws) - 1
				lkw, kws := kws[lki], kws[:lki]

				// keyword search
				var c []int
				re, _ := regexp.Compile("(?i)^" + lkw + ".+")
				for i, k := range e.currenKeys {
					if str := re.FindString(k); str != "" {
						c = append(c, i)
					}
				}
				if len(c) == 1 {
					kws = append(kws, e.currenKeys[c[0]])
					s = strings.Join(kws, ".")
					*f = ([]rune(s[0:len(s)]))
				}
			default:
			}
		case termbox.EventError:
			panic(ev.Err)
			break
		default:
		}
	}
}

func (e *Engine) prettyContents() []string {
	s, _ := e.json.EncodePretty()
	return strings.Split(string(s), "\n")
}

func (e *Engine) filterJson(q string) {
	json := e.orgJson
	if len(q) > 0 {
		keywords := strings.Split(q, ".")

		if keywords[0] != "" {
			e.json = json
			return
		}
		keywords = keywords[1:]

		j := json

		re := regexp.MustCompile("\\[[0-9]*\\]")
		delre := regexp.MustCompile("\\[([0-9]+)?")

		lastIdx := len(keywords) - 1
		//eachFlg := false
		for ki, keyword := range keywords {
			if len(keyword) == 0 {
				break
			}

			// abc[0]
			if keyword[len(keyword)-1:] == "]" {
				matchIndexes := re.FindAllStringIndex(keyword, -1)
				kw := re.ReplaceAllString(keyword, "")

				tj := j.Get(kw)
				if ki != lastIdx {
					j = tj
				} else if !isEmptyJson(tj) {
					j = tj
				}
				lmi := len(matchIndexes) - 1
				for idx, m := range matchIndexes {
					i, _ := strconv.Atoi(keyword[m[0]+1 : m[1]-1])
					if idx == lmi && m[1]-m[0] == 2 {
						//eachFlg = true
					} else if tj := j.GetIndex(i); !isEmptyJson(tj) {
						j = tj
					}
				}
			} else {
				kw := delre.ReplaceAllString(keyword, "")
				tj := j.Get(kw)
				if ki != lastIdx {
					j = tj
				} else if !isEmptyJson(tj) {
					j = tj
				}
			}
		}
		json = j
	}
	e.json = json
	e.setCurrentKeys()
}

func (e *Engine) setCurrentKeys() {
	m, err := e.json.Map()
	if err != nil {
		// is array
	}
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	e.currenKeys = keys
}

func isEmptyJson(j *simplejson.Json) bool {
	switch j.Interface().(type) {
	case nil:
		return true
	default:
		return false
	}
}

func draw(rows []string) {

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	fs := FilterPrompt + string(*f)
	drawln(0, 0, fs)
	termbox.SetCursor(len(fs), 0)

	for idx, row := range rows {
		drawln(0, idx+DefaultY, row)
	}

	termbox.Flush()
}

func drawln(x int, y int, str string) {
	color := termbox.ColorDefault
	backgroundColor := termbox.ColorDefault

	var c termbox.Attribute
	for i, s := range str {
		c = color
		//for _, match := range matches {
		//if i >= match[0] && i < match[1] {
		//c = termbox.ColorGreen
		//}
		//}
		termbox.SetCell(x+i, y, s, c, backgroundColor)
	}
}