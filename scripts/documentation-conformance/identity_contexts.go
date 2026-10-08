// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

const identityContext = `
var ctx=context.Background()
var cfg=config.New()
var logger=hatlog.NewNoopLogger()
var checker auth.PasswordChecker
var admissionQueries auth.CredentialAdmissionQueries
var observer auth.SecurityObserver
var queries auth.Queries
var svc, account, authService *auth.Service
var recovery *auth.RecoveryService
var settingsSvc *settings.Service
var r=httptest.NewRequest("GET","https://localhost/account",nil)
var w=httptest.NewRecorder()
var router=chi.NewRouter()
var required=auth.AccessRequirement{Proof:auth.RequirePassword,Revision:"password-v1"}
var sessionToken, sessionBearer, newPassword string
var email=" READER@EXAMPLE.COM "
var password="a distinct fixture password"
var key=make([]byte,32)
var lookupKey=func()[]byte{key:=make([]byte,32);key[0]=1;return key}()
var user=&auth.User{}
type fixtureInvoiceHTTP struct{}
func (*fixtureInvoiceHTTP) page(http.ResponseWriter,*http.Request){}
var invoiceHandler=&fixtureInvoiceHTTP{}
func accountPage(http.ResponseWriter,*http.Request){}
`

var identityOutcomes = map[string]string{
	"example:crypto/readme.md#block-1": `
if plaintext!="private note" {return fmt.Errorf("documented plaintext changed")}
_,failure:=crypto.DecryptString(value,key,[]byte("contact:other:notes"))
if failure==nil {return fmt.Errorf("foreign associated data accepted")}
`,
	"example:crypto/readme.md#block-2": `
expected, failure:=crypto.DeriveLookupHash("reader@example.com",lookupKey)
if failure!=nil||lookup!=expected {return fmt.Errorf("documented canonical lookup changed")}
`,
	"example:crypto/readme.md#block-3": `
if !ok||len(salt)!=32||len(hash)!=32||len(token)!=44 {return fmt.Errorf("documented salt/hash/token shape changed")}
if crypto.VerifyPassword("a different password",hash,salt) {return fmt.Errorf("wrong password accepted")}
`,
}

func identitySource(id, body string) (string, error) {
	context := identityContext
	declarations := ""
	void := id == "example:docs/how-to/add-authentication/README.md#block-3"

	if strings.Contains(body, "type Queries interface") {
		for _, name := range []string{"User", "CredentialState", "SessionRecord", "AccessRequirement", "Session", "SessionDigest", "SessionPage", "SessionSelection", "ValidatedSession", "SessionActivity"} {
			declarations += fmt.Sprintf("type %s = auth.%s\n", name, name)
		}

		declarations += body + "\nvar _ auth.Queries = (Queries)(nil)\nvar _ Queries = (auth.Queries)(nil)\n"
		body = ""
	}

	if id == "example:auth/readme.md#block-3" {
		end := strings.Index(body, "// Helper methods")
		if end < 0 {
			return "", fmt.Errorf("missing User declaration boundary")
		}

		declarations += body[:end]
		body = body[end:]
	}

	if id == "example:auth/readme.md#block-1" {
		context += "var log=logger\n"
	}

	if id == "example:auth/readme.md#block-2" || id == "example:docs/how-to/add-authentication/README.md#block-4" {
		context += "var r=router\n"
		context = strings.Replace(context, "var r=httptest.NewRequest(\"GET\",\"https://localhost/account\",nil)\n", "", 1)
	}

	if id == "example:examples/ticked/README.md#block-3" {
		return body, nil
	}

	function := "func fragment() (fragmentError error) {\n"
	if void {
		function = "func fragment() {\n"
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), "fragment.go", "package fixture\n"+function+body+"\nreturn\n}\n", 0)
	if err != nil {
		return "", fmt.Errorf("invalid identity fragment %s: %w", id, err)
	}

	var sinks strings.Builder

	for _, statement := range parsed.Decls[0].(*ast.FuncDecl).Body.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE {
			continue
		}

		for _, lhs := range assignment.Lhs {
			name, ok := lhs.(*ast.Ident)
			if ok && name.Name != "_" {
				fmt.Fprintf(&sinks, "_ = %s\n", name.Name)
			}
		}
	}

	source := declarations + context + function + body + sinks.String() + identityOutcomes[id] + "\nreturn\n}\n"

	imports, err := importsFor(source)
	if err != nil {
		return "", err
	}

	return "package fixture\n" + imports + source, nil
}

func compileIdentityContexts(v *verification, discovered inventory) error {
	directory := filepath.Join(v.fixture, "identity-contexts")

	err := v.module(directory)
	if err != nil {
		return err
	}

	var selected []row

	for _, r := range discovered.rows {
		if r.slice != 4 || !strings.HasPrefix(r.id, "example:") {
			continue
		}

		block, err := rowBlock(discovered, r)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(block, "```go\n") {
			continue
		}

		source, err := identitySource(r.id, blockBody(block))
		if err != nil {
			return err
		}

		fragment := filepath.Join(directory, fmt.Sprintf("fragment-%03d", len(selected)))

		err = writeFile(filepath.Join(fragment, "fragment.go"), source)
		if err != nil {
			return err
		}

		if _, executable := identityOutcomes[r.id]; executable {
			err = writeFile(filepath.Join(fragment, "fragment_test.go"), "package fixture\nimport \"testing\"\n// Execute the exact published fragment and its documented positive/negative outcomes.\nfunc TestFragment(t *testing.T){err:=fragment();if err!=nil{t.Fatal(err)}}\n")
			if err != nil {
				return err
			}
		}

		if r.id == "example:examples/ticked/README.md#block-3" {
			err = writeFile(filepath.Join(fragment, "fragment_test.go"), keyGeneratorCheck)
			if err != nil {
				return err
			}
		}

		selected = append(selected, r)
	}

	for _, arguments := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"test", "-count=1", "-timeout=2m", "./..."}} {
		err = v.exec(directory, "go", arguments...)
		if err != nil {
			return err
		}
	}

	for _, r := range selected {
		method := "compiled"
		outcome := "exact fragment compiled with current exported contracts; typed application handlers/adapters supply context and grant no workflow assurance"

		if _, executable := identityOutcomes[r.id]; executable || r.id == "example:examples/ticked/README.md#block-3" {
			method = "executed"
			outcome = "exact crypto fragment or key-generator entrypoint executed; documented shapes and positive/negative outcomes asserted without logging private output"
		}

		v.record(r, method, outcome)
	}

	return nil
}

const keyGeneratorCheck = `package main
import("encoding/base64";"io";"os";"testing")
// Capture the exact published main privately; diagnostics contain no key bytes.
func TestKey(t *testing.T){
 reader,writer,err:=os.Pipe();if err!=nil{t.Fatal(err)}
 defer reader.Close()
 original:=os.Stdout;os.Stdout=writer
 defer func(){os.Stdout=original;writer.Close()}()
 main();writer.Close();os.Stdout=original
 output,err:=io.ReadAll(io.LimitReader(reader,100));if err!=nil{t.Fatal("key output unavailable")}
 if len(output)!=45||output[44]!=10{t.Fatal("key encoding shape changed")}
 text:=string(output[:44]);key,err:=base64.StdEncoding.Strict().DecodeString(text)
 if err!=nil||len(key)!=32||base64.StdEncoding.EncodeToString(key)!=text{t.Fatal("key generation invalid")}
 clear(key);clear(output)
}
`
