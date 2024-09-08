package spbu

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
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/html"
)

type naprav struct {
	Id       int
	Name     string
	Capacity int
	List     []abit
	Payment  string
	Form     string
	EduLevel string
	Url      string
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

func Check(repo *repository.Repo, logger *slog.Logger, u *auth.User) []string {
	napravs, err := retrieveNapravs(repo, u)
	if err != nil {
		logger.Error("failed to retrieve napravs", slog.Any("error", err))
		return nil
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	for i := range napravs {
		wg.Add(1)
		semaphore <- struct{}{}
		go func() {
			defer wg.Done()
			napravs[i].getList(logger, redisClient)
			<-semaphore
		}()
	}
	wg.Wait()

	var response []string

	var userNapravs []int

	for _, n := range napravs {
		var origs int
		for _, abit := range n.List {
			if abit.Snils == u.Snils {
				s := fmt.Sprintf("%s: Ты %d из %d\nПеред тобой %d оригиналов", n.Name, abit.OrderNumber, len(n.List), origs)
				response = append(response, s)
				userNapravs = append(userNapravs, n.Id)
				break
			}
			if abit.HasOriginal {
				origs++
			}
		}

	}

	if len(userNapravs) > len(u.Spbu) || u.Spbu == nil {
		u.Spbu = userNapravs
		args := repository.Args{
			"spbu": u.Spbu,
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

	if response == nil {
		response = append(response, "Не нашел тебя в списках")
	}
	return response

}

// ugly parsing of html but i let it be
func (n *naprav) getList(logger *slog.Logger, r *redis.Client) {
	op := "spbu.getlist"
	logger = logger.With(
		slog.String("op", op),
		slog.Int("naprav_id", n.Id),
	)
	var redisKey = fmt.Sprintf("spbu:%d", n.Id)

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

	resp, err := http.Get(n.Url)
	if err != nil {
		logger.Error(
			"failed making http request",
			slog.String("url", n.Url),
			slog.Any("error", err),
		)
	}

	logger = logger.With(
		slog.Int("response code", resp.StatusCode),
	)

	if n.EduLevel == "Бакалавриат" {
		r, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("failed reading response", slog.Any("error", err))
			return
		}
		var data bachData
		err = json.Unmarshal(r, &data)
		if err != nil {
			logger.Error("error unmarshalling response data", slog.Any("error", err))
			return
		}
		n.List = data.List
		return
	}

	// if n.eduLevel == "Магистратура" || if n.eduLevel == "Аспирантура"
	doc, err := html.Parse(resp.Body)
	if err != nil {
		logger.Error("failed to parse response body (html)", slog.Any("error", err))
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
									logger.Error("error atoi'ing", slog.String("supposed to be int", td.Data), slog.Any("error", err))
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
										logger.Error("error atoi'ing", slog.String("supposed to be int", td.Data), slog.Any("error", err))
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
									logger.Error("error atoi'ing", slog.String("supposed to be int", td.Data), slog.Any("error", err))
								}
								a.Score = num
							case 7:
								if td.FirstChild.Data == "Да" {
									a.HasOriginal = true
								}
							}
						}

					}
					n.List = append(n.List, a)
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

func retrieveNapravs(repo *repository.Repo, u *auth.User) ([]naprav, error) {
	napravs := make([]naprav, 0, len(u.Spbu))

	if u.Spbu != nil {
		rows, err := repo.Db.SelectQuery(context.Background(), "select * from spbu where id = any($1)", u.Spbu)
		if err != nil {
			return nil, fmt.Errorf("failed getting user.spbu: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var n naprav
			rows.Scan(&n.Id, &n.Name, &n.Capacity, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
			napravs = append(napravs, n)
		}
		return napravs, nil
	}

	p, f, el := parseAbitConstraints(u)
	rows, err := repo.Db.SelectQuery(context.Background(), "select * from spbu where payment = any($1) and form = any($2) and edu_level=any($3)", p, f, el)
	if err != nil {
		return nil, fmt.Errorf("failed getting user.spbu: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n naprav
		rows.Scan(&n.Id, &n.Name, &n.Capacity, &n.Payment, &n.Form, &n.EduLevel, &n.Url)
		napravs = append(napravs, n)
	}
	return napravs, nil
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
