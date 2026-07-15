# Spec: scpick — クロスプラットフォーム対話型 SCP/SFTP ファイル転送ツール

## Objective

Windows（ネイティブ PowerShell、WSL/Git Bash 不要）と Linux の両方で、外部コマンド
（`ssh.exe` / `scp.exe` / `fzf.exe` など）に依存せず単一バイナリで動作する、対話的な
scp 代替ツールを作る。

`scpick` を実行すると、ローカルとリモートの2ペインを表示するファイラー画面が起動する。
左右のペインで自由にディレクトリを移動し、片方でヤンク(`y`)、もう片方にカーソルを
移して貼り付け(`p`)するだけで転送が完了する。方向（アップロード/ダウンロード）は
「どちらのペインでヤンクしたか」で自然に決まるため、サブコマンドによる方向指定は不要。

利用者は以下の手間から解放されることを目指す:

- 接続先ホスト・ユーザー名・リモートパス・ローカルパスを毎回手打ちする手間
- OS ごとに異なるツールセット（WSL, Git Bash, ネイティブ PowerShell）を用意する手間
- ファイル/ディレクトリの区別や複数選択状態が分かりにくいUI、キーバインドをカスタマイズ
  できないUIによる操作ミス

**ユーザー像**: SSH で複数サーバーを管理し、日常的にファイルを転送する開発者。
Windows 端末と Linux 端末の両方から同じ体験で操作したい。vim/nvim のキー操作に慣れている。

**成功の定義**: `scpick` を実行すると、ローカル/リモートの2ペインファイラーが起動し、
ホスト選択・ディレクトリ移動・複数選択・転送までをキーボードだけで完結できる。
パスを手でタイプする場面がなく、大量ファイル・深い階層でも動作が重くならない。

## Tech Stack

- 言語: Go 1.25 以上（2026年7月時点の最新安定版は 1.26 系）
- 主要ライブラリ（外部プロセス起動なし、すべて Go ライブラリとして組み込む）
  - `golang.org/x/crypto/ssh` — SSH プロトコル実装
  - `golang.org/x/crypto/ssh/agent` — ssh-agent 連携
  - `golang.org/x/crypto/ssh/knownhosts` — `~/.ssh/known_hosts` によるホスト鍵検証
  - `github.com/pkg/sftp` — SFTP によるファイル一覧取得・転送
  - `github.com/kevinburke/ssh_config` — `~/.ssh/config` の Host エントリ読み込み
  - `github.com/charmbracelet/bubbletea` — TUI本体（Elm的なUpdate/Viewループ、キー
    バインドは完全にカスタム定義）
  - `github.com/charmbracelet/lipgloss` — TUIのスタイリング（色・枠線・レイアウト）
  - `github.com/charmbracelet/bubbles` — `bubbletea`/`lipgloss` と同じ作者による
    部品集。`textinput`（手動ホスト入力・パスワード入力・フィルター入力の1行編集）
    のみ使用し、リスト表示やビューポートは自作する
  - `github.com/sahilm/fuzzy` — フィルターモード(`/`)でのインクリメンタルな
    あいまい検索スコアリング
  - `golang.org/x/term` — パスワードのマスク入力
- ビルド: `CGO_ENABLED=0` の静的バイナリ。Windows/Linux 双方をクロスコンパイルで生成する。

**変更履歴（2026-07-15）**: 当初 `github.com/ktr0731/go-fuzzyfinder` を採用していたが、
キーバインドをカスタマイズできない制約が判明したため、`bubbletea` + `lipgloss` による
自作デュアルペインTUIに変更した。これにより `pull` / `push` のサブコマンド分けも不要に
なった。実装時に手動入力・パスワード入力のテキスト編集を自前実装するコストが見合わない
と判断し、`bubbletea`/`lipgloss` と同じ作者の `github.com/charmbracelet/bubbles`
（`textinput` のみ）を追加した。

## Commands

