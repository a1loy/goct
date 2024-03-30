package report

import (
	"fmt"
	"goct/internal/config"
	"goct/internal/logger"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/ratelimit"
)

const (
	apiRpsLimit = 30
)

type TelegramReportClient struct {
	Name        string
	Token       string
	MsgTemplate string
	Bot         *tgbotapi.BotAPI
	Debug       bool
	ChatIDs     []int64
	RL          ratelimit.Limiter // dummy ratelimiter from https://github.com/uber-go/ratelimit
}

func NewTelegramClient(cfg config.Config) *TelegramReportClient {
	token := os.Getenv("TELEGRAM_APITOKEN")
	return &TelegramReportClient{
		Name:        "TelegramBot",
		Token:       token,
		MsgTemplate: "Found %s",
		Bot:         nil,
		Debug:       cfg.IsVerbose(),
		RL:          ratelimit.New(apiRpsLimit, ratelimit.Per(time.Second*3)),
	}
}

func (c *TelegramReportClient) Init(cfg config.Config) {
	logger.Debugf("running init for %s", c.Name)
	notifications := cfg.GetNotificationsCfg()
	for _, notification := range notifications {
		if notification.Type == "telegram" {
			botPtr, err := tgbotapi.NewBotAPI(c.Token)
			if err != nil {
				panic(fmt.Sprintf("unable to init telegram client due to %s", err.Error()))
			}
			c.Bot = botPtr
			for _, chatIDstr := range notification.Recipients {
				chatID, convErr := strconv.Atoi(chatIDstr)
				if convErr == nil {
					c.ChatIDs = append(c.ChatIDs, int64(chatID))
				}
			}
		}
	}
}

func (c *TelegramReportClient) Report(msg string) error {
	for _, chatID := range c.ChatIDs {
		logger.Debugf("Sending msg %s to chat %d", msg, chatID)
		msg = fmt.Sprintf("\n%s", msg)
		nMsg := tgbotapi.NewMessage(chatID, msg)
		nMsg.ParseMode = tgbotapi.ModeMarkdownV2
		c.RL.Take()
		if _, err := c.Bot.Send(nMsg); err != nil {
			return err
		}
	}
	return nil
}
