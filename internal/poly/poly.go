package poly

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
	List              []Abits
	DirectionCapacity int
}

type Abits struct {
	UserSnils            string
	FullScore            int
	HasOriginalDocuments bool
	Priority             int
}

var napravs = map[string]string{
	"Математика и Компьютерные Науки":                   "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId=2193",
	"Математическое Обеспечение и Администрирование КС": "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId=2199",
	"Прикладная Информатика":                            "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId=2281",
	"Программная Инженерия":                             "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId=2321",
	"Информационныые Системы и Технологии":              "https://enroll.spbstu.ru/applications-manager/api/v1/admission-list/form-rating?applicationEducationLevel=BACHELOR&directionEducationFormId=2&directionId=2156",
}

type parsedData struct {
	data map[string][]Abits
	cap  map[string]int
	mu   sync.Mutex
}

func Check(snils string) []string {
	data := parsedData{
		data: make(map[string][]Abits),
		cap:  make(map[string]int),
	}
	var wg sync.WaitGroup
	response := make([]string, 0, 6)
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
			if v.UserSnils == snils {
				response = append(response, fmt.Sprintf("%s (всего %d мест):\nТы %d из %d, выше тебя %d оригиналов\n", k, data.cap[k], i+1, len(data.data[k]), origs))
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