```
# 依存取得
go mod tidy

# ビルド（現在の OS 向け）
go build -o bin/scpick ./cmd/scpick

# クロスコンパイル
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick.exe ./cmd/scpick
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o bin/scpick     ./cmd/scpick

# テスト
go test ./... -race -cover
go test -tags=integration ./...   # 実際の SSH/SFTP サーバーに繋ぐ統合テスト

# lint / format
gofmt -l .
go vet ./...
golangci-lint run ./...

# 起動
./bin/scpick        # ローカル/リモートの2ペインファイラーが起動
go run ./cmd/scpick  # 開発中の実行
```

## Project Structure

```
cmd/scpick/            → main関数、bubbletea.Program の起動のみ
internal/sshconf/      → ~/.ssh/config のパース、Host一覧の取得
internal/auth/         → 認証チェーン(agent→鍵→パスワード)とknown_hosts検証
internal/remotefs/     → SFTP経由のディレクトリ一覧・転送(pkg/sftpのラッパー)
internal/localfs/      → ローカルファイルシステムの1階層ずつの一覧取得(Windows/Linux差異を吸収)
internal/transfer/     → 転送実行(進捗レポート、キャンセル時の不完全ファイル削除、
                          同名ファイル衝突時の overwrite/skip 処理)
internal/tui/           → bubbletea の Model/Update/View 本体
  ├─ model.go           → 状態(paneState, model, mode enum, yankBuffer)の定義
  ├─ keymap.go          → ヘルプ画面用のキーバインド一覧定義
  ├─ entry.go           → paneEntry型、ローカル/リモート一覧の変換、fuzzy検索
  ├─ browse.go          → Browse(通常)モードの処理・カーソル移動・ヤンク
  ├─ filter.go          → フィルターモード(sahilm/fuzzy 利用)
  ├─ visual.go          → Visual選択モード(範囲選択・マーク)
  ├─ hostselect.go      → ホスト選択・手動入力・パスワード/ホスト鍵確認の画面
  ├─ connect.go         → 接続処理本体(goroutine+channelでの非同期実行)
  ├─ transfer_modal.go  → 転送確認モーダル・進捗表示(同じくgoroutine+channel)
  ├─ view.go            → lipglossによるレンダリング全般
  └─ help.go            → ヘルプ画面(`?`)
testdata/               → テスト用の固定データ(サンプルssh_config等)
*_test.go               → 各パッケージに同居（Go標準の配置）
```

## Code Style

Go の標準スタイル（`gofmt` 必須）に従う。エラーは握り潰さず、呼び出し元に伝わるよう
ラップする。bubbletea の `Update` はキー入力ごとの分岐が中心になるため、モードごとに
関数を分けて委譲する。

```go
// internal/tui/browse.go
func (m model) updateBrowse(msg tea.KeyMsg) (model, tea.Cmd) {
    switch msg.String() {
    case "j", "down":
        m.activePane().cursor = min(m.activePane().cursor+1, m.activePane().lastIndex())
    case "l", "enter":
        if e := m.activePane().current(); e.IsDir {
            return m, m.descend(e.Name)
        }
    case " ":
        m.activePane().toggleSelect()
    case "tab":
        m.focus = 1 - m.focus
    case "y":
        m.yankBuf = m.activePane().selectedOrCursor()
    case "p":
        if len(m.yankBuf) > 0 {
            m.mode = modeTransferConfirm
        }
    case "/":
        m.mode = modeFilter
    case "v":
        m.mode = modeVisual
    }
    return m, nil
}
```

- export される型・関数は CamelCase、パッケージ内部は camelCase
- パッケージ名は短い名詞（`transfer`, `remotefs` など）。`utils` や `common` のような
  汎用名は使わない
- `panic` は main の初期化失敗など、リカバリ不能な場合のみ許可。それ以外は `error` を返す

## Testing Strategy

