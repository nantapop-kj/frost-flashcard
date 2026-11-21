# frost-flashcard

A menu bar flashcard app for macOS and Windows. It uses spaced repetition (SM-2) to help you remember vocabulary.

Built with Go + Wails v3 and a plain HTML/CSS/JS frontend (no npm, no build step). It lives in the menu bar or system tray, has a Liquid Glass UI with light and dark themes, and ships as a universal macOS `.dmg` and a Windows installer.

---

## ✨ Features

- **Review:** click the card or press `Space`/`Enter` to show the answer, then grade with **Again**, **Hard**, **Good** or **Easy** (`1`–`4`). Each time the popup opens, a new shuffled session starts.
- **Add:** a word, a meaning and an optional reading, or import from `.xlsx`, `.xlsm` or `.csv`.
- **Words:** search, filter by status (New, Learning, Known), edit inline, delete.
- **Decks:** create, rename, switch and delete.
- **Settings:** theme, where the reading is shown, which cards to review, CSV export, Move to Applications, Uninstall.
- **Tray menu:** Review, Add Word, Launch at Login, Open Data Folder, About, Quit.

---

## 🧠 Business Rules

1. Grades score Again `1`, Hard `3`, Good `4`, Easy `5`.
2. **Again** resets repetitions to `0` and sets the interval to `1` day. Other grades set the interval to `1`, then `6`, then `round(interval × ease)`.
3. Ease starts at `2.5` and becomes `ease + 0.1 − m × (0.08 + m × 0.02)` with `m = 5 − quality`. The minimum is `1.3`.
4. A card is due when `today ≥ due`, where `due = review date + interval`.
5. Status: **New** = never reviewed, **Known** = at least 2 correct in a row, **Learning** = everything else.
6. A card needs a word and a meaning. The reading is optional.
7. Import reads columns A/B/C (word/meaning/reading) from the first sheet. It skips a header row (`word`, `คำศัพท์`, `単語`…) and blank rows, and counts rows that are empty or duplicates (ignoring case and extra spaces). CSV must be UTF-8.
8. Export writes UTF-8 CSV with a BOM, so Excel shows Thai and Japanese correctly.
9. The last remaining deck cannot be deleted.
10. Every change is saved straight away. If the save fails, the change is undone and an error is shown.
11. Uninstall asks first, then turns off Launch at Login, deletes the data folder and removes the app.

---

## ⚙️ Data

There is no config file. Decks and progress are in `flashcards.json` in the data folder. Theme and review filter are kept in the window's `localStorage`.

