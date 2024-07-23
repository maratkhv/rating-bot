package spbu

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Abits struct {
	Code        string `json:"user_code"`
	OrderNumber int    `json:"order_number"`
	Score       int    `json:"score_overall"`
	HasOriginal bool   `json:"original_document"`
	Priority    int    `json:"priority_number"`
}

var napravs = map[string]string{
	"ПМиИ Фундаментальная информатика и пограммирование": "https://application.spbu.ru/enrollee_lists/lists?id=17",
	"ПМиИ Современное программирование":                  "https://application.spbu.ru/enrollee_lists/lists?id=6",
	"МиКН Науки о данных":                                "https://application.spbu.ru/enrollee_lists/lists?id=27",
	"Математическое Обеспечение и Администрирование КС":  "https://application.spbu.ru/enrollee_lists/lists?id=49",
}

func Check(snils string) []string {
	data := make(map[string][]Abits)
	var wg sync.WaitGroup
	response := make([]string, 0, 5)
	for k, v := range napravs {
		wg.Add(1)
		go func() {
			data[k] = formList(v)
			wg.Done()
		}()
	}
	wg.Wait()

	unique := make(map[string]struct{})
	var uniqueCounter int
	for k := range napravs {
		var origs int
		for _, v := range data[k] {
			if v.Code == snils {
				response = append(response, fmt.Sprintf("%s: %d out of %d, before me %d originals\n", k, v.OrderNumber, len(data[k]), origs))
				break
			}
			if v.HasOriginal {
				origs++
			}
			if _, ok := unique[v.Code]; !ok && v.HasOriginal {
				uniqueCounter++
				unique[v.Code] = struct{}{}
			}
		}
	}

	response = append(response, "      Unique abits with original above me: "+strconv.Itoa(uniqueCounter)+"\n")

	return response
}

func formList(url string) []Abits {
	data := make([]Abits, 0, 1000)
	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("error sending a request: %v", err)
	}
	defer res.Body.Close()
	r, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading the response' body: %v", err)
	}
	err = json.Unmarshal(r, &data)
	if err != nil {
		log.Fatalf("error unmarshalling: %v", err)
	}
	return data
}
