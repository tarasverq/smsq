package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/tink"
	_ "github.com/mattn/go-sqlite3"
)

var errBlockedByUser = errors.New("blocked by user")

// checkErr panics on an error
func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

type smsRequest struct {
	Payload string
	Version int
}

type sms struct {
	Key       string `json:"key"`
	ID        int64  `json:"id"`
	Text      string `json:"text"`
	SIM       string `json:"sim"`
	Carrier   string `json:"carrier"`
	Sender    string `json:"sender"`
	Timestamp int64  `json:"timestamp"`
	Offset    int    `json:"offset"`
}

type deliveryResult int

//go:generate jsonenums -type=deliveryResult
const (
	delivered deliveryResult = iota
	networkError
	blocked
	badRequest
	userNotFound
	apiRetired
	rateLimited
)

type smsResponse struct {
	Error  *string         `json:"error"`
	Result *deliveryResult `json:"result"`
}

type deliverCommand struct {
	sms    sms
	result chan deliveryResult
}

type worker struct {
	bot         *tg.BotAPI
	db          *sql.DB
	cfg         *config
	client      *http.Client
	deliverChan chan deliverCommand
	decryptor   tink.HybridDecrypt
}

func newWorker() *worker {
	if len(os.Args) != 2 {
		panic("usage: smsq <config>")
	}
	cfg := readConfig(os.Args[1])
	client := &http.Client{Timeout: time.Second * time.Duration(cfg.TimeoutSeconds)}
	bot, err := tg.NewBotAPIWithClient(cfg.BotToken, tg.APIEndpoint, client)
	checkErr(err)
	db, err := sql.Open("sqlite3", cfg.DBPath)
	checkErr(err)
	decryptor, err := hybrid.NewHybridDecrypt(cfg.privateKey)
	checkErr(err)
	w := &worker{
		bot:         bot,
		db:          db,
		cfg:         cfg,
		client:      client,
		deliverChan: make(chan deliverCommand),
		decryptor:   decryptor,
	}

	return w
}

func (r deliveryResult) String() string {
	switch r {
	case delivered:
		return "delivered"
	case networkError:
		return "network_error"
	case blocked:
		return "blocked"
	case badRequest:
		return "bad_request"
	case userNotFound:
		return "user_not_found"
	case apiRetired:
		return "api_retired"
	case rateLimited:
		return "rate_limited"
	default:
		return "undefined"
	}
}

func (w *worker) logConfig() {
	cfgString, err := json.MarshalIndent(w.cfg, "", "    ")
	checkErr(err)
	linf("config: " + string(cfgString))
}

func (w *worker) setWebhook() {
	linf("setting webhook...")
	_, err := w.bot.SetWebhook(tg.NewWebhook(path.Join(w.cfg.WebhookDomain, w.cfg.BotToken)))
	checkErr(err)
	info, err := w.bot.GetWebhookInfo()
	checkErr(err)
	if info.LastErrorDate != 0 {
		linf("last webhook error time: %v", time.Unix(int64(info.LastErrorDate), 0))
	}
	if info.LastErrorMessage != "" {
		linf("last webhook error message: %s", info.LastErrorMessage)
	}
	linf("OK")
}

func (w *worker) removeWebhook() {
	linf("removing webhook...")
	_, err := w.bot.RemoveWebhook()
	checkErr(err)
	linf("OK")
}

func (w *worker) mustExec(query string, args ...interface{}) sql.Result {
	stmt, err := w.db.Prepare(query)
	checkErr(err)
	result, err := stmt.Exec(args...)
	checkErr(err)
	checkErr(stmt.Close())
	return result
}

func (w *worker) storedMidnight() int64 {
	if singleInt(w.db.QueryRow("select count(*) from midnight")) == 0 {
		w.mustExec("insert into midnight (unix_time) values (0)")
		return 0
	}
	return singleInt64(w.db.QueryRow("select unix_time from midnight"))
}

