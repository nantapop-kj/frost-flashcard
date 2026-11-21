# frost-flashcard

A menu bar flashcard app for macOS and Windows that uses spaced repetition (SM-2) to help you remember vocabulary.

This project is designed to demonstrate:
- A Go + Wails v3 desktop app that lives in the menu bar or system tray, with no Dock icon
- A plain HTML, CSS and JavaScript frontend with no npm and no build step
- The SM-2 spaced repetition scheduler with multiple decks
- Excel/CSV import and CSV export
- A Liquid Glass UI with light and dark themes
- Packaging as a universal macOS `.dmg` and a Windows installer

---

## ✨ Features

- Open the app from the menu bar icon. The popup hides when it loses focus, and starts a new shuffled review each time it opens.
- Review cards on the **Review** tab:
  - Click the card or press `Space`/`Enter` to show the answer.
  - Grade with **Again**, **Hard**, **Good** or **Easy**, or press `1`–`4`.
  - A progress bar shows how many cards are left in the session.
- Add words on the **Add** tab, with a word, a meaning and an optional reading. Empty required fields get a red border.
- Import words from `.xlsx`, `.xlsm` or `.csv` files.
- Manage words on the **Words** tab:
  - Search
  - Filter by status (New, Learning, Known)
  - Edit inline
  - Delete with two clicks
- Use more than one deck. You can create, rename, switch and delete decks.
- Settings:
  - Theme: System, Light or Dark.
  - Reading: show it with the answer or on the front of the card.
  - Which cards to review: All, Due today, New, Learning or Known.
  - Export the deck to CSV.
  - Move the app to Applications.
  - Uninstall.
- Tray menu:
  - Review and Add Word.
  - Launch at Login.
  - Open Data Folder.
  - About and Quit.

---

## 🧠 Business Rules

1. Each grade has a quality score: Again `1`, Hard `3`, Good `4`, Easy `5`.
2. The next interval, in days:
   - **Again:** repetitions reset to `0` and the interval is `1`.
   - **Other grades:** the interval is `1` on the first review, `6` on the second, then `round(interval × ease)`. Repetitions go up by 1.
3. Ease starts at `2.5`. After each review it becomes `ease + 0.1 − m × (0.08 + m × 0.02)`, where `m = 5 − quality`. It never goes below `1.3`.
4. A card is due when its due date is today or earlier. The due date is `today + interval`.
5. A card's status:
   - **New:** never reviewed (interval `0`).
   - **Known:** answered correctly at least 2 times in a row.
   - **Learning:** everything else.
6. Each review session takes the cards that match the selected filter, in random order.
7. A card needs both a word and a meaning. The reading is optional.
8. When importing:
   - Column A is the word, B the meaning and C the reading. Only the first sheet of an Excel file is read.
   - The first row is skipped when it looks like a header, such as `word`, `คำศัพท์` or `単語`.
   - Blank rows are skipped.
   - Rows without a word or a meaning are counted as empty.
   - Words already in the deck are counted as duplicates. Matching ignores case and extra spaces.
   - A CSV file must be UTF-8.
9. Exported CSV files are UTF-8 with a BOM, so Excel shows Thai and Japanese correctly. The columns are `word`, `meaning` and `reading`.
10. A deck needs a name. The last remaining deck cannot be deleted.
11. Every change is saved straight away. If the save fails, the change is undone and an error is shown.
12. Uninstall asks for confirmation first. It then:
    - Stops saving.
    - Turns off Launch at Login.
    - Deletes the data folder.
    - On macOS, deletes the app. If it cannot, it shows the app in Finder so you can move it to the Trash.
    - On Windows, runs the app's uninstaller.
13. Move to Applications (macOS) copies the app to `/Applications`, replacing an older copy of this app, and opens the new copy.

---

## ⚙️ Configuration

There is no config file. Settings are stored in two places:

| What | Where |
|---|---|
| Decks, cards and review progress | `flashcards.json` in the data folder |
| Theme, reading mode and review filter | The window's `localStorage` |

The data folder:

