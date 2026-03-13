# ADR-002: AST 기반 .grim 코드 포매터

## Status
Accepted

## Date
2026-03-14

## Context
`.grim` 코드를 편집하다 보면 들여쓰기, 공백, 키 순서 등이 제각각이 된다.
`gofmt` / `prettier`처럼 코드를 일관된 형식으로 자동 정규화하는 도구가 필요하다.

포매터 구현 방식으로 두 가지를 검토했다:
1. **토큰 기반 (CST)**: 원본 토큰 스트림을 직접 가공하여 출력 — 주석·공백을 정확히 보존할 수 있으나 구현 복잡도가 높음
2. **AST 기반 (pretty-print)**: Parse() 결과 AST를 받아 직접 문자열로 재구성 — 구현이 단순하고 표준 라이브러리만 사용 가능

`.grim` AST는 `Comment` 노드를 포함한 완전한 구조를 가지며, 이미 충분한 정보를 담고 있어 AST 기반으로 충분히 구현 가능하다.

빈 줄 처리: `gofmt`처럼 빈 줄을 AST에 `BlankLine` statement로 보존한다.
top-level과 블록 내부 모두 추적하며, 연속 빈 줄은 하나로 collapse한다.

## Decision
**AST 기반 pretty-print 방식으로 Formatter를 구현한다.**

### 구현 위치
- `internal/parser/format.go` — `Format(doc *Document) string`
- 표준 라이브러리만 사용 (`strings`, `fmt`, `sort`)

### 포맷 규칙
- **들여쓰기**: 탭(`\t`) 1개 per depth
- **노드**: `nodeName` / `nodeName: "label"` / `nodeName {\n\t...\n}`
- **엣지**: `A -> B` / `A <- B` / `A <-> B` / `A -> B: "label"` (화살표 양쪽 공백)
- **블록 내 순서**: `@metadata` 먼저, 그 다음 child statements
- **주석**: `# ` + TrimLeft(text, " \t")
- **빈 줄**: `BlankLine` AST 노드로 보존 (top-level 및 블록 내부 모두), 연속 빈 줄은 하나로 collapse, 문서 앞뒤 trailing blank lines 제거
- **JSON 오브젝트 키 순서**: 인지하기 좋은 의미론적 순서 우선, 나머지 알파벳순
  - `@layout`: `x → y → w → h` 우선
  - `@style`: `font-color → font-size → fill → stroke` 우선
- **JSON 값 출력**:
  - 단순 식별자 문자열(letters/digits/hyphens/underscores): 따옴표 없이 출력 (`fill: blue`)
  - 공백·특수문자 포함 문자열: 따옴표 출력 (`"hello world"`)
  - 정수로 표현 가능한 float: 소수점 없이 출력 (`100` not `100.0`)

### AST 변경
- `BlankLine` statement 타입 추가 (`ast.go`)
- `stateTopLevel`에 `prevNewline bool` 필드 추가 (`state.go`)
- 연속 newline 감지 시 top-level 및 블록 내부 모두에서 `BlankLine` 추가

### WASM 노출
- `grimFormat(code: string) → string` JS 함수를 `wasm/wasm.go`에 추가
- 파싱 에러 시 원본 코드 반환 (best-effort 대신 안전성 우선)

### UI 통합
- 에디터 상단에 **Format 버튼** 추가
- 키보드 단축키: `Shift+Alt+F` (VS Code 표준)

## Consequences

### 장점
- 구현 단순: 기존 Parse() 재사용, 새 파일 1개 추가
- 멱등성 보장: 동일 AST → 동일 출력
- 표준 라이브러리만 사용 → WASM 호환성 유지
- 빈 줄 보존: top-level 및 블록 내부 구조적 그루핑 유지
- 향후 포맷 규칙 변경 시 format.go 한 파일만 수정

### 단점
- AST에 없는 원본 인라인 공백 정보는 소실됨 (동일 줄의 토큰 간 공백 개수 등)

### 제약사항
- 파싱 실패 시 포맷을 시도하지 않음 (부분 AST 포맷 결과가 더 혼란스러울 수 있음)
- `node: "label" { block }` 인라인 블록 문법은 파서가 미지원