func (w *worker) storeMidnight(midnight int64) {
	if singleInt(w.db.QueryRow("select count(*) from midnight")) == 0 {
		w.mustExec("insert into midnight (unix_time) values (?)", midnight)
		return
	}
	w.mustExec("update midnight set unix_time=?", midnight)
}

func midnight() int64 {
	return time.Now().Truncate(24 * time.Hour).Unix()
}

func (w *worker) keyForChat(chatID int64) *string {
	query, err := w.db.Query("select key from users where chat_id=? and deleted=0", chatID)
	checkErr(err)
	defer func() { checkErr(query.Close()) }()
	if !query.Next() {
		return nil
	}
	var chatKey string
	checkErr(query.Scan(&chatKey))
	return &chatKey
}

func (w *worker) chatForKey(chatKey string) (*int64, int) {
	query, err := w.db.Query("select chat_id, daily_limit from users where key=? and deleted=0", chatKey)
	checkErr(err)
	defer func() { checkErr(query.Close()) }()
	if !query.Next() {
		return nil, 0
	}
	var chatID int64
	var dailyLimit int
	checkErr(query.Scan(&chatID, &dailyLimit))
	return &chatID, dailyLimit
}

func checkKey(key string) bool {
	hash := sha256.Sum256([]byte(key))
	hash = sha256.Sum256(hash[:])
	return hash[0] == 0 && hash[1]&0xf0 == 0
}

func (w *worker) userExists(chatID int64) bool {
	return singleInt(w.db.QueryRow("select count(*) from users where chat_id=? and deleted=0", chatID)) != 0
}

func (w *worker) stop(chatID int64) {
	w.mustExec("update users set deleted=1 where chat_id=?", chatID)
	_ = w.sendText(chatID, false, parseRaw, "Access revoked")
}

func (w *worker) start(chatID int64, key string) {
	if key == "" && w.userExists(chatID) {
		_ = w.sendText(chatID, false, parseRaw, "You are already set up!")
		return
	}
	if key == "" || !checkKey(key) {
		_ = w.sendText(chatID, false, parseRaw, "Install smsQ application on your phone https://smsq.me")
		return
	}
	chatKey := w.keyForChat(chatID)
	if chatKey != nil && *chatKey == key {
		_ = w.sendText(chatID, false, parseRaw, "You are already set up!")
		return
	}
	if chatKey != nil {
		_ = w.sendText(chatID, false, parseRaw, "Your previous subscription is revoked")
	}

	existingChatID, _ := w.chatForKey(key)
	if existingChatID != nil && *existingChatID != chatID {
		_ = w.sendText(chatID, false, parseRaw, "Subscription on other Telegram account has been revoked")
		_ = w.sendText(*existingChatID, false, parseRaw, "Your subscription has been revoked from other Telegram account")
	}

	w.mustExec(`
		insert or replace into users (chat_id, key, daily_limit) values (?, ?, ?)
		on conflict(chat_id) do update set key=excluded.key, deleted=0`,
		chatID,
		key,
		w.cfg.DeliveredLimit)

	_ = w.sendText(chatID, false, parseRaw, "Congratulations! You should see here new SMS messages")
}

func (w *worker) broadcastChats() (chats []int64) {
	chatsQuery, err := w.db.Query(`select chat_id from users where deleted=0`)
	checkErr(err)
	defer func() { checkErr(chatsQuery.Close()) }()
	for chatsQuery.Next() {
		var chatID int64
		checkErr(chatsQuery.Scan(&chatID))
		chats = append(chats, chatID)
	}
	return
}

func (w *worker) broadcast(text string) {
	if text == "" {
		return
	}
	chats := w.broadcastChats()
	for _, chatID := range chats {
		_ = w.sendText(chatID, true, parseRaw, text)
	}
	_ = w.sendText(w.cfg.AdminID, false, parseRaw, "OK")
}

func (w *worker) direct(arguments string) {
	parts := strings.SplitN(arguments, " ", 2)
	if len(parts) < 2 {
		_ = w.sendText(w.cfg.AdminID, false, parseRaw, "Usage: /direct chatID text")
		return
	}
	whom, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		_ = w.sendText(w.cfg.AdminID, false, parseRaw, "First argument is invalid")
		return
	}
	text := parts[1]
	if text == "" {
		return
	}
	_ = w.sendText(whom, true, parseRaw, text)
	_ = w.sendText(w.cfg.AdminID, false, parseRaw, "OK")
}

