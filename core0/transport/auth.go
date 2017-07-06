package transport

import (
	"fmt"

	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/siddontang/ledisdb/config"
	"time"
)

const (
	maxJWTDuration int64 = 3600
)

func validExpiration(claims jwt.MapClaims) error {
	now := time.Now().Unix()
	if !claims.VerifyExpiresAt(now, true) {
		//we call the verify expires at with required=True to make sure that
		//the exp flag is set.
		return fmt.Errorf("jwt token expired")
	}

	exp := claims["exp"]

	var ts int64
	switch exp := exp.(type) {
	case float64:
		ts = int64(exp)
	case json.Number:
		ts, _ = exp.Int64()
	}

	if ts-now > maxJWTDuration {
		return fmt.Errorf("jwt token expiration exceeds max allowed expiration of %v", time.Duration(maxJWTDuration)*time.Second)
	}

	return nil
}

func AuthMethod(organization string, key string) (config.AuthMethod, error) {
	scope := fmt.Sprintf("user:memberof:%s", organization)

	pub, err := jwt.ParseECPublicKeyFromPEM([]byte(key))
	if err != nil {
		return nil, err
	}

	return func(_ *config.Config, token string) bool {
		log.Debugf("checking token: %s", token)
		t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			m, ok := t.Method.(*jwt.SigningMethodECDSA)
			if !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
			}
			if t.Header["alg"] != m.Alg() {
				return nil, fmt.Errorf("Unexpected signing algorithm: %v", t.Header["alg"])
			}
			return pub, nil
		})

		if err != nil {
			log.Errorf("JWT parse error: %s", err)
			return false
		}

		if !t.Valid {
			return false
		}

		claims := t.Claims.(jwt.MapClaims)

		if err := claims.Valid(); err != nil {
			log.Errorf("itsyouonline calim validation error: %s", err)
			return false
		}

		if err := validExpiration(claims); err != nil {
			log.Errorf("itsyouonline calim validation error: %s", err)
			return false
		}

		if claims["azp"] == organization {
			return true
		}

		var scopes []interface{}
		if value, ok := claims["scope"]; ok {
			if scopes, ok = value.([]interface{}); !ok {
				return false
			}
		} else {
			return false
		}

		for _, s := range scopes {
			switch s.(type) {
			case string:
				if s == scope {
					return true
				}
			}
		}

		return false
	}, nil
}
