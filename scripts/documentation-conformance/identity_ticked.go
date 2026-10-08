// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (v *verification) identityTicked(discovered inventory) error {
	if os.Getenv("HATMAX_DOC_PGDATA") != filepath.Join(v.fixture, "postgres-data") || !strings.HasPrefix(os.Getenv("DB_HOST"), "/tmp/hatmax-doc-postgres.") {
		return fmt.Errorf("Ticked conformance requires the invocation-owned PostgreSQL fixture")
	}

	project := filepath.Join(v.fixture, "ticked-project")

	err := v.exec(v.root, "git", "clone", "--quiet", "--no-hardlinks", "--no-checkout", v.root, project)
	if err != nil {
		return err
	}

	err = v.exec(project, "git", "checkout", "--quiet", "--detach", v.receipt.Head)
	if err != nil {
		return err
	}

	err = os.RemoveAll(filepath.Join(project, "examples/ticked"))
	if err != nil {
		return err
	}

	// The local immutable clone supplies the module and Make context. Overlay
	// inventoried example bytes so development runs also use the candidate tree.
	for file, contents := range discovered.contents {
		if strings.HasPrefix(file, "examples/ticked/") {
			err = writeFile(filepath.Join(project, file), contents)
			if err != nil {
				return err
			}
		}
	}

	directory := filepath.Join(project, "examples/ticked")
	socket, port := os.Getenv("DB_HOST"), os.Getenv("DB_PORT")

	err = v.execBound(directory, time.Minute, "fixture-only\nfixture-only\n", "createuser", "-h", socket, "-p", port, "-U", "postgres", "--pwprompt", "documentation_reader")
	if err != nil {
		return err
	}

	err = v.exec(directory, "createdb", "-h", socket, "-p", port, "-U", "postgres", "--owner=documentation_reader", "tickedhm")
	if err != nil {
		return err
	}

	err = v.exec(directory, "make", "db-init", "DB_HOST="+socket, "DB_PORT="+port, "DB_USER=postgres", "DB_PASS=fixture-only", "DB_NAME=documentation_make")
	if err != nil {
		return err
	}

	err = v.exec(directory, "psql", "-h", socket, "-p", port, "-U", "postgres", "-d", "documentation_make", "-v", "ON_ERROR_STOP=1", "-c", "SELECT 1 FROM pg_namespace WHERE nspname='public'")
	if err != nil {
		return err
	}

	for _, target := range []string{"sqlc", "test", "build"} {
		err = v.exec(directory, "make", target)
		if err != nil {
			return err
		}
	}

	// Admission material is private, generated once and retained across restart.
	key := make([]byte, 32)

	_, err = rand.Read(key)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(key)
	clear(key)

	var cookie string
	for attempt := 1; attempt <= 3; attempt++ {
		cookie, err = v.tickedAttempt(directory, discovered.contents["examples/ticked/config.yaml"], encoded, cookie, attempt)
		if err != nil {
			return err
		}
	}

	err = v.exec(directory, "make", "clean")
	if err != nil {
		return err
	}

	_, err = os.Stat(filepath.Join(directory, "ticked"))
	if !os.IsNotExist(err) {
		return fmt.Errorf("published clean target retained the owned binary")
	}

	v.receipt.Observations = append(v.receipt.Observations,
		"Ticked exact entrypoint/Makefile in an owned immutable local copy: native role/database creation, db-init result, SQLC generation, Make test/build/clean and foreground Make process",
		"owned Unix socket, disposable role/database and fresh localhost HTTP coordinates replace deployment examples; shipped YAML origin updated consistently with its explicit prerequisite",
		"registration issues no session; ordinary password sign-in issues Secure/HttpOnly/Lax cookie; persisted session works after restart with stable private admission material; signout retires it; coordinated shutdown closes listeners",
	)

	return nil
}

