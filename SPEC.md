# Spec: scpick — クロスプラットフォーム対話型 SCP/SFTP ファイル転送ツール

## Objective

Windows（ネイティブ PowerShell、WSL/Git Bash 不要）と Linux の両方で、外部コマンド
（`ssh.exe` / `scp.exe` / `fzf.exe` など）に依存せず単一バイナリで動作する、対話的な
scp 代替ツールを作る。

利用者は以下の手間から解放されることを目指す:

- 接続先ホスト・ユーザー名・リモートパス・ローカルパスを毎回手打ちする手間
- OS ごとに異なるツールセット（WSL, Git Bash, ネイティブ PowerShell）を用意する手間

**ユーザー像**: SSH で複数サーバーを管理し、日常的にファイルを転送する開発者。
Windows 端末と Linux 端末の両方から同じ体験で操作したい。

**成功の定義**: `scpick pull` / `scpick push` を実行すると、ホスト選択 →
（リモート/ローカルの）パス選択 → 転送、という一連の操作をキーボードだけで完了できる。
パスを手でタイプする場面がない。

## Tech Stack

- 言語: Go 1.25 以上（2026年7月時点の最新安定版は 1.26 系）
- 主要ライブラリ（外部プロセス起動なし、すべて Go ライブラリとして組み込む）
  - `golang.org/x/crypto/ssh` — SSH プロトコル実装
  - `golang.org/x/crypto/ssh/agent` — ssh-agent 連携
  - `golang.org/x/crypto/ssh/knownhosts` — `~/.ssh/known_hosts` によるホスト鍵検証
  - `github.com/pkg/sftp` — SFTP によるファイル一覧取得・転送
  - `github.com/kevinburke/ssh_config` — `~/.ssh/config` の Host エントリ読み込み
  - `github.com/ktr0731/go-fuzzyfinder` — fzf 風の対話選択 UI（外部 fzf バイナリ不要）
  - `golang.org/x/term` — パスワードのマスク入力
  - `github.com/spf13/cobra` — サブコマンド・フラグ解析、`--help` の自動生成
- ビルド: `CGO_ENABLED=0` の静的バイナリ。Windows/Linux 双方をクロスコンパイルで生成する。

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