func (w *worker) limit(arguments string) {
	parts := strings.SplitN(arguments, " ", 2)
	if len(parts) < 2 {
		_ = w.sendText(w.cfg.AdminID, false, parseRaw, "Usage: /limit chatID text")
		return
	}
	whom, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		_ = w.sendText(w.cfg.AdminID, false, parseRaw, "First argument is invalid")
		return
	}
	limit, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		_ = w.sendText(w.cfg.AdminID, false, parseRaw, "Second argument is invalid")
		return
	}
	result := w.mustExec("update users set daily_limit=? where chat_id=?", limit, whom)
	answer := "OK"
	rows, err := result.RowsAffected()
	checkErr(err)
	if rows != 1 {
		answer = "User not found"
	}
	_ = w.sendText(w.cfg.AdminID, false, parseRaw, answer)
}

func (w *worker) processAdminMessage(chatID int64, command, arguments string) bool {
	switch command {
	case "stat":
		w.stat()
		return true
	case "broadcast":
		w.broadcast(arguments)
		return true
	case "direct":
		w.direct(arguments)
		return true
	case "limit":
		w.limit(arguments)
		return true
	}
	return false
}

func (w *worker) processIncomingCommand(chatID int64, command, arguments string) {
	command = strings.ToLower(command)
	if chatID == w.cfg.AdminID && w.processAdminMessage(chatID, command, arguments) {
		return
	}
	switch command {
	case "stop":
		w.stop(chatID)
	case "feedback":
		w.feedback(chatID, arguments)
	case "start":
		w.start(chatID, arguments)
	case "challenge":
		if reply, ok := w.cfg.Challenges[arguments]; ok {
			_ = w.sendText(chatID, false, parseRaw, reply)
		} else {
			_ = w.sendText(chatID, false, parseRaw, "Unknown command")
		}
	case "help":
		_ = w.sendText(chatID, false, parseHTML,
			""+
				"smsq: Receive SMS messages in Telegram\n"+
				"1. Install Android app\n"+
				"2. Open app, start forwarding, connect Telegram\n"+
				"3. Now you receive your SMS messages in this bot!\n"+
				"Project page: https://smsq.me\n"+
				"Source code: https://github.com/igrmk/smsq\n"+
				"\n"+
				"Bot commands:\n"+
				"<b>/help</b> — Help\n"+
				"<b>/stop</b> — Revoke access\n"+
				"<b>/feedback</b> — Send feedback")
	default:
		_ = w.sendText(chatID, false, parseRaw, "Unknown command")
	}
}

func (w *worker) ourID() int64 {
	if idx := strings.Index(w.cfg.BotToken, ":"); idx != -1 {
		id, err := strconv.ParseInt(w.cfg.BotToken[:idx], 10, 64)
		checkErr(err)
		return id
	}
	checkErr(errors.New("cannot get our ID"))
	return 0
}

func (w *worker) processTGUpdate(u tg.Update) {
	onlyInAPrivateChat := "smsq_bot works only in a private chat"
	if u.Message != nil && u.Message.Chat != nil {
		if newMembers := u.Message.NewChatMembers; newMembers != nil && len(*newMembers) > 0 {
			ourID := w.ourID()
			for _, m := range *newMembers {
				if int64(m.ID) == ourID {
					_ = w.sendText(u.Message.Chat.ID, false, parseRaw, onlyInAPrivateChat)
					break
				}
			}
		} else if u.Message.IsCommand() {
			if u.Message.Chat.Type != "private" {
				_ = w.sendText(u.Message.Chat.ID, false, parseRaw, onlyInAPrivateChat)
				return
			}
			w.processIncomingCommand(u.Message.Chat.ID, u.Message.Command(), u.Message.CommandArguments())
		}
	} else if u.ChannelPost != nil && u.ChannelPost.Chat != nil && u.ChannelPost.IsCommand() {
		_ = w.sendText(u.ChannelPost.Chat.ID, false, parseRaw, onlyInAPrivateChat)
		return
	}
}

