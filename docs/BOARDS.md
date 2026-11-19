# Boards, the shared board and kiosk mode

A board is one dashboard: its own groups, services and widgets. Every user starts with one,
the **default board** at `/dashboard`. As long as you have only that one, nothing changes: no
switcher, the same pages and the same API answers as before boards existed.

## Your boards

Open the user menu and choose **New board**. The boards dialog creates, renames, reorders and
deletes boards. A board gets an address from its name: "Media" lives at `/board/media`.

Once you see more than one board, a switcher appears: next to the logo in the canvas top bar,
and under the navigation in the classic sidebar. Its **Manage boards** entry opens the same
dialog.

- **Moving tiles.** Edit a service or widget and pick another board under **Board**. It lands
  at the end of that board, outside any group. A service or widget added on a board stays on
  that board; discovered containers, importers from other dashboards and API clients that name
  no board add to the default board.
- **Deleting a board.** Choose whether its services, widgets and groups move to another of your
  boards or go with it. The default board can't be deleted.
- **Where `/dashboard` takes you.** Nimbus remembers the board you chose last in this browser
  and opens it from `/dashboard` (and from the installed app). Pick the default board in the
  switcher to go back to it.
- **Search.** `/` or Ctrl+K searches the services on all boards you see and names the board of
  each result.

Users can have up to 20 boards. Each board has its own default group, which holds the board's
services without a group.

## The shared board

An admin can share a board with every user: tick **Shared with every user, read-only** in the
boards dialog. Everyone then finds it in their switcher.

- Only its owner, the admin who shared it, edits it. Other users, other admins included, see it
  read-only: no edit mode, no add buttons, no moving or resizing.
- Viewers see what the tiles show: names, links, status, uptime on hover, widget data and live
  numbers. They don't see integration credentials, a user name and password in a link,
  addresses of the apps behind the live numbers, widget secrets, or what a failed check or
  fetch ran into (that shows as a general error).
- A new user whose own board is still empty opens the shared board on their first visit.
- Unsharing, or deleting, the board takes it away from viewers at once: an open dashboard moves
  to their own board within a minute.
- The default board can't be shared, so nothing reaches every user by accident through
  discovery, an import or an API client. Create a board to share.
- A board is linked by its address name. When a viewer has a board of their own with the same
  name, the shared one is linked by its ID instead (`/board/<id>`).

## Kiosk mode

Add `?kiosk=1` to a board address for a wall screen or a tablet:
`https://nimbus.example.com/board/family?kiosk=1` (or `/dashboard?kiosk=1`). The page shows the
tiles and nothing around them: no top bar, sidebar, banners, buttons, search or edit mode.
Health checks and widget data keep arriving live, and the whole board is fetched again every
five minutes. A kiosk address is not remembered as your last board and `/dashboard?kiosk=1`
does not move on to another board. Only when its board is no longer shared with the kiosk user
does it fall back to `/dashboard?kiosk=1`.

Kiosk mode still needs a signed-in user. A browser without a session goes to the login page,
also with `?kiosk=1`.

### Wall screen recipe

1. As an admin, create the board to show and share it (or use any board of the kiosk user).
2. Create a dedicated user for the screen, for example `wall@home.lan`, as a normal user, not an
   admin. It sees only its own boards and the shared ones, read-only, and can't change anyone's
   dashboard.
3. On the screen, sign in as that user once with **Remember me** on, then open the kiosk
   address. After a new login Nimbus opens `/dashboard`, not the kiosk address, so keep the
   address in the browser's start page or kiosk command.
4. Start the browser full screen on that address, for example:

   ```bash
   chromium --kiosk --noerrdialogs --disable-session-crashed-bubble \
     "https://nimbus.example.com/board/family?kiosk=1"
   ```

   Firefox has `firefox --kiosk <address>`. On a tablet, add the page to the home screen.

## API, export and MCP

- `GET /api/v1/boards` lists the boards you see (your own first), with `owned` and `can_edit`.
  It takes a token with `read:services`. Creating, changing and deleting boards needs a browser
  session.
- `GET /services`, `/groups`, `/widgets` and `/dashboard/data` take `?board=<id>`; without it
  they are about your default board. Creating a service, widget or group takes `board_id`;
  changing the `board_id` of a service or widget moves it. See
  [CONFIGURATION.md](CONFIGURATION.md#boards-in-the-file) for the export file.
- MCP tools take an optional `board` (name, address name or ID); see [MCP.md](MCP.md#boards).
