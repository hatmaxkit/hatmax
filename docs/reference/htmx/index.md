# HTMX

`htmx` builds `hx-*` attributes and reads or writes HTMX headers. The
implementation note is [htmx/readme.md](../../../htmx/readme.md).

## Requests

| Function | Header | True when |
| --- | --- | --- |
| `IsHTMXRequest` | `HX-Request` | the value is `true` |
| `IsBoosted` | `HX-Boosted` | the value is `true` |
| `IsHistoryRestore` | `HX-History-Restore-Request` | the value is `true` |

`GetCurrentURL`, `GetPromptResponse`, `GetRequestTarget`,
`GetRequestTrigger`, and `GetTriggerName` return `HX-Current-URL`,
`HX-Prompt`, `HX-Target`, `HX-Trigger`, and `HX-Trigger-Name`. An absent
header is `""`.

## Responses

| Function | Header or status |
| --- | --- |
| `Redirect` | `HX-Redirect` |
| `Refresh` | `HX-Refresh: true` |
| `Retarget` | `HX-Retarget` |
| `Reswap` | `HX-Reswap` from `Swap.String` |
| `ReswapStr` | `HX-Reswap` from the supplied string |
| `Reselect` | `HX-Reselect` |
| `PushURL` | `HX-Push-Url` |
| `PreventPushURL` | `HX-Push-Url: false` |
| `ReplaceURL` | `HX-Replace-Url` |
| `PreventReplaceURL` | `HX-Replace-Url: false` |
| `StopPolling` | status `286` and no body |
| `RespondEmpty` | status `200` and no body |
| `RespondDelete` | status `200` and no body |

`TriggerEvent` sets `HX-Trigger` to the event name.
`TriggerEventAfterSettle` and `TriggerEventAfterSwap` set
`HX-Trigger-After-Settle` and `HX-Trigger-After-Swap` to the event name.
`TriggerEventWithData` sets `HX-Trigger` to JSON `{"<event>": data}`. A
marshal error sets the header to the event name. `TriggerEvents` sets
`HX-Trigger` to the JSON object. A marshal error leaves the header unset.

`Location` sets `HX-Location`. A `LocationConfig` that has only `Path` is
written as that path. Any other populated field is written as JSON. A marshal
error falls back to `Path`. `LocationSimple` calls `Location` with `Path` and
`Target`, or writes the path alone when `target` is empty.

`LocationConfig` fields are `Path`, `Source`, `Event`, `Handler`, `Target`,
`Swap`, `Values`, `Headers`, and `Select`.

## Attributes

`HX` returns an `Attrs` builder. `String` joins `Map` as `key="value"` pairs.
`HTML` returns that string as `template.HTMLAttr`. `Set` stores an extra
attribute. Later calls replace the same key.

`Map` emits an attribute only when its value is set:

| Builder | Attribute |
| --- | --- |
| `Get`, `Post`, `Put`, `Patch`, `Delete`, `Action` | `hx-get`, `hx-post`, `hx-put`, `hx-patch`, or `hx-delete` |
| action values and headers | `hx-vals`, `hx-headers` |
| `Trigger` | `hx-trigger` |
| `Target`, `TargetID`, `TargetThis` | `hx-target` |
| `Swap`, `SwapOuter`, `SwapInner`, `SwapDelete` | `hx-swap` |
| `Confirm` | `hx-confirm` |
| `Disable`, `DisableSelf` | `hx-disabled-elt` |
| `Indicator` | `hx-indicator` |
| `Sync` | `hx-sync` |
| `Validate` | `hx-validate="true"` |
| `Boost`, `NoBoost` | `hx-boost` `true` or `false` |
| `PushURL`, `ReplaceURL` | `hx-push-url`, `hx-replace-url` |
| `Select`, `SelectOOB` | `hx-select`, `hx-select-oob` |
| `Ext` | `hx-ext` |
| `Encoding` | `hx-encoding` |
| `Preserve` | `hx-preserve="true"` |
| `History` | `hx-history` |

`Get`, `Post`, `Put`, `Patch`, and `Delete` build an `Action`. `WithParam`
and `WithParams` add query parameters. `WithHeader`, `WithHeaders`,
`WithVal`, and `WithVals` add `hx-headers` and `hx-vals`. `URL` includes the
query. `Vals` is JSON.

Trigger constructors are `OnClick`, `OnSubmit`, `OnLoad`, `OnRevealed`,
`OnIntersect`, `Every`, `OnEvent`, `OnChange`, `OnInput`, `OnKeyup`,
`OnFocus`, and `OnBlur`. Modifiers are `Once`, `Changed`, `Delay`,
`Throttle`, `From`, `Queue`, `Target`, and `Consume`. `Triggers` joins
several triggers with `, `.

Target constructors are `TargetSelf`, `TargetID`, `TargetSelector`,
`TargetClosest`, `TargetNext`, `TargetPrevious`, `TargetFind`, `TargetBody`,
and `ItemTarget`. `ItemTarget(prefix, id)` selects `#<prefix>-<id>`.

Swap strategies are `innerHTML`, `outerHTML`, `beforeend`, `afterend`,
`beforebegin`, `afterbegin`, `delete`, and `none`, from `SwapInner`,
`SwapOuter`, `SwapBeforeEnd`, `SwapAfterEnd`, `SwapBeforeBegin`,
`SwapAfterBegin`, `SwapDelete`, and `SwapNone`. Modifiers are `After`,
`Settle`, `ScrollTop`, `ScrollBottom`, `Scroll`, `Show`, `ShowTop`,
`ShowBottom`, `FocusScroll`, `Transition`, and `IgnoreTitle`.

`OOB` constructors are `OOBSwap`, `OOBInner`, `OOBOuter`, `OOBBeforeEnd`,
`OOBAfterBegin`, `OOBBeforeBegin`, `OOBAfterEnd`, `OOBDelete`, and `OOBNone`.
`Target` adds the selector. `String` renders the `hx-swap-oob` value. `Attr`
returns it as `template.HTMLAttr`.

`OOBWrap(id)` builds a wrapper. `Tag` defaults to `div` when empty. `Swap`
selects the out-of-band swap. `Class` adds classes. `Open` returns the start
tag with `id` and `hx-swap-oob`. `Close` returns the end tag.

## Template functions

`FuncMap` registers `hx`, `hxGet`, `hxPost`, `hxPut`, `hxPatch`, `hxDelete`,
the `hxOn*` triggers listed above except `OnIntersect`, `OnFocus`, and
`OnBlur`, `hxEvery`, `hxTriggers`, the target and swap constructors, `ms`,
`s`, `hxAttrs`, `hxPostAttrs`, and `hxGetAttrs`.

`hxAttrs(action, target, swap)` returns those three attributes. `hxPostAttrs`
and `hxGetAttrs` use the URL, the target id, and `outerHTML`.