| OS | Path |
|---|---|
| macOS | `~/Library/Application Support/frostflashcard.app/` |
| Windows | `%AppData%\frostflashcard.app\` |

- The file is written to a temporary file and then renamed into place, so a crash cannot leave it half written.
- A data folder from the old app ID, `dev.frostflashcard.app`, is moved to the new folder on first launch.
- A `flashcards.json` from the old single-deck format becomes a library with one deck.
- Imported files are only read. Their words are copied into `flashcards.json`, and the files themselves are not kept.
- Removing the app by hand, such as dragging it to the Trash on macOS, keeps the data folder. Installing again brings your decks back.

---

## 🚀 How to Run

### Requirements
- Go 1.27 or newer
- macOS 11 or newer with Xcode Command Line Tools (`xcode-select --install`)
- To build a Windows installer: Windows with [Inno Setup 6](https://jrsoftware.org/isinfo.php)

### 1. Run in development
```bash
make run
```

The app shows an icon in the menu bar. Click it to open the popup.

### 2. Test
```bash
make test     # run all tests
make check    # gofmt, go vet, and tests with the race detector
```

### 3. Build the macOS app
```bash
make app                      # build/bin/Frost Flashcard.app
make dmg                      # build/bin/Frost-Flashcard-<version>.dmg
make dmg VERSION=1.0.0        # set the version shown in About
```

- The app is a universal binary for Apple Silicon and Intel.
- The app is signed ad hoc, not with a Developer ID. The first time it opens, macOS may block it. Right-click the app and choose **Open**, or allow it in **System Settings → Privacy & Security**.

### 4. Build for Windows
```bash
make windows                  # build/bin/frost-flashcard.exe
```

Then build the installer on Windows:
```bash
iscc /DAppVersion=1.0.0 build\windows\installer.iss
```

- The installer installs for the current user only, so it does not need admin rights.
- The uninstaller asks whether to delete your data folder too. The default answer is **No**.
- The `AppId` GUID in `installer.iss` must match `innoUninstallKey` in `internal/services/uninstaller_windows.go`. Otherwise Uninstall in the app deletes the data but leaves the program installed.

### Development notes
- The frontend is in `frontend/` and is embedded into the binary. Restart `make run` after changing it.
- Regenerate the icons after changing `assets/icons/appicon.png`:
  ```bash
  make icons
  ```
- The `private_mac_apis` build tag is needed for the see-through glass window on macOS. The make targets and build scripts add it for you.
- `make clean` removes `build/bin`.

---

## 🔌 Services API

The frontend calls Go methods through the Wails runtime, for example `Call.ByName("github.com/nantapop-kj/frost-flashcard/internal/services.Service.AddCard", front, back, reading)`.

Errors use the shape `{ "code": "...", "detail": "..." }`. The frontend shows a message for each `code`.

**`Service`**: cards and decks

| Method | Description |
|---|---|
| `AddCard(front, back, reading)` | Add a card to the active deck |
| `UpdateCard(id, front, back, reading)` | Edit a card |
| `DeleteCard(id)` | Delete a card |
| `ListCards()` | Cards in the active deck, with status and due date |
| `Stats()` | Card counts per status, plus the number due today |
| `StartSession(filter)` | Shuffled cards for a review (`all`, `due`, `new`, `learning`, `known`) |
| `GradeCard(id, grade)` | Grade a card (`again`, `hard`, `good`, `easy`) and return the next interval |
| `ListDecks()` | All decks and the active one |
| `CreateDeck(name)` | Create a deck and switch to it |
| `RenameDeck(id, name)` | Rename a deck |
| `DeleteDeck(id)` | Delete a deck |
| `SetActiveDeck(id)` | Switch decks |
| `ImportFile(title)` | Pick an Excel/CSV file and import it into the active deck |
| `ExportCSV(title)` | Pick where to save, then export the active deck |

**`System`**: the app itself

| Method | Description |
|---|---|
| `AppInfo()` | Version, OS, and whether the app can be moved to Applications |
| `InstallApp()` | Copy the app to `/Applications` and open that copy (macOS) |
| `Uninstall()` | Delete the data and the app, turn off Launch at Login, and quit |
| `Quit()` | Quit the app |

**Error codes**

| Code | When |
|---|---|
| `empty_fields` | The word or the meaning is empty |
| `empty_name` | The deck name is empty |
| `not_found` | The card or deck no longer exists |
| `last_deck` | Trying to delete the only deck |
| `save_failed` | Saving to disk failed, so the change was undone |
| `unsupported_file` | The import file is not `.xlsx`, `.xlsm` or `.csv` |
| `not_utf8` | The CSV is not UTF-8 |
| `no_sheet` | The Excel file has no sheets |
| `read_failed` | The import file could not be read |
| `write_failed` | The export file could not be written |
| `bad_input` | The request was not valid, such as an unknown filter or grade |

---

## 📊 Example Output

`flashcards.json`:

```json
{
  "decks": [
    {
      "id": 1,
      "name": "Japanese",
      "cards": [
        {
          "id": 1,
          "front": "猫",
          "back": "แมว",
          "reading": "ねこ",
          "interval": 6,
          "ease": 2.5,
          "repetitions": 2,
          "due": "2026-11-26"
        }
      ],
      "next_id": 2
    }
  ],
  "active": 1,
  "next_deck_id": 2
}
```

Message after an import:

```
Imported 120 words · 3 already in deck · 1 row missing word or meaning
```

---

## 📚 Import Template

`assets/vocab-template.xlsx` has the columns `word`, `meaning` and `reading`, with a few sample rows. Fill it in and import it from the **Add** tab.

To make a CSV in Excel, choose **File → Save As → CSV UTF-8**.

---

## 📁 Project Structure

```
frost-flashcard/
├── cmd/
│   └── main.go                  # Load data, create services, start the app
├── frontend/                    # Embedded into the binary
│   ├── index.html
│   ├── style.css                # Glass theme, light and dark
│   ├── app.js                   # Views, review session, decks, words list
│   └── lib/
│       ├── api.js               # Calls to Go services and events
│       └── helpers.js           # Error messages, toast, validation, confirm dialog
├── internal/
│   ├── model/                   # Card, Deck, Library, Filter, Date
│   ├── scheduler/               # SM-2
│   ├── storage/                 # JSON load/save and old format migration
│   ├── importer/                # Excel/CSV reading and import rules
│   ├── exporter/                # CSV export
│   ├── services/                # Methods called by the frontend (Service, System)
│   ├── dto/                     # Data sent to the frontend and error codes
│   └── helper/                  # Tray, popup window, file dialogs, data folder
├── assets/
│   ├── icons/                   # App and tray icons
│   └── vocab-template.xlsx      # Import template
├── build/
│   ├── darwin/                  # Info.plist, icon.icns
│   ├── windows/                 # icon.ico, manifest, Inno Setup script
│   └── bin/                     # Build output (not committed)
├── scripts/                     # build-mac.sh, build-windows.sh, make-icons.sh
└── Makefile
```

---

## ⚠️ Known Limitations

- The app is not signed with an Apple Developer ID or notarized, so macOS warns about it the first time it opens.
- The Windows build is not code signed, so SmartScreen may warn about it. Choose **More info → Run anyway**.
- The Windows build has only been cross-compiled from macOS. The installer has not been tested on Windows.
- Launch at Login is tied to the app ID. After the app ID changed from `dev.frostflashcard.app`, turn it on again from the tray menu.
- The glass effect follows the system appearance, not the theme picked in Settings, so it can look different when they do not match.
- Only one copy of the app should run at a time. Two copies would overwrite each other's saves.
- The interface is in English only.

---

## 🤖 AI Assistance

- **UX/UI design:**
  - The popup layout and the review, add, words and settings views.
  - The Liquid Glass look, with light and dark themes.
  - The glass confirm dialog.
- **Formulas:** the SM-2 rules in [🧠 Business Rules](#-business-rules).
- **Packaging:**
  - The macOS build and `.dmg` scripts.
  - The Windows manifest and Inno Setup script.
  - The icon generation script.
- **Tests:** unit tests for the models, scheduler, storage, importer, exporter, services and helpers.
- **Code and security review:** reviewing the code for bugs and security issues, and fixing what was found.

All code written or reviewed with AI help was checked again before it was used.

---

## 📄 License
MIT License. See [LICENSE](LICENSE).
