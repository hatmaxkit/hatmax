<!--
SPDX-FileCopyrightText: 2026 Adrian PK
SPDX-License-Identifier: Apache-2.0

This file is part of Hatmax. See LICENSE for license terms.
-->

# Internationalization

`i18n` loads YAML translations and looks up a flattened key. The
implementation note is [i18n/readme.md](../../../i18n/readme.md).

`DefaultLocale` is `en`. `New` uses that default and stores no locales.

`LoadFromFS` reads `basePath` from an embedded filesystem. Subdirectories are
skipped. A name ending in `.yml` or `.yaml` is a locale. The locale code is
the filename without that suffix, so `pt-BR.yaml` is locale `pt-BR`. Any
other file is skipped. A directory read, file read, or YAML parse error is
returned. Loaded locale codes are appended to `AvailableLocales` in directory
order.

Nested YAML maps are flattened with `.` between keys. A non-map value is
stored at the flattened key.

`Get` returns the locale's value when the key exists. A string is returned as
itself. Any other stored value is formatted with `%v`. A missing key is looked
up in the default locale when the requested locale is different. A missing key
in both places returns the key unchanged.

`SetDefaultLocale` replaces the fallback locale. `DefaultLocaleValue` returns
it. `HasLocale` reports whether that locale was loaded. `AvailableLocales`
returns a copy of the loaded codes.

`TranslateFunc(locale)` returns a function that calls `Get` with that locale.
