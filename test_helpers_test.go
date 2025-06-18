package bon

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Extended Verify helper - supports HTTP methods
func VerifyExtended(h http.Handler, ws []*Want) error {
	sv := httptest.NewServer(h)
	defer sv.Close()

	for _, v := range ws {
		method := "GET"
		path := v.Path
		
		// Separate method if included in path
		if strings.Contains(path, ":") {
			parts := strings.SplitN(path, ":", 2)
			if len(parts) == 2 {
				method = parts[0]
				path = parts[1]
			}
		}
		
		// Create HTTP request
		req, err := http.NewRequest(method, sv.URL+path, nil)
		if err != nil {
			return err
		}
		
		// Execute request
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode != v.StatusCode {
			return fmt.Errorf("Method=%s, Path=%s, StatusCode=%d, WantStatusCode=%d", method, path, res.StatusCode, v.StatusCode)
		}

		if len(v.Body) > 0 {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(res.Body); err != nil {
				return err
			}

			if buf.String() != v.Body {
				return fmt.Errorf("Method=%s, Path=%s, Body=%s, WantBody=%s", method, path, buf.String(), v.Body)
			}
		}
	}

	return nil
}