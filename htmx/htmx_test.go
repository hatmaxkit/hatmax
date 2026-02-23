package htmx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTrigger(t *testing.T) {
	tests := []struct {
		name     string
		trigger  Trigger
		expected string
	}{
		{"click", OnClick(), "click"},
		{"submit", OnSubmit(), "submit"},
		{"load", OnLoad(), "load"},
		{"revealed", OnRevealed(), "revealed"},
		{"change", OnChange(), "change"},
		{"input", OnInput(), "input"},
		{"keyup", OnKeyup(), "keyup"},
		{"custom event", OnEvent("myEvent"), "myEvent"},
		{"every 1s", Every(time.Second), "every 1s"},
		{"every 500ms", Every(500 * time.Millisecond), "every 500ms"},
		{"click once", OnClick().Once(), "click once"},
		{"click changed", OnClick().Changed(), "click changed"},
		{"click delay", OnClick().Delay(500 * time.Millisecond), "click delay:500ms"},
		{"click throttle", OnClick().Throttle(time.Second), "click throttle:1s"},
		{"click from", OnClick().From("#other"), "click from:#other"},
		{"click queue", OnClick().Queue("last"), "click queue:last"},
		{"combined modifiers", OnClick().Once().Delay(200 * time.Millisecond), "click once delay:200ms"},
		{"intersect with root", OnIntersect("#container"), "intersect root:#container"},
		{"intersect without root", OnIntersect(""), "intersect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.trigger.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTriggers(t *testing.T) {
	combined := Triggers(OnClick(), OnKeyup().Delay(500*time.Millisecond))

	expected := "click, keyup delay:500ms"
	if combined != expected {
		t.Errorf("got %q, want %q", combined, expected)
	}
}

func TestAction(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		method string
		url    string
		vals   string
	}{
		{"get", Get("/items"), "get", "/items", ""},
		{"post", Post("/items"), "post", "/items", ""},
		{"put", Put("/items/1"), "put", "/items/1", ""},
		{"patch", Patch("/items/1"), "patch", "/items/1", ""},
		{"delete", Delete("/items/1"), "delete", "/items/1", ""},
		{
			"get with param",
			Get("/items").WithParam("page", "2"),
			"get", "/items?page=2", "",
		},
		{
			"post with vals",
			Post("/items").WithVal("id", "123"),
			"post", "/items", `{"id":"123"}`,
		},
		{
			"post with multiple vals",
			Post("/items").WithVals(map[string]any{"id": "123", "name": "test"}),
			"post", "/items", "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.action.Method(); got != tt.method {
				t.Errorf("Method() = %q, want %q", got, tt.method)
			}

			if got := tt.action.URL(); got != tt.url {
				t.Errorf("URL() = %q, want %q", got, tt.url)
			}

			if tt.vals != "" {
				if got := tt.action.Vals(); got != tt.vals {
					t.Errorf("Vals() = %q, want %q", got, tt.vals)
				}
			}
		})
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   Target
		expected string
	}{
		{"self", TargetSelf(), "this"},
		{"id", TargetID("my-element"), "#my-element"},
		{"selector", TargetSelector(".items"), ".items"},
		{"closest", TargetClosest("tr"), "closest tr"},
		{"next", TargetNext(".sibling"), "next .sibling"},
		{"previous", TargetPrevious(".sibling"), "previous .sibling"},
		{"find", TargetFind(".child"), "find .child"},
		{"body", TargetBody(), "body"},
		{"item target", ItemTarget("item", 123), "#item-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSwap(t *testing.T) {
	tests := []struct {
		name     string
		swap     Swap
		expected string
	}{
		{"inner", SwapInner(), "innerHTML"},
		{"outer", SwapOuter(), "outerHTML"},
		{"beforeend", SwapBeforeEnd(), "beforeend"},
		{"afterend", SwapAfterEnd(), "afterend"},
		{"beforebegin", SwapBeforeBegin(), "beforebegin"},
		{"afterbegin", SwapAfterBegin(), "afterbegin"},
		{"delete", SwapDelete(), "delete"},
		{"none", SwapNone(), "none"},
		{"with after", SwapOuter().After(time.Second), "outerHTML swap:1s"},
		{"with settle", SwapOuter().Settle(500 * time.Millisecond), "outerHTML settle:500ms"},
		{"scroll top", SwapOuter().ScrollTop(), "outerHTML scroll:top"},
		{"scroll bottom", SwapOuter().ScrollBottom(), "outerHTML scroll:bottom"},
		{"show element", SwapOuter().Show("#target"), "outerHTML show:#target"},
		{"focus scroll", SwapOuter().FocusScroll(false), "outerHTML focus-scroll:false"},
		{"transition", SwapOuter().Transition(true), "outerHTML transition:true"},
		{"combined", SwapOuter().After(100 * time.Millisecond).ScrollTop(), "outerHTML swap:100ms scroll:top"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.swap.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestOOB(t *testing.T) {
	tests := []struct {
		name     string
		oob      OOB
		expected string
	}{
		{"swap", OOBSwap(), "true"},
		{"inner", OOBInner(), "innerHTML"},
		{"outer", OOBOuter(), "outerHTML"},
		{"beforeend", OOBBeforeEnd(), "beforeend"},
		{"afterbegin", OOBAfterBegin(), "afterbegin"},
		{"delete", OOBDelete(), "delete"},
		{"with target", OOBSwap().Target("#counter"), "true:#counter"},
		{"inner with target", OOBInner().Target("#content"), "innerHTML:#content"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.oob.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAttrs(t *testing.T) {
	t.Run("basic attrs", func(t *testing.T) {
		attrs := HX().Get("/items").TargetID("list").SwapOuter()
		m := attrs.Map()

		if m["hx-get"] != "/items" {
			t.Errorf("hx-get = %q, want %q", m["hx-get"], "/items")
		}

		if m["hx-target"] != "#list" {
			t.Errorf("hx-target = %q, want %q", m["hx-target"], "#list")
		}

		if m["hx-swap"] != "outerHTML" {
			t.Errorf("hx-swap = %q, want %q", m["hx-swap"], "outerHTML")
		}
	})

	t.Run("with trigger", func(t *testing.T) {
		attrs := HX().Post("/toggle").Trigger(OnClick().Once())
		m := attrs.Map()

		if m["hx-post"] != "/toggle" {
			t.Errorf("hx-post = %q, want %q", m["hx-post"], "/toggle")
		}

		if m["hx-trigger"] != "click once" {
			t.Errorf("hx-trigger = %q, want %q", m["hx-trigger"], "click once")
		}
	})

	t.Run("with confirm", func(t *testing.T) {
		attrs := HX().Delete("/item/1").Confirm("Are you sure?")
		m := attrs.Map()

		if m["hx-confirm"] != "Are you sure?" {
			t.Errorf("hx-confirm = %q, want %q", m["hx-confirm"], "Are you sure?")
		}
	})

	t.Run("with vals", func(t *testing.T) {
		attrs := HX().Post("/update").WithVal("id", "123")
		m := attrs.Map()

		if m["hx-vals"] != `{"id":"123"}` {
			t.Errorf("hx-vals = %q, want %q", m["hx-vals"], `{"id":"123"}`)
		}
	})

	t.Run("boost", func(t *testing.T) {
		attrs := HX().Boost()
		m := attrs.Map()

		if m["hx-boost"] != "true" {
			t.Errorf("hx-boost = %q, want %q", m["hx-boost"], "true")
		}
	})

	t.Run("no boost", func(t *testing.T) {
		attrs := HX().NoBoost()
		m := attrs.Map()

		if m["hx-boost"] != "false" {
			t.Errorf("hx-boost = %q, want %q", m["hx-boost"], "false")
		}
	})

	t.Run("string output", func(t *testing.T) {
		attrs := HX().Get("/items").TargetID("list")
		s := attrs.String()

		// Check that it contains expected attributes
		if s == "" {
			t.Error("String() returned empty")
		}
	})

	t.Run("HTML output", func(t *testing.T) {
		attrs := HX().Get("/items")
		html := attrs.HTML()

		if html == "" {
			t.Error("HTML() returned empty")
		}
	})
}

func TestHeaders(t *testing.T) {
	t.Run("redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		Redirect(w, "/dashboard")

		if got := w.Header().Get(HeaderRedirect); got != "/dashboard" {
			t.Errorf("HX-Redirect = %q, want %q", got, "/dashboard")
		}
	})

	t.Run("refresh", func(t *testing.T) {
		w := httptest.NewRecorder()
		Refresh(w)

		if got := w.Header().Get(HeaderRefresh); got != "true" {
			t.Errorf("HX-Refresh = %q, want %q", got, "true")
		}
	})

	t.Run("retarget", func(t *testing.T) {
		w := httptest.NewRecorder()
		Retarget(w, "#new-target")

		if got := w.Header().Get(HeaderRetarget); got != "#new-target" {
			t.Errorf("HX-Retarget = %q, want %q", got, "#new-target")
		}
	})

	t.Run("reswap", func(t *testing.T) {
		w := httptest.NewRecorder()
		Reswap(w, SwapOuter())

		if got := w.Header().Get(HeaderReswap); got != "outerHTML" {
			t.Errorf("HX-Reswap = %q, want %q", got, "outerHTML")
		}
	})

	t.Run("push url", func(t *testing.T) {
		w := httptest.NewRecorder()
		PushURL(w, "/new-page")

		if got := w.Header().Get(HeaderPushURL); got != "/new-page" {
			t.Errorf("HX-Push-Url = %q, want %q", got, "/new-page")
		}
	})

	t.Run("replace url", func(t *testing.T) {
		w := httptest.NewRecorder()
		ReplaceURL(w, "/replaced")

		if got := w.Header().Get(HeaderReplaceURL); got != "/replaced" {
			t.Errorf("HX-Replace-Url = %q, want %q", got, "/replaced")
		}
	})

	t.Run("trigger", func(t *testing.T) {
		w := httptest.NewRecorder()
		TriggerEvent(w, "itemAdded")

		if got := w.Header().Get(HeaderTrigger); got != "itemAdded" {
			t.Errorf("HX-Trigger = %q, want %q", got, "itemAdded")
		}
	})

	t.Run("trigger with data", func(t *testing.T) {
		w := httptest.NewRecorder()
		TriggerEventWithData(w, "notify", map[string]string{"message": "success"})

		got := w.Header().Get(HeaderTrigger)
		if got == "" {
			t.Error("HX-Trigger is empty")
		}
	})

	t.Run("stop polling", func(t *testing.T) {
		w := httptest.NewRecorder()
		StopPolling(w)

		if w.Code != 286 {
			t.Errorf("status code = %d, want %d", w.Code, 286)
		}
	})
}

func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{"htmx request", "true", true},
		{"not htmx request", "", false},
		{"invalid value", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				r.Header.Set(HeaderRequest, tt.header)
			}

			if got := IsHTMXRequest(r); got != tt.expected {
				t.Errorf("IsHTMXRequest() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFuncMap(t *testing.T) {
	fm := FuncMap()

	// Check that key functions exist
	funcs := []string{
		"hx", "hxGet", "hxPost", "hxPut", "hxPatch", "hxDelete",
		"hxOnClick", "hxOnSubmit", "hxOnLoad",
		"hxTargetID", "hxTargetSelf",
		"hxSwapInner", "hxSwapOuter", "hxSwapDelete",
		"ms", "s",
	}

	for _, name := range funcs {
		if _, ok := fm[name]; !ok {
			t.Errorf("FuncMap missing %q", name)
		}
	}
}

func TestOOBWrapper(t *testing.T) {
	t.Run("basic wrapper", func(t *testing.T) {
		w := OOBWrap("counter")
		open := string(w.Open())
		close := string(w.Close())

		if open != `<div id="counter" hx-swap-oob="true">` {
			t.Errorf("Open() = %q", open)
		}

		if close != "</div>" {
			t.Errorf("Close() = %q", close)
		}
	})

	t.Run("wrapper with custom tag", func(t *testing.T) {
		w := OOBWrap("nav").Tag("nav")
		open := string(w.Open())

		if open != `<nav id="nav" hx-swap-oob="true">` {
			t.Errorf("Open() = %q", open)
		}
	})

	t.Run("wrapper with class", func(t *testing.T) {
		w := OOBWrap("box").Class("hidden", "md:block")
		open := string(w.Open())

		if open != `<div id="box" hx-swap-oob="true" class="hidden md:block">` {
			t.Errorf("Open() = %q", open)
		}
	})

	t.Run("wrapper with swap strategy", func(t *testing.T) {
		w := OOBWrap("list").Swap(OOBBeforeEnd())
		open := string(w.Open())

		if open != `<div id="list" hx-swap-oob="beforeend">` {
			t.Errorf("Open() = %q", open)
		}
	})
}