func (v *verification) tickedAttempt(directory, configuration, key, cookie string, attempt int) (resultCookie string, result error) {
	address, err := freeAddress()
	if err != nil {
		return "", err
	}

	_, number, ok := strings.Cut(address, ":")
	if !ok {
		return "", fmt.Errorf("missing owned HTTP port")
	}

	configuration = strings.ReplaceAll(configuration, "http://localhost:8080", "http://localhost:"+number)

	err = writeFile(filepath.Join(directory, "config.yaml"), configuration)
	if err != nil {
		return "", err
	}

	// Save immutable per-attempt configurations for receipt reconciliation.
	configurationFile := filepath.Join(v.fixture, fmt.Sprintf("ticked-config-%d.yaml", attempt))

	err = writeFile(configurationFile, configuration)
	if err != nil {
		return "", err
	}

	environment := []string{
		"TICKED_CREDENTIAL_NAMESPACE=documentation-conformance-owned", "TICKED_CREDENTIAL_KEY=" + key,
		"TICKED_DATABASE_HOST=" + os.Getenv("DB_HOST"), "TICKED_DATABASE_PORT=" + os.Getenv("DB_PORT"),
		"TICKED_DATABASE_USER=documentation_reader", "TICKED_DATABASE_PASSWORD=fixture-only",
		"TICKED_DATABASE_DATABASE=tickedhm", "TICKED_DATABASE_SCHEMA=public", "TICKED_SERVER_PORT=:" + number,
	}

	arguments := []string{filepath.Join(directory, "ticked")}

	name := fmt.Sprintf("ticked-direct-%d", attempt)
	if attempt == 3 {
		name = "ticked-make-3"
		arguments = []string{"make", "run-fg", "DB_HOST=" + os.Getenv("DB_HOST"), "DB_PORT=" + os.Getenv("DB_PORT"), "DB_USER=documentation_reader", "DB_PASS=fixture-only", "DB_NAME=tickedhm", "SERVER_PORT=:" + number}
	}

	process, err := v.startCommand(arguments, directory, environment, name)
	if err != nil {
		return "", err
	}

	v.receipt.Processes[process.index].ConfigurationFile = configurationFile

	endpoint := "http://localhost:" + number

	defer func() {
		stopErr := process.stop(true)
		if result == nil {
			result = stopErr
		}

		log, readErr := os.ReadFile(filepath.Join(v.fixture, name+"-process.log"))
		if readErr == nil {
			v.receipt.Processes[process.index].LogDigest = digest(string(log))
		}

		if readErr == nil && (strings.Contains(string(log), key) || cookie != "" && strings.Contains(string(log), cookie)) {
			result = fmt.Errorf("private fixture material appeared in Ticked diagnostics")
		}

		if result == nil && (readErr != nil || !strings.Contains(string(log), "Shutting down gracefully") || strings.Contains(string(log), "error stopping component") || strings.Contains(string(log), "server shutdown failed")) {
			result = fmt.Errorf("Ticked coordinator completion missing")
		}

		_, _, _, requestErr := request(localClient(), "GET", endpoint+"/ping", "", nil)
		if result == nil && requestErr == nil {
			result = fmt.Errorf("owned Ticked listener remains available after stop")
		}
	}()

	client := localClient()
	client.Timeout = 12 * time.Second

	err = awaitHealth(client, endpoint)
	if err != nil {
		return "", err
	}

	if attempt == 1 {
		cookie, err = tickedSignup(client, endpoint)
		if err != nil {
			return "", err
		}
	}

	status, _, _, err := request(client, "GET", endpoint+"/list-items", "", map[string]string{"Cookie": cookie})
	if err != nil || status != http.StatusOK {
		return "", fmt.Errorf("Ticked signed-in/restarted session failed")
	}

	if attempt == 3 {
		status, headers, _, err := request(client, "POST", endpoint+"/signout", "", map[string]string{"Cookie": cookie, "Origin": endpoint})
		if err != nil || status != http.StatusOK || headers.Get("HX-Redirect") != "/signin" || !strings.Contains(headers.Get("Set-Cookie"), "Max-Age=0") {
			return "", fmt.Errorf("Ticked signout failed cookie retirement")
		}

		status, _, _, err = request(client, "GET", endpoint+"/list-items", "", map[string]string{"Cookie": cookie})
		if err != nil || status != http.StatusSeeOther {
			return "", fmt.Errorf("Ticked retired session retained access")
		}
	}

	return cookie, nil
}

func tickedSignup(client *http.Client, endpoint string) (string, error) {
	status, _, _, err := request(client, "GET", endpoint+"/list-items", "", nil)
	if err != nil || status != http.StatusSeeOther {
		return "", fmt.Errorf("Ticked anonymous protection failed")
	}

	form := url.Values{"email": {"reader@example.com"}, "password": {"a distinct safe password"}, "confirm_password": {"a distinct safe password"}}.Encode()
	headers := map[string]string{"Origin": endpoint, "Content-Type": "application/x-www-form-urlencoded"}

	status, response, _, err := request(client, "POST", endpoint+"/signup", form, headers)
	if err != nil || status != http.StatusSeeOther || response.Get("Set-Cookie") != "" {
		return "", fmt.Errorf("Ticked registration failed/no-session assertion")
	}

	form = url.Values{"email": {"reader@example.com"}, "password": {"a distinct safe password"}}.Encode()

	status, response, _, err = request(client, "POST", endpoint+"/signin", form, headers)
	if err != nil || status != http.StatusOK || response.Get("HX-Redirect") != "/list-items" {
		return "", fmt.Errorf("Ticked ordinary password sign-in failed")
	}

	cookies := (&http.Response{Header: response}).Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		return "", fmt.Errorf("Ticked session cookie metadata failed")
	}

	return cookies[0].Name + "=" + cookies[0].Value, nil
}
