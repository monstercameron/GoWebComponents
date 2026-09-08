//go:build js && wasm

package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
)

const authCookieSyncPath = "/api/public/auth/session/sync"

// parseSyncAuthCookie mirrors one bearer token into the same-origin auth cookie used by HTTP deep-link guards.
func parseSyncAuthCookie(parseToken string) error {
	parseToken = strings.TrimSpace(parseToken)
	if parseToken == "" {
		return parseClearAuthCookie()
	}
	parseResult := <-fetch.Fetch(authCookieSyncPath, fetch.Options{
		Method: "POST",
		Headers: map[string]interface{}{
			"Authorization": "Bearer " + parseToken,
		},
	})
	if parseResult.Err != nil {
		return parseResult.Err
	}
	if parseResult.Status != 0 && parseResult.Status != 204 {
		return fmt.Errorf("sync auth cookie: unexpected status %d", parseResult.Status)
	}
	return nil
}

// parseClearAuthCookie clears one same-origin auth cookie used by HTTP deep-link guards.
func parseClearAuthCookie() error {
	parseResult := <-fetch.Fetch(authCookieSyncPath, fetch.Options{Method: "DELETE"})
	if parseResult.Err != nil {
		return parseResult.Err
	}
	if parseResult.Status != 0 && parseResult.Status != 204 {
		return errors.New("clear auth cookie: unexpected response")
	}
	return nil
}
