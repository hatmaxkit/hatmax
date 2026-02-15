package htmx

import (
	"html/template"
	"time"
)

// FuncMap returns a template.FuncMap with HTMX helper functions.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// Attribute builder
		"hx": HX,

		// Actions
		"hxGet":    Get,
		"hxPost":   Post,
		"hxPut":    Put,
		"hxPatch":  Patch,
		"hxDelete": Delete,

		// Triggers
		"hxOnClick":    OnClick,
		"hxOnSubmit":   OnSubmit,
		"hxOnLoad":     OnLoad,
		"hxOnRevealed": OnRevealed,
		"hxOnChange":   OnChange,
		"hxOnInput":    OnInput,
		"hxOnKeyup":    OnKeyup,
		"hxOnEvent":    OnEvent,
		"hxEvery":      Every,
		"hxTriggers":   Triggers,

		// Targets
		"hxTargetID":       TargetID,
		"hxTargetSelf":     TargetSelf,
		"hxTargetSelector": TargetSelector,
		"hxTargetClosest":  TargetClosest,
		"hxTargetNext":     TargetNext,
		"hxTargetPrevious": TargetPrevious,
		"hxTargetFind":     TargetFind,
		"hxItemTarget":     ItemTarget,

		// Swaps
		"hxSwapInner":       SwapInner,
		"hxSwapOuter":       SwapOuter,
		"hxSwapBeforeEnd":   SwapBeforeEnd,
		"hxSwapAfterEnd":    SwapAfterEnd,
		"hxSwapBeforeBegin": SwapBeforeBegin,
		"hxSwapAfterBegin":  SwapAfterBegin,
		"hxSwapDelete":      SwapDelete,
		"hxSwapNone":        SwapNone,

		// Duration helper for templates
		"ms": func(n int) time.Duration { return time.Duration(n) * time.Millisecond },
		"s":  func(n int) time.Duration { return time.Duration(n) * time.Second },

		// Quick attribute generators
		"hxAttrs":     hxAttrs,
		"hxPostAttrs": hxPostAttrs,
		"hxGetAttrs":  hxGetAttrs,
	}
}

// hxAttrs creates attributes for a given action with target and swap.
func hxAttrs(action Action, target Target, swap Swap) template.HTMLAttr {
	return HX().Action(action).Target(target).Swap(swap).HTML()
}

// hxPostAttrs creates POST attributes with target and swap.
func hxPostAttrs(url string, targetID string) template.HTMLAttr {
	return HX().Post(url).TargetID(targetID).SwapOuter().HTML()
}

// hxGetAttrs creates GET attributes with target and swap.
func hxGetAttrs(url string, targetID string) template.HTMLAttr {
	return HX().Get(url).TargetID(targetID).SwapOuter().HTML()
}