| OS | Data folder |
|---|---|
| macOS | `~/Library/Application Support/frostflashcard.app/` |
| Windows | `%AppData%\frostflashcard.app\` |

- Saves go to a temporary file first and are then renamed into place, so a crash cannot leave the file half written.
- Data from the old app ID (`dev.frostflashcard.app`) and the old single-deck format is migrated on first launch.
- Removing the app by hand keeps the data folder, so reinstalling brings your decks back.

---

## 🚀 How to Run

**Requirements:** Go 1.27+. On macOS, Xcode Command Line Tools (`xcode-select --install`).

```bash
make run        # run in development (icon appears in the menu bar)
make test       # tests
make check      # gofmt, go vet, tests with -race
```

### macOS
```bash
make app                  # build/bin/Frost Flashcard.app (universal)
make dmg VERSION=1.0.0    # build/bin/Frost-Flashcard-1.0.0.dmg
```

### Windows
```bash
make windows                              # build/bin/frost-flashcard.exe
make windows-installer VERSION=1.0.0      # build/bin/Frost-Flashcard-Setup-1.0.0.exe
```

- Runs on Windows (Git Bash) or WSL. In WSL, Windows `go.exe` is used if Linux Go is not installed.
- If Inno Setup is not installed, the script downloads a portable copy into `build/tools/` (checksum verified). Nothing is installed on the system.
- The installer is per user and does not need admin rights. The uninstaller asks whether to delete your data too (default **No**).
- The `AppId` in `build/windows/installer.iss` must match `innoUninstallKey` in `internal/services/uninstaller_windows.go`.

### Releases
Each push to `main` runs `.github/workflows/release.yml`. It takes the latest `v*` tag plus one patch version (starting at `v0.0.1`), builds the `.dmg` and the Windows installer, and publishes a GitHub release. To rebuild an existing tag, run the workflow manually with that tag.

### Development notes
- `frontend/` is embedded into the binary. Restart `make run` after changing it.
- `make icons` regenerates the icons from `assets/icons/appicon.png`.
- The `private_mac_apis` build tag enables the see-through glass window on macOS. The make targets add it for you.

---

## 🔏 Code Signing

The builds are **not signed with a paid certificate** yet, so both systems warn the first time the app opens.

### macOS (Apple)
The app is signed **ad hoc** (`codesign --sign -` in `scripts/build-mac.sh`) and is not notarized.

- **For users:** right-click the app → **Open**, or allow it in **System Settings → Privacy & Security → Open Anyway**.
- **To sign it properly:**
  1. Join the Apple Developer Program (US$99/year) and create a **Developer ID Application** certificate.
  2. Sign with the hardened runtime:
     `codesign --force --deep --options runtime --timestamp --sign "Developer ID Application: <Name> (<TEAMID>)" "Frost Flashcard.app"`
  3. Notarize and staple the `.dmg`:
     `xcrun notarytool submit Frost-Flashcard-1.0.0.dmg --apple-id <id> --team-id <TEAMID> --password <app-specific-password> --wait`
     `xcrun stapler staple Frost-Flashcard-1.0.0.dmg`
  4. In CI, store the `.p12` certificate and the notarization password as GitHub secrets and import them into a temporary keychain before building.

### Windows
The `.exe` and the installer are not signed, so SmartScreen shows **"Windows protected your PC"**.

- **For users:** choose **More info → Run anyway**.
- **To sign it properly:**
  1. Get a code signing certificate: **Azure Trusted Signing** (cheapest, monthly fee), or an OV/EV certificate from a CA such as DigiCert or Sectigo. Since 2023 the private key must live on a hardware token or a cloud HSM. EV and Trusted Signing build SmartScreen reputation faster than OV.
  2. Sign the app and the installer with a timestamp:
     `signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /a frost-flashcard.exe`
  3. To have Inno Setup sign the installer and uninstaller, add `SignTool=...` under `[Setup]` in `installer.iss`.

---

## 🔌 Services API

The frontend calls Go methods through the Wails runtime, for example `Call.ByName("github.com/nantapop-kj/frost-flashcard/internal/services.Service.AddCard", front, back, reading)`. Errors have the shape `{ "code": "...", "detail": "..." }`.

| `Service` method | Description |
|---|---|
| `AddCard` / `UpdateCard` / `DeleteCard` | Change cards in the active deck |
| `ListCards()` | Cards with status and due date |
| `Stats()` | Counts per status and due today |
| `StartSession(filter)` | Shuffled cards (`all`, `due`, `new`, `learning`, `known`) |
| `GradeCard(id, grade)` | Grade a card (`again`, `hard`, `good`, `easy`) |
| `ListDecks` / `CreateDeck` / `RenameDeck` / `DeleteDeck` / `SetActiveDeck` | Manage decks |
| `ImportFile(title)` / `ExportCSV(title)` | Import Excel/CSV, export CSV |

| `System` method | Description |
|---|---|
| `AppInfo()` | Version, OS, and whether the app can be moved to Applications |
| `InstallApp()` | Copy to `/Applications` and reopen (macOS) |
| `Uninstall()` | Delete data and app, turn off Launch at Login, quit |
| `Quit()` | Quit |

Error codes: `empty_fields`, `empty_name`, `not_found`, `last_deck`, `save_failed`, `unsupported_file`, `not_utf8`, `no_sheet`, `read_failed`, `write_failed`, `bad_input`.

---

## 📚 Import Template

`assets/vocab-template.xlsx` has the columns `word`, `meaning` and `reading` with sample rows. For CSV from Excel, use **File → Save As → CSV UTF-8**.

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

- Not signed with a paid certificate (see [🔏 Code Signing](#-code-signing)).
- Launch at Login is tied to the app ID. After the change from `dev.frostflashcard.app`, turn it on again.
- The glass effect follows the system appearance, not the theme picked in Settings.
- Run only one copy at a time, or the two copies overwrite each other's saves.
- English UI only.

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

---

## 📄 License
MIT License. See [LICENSE](LICENSE).