func (w *worker) feedback(chatID int64, text string) {
	if text == "" {
		_ = w.sendText(chatID, false, parseRaw, "Command format: /feedback <text>")
		return
	}
	w.mustExec("insert into feedback (chat_id, text) values (?, ?)", chatID, text)
	_ = w.sendText(chatID, false, parseRaw, "Thank you for your feedback")
	_ = w.sendText(w.cfg.AdminID, true, parseRaw, fmt.Sprintf("Feedback from %d: %s", chatID, text))
}

func (w *worker) userCount() int {
	query := w.db.QueryRow("select count(*) from users where deleted=0")
	return singleInt(query)
}

func (w *worker) activeUserCount() int {
	query := w.db.QueryRow("select count(*) from users where delivered > 0 and deleted=0")
	return singleInt(query)
}

func (w *worker) smsCount() int {
	query := w.db.QueryRow("select coalesce(sum(delivered), 0) from users")
	return singleInt(query)
}

func (w *worker) smsTodayCount() int {
	query := w.db.QueryRow("select coalesce(sum(delivered_today), 0) from users")
	return singleInt(query)
}

func (w *worker) stat() {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("users: %d", w.userCount()))
	lines = append(lines, fmt.Sprintf("active users: %d", w.activeUserCount()))
	lines = append(lines, fmt.Sprintf("smses: %d", w.smsCount()))
	lines = append(lines, fmt.Sprintf("smses today: %d", w.smsTodayCount()))
	_ = w.sendText(w.cfg.AdminID, false, parseRaw, strings.Join(lines, "\n"))
}

func (w *worker) sendText(chatID int64, notify bool, parse parseKind, text string) error {
	msg := tg.NewMessage(chatID, text)
	msg.DisableNotification = !notify
	switch parse {
	case parseHTML, parseMarkdown:
		msg.ParseMode = parse.String()
	}
	return w.send(&messageConfig{msg})
}

func (w *worker) send(msg baseChattable) error {
	if _, err := w.bot.Send(msg); err != nil {
		switch err := err.(type) {
		case tg.Error:
			if err.Code == 403 {
				linf("bot is blocked by the user %d, %v", msg.baseChat().ChatID, err)
				return errBlockedByUser
			}
			lerr("cannot send a message to %d, code %d, %v", msg.baseChat().ChatID, err.Code, err)
		default:
			lerr("unexpected error type while sending a message to %d, %v", msg.baseChat().ChatID, err)
		}
		return err
	}
	return nil
}

func (w *worker) handleRetired(writer http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(writer, "404 not found", http.StatusNotFound)
		return
	}

	w.ldbg("got retired API call")
	writer.Header().Set("Content-Type", "application/json")
	w.apiReply(writer, apiRetired)
}

