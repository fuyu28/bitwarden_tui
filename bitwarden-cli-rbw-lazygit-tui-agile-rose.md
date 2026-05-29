# Bitwarden TUI (rbw + Go + Bubble Tea)

## Context

rbw（非公式 Bitwarden CLI）を使い、lazygit 風の 2 ペインレイアウトで Bitwarden の内容を閲覧・コピーできる TUI を作る。
表示フェーズのみ（編集は後回し）。機密フィールドはデフォルト伏せ表示で、キー操作で表示トグル・コピー可能。

---

## UI レイアウト

```
┌─────────────────────────────────────────────────┐
│ bitwarden-tui                      [🔓 unlocked] │
├───────────────────┬─────────────────────────────┤
│ All Login Card    │ SAISON VISA          [Card]  │
│ Note SSH          │                             │
│                   │ Cardholder  TARO YAMADA      │
│ > SAISON VISA Card│ Number      4111 **** 1111   │
│   Canva     Login │ Expiry      12/2028          │
│   R2 Token  Note  │ Code/PIN    ****             │
│   ...             │                             │
│                   │ ─ URIs ─                    │
├───────────────────┴─────────────────────────────┤
│ /search  1-5 filter  y copy  tab focus  q quit  │
└─────────────────────────────────────────────────┘
```

フォーカスは list と detail 間で Tab で切り替え。

---

## ディレクトリ構成

```
bitwarden_tui/
├── main.go                      # DI: rbw 実装を UI に注入
├── go.mod
├── internal/
│   ├── model/
│   │   └── item.go              # 共通データ型（全層参照可）
│   ├── repository/
│   │   ├── vault.go             # VaultRepository インターフェース（Port）
│   │   └── clipboard.go         # ClipboardRepository インターフェース（Port）
│   ├── infra/
│   │   ├── rbw/
│   │   │   └── client.go        # rbw による VaultRepository 実装（Adapter）
│   │   ├── copyq/
│   │   │   └── client.go        # copyq による ClipboardRepository 実装（hidden 対応）
│   │   └── clipboard/
│   │       └── fallback.go      # 標準クリップボード実装（機密コピー時に確認あり）
│   └── ui/
│       ├── app.go               # ルート bubbletea Model（VaultRepository を受け取る）
│       ├── list.go              # 左ペイン: bubbles/list
│       ├── detail.go            # 右ペイン: bubbles/viewport
│       ├── unlock.go            # アンロック画面
│       ├── keys.go              # キーバインド定義
│       └── style.go             # lipgloss スタイル定数
└── Makefile
```

### 依存の方向（Repository パターン）

```
ui ──→ repository.VaultRepository     (interface)
                   ↑
          infra/rbw/client.go          (実装)

ui ──→ repository.ClipboardRepository (interface)
                   ↑                ↑
    infra/copyq/client.go   infra/clipboard/fallback.go
```

- `ui` は両インターフェースのみに依存し、`infra/` 配下を直接参照しない
- `main.go` で実装を生成して `ui.NewApp(vaultRepo, clipRepo)` に渡す（DI）
- 将来 `bw` や mock に差し替える場合は `infra/` 以下に新実装を追加するだけでよい

---

## 依存パッケージ

```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles     # list / viewport / textinput
github.com/charmbracelet/lipgloss
github.com/atotto/clipboard          # infra/clipboard/fallback.go で使用
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

### Step 2 — VaultRepository インターフェース (`internal/repository/vault.go`)

```go
type VaultRepository interface {
    IsUnlocked() bool
    Unlock(password string) error
    List() ([]model.ListItem, error)
    GetDetail(id string) (*model.Item, error)
    Sync() error
}
```

### Step 3 — rbw クライアント (`internal/infra/rbw/client.go`)

| 関数 | rbw コマンド |
|------|------------|
| `IsUnlocked() bool` | `rbw unlocked` (exit code 0 = unlocked, non-0 = locked ※要確認) |
| `Unlock(password string) error` | `echo "<pw>" \| rbw unlock` |
| `List() ([]model.ListItem, error)` | `rbw list --raw` |
| `GetDetail(id string) (*model.Item, error)` | `rbw get <id> --raw` |
| `Sync() error` | `rbw sync` |

TOTP の現在コード（6桁）は `rbw code <id>` で取得し `LoginDetail.TOTP` に格納する。`rbw get --raw` の totp フィールドはシークレットキーなので使わない。
`GetDetail` は `rbw get <id> --raw` の name/folder/notes/data をすべてパースして `*model.Item` を返す。

`rbw list --raw` 実測レスポンス（各アイテム）:

```json
{ "id": "...", "name": "...", "user": "...", "folder": null, "uris": [...], "type": "Login" }
{ "id": "...", "name": "...", "user": null,  "folder": null, "uris": null,  "type": "Note" }
{ "id": "...", "name": "...", "user": null,  "folder": null, "uris": null,  "type": "Card" }
{ "id": "...", "name": "...", "user": null,  "folder": null, "uris": null,  "type": "SSH Key" }
```

`rbw get <id> --raw` 実測レスポンス（type 別）:

**Login**:

```json
{ "id": "...", "folder": null, "name": "...",
  "data": { "username": "...", "password": "...", "totp": null, "uris": [] },
  "fields": [], "notes": null, "history": [] }
