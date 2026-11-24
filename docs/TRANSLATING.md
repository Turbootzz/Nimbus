# Translating Nimbus

Nimbus speaks English, Dutch (`nl`), German (`de`), French (`fr`) and Spanish (`es`). The dashboard, boards, settings and sign-in pages are translated, with every component on them.

> **The Dutch, German, French and Spanish texts are machine translations.** They were written by an AI model and have not been reviewed by native speakers yet. Corrections are very welcome: see [Improving a translation](#improving-a-translation). Issues about translations carry the `help wanted` label.

## Which language you see

1. The language you picked under **Settings → Profile → Language**. It is saved with your account (and in dashboard exports), so it follows you to other browsers.
2. **Automatic (browser)**, the default: the first language in your browser's preferred languages that Nimbus has.
3. English, when Nimbus has none of your browser's languages.

Dates, times and numbers use your browser's regional format of that language: with English and a British browser the clock shows 14:05, with an American one 02:05 PM.

How it works: there is no language in the URL. Nimbus keeps your choice in a `NEXT_LOCALE` cookie, so pages are rendered in your language on the server; without the cookie it reads the browser's `Accept-Language` header. Signing out keeps the cookie, so the sign-in page stays in your language on that browser.

Not translated (yet): the public status page (`/status` is always English, whoever visits it), the services list and service pages, metrics, user management, the landing page and the offline page. Error messages from the server are English. Names and labels that come from your apps (integration numbers, Docker states) are shown as the app reports them.

## Where the texts are

```
frontend/messages/en.json   English: the source, every key starts here
frontend/messages/nl.json   one file per language, the same keys
...
```

Keys are grouped per page and component, e.g. `dashboard.header.edit` or `settings.theme.wallpaper.title`. Messages use [ICU syntax](https://next-intl.dev/docs/usage/messages): `{name}` is a value Nimbus fills in, `{count, plural, one {# service} other {# services}}` picks the right form for the number, and `<link>...</link>` marks text that becomes a link. Keep those parts as they are and translate the words around them. A literal `{` or `}` must be quoted (`'{'`).

## Improving a translation

Edit the value in `frontend/messages/<language>.json`, keep the key, then check it:

```bash
cd frontend
npm run check-messages
```

The check fails when a language misses a key of `en.json`, has a key `en.json` doesn't have, or a message has other `{values}` or `<tags>` than the English one. It runs in CI as part of `npm run ci-check`.

## Adding a language

1. Copy `frontend/messages/en.json` to `frontend/messages/<code>.json` (a two-letter [ISO 639-1](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) code) and translate the values.
2. Add the code to `languages` in `frontend/i18n/locale.ts`, and its own name (e.g. `"it": "Italiano"`) to `language.names` in every messages file.
3. Add the code to `Languages` in `backend/internal/models/preferences.go`, so the preferences API and dashboard imports accept it.
4. Run `npm run check-messages` and `npm test` in `frontend/`, and `make test` in `backend/`.

## Adding or changing English text

New text in a translated page goes into `en.json` under the component's key, and into every other language file (the check fails otherwise). Use `useTranslations('<namespace>')` in client components and `getTranslations` in server components. Keys are type-checked: `t('a.b')` does not compile when `en.json` lacks it. Format dates and numbers with the active locale (`useLocale()`), never with a fixed one.

If you can't translate the new text yourself, put the English text in the other files too and mention it in the pull request; a translator picks it up later.

## Translation platforms (later)

Nimbus has no Weblate or Crowdin project yet. Both work with these files as they are (nested JSON with ICU messages), so one can be set up without code changes:

- **[Weblate](https://weblate.org)** (free hosting for open-source projects): create a component with file format *JSON nested structure file*, file mask `frontend/messages/*.json`, monolingual base file `frontend/messages/en.json`, and connect it to the GitHub repository. Weblate commits translations back through pull requests. Enable the ICU MessageFormat checks.
- **[Crowdin](https://crowdin.com)** (free for open source): add a `crowdin.yml` at the repository root with `files: [{ source: /frontend/messages/en.json, translation: /frontend/messages/%two_letters_code%.json }]` and install the Crowdin GitHub integration; it opens pull requests with new translations.

Either way `npm run check-messages` in CI guards what comes back.