func (w *worker) handleV1SMS(writer http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(writer, "404 not found", http.StatusNotFound)
		return
	}

	w.ldbg("got new SMS")

	var request smsRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.ldbg("cannot decode v1 request")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Version != 1 {
		lerr("version is not 1")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	decrypted, err := w.decrypt(request.Payload)
	if err != nil {
		lerr("decryption error")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	var sms sms
	err = json.NewDecoder(bytes.NewReader(decrypted)).Decode(&sms)
	if err != nil {
		lerr("cannot decode SMS")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	if false ||
		!utf8.ValidString(sms.Text) ||
		!utf8.ValidString(sms.Carrier) ||
		!utf8.ValidString(sms.SIM) ||
		!utf8.ValidString(sms.Sender) {
		w.apiReply(writer, badRequest)
		lerr("invalid text")
		return
	}

	deliver := deliverCommand{sms: sms, result: make(chan deliveryResult)}
	defer close(deliver.result)
	w.deliverChan <- deliver
	result := <-deliver.result
	w.apiReply(writer, result)
}

func (w *worker) apiReply(writer http.ResponseWriter, result deliveryResult) {
	writer.WriteHeader(http.StatusOK)
	res := smsResponse{Result: &result}
	resString, err := json.Marshal(res)
	checkErr(err)
	_, err = writer.Write(resString)
	checkErr(err)
}

func (w *worker) handleEndpoints() {
	http.HandleFunc("/v0/sms", w.handleRetired)
	http.HandleFunc("/v1/sms", w.handleV1SMS)
}

func (w *worker) deliver(sms sms) deliveryResult {
	chatID, dailyLimit := w.chatForKey(sms.Key)
	if chatID == nil {
		w.ldbg("cannot found user")
		return userNotFound
	}

	w.mustExec("update users set received_today=received_today+1 where chat_id=?", *chatID)
	receivedToday := singleInt(w.db.QueryRow("select received_today from users where chat_id=?", *chatID))
	if receivedToday >= w.cfg.ReceivedLimit+1 {
		return rateLimited
	}

	deliveredToday := singleInt(w.db.QueryRow("select delivered_today from users where chat_id=?", *chatID))
	if deliveredToday >= dailyLimit {
		if deliveredToday == dailyLimit {
			w.mustExec("update users set delivered_today=delivered_today+1 where chat_id=?", *chatID)
			_ = w.sendText(*chatID, true, parseRaw, fmt.Sprintf("We cannot deliver more than %d messages a day", w.cfg.DeliveredLimit))
		}
		return rateLimited
	}

	var lines []string
	loc := time.FixedZone("", sms.Offset)
	tm := time.Unix(sms.Timestamp, 0).In(loc)
	lines = append(lines, tm.Format("2006-01-02 15:04:05"))
	var sender = html.EscapeString(sms.Sender)
	var sim = html.EscapeString(sms.SIM)
	if sim == "" {
		sim = html.EscapeString(sms.Carrier)
	}
	if sim != "" {
		sender = strings.Join([]string{sender, sim}, " ")
	}
	if sender != "" {
		lines = append(lines, sender)
	}

	for i, l := range lines {
		lines[i] = "<i>" + l + "</i>"
	}

	lines = append(lines, html.EscapeString(sms.Text))
	text := strings.Join(lines, "\n")

	if err := w.sendText(*chatID, true, parseHTML, text); err != nil {
		switch err {
		case errBlockedByUser:
			return blocked
		default:
			return networkError
		}
	}
	w.mustExec("update users set delivered=delivered+1, delivered_today=delivered_today+1 where chat_id=?", chatID)
	return delivered
}

func (w *worker) decrypt(str string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	result, err := w.decryptor.Decrypt(data, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (w *worker) periodic() {
	if m := midnight(); m > w.storedMidnight() {
		w.storeMidnight(m)
		w.mustExec("update users set delivered_today=0, received_today=0")
	}
}

func main() {
	w := newWorker()
	w.logConfig()
	w.setWebhook()
	w.createDatabase()

	incoming := w.bot.ListenForWebhook("/" + w.cfg.BotToken)
	w.handleEndpoints()

	go func() {
		checkErr(http.ListenAndServe(w.cfg.ListenAddress, nil))
	}()

	_ = w.sendText(w.cfg.AdminID, false, parseRaw, "Bot started")

	signals := make(chan os.Signal, 16)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	var periodicTimer = time.NewTicker(time.Minute * 10)
	for {
		select {
		case <-periodicTimer.C:
			w.periodic()
		case s := <-w.deliverChan:
			s.result <- w.deliver(s.sms)
		case m := <-incoming:
			chatString := ""
			if m.Message != nil && m.Message.Chat != nil {
				chatString = fmt.Sprintf("chat: %d", m.Message.Chat.ID)
			}
			textString := ""
			if m.Message != nil {
				textString = fmt.Sprintf("text: %s", m.Message.Text)
			}
			linf(strings.Join([]string{"got TG update", chatString, textString}, ", "))
			w.processTGUpdate(m)
		case s := <-signals:
			linf("got signal %v", s)
			w.removeWebhook()
			return
		}
	}
}
