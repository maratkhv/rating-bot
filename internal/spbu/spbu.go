package spbu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"ratinger/pkg/auth"
	"ratinger/pkg/db"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

type naprav struct {
	id       int
	name     string
	capacity int
	list     []abit
	payment  string
	form     string
	eduLevel string
	link     string
}

type abit struct {
	Snils       string `json:"user_code"`
	IsBVI       bool   `json:"without_trials"`
	OrderNumber int    `json:"order_number"`
	Score       int    `json:"score_overall"`
	Priority    int    `json:"priority_number"`
	HasOriginal bool   `json:"original_document"`
}

type bachData struct {
	List []abit
}

// TODO: also think of limiting goroutines
func Check(u *auth.User) []string {
	napravs := retrieveNapravs(u)
	var wg sync.WaitGroup
	for i := range napravs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			napravs[i].getList()
		}()
	}
	wg.Wait()

	var response []string

	for _, n := range napravs {
		var origs int
		for _, abit := range n.list {
			if abit.Snils == u.Snils {
				s := fmt.Sprintf("%s: Ты %d из %d\nПеред тобой %d оригиналов", n.name, abit.OrderNumber, len(n.list), origs)
				response = append(response, s)
				break
			}
			if abit.HasOriginal {
				origs++
			}
		}

	}
	if response == nil {
		response = append(response, "Не нашел тебя в списках")
	}
	return response

}

// ugly but parsing html is such a pita
func (n *naprav) getList() {
	resp, err := http.Get(n.link)
	if err != nil {
		log.Fatalf("get request fail: %v", err)
	}

	switch n.eduLevel {
	case "Бакалавриат":
		r, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("error reading response: %v", err)
		}
		var data bachData
		err = json.Unmarshal(r, &data)
		if err != nil {
			log.Fatalf("error unmarshalling data: %v", err)
		}
		n.list = data.List

	default:
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Fatalf("error parsing data (html): %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		var parser func(context.Context, *html.Node)
		parser = func(ctx context.Context, nd *html.Node) {
			if nd.Type == html.ElementNode && nd.Data == "tbody" {
				for tr := nd.FirstChild; tr != nil; tr = tr.NextSibling {
					if tr.Type == html.ElementNode && tr.Data == "tr" {
						var a abit
						var counter int

						for td := tr.FirstChild; td != nil; td = td.NextSibling {
							if td.Type == html.ElementNode {
								counter++
								if td.FirstChild == nil {
									continue
								}
								switch counter {
								case 1:
									num, err := strconv.Atoi(td.FirstChild.Data)
									if err != nil {
										log.Fatalf("error  1st atoi'ing %v: %v", td.Data, err)
									}
									a.OrderNumber = num
								case 2:
									a.Snils = td.FirstChild.Data
								case 4:
									d := td.FirstChild.Data
									if len(d) == 0 {
										continue
									}
									if !strings.Contains(d, ",") {
										num, err := strconv.Atoi(td.FirstChild.Data)
										if err != nil {
											log.Fatalf("error (get %v) atoi'ing %v: %v", n.link, td.Data, err)
										}
										a.Priority = num
									}
									counter++
									fallthrough
								case 5:
									d := td.FirstChild.Data
									if len(d) < 4 {
										continue
									}
									if d[0] == ' ' {
										d = d[1:]
									}
									num, err := strconv.Atoi(d[:len(d)-3])
									if err != nil {
										log.Fatalf("error 3rd atoi'ing %v: %v", td.Data, err)
									}
									a.Score = num
								case 7:
									if td.FirstChild.Data == "Да" {
										a.HasOriginal = true
									}
								}
							}

						}
						n.list = append(n.list, a)
					}

				}
				cancel()
			} else {
				for c := nd.FirstChild; c != nil; c = c.NextSibling {
					select {
					case <-ctx.Done():
						return
					default:
						parser(ctx, c)
					}
				}
			}
		}
		parser(ctx, doc)
	}

}

func retrieveNapravs(u *auth.User) []naprav {
	napravs := make([]naprav, 0, len(u.Spbu))
	conn := db.NeonConnect()
	defer conn.Close(context.Background())
	if u.Spbu != nil {
		rows, err := conn.Query(context.Background(), "select * from spbu where id = any($1)", u.Spbu)
		if err != nil {
			log.Fatalf("failed getting spbu: %v", err)
		}
		for rows.Next() {
			var n naprav
			rows.Scan(&n.id, &n.name, &n.capacity, &n.payment, &n.form, &n.eduLevel, &n.link)
			napravs = append(napravs, n)
		}
		return napravs
	}

	p, f, el := parseAbitConstraints(u)
	rows, err := conn.Query(context.Background(), "select * from spbu where payment = any($1) and form = any($2) and edu_level=any($3)", p, f, el)
	if err != nil {
		log.Fatalf("failed getting spbu: %v", err)
	}
	for rows.Next() {
		var n naprav
		rows.Scan(&n.id, &n.name, &n.capacity, &n.payment, &n.form, &n.eduLevel, &n.link)
		napravs = append(napravs, n)
	}
	return napravs
}

func parseAbitConstraints(u *auth.User) ([]string, []string, []string) {
	p := make([]string, 0, len(u.Payments))
	for _, v := range u.Payments {
		switch v {
		case "Бюджет":
			p = append(p, "Бюджетная основа", "Полное возмещение затрат")
		case "Контракт":
			p = append(p, "Контракт")
		case "Целевое":
			p = append(p, "Целевой прием", "Целевая квота")
		}
	}

	f := make([]string, 0, len(u.Forms))
	for _, v := range u.Forms {
		switch v {
		case "Очная":
			f = append(f, "очная")
		case "Очно-заочная":
			f = append(f, "Очно-заочная", "очно-заочная")
		case "Заочная":
		}
	}

	return p, f, u.EduLevel
}
