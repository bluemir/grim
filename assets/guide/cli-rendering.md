# CLI 렌더링

grim은 브라우저 에디터뿐 아니라 명령줄 도구로도 동작합니다. `.grim` 파일을 SVG로 변환하거나, 로컬에서 직접 에디터 서버를 띄울 수 있습니다.

> 브라우저 에디터와 동일한 Go 코어를 사용하므로, CLI로 렌더링한 결과는 에디터 미리보기와 동일합니다.

---

## 바이너리 준비

배포된 바이너리가 없다면 소스에서 직접 빌드합니다.

```sh
# 저장소를 클론한 뒤 프로젝트 루트에서
make build
# → build/grim 생성

# 또는 Go 툴체인으로 직접
go build -o grim .
```

빌드된 `grim` 실행 파일로 아래 명령을 사용합니다.

---

## `grim render` — SVG로 변환

`.grim` 파일을 읽어 SVG로 렌더링합니다.

```sh
# stdout으로 출력
grim render diagram.grim

# 파일로 저장 (-o / --output)
grim render diagram.grim -o out.svg
grim render diagram.grim --output out.svg
```

| 인자 / 플래그 | 설명 |
|------|------|
| `<file>` | 입력 `.grim` 파일 경로 (필수) |
| `-o`, `--output` | 출력 파일 경로 (생략 시 stdout) |

출력은 표준 SVG이므로 파이프라인이나 다른 도구와 조합할 수 있습니다.

```sh
# 빌드 스크립트에서 다이어그램을 일괄 변환
for f in docs/*.grim; do
    grim render "$f" -o "${f%.grim}.svg"
done
```

---

## `grim server` — 로컬 에디터 서버

브라우저 에디터를 로컬에서 직접 실행합니다.

```sh
# 기본 :8080 포트로 실행
grim server

# 포트 변경
grim server --bind :3000
```

| 플래그 | 설명 | 기본값 |
|--------|------|--------|
| `--bind` | 서비스 HTTP 바인드 주소 | `:8080` |
| `-c`, `--config` | 설정 파일 경로 | — |
| `--db-path` | 저장소 DB 경로 | `:memory:` |
| `--cert`, `--key` | TLS 인증서 / 키 파일 | — |

실행 후 브라우저에서 `http://localhost:8080/` 으로 접속합니다.

---

## `grim guide` — 가이드 출력

이 가이드 문서를 터미널에서 확인합니다.

```sh
grim guide
```

---

## 전역 옵션

모든 서브커맨드와 함께 사용할 수 있습니다.

| 플래그 | 설명 |
|--------|------|
| `-v`, `--verbose` | 로그 레벨 증가 (여러 번 지정 가능: `-vvv`) |
| `--log-format` | 로그 형식 (`text`, `text-color`, `json`) |
| `--version` | 버전 정보 출력 |
| `--help` | 도움말 출력 |

---

## 다음 단계

- [기본 문법](syntax/basics) — `.grim` 파일에 작성할 노드·엣지 문법을 배웁니다.
- [다이어그램 공유](sharing) — 브라우저 에디터에서 URL로 공유하는 방법을 확인합니다.
