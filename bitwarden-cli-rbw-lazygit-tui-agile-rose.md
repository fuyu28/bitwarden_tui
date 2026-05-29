# Bitwarden TUI (rbw + Go + Bubble Tea)

## Context

rbw（非公式 Bitwarden CLI）を使い、lazygit 風の 2 ペインレイアウトで Bitwarden の内容を閲覧・コピーできる TUI を作る。
表示フェーズのみ（編集は後回し）。**パスワードフィールドは絶対に表示・クリップボードコピーしない。**

---

## UI レイアウト

```
┌─────────────────────────────────────────────────┐
│ bitwarden-tui                      [🔓 unlocked] │
├───────────────────┬─────────────────────────────┤
│ All Login Card    │ SAISON VISA          [Card]  │
│ Note              │                             │
│                   │ Cardholder  TARO YAMADA      │
│ > SAISON VISA Card│ Number      4111 **** 1111   │
│   Canva     Login │ Expiry      12/2028          │
│   R2 Token  Note  │ Code/PIN    ****             │
│   ...             │                             │
│                   │ ─ URIs ─                    │
├───────────────────┴─────────────────────────────┤
│ /search  j/k nav  y copy  tab focus  q quit     │
└─────────────────────────────────────────────────┘
```

フォーカスはリストとデテール間で Tab で切り替え。

---

## ディレクトリ構成

```
bitwarden_tui/
├── main.go
├── go.mod
├── internal/
│   ├── rbw/
│   │   └── client.go      # rbw CLI ラッパー（JSON パース含む）
│   ├── model/
│   │   └── item.go        # データ型定義
│   └── ui/
│       ├── app.go         # ルート bubbletea Model（レイアウト・フォーカス管理）
│       ├── list.go        # 左ペイン: bubbles/list
│       ├── detail.go      # 右ペイン: bubbles/viewport
│       ├── keys.go        # キーバインド定義
│       └── style.go       # lipgloss スタイル定数
└── Makefile
```

---

## 依存パッケージ

```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles     # list / viewport / textinput
github.com/charmbracelet/lipgloss
github.com/atotto/clipboard
```

---

## 実装ステップ

### Step 1 — プロジェクト初期化

```bash
go mod init github.com/fuyu28/bitwarden_tui
go get github.com/charmbracelet/bubbletea \
       github.com/charmbracelet/bubbles \
       github.com/charmbracelet/lipgloss \
       github.com/atotto/clipboard
```

### Step 2 — rbw クライアント (`internal/rbw/client.go`)

| 関数 | rbw コマンド |
|------|------------|
| `IsUnlocked() bool` | `rbw unlocked` (exit code 0 = locked, 1 = unlocked ※要確認) |
| `List() ([]RawItem, error)` | `rbw list --raw` |
| `GetDetail(id string) (*Detail, error)` | `rbw get <id> --raw` |
| `CopyToClipboard(value string)` | `atotto/clipboard` |

**重要**: `GetDetail` で返す構造体には `Password` フィールドを持たせない。
JSON パース時に password キーを完全に無視（`json:"-"` または専用 struct で除外）。

`rbw list --raw` のレスポンス例（実測）:
```json
{ "id": "...", "name": "...", "user": "...", "folder": null, "uris": [...], "type": "Login" }
```

`rbw get <id> --raw` の想定フィールド（type 別）:
- **Login**: username, totp, uris, notes  ← password は除外
- **Card**: cardholder_name, number, brand, exp_month, exp_year, code
- **Note**: notes
- **Identity**: title, first_name, last_name, email, phone, ...
- **SSH Key**: private_key, public_key, fingerprint

### Step 3 — データモデル (`internal/model/item.go`)

```go
type ItemType string
const (TypeLogin, TypeCard, TypeNote, TypeIdentity, TypeSSH ItemType = ...)

type ListItem struct { ID, Name string; User string; Type ItemType }
type LoginDetail    struct { Username string; TOTP string; URIs []string; Notes string }
type CardDetail     struct { CardholderName, Number, Brand, ExpMonth, ExpYear, Code string }
type NoteDetail     struct { Notes string }
type IdentityDetail struct { FullName, Email, Phone, Address string; ... }
type SSHKeyDetail   struct { PublicKey, Fingerprint string } // 秘密鍵は表示しない
type Item struct { ListItem; Detail any }
```

