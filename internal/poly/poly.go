package poly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
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

type parsedData struct {
	data map[int][]Abits
	cap  map[int]int
	mu   sync.Mutex
}

type naprav struct {
	name string
	url  string
}

func findNapravs(payment, form string) *map[int]naprav {
	napravs := make(map[int]naprav)
	godotenv.Load()
	var PSWD = os.Getenv("PSWD")
	var connString = "postgresql://myneondb_owner:" + PSWD + "@ep-shiny-sun-a2swl5c2.eu-central-1.aws.neon.tech/myneondb?sslmode=require"
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}
	rows, err := conn.Query(context.Background(), "select id, name from napravs where payment=$1 and form=$2", payment, form)
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

	return &napravs
}

func Check(snils string) []string {
	godotenv.Load()
	var PSWD = os.Getenv("PSWD")
	var connString = "postgresql://myneondb_owner:" + PSWD + "@ep-shiny-sun-a2swl5c2.eu-central-1.aws.neon.tech/myneondb?sslmode=require"
	_, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal(err)
	}

	napravs := *findNapravs("Бюджетная основа", "Очная")

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
	for id := range napravs {
		var origs int
		for i, v := range data.data[id] {
			if v.UserSnils == snils {
				response = append(response, fmt.Sprintf("%d (всего %d мест):\nТы %d из %d, выше тебя %d оригиналов\n", id, data.cap[id], i+1, len(data.data[id]), origs))
				break
			}
			if v.HasOriginalDocuments {
				origs++
			}
			if _, ok := unique[v.UserSnils]; !ok && v.HasOriginalDocuments {
				uniqueCounter++
				unique[v.UserSnils] = struct{}{}
			}
		}
	}

	if len(response) != 0 {
		response = append(response, "Количество уникальных* аттестатов: "+strconv.Itoa(uniqueCounter)+"\n")
	} else {
		response = append(response, fmt.Sprintf("Не нашел Тебя в списках.\n\nПроверь, верен ли введенный СНИЛС (%v).\n\n*возможна также проблема в сайте вуза, тогда остается только ждать*", snils))
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
