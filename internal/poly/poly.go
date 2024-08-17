package poly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"ratinger/pkg/auth"
	"ratinger/pkg/db"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type naprav struct {
	Id                int
	Name              string
	DirectionCapacity int
	List              []abit
	Payment           string
	Form              string
	EduLevel          string
	Url               string
}

type abit struct {
	UserSnils            string
	FullScore            float32
	HasOriginalDocuments bool
	Priority             int
}

func Check(u *auth.User) []string {
	napravs := retrieveNapravs(u)
	var wg sync.WaitGroup
	response := make([]string, 0)
	semaphore := make(chan struct{}, 20)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	for i := range napravs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			napravs[i].getList(redisClient)
			wg.Done()
			<-semaphore
		}()
	}
	wg.Wait()

	unique := make(map[string]struct{})
	var uniqueCounter int
	var abitNapravs []int
	for _, n := range napravs {
		var origs, uc int
		for i, v := range n.List {
			if v.UserSnils == u.Snils {
				response = append(response, fmt.Sprintf("%s (всего %d мест):\nТы %d из %d, выше тебя %d оригиналов\n", n.Name, n.DirectionCapacity, i+1, len(n.List), origs))
				uniqueCounter += uc
				abitNapravs = append(abitNapravs, n.Id)
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

func (n *naprav) getList(r *redis.Client) {
	var redisKey = fmt.Sprintf("spbstu:%d", n.Id)
	if jsonList, err := r.Get(context.Background(), redisKey).Result(); err == nil {
		err = json.Unmarshal([]byte(jsonList), &n)
		if err != nil {
			log.Fatal(err)
		}
		return
	} else if !errors.Is(err, redis.Nil) {
		log.Fatal(err)
	}

	defer func() {
		data, err := json.Marshal(n)
		if err != nil {
			log.Fatal(err)
		}

		err = r.SetNX(context.Background(), redisKey, data, 10*time.Minute).Err()
		if err != nil {
			log.Fatal(err)
		}
	}()

	client := &http.Client{}
	req, err := http.NewRequest("GET", n.Url, nil)
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

	read, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("error reading: %v", err)
	}

	err = json.Unmarshal(read, &n)
	if err != nil {
		log.Fatalf("error unmarshalling: %v", err)
	}
}

func retrieveNapravs(u *auth.User) []naprav {
	napravs := make([]naprav, 0, len(u.Spbstu))
	conn := db.NeonConnect()
	defer conn.Close(context.Background())
	if u.Spbstu != nil {
		rows, err := conn.Query(context.Background(), "select * from spbstu where id = any($1)", u.Spbstu)
		if err != nil {
			log.Fatalf("failed getting spbstu: %v", err)
		}
		for rows.Next() {
			var n naprav
			rows.Scan(&n.Id, &n.Name, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
			napravs = append(napravs, n)
		}
		return napravs
	}

	p, f, el := parseAbitConstraints(u)
	rows, err := conn.Query(context.Background(), "select * from spbstu where payment = any($1) and form = any($2) and edu_level=any($3)", p, f, el)
	if err != nil {
		log.Fatalf("failed getting spbstu: %v", err)
	}
	for rows.Next() {
		var n naprav
		rows.Scan(&n.Id, &n.Name, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
		napravs = append(napravs, n)
	}
	return napravs
}

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
