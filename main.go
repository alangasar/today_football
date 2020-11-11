package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/ilyakaznacheev/cleanenv"
)

// Config предосталяет различные параматеры приложения
type Config struct {
	Telegram struct {
		Token string `yaml:"token"`
		Debug bool   `yaml:"debug"`
	} `yaml:"telegram"`
	Database struct {
		Host     string `yaml:"host"`
		Dbname   string `yaml:"dbname"`
		Username string `yaml:"user"`
		Password string `yaml:"pass"`
	} `yaml:"database"`
}

var (
	cfg           Config
	id            int
	home, visitor string
	matchtime     int
)

func main() {

	// прочитать конфиг
	err := cleanenv.ReadConfig("config.yml", &cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	db, err := sql.Open("mysql",
		fmt.Sprintf("%s:%s@%s/%s",
			cfg.Database.Username, cfg.Database.Password, cfg.Database.Host, cfg.Database.Dbname))

	if err != nil {
		panic(err)
	}

	bot, _ := tgbotapi.NewBotAPI(cfg.Telegram.Token)

	bot.Debug = cfg.Telegram.Debug

	log.Printf("Бот %s авторизован", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	//Получаем обновления от бота
	updates, err := bot.GetUpdatesChan(u)

	go func() {
		for range time.Tick(time.Minute) {
			currentTime := time.Now()
			//отправить сообщение в определенное время
			if currentTime.Format("15:04") == "09:05" {
				message := getTodayMatch(db)
				msg := tgbotapi.NewMessageToChannel("@today_football", message[0])

				if msg.Text != "" {
					bot.Send(msg)
				}
			}
		}
	}()

	for update := range updates {

		if update.Message == nil {
			continue
		}
		var command = ""
		command = update.Message.Command()
		args := update.Message.CommandArguments()

		// выполнить, если не команда
		if command == "" {

		} else {
			switch command {
			case "start":
				firstMsg := fmt.Sprintf("Buenas, %s! Mi nombre es Bruno, и я, пожалуй, самый нужный бот в телеге! Шли /today", update.Message.From.FirstName)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, firstMsg)
				bot.Send(msg)
			case "add":
				match := strings.Split(args, ",")
				if update.Message.From.UserName == "yura_gushchin" {
					if len(match) != 3 {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не верный формат матча. Пример: Зенит,Реал,20.12.2020 21:00")
						bot.Send(msg)
					} else {
						insert(db, strings.TrimSpace(match[0]), strings.TrimSpace(match[1]), strings.TrimSpace(match[2]))
					}
				}

			case "list":
				matches := getAllMatches(db)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, strings.Join(matches, "\n"))
				bot.Send(msg)

			case "today":
				matches := getTodayMatch(db)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, strings.Join(matches, "\n"))
				bot.Send(msg)
			}
		}

	}

	select {} // зациклить выполнение программы
}

func insert(db *sql.DB, home, visitor string, matchtime string) {
	res, err := db.Prepare("INSERT INTO matches(home, visitor, time) VALUES( ?, ?, ? )")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()

	layout := "02.01.2006 15:04"
	t, err := time.Parse(layout, matchtime)
	if err != nil {
		fmt.Println(err)
	}

	if _, err := res.Exec(home, visitor, t.Unix()); err != nil {
		log.Fatal(err)
	}

}

func getAllMatches(db *sql.DB) []string {
	allMathes := []string{}

	rows, err := db.Query("SELECT id, home, visitor, time FROM matches ORDER BY time")
	if err != nil {
		panic(err.Error())
	}

	for rows.Next() {
		err := rows.Scan(&id, &home, &visitor, &matchtime)
		if err != nil {
			log.Fatal(err)
		}
		allMathes = append(allMathes, fmt.Sprintf("- %s - %s, %s", home, visitor, dateFormat("02.01.2006 в 15:04", matchtime)))
	}
	allMathes = append(allMathes, "\n___\nУведомления в день матча на канале @today_football")
	return allMathes
}

func getTodayMatch(db *sql.DB) []string {
	allMathes := []string{}

	rows, err := db.Query("SELECT id, home, visitor, time FROM matches WHERE FROM_UNIXTIME(time,'%Y-%m-%d') = CURDATE()")
	if err != nil {
		panic(err.Error())
	}

	for rows.Next() {
		err := rows.Scan(&id, &home, &visitor, &matchtime)
		if err != nil {
			log.Fatal(err)
		}
		allMathes = append(allMathes, fmt.Sprintf("Сегодня Футбол! %s - %s, %s", home, visitor, dateFormat("02.01.2006 в 15:04", matchtime)))
		return allMathes
	}
	allMathes = append(allMathes, "Сожалею, сегодня нет матчей.\n___\nУведомления в день матча на канале @today_football")
	return allMathes
}

func dateFormat(layout string, d int) string {
	intTime := int64(d)
	t := time.Unix(intTime, 0)
	if layout == "" {
		layout = "02.01.2006 в 15:04"
	}
	return t.Format(layout)
}
