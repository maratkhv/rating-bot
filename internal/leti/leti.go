package leti

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type info struct {
	Data struct {
		Competition struct {
			TotalNum int `json:"total_num"`
		}
		List []Abits
	}
}

type Abits struct {
	Code        string
	Priority    int
	TotalPoints int    `json:"total_points"`
	HasOriginal bool   `json:"has_original"`
	Condition   string `json:"enroll_condition"`
}

var napravs = map[string]string{
	"Программная Инженерия":                           "https://lists.priem.etu.ru/public/list/2ff7fa53-ffd4-4ec3-a8e2-dc600756af4f",
	"Прикладная Математика и Информатика":             "https://lists.priem.etu.ru/public/list/d249be39-64ac-4672-af1d-448ca1eb0006",
	"Информатика и Вычислительная техника":            "https://lists.priem.etu.ru/public/list/0bf4de14-49d4-4fe2-989f-64dd4d9e1b0c",
	"Инфокуммуникационные Технологии и Системы Связи": "https://lists.priem.etu.ru/public/list/1e25208b-c141-4dc1-b118-b7df2b9d9988",
	"Информационные Системы и Технологии":             "https://lists.priem.etu.ru/public/list/77795752-fee7-42e5-9ff2-f93f9f2b8b95",
}

type parsedData struct {
	data map[string][]Abits
	cap  map[string]int
	mu   sync.Mutex
}

func Check(snils string) []string {
	response := make([]string, 0, 5)
	var wg sync.WaitGroup
	data := parsedData{
		data: make(map[string][]Abits),
		cap:  make(map[string]int),
	}
	for k, v := range napravs {
		wg.Add(1)
		go func() {
			tmp, cap := formList(v)
			data.mu.Lock()
			data.data[k], data.cap[k] = tmp, cap
			data.mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	unique := make(map[string]struct{})
	var uniqueCounter int
	for k := range napravs {
		var origs int
		for i, v := range data.data[k] {
			if v.Code == snils {
				response = append(response, fmt.Sprintf("%s (всего %d мест):\nТы %d из %d, выше тебя %d оригиналов\n", k, data.cap[k], i+1, len(data.data[k]), origs))
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

	if len(response) != 0 {
		response = append(response, "Количество уникальных* аттестатов: "+strconv.Itoa(uniqueCounter)+"\n")
	} else {
		response = append(response, fmt.Sprintf("Не нашел Тебя в списках.\n\nПроверь, верен ли введенный СНИЛС (%v).\n\n*возможна также проблема в сайте вуза, тогда остается только ждать*", snils))
	}

	return response
}

func formList(url string) ([]Abits, int) {
	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("problem with sending a request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Printf("leti: response with status code %v\n", res.StatusCode)
	}
	r, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading from a resnose body: %v\n", err)
	}
	var data info
	err = json.Unmarshal(r, &data)
	if err != nil {
		log.Fatalf("error unmarshalling: %v\n", err)
	}
	return data.Data.List, data.Data.Competition.TotalNum
}
