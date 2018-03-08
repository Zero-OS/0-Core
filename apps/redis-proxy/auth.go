package main

import (
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

func AuthMethod(organization string, key string) (func(string) bool, error) {
	scope := fmt.Sprintf("user:memberof:%s", organization)

	pub, err := jwt.ParseECPublicKeyFromPEM([]byte(key))
	if err != nil {
		return nil, err
	}

	return func(token string) bool {
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
