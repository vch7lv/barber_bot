package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config — конфигурация приложения. Загружается из env.
type Config struct {
	TelegramBotToken     string
	BarberTelegramIDs    []int64
	DatabaseDSN          string
	TZ                   *time.Location
	BackupTime           string // HH:MM по МСК
	BackupDir            string
	ReminderBeforeHours       int // напоминание клиенту за N часов до визита
	BarberReminderBeforeHours int // напоминание барберу за N часов до визита (0 — выключено)
	LogLevel             string
	BotMode              string
	WebhookURL           string
	WebhookPort          int
}

// Load читает конфиг из окружения и валидирует его.
// Если в текущей директории есть файл .env, переменные из него подставляются в env.
func Load() (*Config, error) {
	loadEnvFile(".env")
	c := &Config{
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseDSN:         getEnv("DATABASE_DSN", "file:barber_bot.db?_journal_mode=WAL"),
		BackupTime:          getEnv("BACKUP_TIME", "22:00"),
		BackupDir:           getEnv("BACKUP_DIR", "./backups"),
		ReminderBeforeHours:       getEnvInt("REMINDER_BEFORE_HOURS", 2),
		BarberReminderBeforeHours: getEnvInt("BARBER_REMINDER_BEFORE_HOURS", 1),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
		BotMode:             getEnv("BOT_MODE", "polling"),
		WebhookURL:          os.Getenv("WEBHOOK_URL"),
		WebhookPort:         getEnvInt("WEBHOOK_PORT", 8080),
	}

	if c.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	barberIDsRaw := os.Getenv("BARBER_TELEGRAM_IDS")
	if barberIDsRaw == "" {
		return nil, fmt.Errorf("BARBER_TELEGRAM_IDS is required and must not be empty")
	}
	for _, s := range strings.Split(barberIDsRaw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("BARBER_TELEGRAM_IDS: invalid id %q: %w", s, err)
		}
		c.BarberTelegramIDs = append(c.BarberTelegramIDs, id)
	}
	if len(c.BarberTelegramIDs) == 0 {
		return nil, fmt.Errorf("BARBER_TELEGRAM_IDS must contain at least one id")
	}

	tzName := getEnv("TZ", "Europe/Moscow")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("TZ: invalid timezone %q: %w", tzName, err)
	}
	c.TZ = loc

	if err := validateBackupTime(c.BackupTime); err != nil {
		return nil, err
	}

	allowedLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !allowedLevels[c.LogLevel] {
		return nil, fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	allowedModes := map[string]bool{"polling": true, "webhook": true}
	if !allowedModes[c.BotMode] {
		return nil, fmt.Errorf("BOT_MODE must be polling or webhook")
	}

	if c.BotMode == "webhook" && c.WebhookURL == "" {
		return nil, fmt.Errorf("WEBHOOK_URL is required when BOT_MODE=webhook")
	}

	return c, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func validateBackupTime(s string) error {
	if len(s) != 5 || s[2] != ':' {
		return fmt.Errorf("BACKUP_TIME must be HH:MM, got %q", s)
	}
	h, err1 := strconv.Atoi(s[:2])
	m, err2 := strconv.Atoi(s[3:5])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return fmt.Errorf("BACKUP_TIME must be valid HH:MM, got %q", s)
	}
	return nil
}

// IsBarber возвращает true, если telegramID в списке барберов.
func (c *Config) IsBarber(telegramID int64) bool {
	for _, id := range c.BarberTelegramIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}

// loadEnvFile читает файл .env и выставляет переменные в os.Environ (только если они ещё не заданы).
func loadEnvFile(name string) {
	path := name
	if !filepath.IsAbs(name) {
		if wd, err := os.Getwd(); err == nil {
			path = filepath.Join(wd, name)
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if key == "" {
			continue
		}
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