- フレームワーク: 標準 `testing` パッケージ（追加の外部テストフレームワークは導入しない）
- 配置: 各パッケージ内に `_test.go` として同居
- テストレベル
  - **単体テスト**（`go test ./...` で常に実行）:
    - `internal/sshconf` のパース
    - `internal/localfs` / `internal/remotefs` のディレクトリ列挙
    - `internal/tui` の `Update` 関数群 — bubbletea の Model は素の構造体なので、
      `tea.KeyMsg{...}` を送って返ってきた `model` の状態(カーソル位置・選択集合・
      モード遷移)をアサートする形でテストできる。実端末は不要
    - `internal/tui/filter.go` の `sahilm/fuzzy` を使った絞り込みロジック
    - `internal/transfer` の衝突検出・overwrite/skip 分岐ロジック
  - **統合テスト**（`-tags=integration` で分離、CI 上で一時起動する SSH/SFTP サーバーに
    対して実行）: 認証チェーン全体、実ファイルの転送、転送キャンセル時に不完全ファイルが
    残らないことの確認
  - `View()` のレンダリング結果そのものは厳密な単体テスト対象にしないが、各モードで
    `View()` がパニックしないことをスモークテストする
- カバレッジ目標: `internal/tui`（Update系）と `internal/sshconf` は 70% 以上を目標。
  レンダリング(`View`)は数値目標を設けない

## Boundaries

- **Always**: コミット前に `gofmt` / `go vet` / `go test ./...` を通す。パスワードや
  パスフレーズを一切ログ・標準出力に出さない。デフォルトでは `known_hosts` による
  ホスト鍵検証を行う。転送キャンセル時は書きかけの転送先ファイルを削除する
- **Ask first**: Tech Stackに記載した以外の外部依存を追加する。`InsecureIgnoreHostKey`
  をデフォルト動作にする変更。主要キーバインド(`y`/`p`/`v`/`/`/`Tab`など)の割り当てを
  変更する。S3・FTP など対象プロトコルを追加する
- **Never**: 秘密鍵・パスワード・テスト用認証情報をリポジトリにコミットする。ホスト鍵の
  不一致を警告なしに通す。ユーザーの確認なしにリモート/ローカルのファイルを削除する
  （本ツールは転送専用で削除機能は持たない）。転送確認モーダルを経由せずに上書きする

## Success Criteria

- `GOOS=windows` と `GOOS=linux` それぞれで `CGO_ENABLED=0` の単一バイナリがビルドできる
- `scpick` を引数なしで実行すると、ローカル(即座に表示)/リモート(未接続プレースホルダー)
  の2ペインファイラーが起動する
- リモートペインで `C` を押すと、ssh_config一覧＋手動入力からホストを選び、
  agent→鍵→パスワード(マスク入力)の優先順で認証し、接続できる。既に接続済みの
  場合は現在の接続を切ってから繋ぎ直す(リモートペインは常に1接続のみ)
- 未知のホストは fingerprint を表示した信頼確認プロンプトを経て `known_hosts` に追記
  される。既知のホスト鍵と不一致の場合は接続を中断し、確認プロンプトなしでエラーになる
- `j`/`k`/`h`/`l` でのペイン内移動、`Tab` でのペイン切り替え、`Space`/`v` での複数選択、
  `/` でのインクリメンタルなあいまい絞り込みが、キー入力から遅延なく反映される
- `y` でヤンクした項目を別ペインで `p` すると転送確認モーダルが出て、同名ファイルが
  存在する場合は `o`(上書き) / `s`(スキップ) / `Esc`(中止) を選べる
- 転送中は進捗（ファイルごとのバイト数、全体の何件目か）が表示され、キャンセル時は
  書きかけの転送先ファイルが残らない
- 実行時に `ssh` / `scp` / `fzf` などの外部バイナリを一切必要としない
  （Windows のネイティブ環境で追加インストールなしに動く）
- 数百〜数千件のファイルがあるディレクトリでも、一覧取得は常に1階層分のみで、
  操作が体感的に重くならない

## Open Questions

- 接続先のブックマーク（よく使う host/path の保存）を持たせるか、`~/.ssh/config` の
  情報だけに頼るか
- `go.mod` のモジュールパス（公開する場合のリポジトリ URL など）は未確定のため
  プレースホルダーを使用している
- lint ツールとして `golangci-lint` を仮定したが、導入コストが問題であれば `go vet`
  のみに縮小してよいか
- 転送確認モーダルの `o`/`s` は「今回の貼り付け全体」に対する一括選択だが、
  項目ごとに個別選択したいケースが出てきたら拡張するか