```

**Note**:

```json
{ "id": "...", "folder": null, "name": "...",
  "data": null,
  "fields": [], "notes": "...", "history": [] }
```

**SSH Key**:

```json
{ "id": "...", "folder": null, "name": "...",
  "data": { "public_key": "...", "fingerprint": "...", "private_key": "..." },
  "fields": [], "notes": null, "history": [] }
```

**Card**:

```json
{ "id": "...", "folder": null, "name": "...",
  "data": { "cardholder_name": "...", "number": "...", "brand": "...", "exp_month": "...", "exp_year": "...", "code": "..." },
  "fields": [], "notes": null, "history": [] }
```

- `notes` はトップレベル（`data` の中ではない）、全タイプに存在（未設定時は `null`）
- Note のみ `data: null`（コンテンツはすべて `notes` に入る）
- type 文字列: `"Login"`, `"Note"`, `"Card"`, `"SSH Key"`（スペースあり）

### Step 4 — データモデル (`internal/model/item.go`)

```go
type ItemType string
const (
    TypeLogin    ItemType = "Login"
    TypeNote     ItemType = "Note"
    TypeCard     ItemType = "Card"
    TypeSSH      ItemType = "SSH Key"   // スペースあり
    TypeIdentity ItemType = "Identity"
)

type ListItem struct { ID, Name, Folder string; User string; Type ItemType }  // Folder: null → "" (未設定)、将来のフォルダナビ用に保持

// notes はトップレベルフィールドなので全タイプ共通で Item に持たせる（null = 未設定）
type LoginDetail    struct { Username, Password, TOTP string; URIs []string }
type CardDetail     struct { CardholderName, Number, Brand, ExpMonth, ExpYear, Code string }
type NoteDetail     struct{}  // コンテンツは Item.Notes のみ
type IdentityDetail struct { FullName, Email, Phone, Address string }
type SSHKeyDetail   struct { PrivateKey, PublicKey, Fingerprint string }
type Item struct { ListItem; Notes string; Detail any }
```

### Step 5 — UI コンポーネント

**`ui/app.go`** — ルートモデル

- `tea.Model` を実装
- 状態: `items []model.ListItem`, `selected *model.Item`, `focus` (left/right), `typeFilter ItemType`
- `Init()`: `VaultRepository.List()` を非同期 `tea.Cmd` で実行
- アイテム選択時に `VaultRepository.GetDetail()` を非同期で呼び出し

**`ui/list.go`** — 左ペイン

- `bubbles/list` ラッパー
- タイプ別フィルタ: `[All][Login][Card][Note][SSH]` を `1-5` で切り替え
- `/` でインクリメンタルサーチ（bubbles/list の FilteringEnabled）
- app.go の状態: `typeFilter ItemType`

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
1-5        タイプフィルタ切り替え（All/Login/Card/Note/SSH）
y          フィールド/行をクリップボードへコピー（マスク中も実値をコピー）
Y          Note全文コピー
space      フォーカス中フィールドのマスク/表示トグル
r          rbw sync 実行
q/Ctrl+C   終了
```

### Step 6 — アンロック画面 (`ui/unlock.go`)

- `bubbles/textinput`（EchoModePassword でマスク入力）
- Enter で `VaultRepository.Unlock(password)` を呼び出し、成功でメイン画面へ切り替え
- 失敗時はエラーメッセージ再表示

### Step 7 — main.go

