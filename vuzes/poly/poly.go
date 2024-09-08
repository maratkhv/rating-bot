package poly

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"ratinger/internal/models/auth"
	"ratinger/internal/repository"
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

func Check(repo *repository.Repo, logger *slog.Logger, u *auth.User) []string {
	napravs, err := retrieveNapravs(repo, u)
	if err != nil {
		logger.Error("failed to retrieve napravs", slog.Any("error", err))
		return nil
	}

	var wg sync.WaitGroup
	response := make([]string, 0)
	semaphore := make(chan struct{}, 20)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	for i := range napravs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			napravs[i].getList(logger, redisClient)
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

	if len(abitNapravs) > len(u.Spbstu) || u.Spbstu == nil {
		u.Spbstu = abitNapravs
		args := repository.Args{
			"spbstu": u.Spbstu,
		}
		err := repo.Db.UpdateUser(context.Background(), u.Id, args)
		if err != nil {
			logger.Error(
				"failed updating user",
				slog.Int64("user_id", u.Id),
				slog.Any("args", args),
			)
		}
	}

	if len(response) != 0 {
		response = append(response, "Количество уникальных* аттестатов: "+strconv.Itoa(uniqueCounter)+"\n")
	} else {
		response = append(response, fmt.Sprintf("Не нашел Тебя в списках.\n\nПроверь, верен ли введенный СНИЛС (%v).\n\n*возможна также проблема в сайте вуза, тогда остается только ждать*", u.Snils))
	}

	return response

}

func (n *naprav) getList(logger *slog.Logger, r *redis.Client) {
	op := "spbstu.getlist"
	logger = logger.With(
		slog.String("op", op),
		slog.Int("naprav_id", n.Id),
	)

	var redisKey = fmt.Sprintf("spbstu:%d", n.Id)
	if jsonList, err := r.Get(context.Background(), redisKey).Result(); err == nil {
		err = json.Unmarshal([]byte(jsonList), &n)
		if err != nil {
			logger.Error("failed to unmarshal data from redis", slog.Any("error", err))
		} else {
			return
		}
	} else if !errors.Is(err, redis.Nil) {
		logger.Error("failed to get data from redis", slog.Any("error", err))
	}

	defer func() {
		data, err := json.Marshal(n)
		if err != nil {
			logger.Error("failed to marshal data", slog.Any("error", err), slog.Any("naprav", n))
			return
		}

		err = r.SetNX(context.Background(), redisKey, data, 10*time.Minute).Err()
		if err != nil {
			logger.Error("failed to set data to redis", slog.Any("error", err))
			return
		}
	}()

	client := &http.Client{}
	req, err := http.NewRequest("GET", n.Url, nil)
	if err != nil {
		logger.Error("failed creating request", slog.Any("error", err))
	}
	req.Header.Add("Accept", `application/json,text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8`)
	req.Header.Add("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.64 Safari/537.11`)
	res, err := client.Do(req)
	if err != nil {
		logger.Error("failed doing request", slog.Any("error", err), slog.String("url", n.Url))
	}
	defer res.Body.Close()

	logger = logger.With(
		slog.Int("response code", res.StatusCode),
	)

	read, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error("failed reading response", slog.Any("error", err))
		return
	}

	err = json.Unmarshal(read, &n)
	if err != nil {
		logger.Error("error unmarshalling response data", slog.Any("error", err))
		return
	}
}

func retrieveNapravs(repo *repository.Repo, u *auth.User) ([]naprav, error) {
	napravs := make([]naprav, 0, len(u.Spbstu))

	if u.Spbstu != nil {
		rows, err := repo.Db.SelectQuery(context.Background(), "select * from spbstu where id = any($1)", u.Spbstu)
		if err != nil {
			return nil, fmt.Errorf("failed getting user.spbstu: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var n naprav
			rows.Scan(&n.Id, &n.Name, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
			napravs = append(napravs, n)
		}
		return napravs, nil
	}

	p, f, el := parseAbitConstraints(u)
	rows, err := repo.Db.SelectQuery(context.Background(), "select * from spbstu where payment = any($1) and form = any($2) and edu_level=any($3)", p, f, el)
	if err != nil {
		return nil, fmt.Errorf("failed getting user.spbstu: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n naprav
		rows.Scan(&n.Id, &n.Name, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
		napravs = append(napravs, n)
	}
	return napravs, nil
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
