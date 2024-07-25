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

type Data struct {
	List []Abits
}

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

type parsedData struct {
	data map[string][]Abits
	mu   sync.Mutex
}

func Check(snils string) []string {
	data := parsedData{
		data: make(map[string][]Abits),
	}
	var wg sync.WaitGroup
	response := make([]string, 0, 5)
	for k, v := range napravs {
		wg.Add(1)
		go func() {
			tmp := formList(v)
			data.mu.Lock()
			data.data[k] = tmp
			data.mu.Unlock()
			wg.Done()
		}()
	}
	wg.Wait()

	unique := make(map[string]struct{})
	var uniqueCounter int
	for k := range napravs {
		var origs int
		for _, v := range data.data[k] {
			if v.Code == snils {
				response = append(response, fmt.Sprintf("%s:\nТы %d из %d, выше тебя %d оригиналов\n", k, v.OrderNumber, len(data.data[k]), origs))
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

func formList(url string) []Abits {
	var data Data
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
	return data.List
}
