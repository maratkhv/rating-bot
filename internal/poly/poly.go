package poly

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
	"sync"
)

type Data struct {
	List              []Abits
	DirectionCapacity int
}

type Abits struct {
	UserSnils            string
	FullScore            int
	HasOriginalDocuments bool
	Priority             int
}

var apiLink = "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId="

// TODO: i think i can just use []Structs with len=len(napravs)
type parsedData struct {
	data map[int][]Abits
	cap  map[int]int
	mu   sync.Mutex
}

type naprav struct {
	name string
	url  string
}

// TODO: needs to be rewritten but im too lazy
// TODO: use workers / semphore to avoid using 100s of goroutines
func Check(u *auth.User) []string {
	napravs := retrieveNapravs(u)

	data := parsedData{
		data: make(map[int][]Abits),
		cap:  make(map[int]int),
	}
	var wg sync.WaitGroup
	response := make([]string, 0)
	for id, n := range napravs {
		wg.Add(1)
		go func() {
			tmp, cap := formList(n.url)
			data.mu.Lock()
			data.data[id], data.cap[id] = tmp, cap
			data.mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	unique := make(map[string]struct{})
	var uniqueCounter int
	var abitNapravs []int
	for id := range napravs {
		var origs, uc int
		for i, v := range data.data[id] {
			if v.UserSnils == u.Snils {
				response = append(response, fmt.Sprintf("%s (всего %d мест):\nТы %d из %d, выше тебя %d оригиналов\n", napravs[id].name, data.cap[id], i+1, len(data.data[id]), origs))
				uniqueCounter += uc
				abitNapravs = append(abitNapravs, id)
				break
			}
			if v.HasOriginalDocuments {
				origs++
			}
			if _, ok := unique[v.UserSnils]; !ok && v.HasOriginalDocuments {
				uc++
				unique[v.UserSnils] = struct{}{}
			}
		}
	}

	if len(abitNapravs) != 0 && u.Spbstu == nil {
		conn := db.NeonConnect()
		defer conn.Close(context.Background())
		_, err := conn.Exec(context.Background(), "update users set spbstu=$1 where snils=$2", abitNapravs, u.Snils)
		if err != nil {
			log.Fatal(err)
		}
		u.Spbstu = abitNapravs
	}

	if len(response) != 0 {
		response = append(response, "Количество уникальных* аттестатов: "+strconv.Itoa(uniqueCounter)+"\n")
	} else {
		response = append(response, fmt.Sprintf("Не нашел Тебя в списках.\n\nПроверь, верен ли введенный СНИЛС (%v).\n\n*возможна также проблема в сайте вуза, тогда остается только ждать*", u.Snils))
	}

	return response

}

func formList(url string) ([]Abits, int) {
	var data Data
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("error creating req: %v", err)
	}
	req.Header.Add("Accept", `application/json,text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8`)
	req.Header.Add("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.64 Safari/537.11`)
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("error doing req: %v", err)
	}
	defer res.Body.Close()

	r, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading: %v", err)
	}

	err = json.Unmarshal(r, &data)
	if err != nil {
		log.Fatalf("error unmarshalling: %v", err)
	}

	return data.List, data.DirectionCapacity
}

func retrieveNapravs(u *auth.User) map[int]naprav {
	napravs := make(map[int]naprav)
	conn := db.NeonConnect()
	defer conn.Close(context.Background())
	if u.Spbstu != nil {
		rows, err := conn.Query(context.Background(), "select id, name from spbstu where id = any($1)", u.Spbstu)
		if err != nil {
			log.Fatalf("failed getting spbstu that already known: %v", err)
		}
		for rows.Next() {
			var (
				id   int
				name string
			)
			rows.Scan(&id, &name)
			napravs[id] = naprav{
				name: name,
				url:  apiLink + strconv.Itoa(id),
			}
		}
		return napravs
	}

	p, f, el := parseAbitConstraints(u)
	fmt.Println(p, f, el)
	rows, err := conn.Query(context.Background(), "select id, name from spbstu where payment = any($1) and form = any($2) and edu_level=any($3)", p, f, el)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			log.Fatalf("scan err: %v", err)
		}
		napravs[id] = naprav{
			name: name,
			url:  apiLink + strconv.Itoa(id),
		}
	}
	fmt.Println(napravs)
	return napravs
}

// prolly should just make data in tables fit defaults TODO? also add links in tables
func parseAbitConstraints(u *auth.User) ([]string, []string, []string) {
	p := make([]string, 0, len(u.Payments))
	for _, v := range u.Payments {
		switch v {
		case "Бюджет":
			p = append(p, "Бюджетная основа")
		case "Контракт":
			p = append(p, "Контракт")
		case "Целевое":
			p = append(p, "Целевой прием")
		}
	}
	return p, u.Forms, u.EduLevel
}
