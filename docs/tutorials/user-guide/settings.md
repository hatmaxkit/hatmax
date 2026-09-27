# Change Settings at Runtime

Read one greeting that is not in `config.yaml`. Change it while the process
is running. The next request returns the new greeting. Restarting the process
returns the schema default again.

The contract is in
[Configuration](../../reference/configuration/index.md).

`config.Config` is loaded once at startup. A setting is a runtime value
checked against a `settings.Schema`. This package does not ship a Postgres
store. The store below keeps values in memory for the life of the process.

## Register the greeting

```go
type memorySettings struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memorySettings) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok := m.values[key]
	if !ok {
		return "", errors.New("missing")
	}

	return value, nil
}

func (m *memorySettings) Set(ctx context.Context, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.values == nil {
		m.values = map[string]string{}
	}

	m.values[key] = value

	return nil
}

func (m *memorySettings) All(context.Context) ([]settings.Value, error) {
	return nil, nil
}

func (m *memorySettings) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, key)

	return nil
}
```

```go
registry := settings.NewRegistry()
registry.Register(settings.Schema{
	Key:     "guide.greeting",
	Type:    settings.String,
	Default: "Hello",
})

greetings := settings.NewService(registry, &memorySettings{})
```

`GetString` returns the schema default when the store has no value or returns
an error. `Set` validates the value because the key is registered, then
stores it. Nothing in `config.yaml` is rewritten.

## Serve it

```go
func (p *pages) greeting(w http.ResponseWriter, r *http.Request) {
	value, err := p.greetings.GetString(r.Context(), "guide.greeting")
	if err != nil {
		http.Error(w, "cannot read greeting", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, value)
}

func (p *pages) setGreeting(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	err = p.greetings.Set(r.Context(), "guide.greeting", r.FormValue("greeting"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.greeting(w, r)
}
```

Register `GET /greeting` and `POST /greeting`. Keep
`middleware.RequireSameOrigin` on the router.

## Check the result

`config.yaml` still has no `guide.greeting` key. Run the process.

```sh
curl -sS http://localhost:8080/greeting
curl -sS -H 'Origin: http://localhost:8080' \
  -d 'greeting=Hi' http://localhost:8080/greeting
curl -sS http://localhost:8080/greeting
```

The first response is `Hello`. The next two are `Hi`.

Stop the process and start it again.

```sh
curl -sS http://localhost:8080/greeting
```

The response is `Hello` again. `config.yaml` is unchanged.
