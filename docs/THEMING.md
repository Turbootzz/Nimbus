# Theming

Everything here is per user and lives in **Settings → Theme**. Changes apply at once and are saved with your account, so they follow you to other browsers.

## Themes

A theme is a full set of colours: page, cards, borders, the three text levels, the accent, and the status colours (success, warning, error, info). Each one uses the official palette of the project it is named after.

| Theme | Mode | Palette |
|---|---|---|
| Default | light or dark | Nimbus |
| Catppuccin Mocha | dark | [catppuccin/palette](https://github.com/catppuccin/palette) |
| Catppuccin Latte | light | [catppuccin/palette](https://github.com/catppuccin/palette) |
| Nord | dark | [nordtheme.com](https://www.nordtheme.com/docs/colors-and-palettes) |
| Dracula | dark | [draculatheme.com](https://draculatheme.com/contribute#color-palette) |
| Gruvbox Dark | dark | [morhetz/gruvbox](https://github.com/morhetz/gruvbox) |
| Gruvbox Light | light | [morhetz/gruvbox](https://github.com/morhetz/gruvbox) |
| Tokyo Night | dark | [folke/tokyonight.nvim](https://github.com/folke/tokyonight.nvim) |

How a theme works with the other settings:

- **Mode.** A theme is light or dark. Picking one sets the mode to match. Picking a mode the theme doesn't have (or Automatic) goes back to Default.
- **Accent colour.** A theme brings its own accent, so picking one drops a custom accent. You can then pick another accent on top of the theme.
- **Default** is Nimbus as it was before themes: it follows light, dark or automatic mode.
- **Wallpaper and glass cards** work with every theme: cards take the theme's card colour at the opacity you set.

## Font

Default, System (the font of your device), Inter or JetBrains Mono. The font files ship with Nimbus under the SIL Open Font License; your browser never contacts a font service. A font is only downloaded once you pick it.

**Monospace KPI labels** shows the small labels above the numbers on tiles, widgets and the status strip in JetBrains Mono.

## Icon source

Where **Auto-fetch icon** on a service looks for an icon by the service's name:

| Source | Format |
|---|---|
| [Dashboard Icons](https://github.com/homarr-labs/dashboard-icons) | SVG (default), PNG or WebP |
| [selfh.st icons](https://selfh.st/icons/) | SVG |

It applies to icons you fetch from then on. The server downloads the icon once and keeps it, so pages never load icons from another site. If the server has `DASHBOARD_ICONS_BASE_URL` set (a mirror, or empty to turn the lookup off), that wins over this setting.

## Custom CSS

Your own CSS on top of the theme, up to 20 KB. For example:

```css
.glass-card {
  border-radius: 1rem;
}
```

The colours are CSS variables on `:root` (`--color-background`, `--color-card`, `--color-card-border`, `--color-card-hover`, `--color-text-primary`, `--color-text-secondary`, `--color-text-muted`, `--color-primary`, `--color-primary-hover`, `--color-success`, `--color-warning`, `--color-error`, `--color-info`), so you can change one without restyling every element.

### Safe mode

If your CSS hides the page, add `?safe=1` to the address of any page while logged in, for example `/settings/theme?safe=1`. That load, and the pages you click through to, leave your CSS out so you can fix it. A normal reload turns it back on.

### What is refused, and why

Your CSS only applies to you: it is stored with your account and only added on the pages behind the login (not the login page, the offline page, or for other users). It is never put in the browser's local storage, which the login page reads; the installed app keeps it with your other preferences for offline use, in a cache that is tied to your account and emptied when you log out or someone else logs in. It still must be safe to share, because a theme file contains it. So the server checks it on every save, including an import, and refuses the whole save when it finds:

- `@import` in any form;
- `url()` holding anything other than an inline `data:image/...`. That covers `background`, `cursor`, `mask`, `@font-face src` and every other place a URL can go;
- `image-set()`, `image()` and `src()`, which can also name a URL;
- `expression()`, `javascript:` and `-moz-binding` (script hooks of long-gone browsers);
- `</style`.

The check runs on the CSS the way a browser reads it: escapes such as `\75rl(` are decoded and case is ignored. It is deliberately stricter than a browser, so a `url(` inside a string or a comment is refused too, and so is a reference like `url(#gradient)`.

With that, CSS can't load anything from another site, which also means it can't send anything out: the well-known trick of reading a field with attribute selectors (`input[value^="a"]`) needs a request per match to leak what it found.

CSS can't run scripts either. Nimbus adds it to the page as the text of a `<style>` element, never as HTML, so it can't close the element or add a tag, and current browsers have no way to run script from CSS. Nimbus does not send a Content-Security-Policy header for the app at the moment; this does not depend on one.

## Export and import

**Export theme** downloads `nimbus-theme.json` with the theme, mode, accent, font, monospace KPI labels, icon source, the glass settings (wallpaper blur and dim, card opacity and blur) and your custom CSS. The wallpaper image is not in it.

**Import theme** reads such a file and first shows what it would set, including the full custom CSS. Read the CSS of a file you got from someone else before you apply it. Nothing changes until you choose **Apply theme**; the server then checks every value the same way as when you change it by hand, and applies all of it or nothing.

A [dashboard export](CONFIGURATION.md#dashboard-export-and-import) carries the same theme settings, custom CSS included, next to the dashboard itself. Importing one in replace mode shows the CSS first as well, and the server checks it the same way.

The file is versioned (`"nimbus_theme": 1`). Fields it doesn't have keep their current value, and anything else in the file is ignored.
