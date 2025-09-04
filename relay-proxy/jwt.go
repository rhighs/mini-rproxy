package main

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

func DecodeJWTHS256(token, secret string) (map[string]any, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected alg %s", t.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	m, ok := t.Claims.(jwt.MapClaims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token")
	}

	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out, nil
}