### Step 4 — UI コンポーネント

**`ui/app.go`** — ルートモデル
- `tea.Model` を実装
- 状態: `items []model.ListItem`, `selected *model.Item`, `focus` (left/right), `filter ItemType`
- `Init()`: `rbw list --raw` を非同期 `tea.Cmd` で実行
- アイテム選択時に `rbw get --raw` を非同期で呼び出し

**`ui/list.go`** — 左ペイン
- `bubbles/list` ラッパー
- タイプ別フィルタ: `[All][Login][Card][Note]` タブ切り替え (`1`/`2`/`3`/`4` キー)
- `/` でインクリメンタルサーチ（bubbles/list の FilteringEnabled）

**`ui/detail.go`** — 右ペイン
- `bubbles/viewport` でスクロール対応
- Note の場合: 行番号付き表示、行選択カーソル、`y` で選択行コピー、`Y` で全文コピー
- その他: フィールド一覧表示、`y` でフォーカス中フィールドの値をクリップボードへ

**`ui/keys.go`**:
```
j/↓        リスト下移動
k/↑        リスト上移動  
Tab        左右ペインフォーカス切り替え
/          リストサーチ
1-4        タイプフィルタ切り替え
y          フィールド/行をクリップボードへコピー（マスク中も実値をコピー）
Y          Note全文コピー
space      フォーカス中フィールドのマスク/表示トグル
r          rbw sync 実行
q/Ctrl+C   終了
```

### Step 5 — アンロック画面 (`ui/unlock.go`)

- `bubbles/textinput`（EchoModePassword でマスク入力）
- Enter で `echo "<pw>" | rbw unlock` を実行、成功でメイン画面へ切り替え
- 失敗時はエラーメッセージ再表示

### Step 6 — main.go

1. `tea.NewProgram(ui.NewApp()).Run()`
2. `NewApp()` 内で初期状態を `StateUnlock` or `StateMain` に設定（`rbw unlocked` チェック結果による）

---

## パスワード・機密フィールドの扱い

| フィールド | デフォルト表示 | キー操作 | コピー |
|-----------|--------------|---------|-------|
| Login: password | `••••••••` (マスク) | `space` で表示/非表示トグル | `y` で可 |
| Card: number | `**** **** **** 1111` (末4桁のみ) | `space` でフル表示トグル | `y` で可 |
| Card: code/PIN | `••••` | `space` でトグル | `y` で可 |
| SSH Key: private_key | 非表示 | `space` でトグル | `y` で可 |

- フィールドにフォーカスがある状態で `y` を押すと、**マスク中でも**実際の値をクリップボードへコピー
- クリップボードへのコピーは明示的なキー操作のみ（自動コピーなし）

---

## 起動フロー（マスターパスワード要求）

1. 起動時に `rbw unlocked` チェック
2. **ロック中** → TUI 内でパスワード入力プロンプト表示（bubbles/textinput、入力は `*` でマスク）
3. 入力確定で `rbw unlock` を実行（パイプ経由でパスワードを渡す）
4. アンロック成功 → メイン画面へ遷移
5. 失敗 → エラーメッセージ表示 → 再入力

`rbw unlock` にパスワードを渡す方法:
```bash
echo "<password>" | rbw unlock
# または
rbw unlock  # 対話式（TUI 内でパイプ渡し）
```

---

## 実装しない機能（今回）

- アイテムの追加・編集・削除
- TOTP コードの表示（`rbw code` コマンドは使わない）

---

## 検証方法

1. `go build -o bwtui . && ./bwtui` で起動確認
2. `rbw unlocked` がロック状態のときにエラー終了するか確認
3. リスト表示 → タイプフィルタ切り替え → アイテム選択 → 詳細表示の一連フローを手動確認
4. Note アイテムで行選択コピー（`y`）が正しくクリップボードに入るか確認
5. Login アイテムの詳細にパスワードが表示されないことを確認