# 開発中の実行
go run ./cmd/scpick pull
go run ./cmd/scpick push
```

## Project Structure

```
cmd/scpick/            → main関数、サブコマンド(pull/push)のルーティングのみ
internal/sshconf/      → ~/.ssh/config のパース、Host一覧の取得
internal/auth/         → 認証チェーン(agent→鍵→パスワード)とknown_hosts検証
internal/picker/       → go-fuzzyfinderのラッパー。file-pickモード/dir-pickモードのリスト生成
internal/remotefs/     → SFTP経由のディレクトリ一覧・転送(pkg/sftpのラッパー)
internal/localfs/      → ローカルファイルシステムの1階層ずつの一覧取得(Windows/Linux差異を吸収)
internal/transfer/     → pull/pushそれぞれの転送フロー(上位層の組み立て役)
testdata/              → テスト用の固定データ(サンプルssh_config等)
*_test.go              → 各パッケージに同居（Go標準の配置）
```

## Code Style

Go の標準スタイル（`gofmt` 必須）に従う。エラーは握り潰さず、呼び出し元に伝わるよう
ラップする。

```go
// internal/remotefs/list.go
func (c *Client) ListDir(path string) ([]Entry, error) {
    infos, err := c.sftp.ReadDir(path)
    if err != nil {
        return nil, fmt.Errorf("remotefs: list %q: %w", path, err)
    }
    entries := make([]Entry, 0, len(infos))
    for _, fi := range infos {
        entries = append(entries, Entry{
            Name:  fi.Name(),
            IsDir: fi.IsDir(),
        })
    }
    return entries, nil
}
```

- export される型・関数は CamelCase、パッケージ内部は camelCase
- パッケージ名は短い名詞（`picker`, `remotefs` など）。`utils` や `common` のような
  汎用名は使わない
- `panic` は main の初期化失敗など、リカバリ不能な場合のみ許可。それ以外は `error` を返す

## Testing Strategy

- フレームワーク: 標準 `testing` パッケージ（追加の外部テストフレームワークは導入しない）
- 配置: 各パッケージ内に `_test.go` として同居
- テストレベル
  - **単体テスト**（`go test ./...` で常に実行）: `internal/sshconf` のパース、
    `internal/picker` のリスト生成ロジック（`..` や `★ ここを使う` の出し分けを含む）、
    `internal/localfs` のディレクトリ列挙
  - **統合テスト**（`-tags=integration` で分離、CI 上で一時起動する SSH/SFTP サーバーに
    対して実行）: 認証チェーン・pull/push 全体の E2E 確認
  - go-fuzzyfinder 自体の TUI 描画は自動テスト対象外とし、手動確認に留める
- カバレッジ目標: `internal/picker` と `internal/sshconf` は 70% 以上を目標。
  TUI/IO 境界に近い層は数値目標を設けない

## Boundaries

- **Always**: コミット前に `gofmt` / `go vet` / `go test ./...` を通す。パスワードや
  パスフレーズを一切ログ・標準出力に出さない。デフォルトでは `known_hosts` による
  ホスト鍵検証を行う
- **Ask first**: 上記リスト以外の外部依存を追加する。`InsecureIgnoreHostKey` を
  デフォルト動作にする変更。サブコマンド名を変更する、または `-r`/`--recursive`
  以外のフラグを追加する。S3・FTP など対象プロトコルを追加する
- **Never**: 秘密鍵・パスワード・テスト用認証情報をリポジトリにコミットする。
  ホスト鍵の不一致を警告なしに通す。ユーザーの確認なしにリモート/ローカルのファイルを
  削除する（本ツールは転送専用で削除機能は持たない）

## Success Criteria

- `GOOS=windows` と `GOOS=linux` それぞれで `CGO_ENABLED=0` の単一バイナリがビルドできる
- `scpick pull` で「ホスト選択（ssh_config 一覧 or 手動入力）→ 認証
  （agent → 鍵 → パスワードの優先順）→ リモートのファイルを1階層ずつ辿って選択 →
  ローカルの保存先ディレクトリを1階層ずつ辿って選択 → 転送」が完了する
- `scpick push` で上記の逆方向（ローカルファイル選択 → リモート保存先選択 → 転送）が
  完了する
- パスワード入力時、画面に入力文字が表示されない
- 未知のホストに接続する際、fingerprint を表示した上で信頼するかどうかを確認する
  プロンプトが出る。既知のホスト鍵と一致しない場合は接続を中断し、明確なエラーを表示する
- 実行時に `ssh` / `scp` / `fzf` などの外部バイナリを一切必要としない
  （Windows のネイティブ環境で追加インストールなしに動く）

## Open Questions

- CLI のバイナリ名・サブコマンド名（本 spec では `scpick` / `pull` / `push` と仮定）は
  確定でよいか？
- 複数ファイルの同時選択・転送は初期スコープに含めるか？（本 spec では1回の実行=1ファイル
  と仮定）
- 転送先に同名ファイルが既に存在する場合の挙動（上書き確認 / 自動リネーム / エラー終了）は
  未定
- 接続先のブックマーク（よく使う host/path の保存）を持たせるか、`~/.ssh/config` の情報
  だけに頼るか
- 大きいファイル転送時の進捗表示（プログレスバーの有無・形式）
- Windows でルートより上に辿ろうとした場合の挙動（ドライブレター一覧を出すか等）
- `go.mod` のモジュールパス（公開する場合のリポジトリ URL など）は未確定のため
  プレースホルダーを使用している
- lint ツールとして `golangci-lint` を仮定したが、導入コストが問題であれば `go vet`
  のみに縮小してよいか
