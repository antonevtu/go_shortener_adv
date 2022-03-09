package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"net/http"
)

const userIDCookieName = "user_id"

var secretKey = []byte("the super-puper secret key")

func getUserID(r *http.Request) (userID uuid.UUID, err error) {
	// проверяем куки на наличие достоверного идентификатора пользователя
	var valid bool
	userID, valid = extractUserID(r)
	if valid {
		return userID, nil
	}

	// Куки не содержит валидного идентификатора пользователя - создаем новый
	userID, err = uuid.NewUUID()
	if err != nil {
		return userID, err
	}
	return userID, err
}

func extractUserID(r *http.Request) (userID uuid.UUID, valid bool) {
	cuca, errNoCookie := r.Cookie(userIDCookieName)
	if (cuca != nil) && (errNoCookie == nil) {
		data, err := hex.DecodeString(cuca.Value)
		if (err == nil) && (len(data) > len(userID)) {
			userID_ := data[:len(userID)]
			sign := data[len(userID):]
			h := hmac.New(sha256.New, secretKey)
			h.Write(userID_)
			dst := h.Sum(nil)
			if hmac.Equal(dst, sign) {
				copy(userID[:], data[:len(userID)])
				return userID, true
			}
		}
	}
	return userID, false
}

func setCookie(w http.ResponseWriter, userID uuid.UUID) {
	token := encodeToken(secretKey, userID)
	cuca := http.Cookie{
		Name:  userIDCookieName,
		Value: token,
	}
	http.SetCookie(w, &cuca)
}

func encodeToken(key []byte, id16 uuid.UUID) string {
	id := id16[:]
	h := hmac.New(sha256.New, key)
	h.Write(id)
	dst := h.Sum(nil)

	id = append(id, dst...)
	token := hex.EncodeToString(id)
	return token
}
