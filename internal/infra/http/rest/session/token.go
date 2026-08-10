// Package session выдаёт и проверяет подписанный токен тестовой сессии.
//
// user_id не принимается от клиента в открытом виде — сервер выдаёт его сам и
// подписывает HMAC-SHA256 на секрете, которого у клиента нет: иначе клиент мог
// бы назваться кем угодно, и право на покупку ничего бы не защищало.
//
// В подпись входит и срок жизни: Max-Age куки — лишь подсказка браузеру,
// которую он может игнорировать, поэтому истечение проверяется на сервере.
package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// CookieName — имя куки с токеном.
	CookieName = "session"

	// TTL — срок жизни токена, сутки: демо-стенд должен переживать ночь между
	// сдачей и защитой без перелогина.
	TTL = 24 * time.Hour

	separator = "."
)

var (
	ErrInvalidToken = errors.New("invalid session token")
	ErrExpiredToken = errors.New("session token expired")
)

// Issue собирает токен вида "<user_id>.<exp>.<signature>".
func Issue(userID uuid.UUID, secret string, now time.Time) string {
	payload := payloadOf(userID.String(), now.Add(TTL).Unix())

	return payload + separator + sign(payload, secret)
}

// Parse проверяет подпись и срок, возвращая user_id. Подпись проверяется до
// разбора полей: содержимое неподписанного токена не заслуживает доверия.
func Parse(token, secret string, now time.Time) (uuid.UUID, error) {
	cut := strings.LastIndex(token, separator)
	if cut < 0 {
		return uuid.Nil, ErrInvalidToken
	}

	payload, signature := token[:cut], token[cut+1:]
	if !hmac.Equal([]byte(signature), []byte(sign(payload, secret))) {
		return uuid.Nil, ErrInvalidToken
	}

	rawUserID, rawExp, ok := strings.Cut(payload, separator)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	if now.After(time.Unix(exp, 0)) {
		return uuid.Nil, ErrExpiredToken
	}

	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	return userID, nil
}

func payloadOf(rawUserID string, exp int64) string {
	return rawUserID + separator + strconv.FormatInt(exp, 10)
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))

	return hex.EncodeToString(mac.Sum(nil))
}