```go
vaultRepo := rbw.NewClient()
clipRepo  := selectClipboard()  // copyq あり→copyq実装、なし→fallback
tea.NewProgram(ui.NewApp(vaultRepo, clipRepo)).Run()
```

`ui.NewApp()` 内で `vaultRepo.IsUnlocked()` を呼び、結果に応じて初期状態を `StateUnlock` or `StateMain` に設定。

---

## パスワード・機密フィールドの扱い

| フィールド | デフォルト表示 | キー操作 | コピー |
|-----------|--------------|---------|-------|
| Login: password | `••••••••` (マスク) | `space` で表示/非表示トグル | `y` で可 |
| Login: totp | `••••••` (マスク) | `space` でトグル | `y` で可 |
| Card: number | `**** **** **** 1111` (末4桁のみ) | `space` でフル表示トグル | `y` で可 |
| Card: code/PIN | `••••` | `space` でトグル | `y` で可 |
| SSH Key: private_key | `••••••••` (マスク) | `space` でトグル | `y` で可 |

- フィールドにフォーカスがある状態で `y` を押すと、**マスク中でも**実際の値をクリップボードへコピー
- クリップボードへのコピーは明示的なキー操作のみ（自動コピーなし）

### クリップボード実装（Repository パターンで抽象化）

クリップボード操作も差し替え可能にするため `repository/clipboard.go` にインターフェースを置く。

```go
// repository/clipboard.go
type ClipboardRepository interface {
    Copy(value string) error
    CopySensitive(value string) error  // 機密用（履歴で伏せ表示）
}
```

#### copyq が **ある** 場合（`infra/copyq/client.go`）

`application/x-copyq-hidden` を付与して書き込む（実測で動作確認済み）:

```bash
copyq write 0 "application/x-copyq-hidden" "1" "text/plain" "<value>"
```

#### copyq が **ない** 場合（`infra/clipboard/fallback.go`）

`atotto/clipboard` で標準クリップボードに書き込む。
ただし機密フィールドのコピー時は **確認ダイアログを表示**:

```
copyq が見つかりません。
クリップボード履歴に平文で保存されます。
続けますか？ [y/N]
```

`y` で確定コピー、それ以外でキャンセル。

#### 起動時の検出と DI

```go
// main.go
var clipRepo repository.ClipboardRepository
if isCopyqAvailable() {  // exec.LookPath("copyq") で判定
    clipRepo = copyq.NewClient()
} else {
    clipRepo = clipboard.NewFallback()
}
ui.NewApp(vaultRepo, clipRepo)
```

非機密フィールド（username, URI, Note 本文など）はどちらの実装でも確認なしでコピー。

---

## 起動フロー（マスターパスワード要求）

1. 起動時に `VaultRepository.IsUnlocked()` チェック
2. **ロック中** → TUI 内でパスワード入力プロンプト表示（bubbles/textinput、入力は `*` でマスク）
3. 入力確定で `VaultRepository.Unlock(password)` を呼び出し
4. アンロック成功 → メイン画面へ遷移
5. 失敗 → エラーメッセージ表示 → 再入力

---

## 対応アイテムタイプ（表示）

| タイプ | 表示フィールド | 機密フィールド（デフォルト伏せ） |
|--------|--------------|-------------------------------|
| Login | name, username, URIs, notes | password, totp |
| Card | name, cardholder_name, brand, exp_month/year, notes | number, code/PIN |
| Note | name, notes（行単位コピー対応） | — |
| SSH Key | name, public_key, fingerprint, notes | private_key |
| Identity | title, full_name, email, phone, address | — |

---

## 実装しない機能（今回）

- アイテムの追加・編集・削除
- フォルダーナビゲーション（将来: `f` キーで左ペインがフォルダ一覧に切り替わり `j/k` + `Enter` で選択する B 方式）

---

## 検証方法

1. `go build -o bwtui . && ./bwtui` で起動確認
2. `rbw unlocked` がロック状態のときにエラー終了するか確認
3. リスト表示 → タイプフィルタ切り替え → アイテム選択 → 詳細表示の一連フローを手動確認
4. Note アイテムで行選択コピー（`y`）が正しくクリップボードに入るか確認
5. Login アイテムのパスワードがデフォルトで `••••` になっていること、`space` で表示トグルできることを確認
6. パスワードコピー時に copyq 履歴で伏せ表示になっていることを確認
